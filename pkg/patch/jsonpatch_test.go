package patch

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEscapePointerToken(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		escaped string
	}{
		{name: "no reserved character", token: "chartValues", escaped: "chartValues"},
		{name: "slash", token: "objectset.rio.cattle.io/applied", escaped: "objectset.rio.cattle.io~1applied"},
		{name: "tilde", token: "a~b", escaped: "a~0b"},
		{name: "tilde before one", token: "a~1b", escaped: "a~01b"},
		{name: "both", token: "a~/b", escaped: "a~0~1b"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.escaped, EscapePointerToken(test.token))
		})
	}
}

func TestIsJSONPatch(t *testing.T) {
	assert.True(t, IsJSONPatch([]byte(`[{"op":"remove","path":"/spec"}]`)))
	assert.False(t, IsJSONPatch([]byte(`{"spec":null}`)))
	assert.False(t, IsJSONPatch(nil))
}

func TestCreateJSONPatchFromMergePatch(t *testing.T) {
	tests := []struct {
		name string
		// mergePatch is what apply sends today, and the three-way merge that
		// produced it is what loses the null.
		mergePatch string
		modified   string
		current    string
		// expected is the translated patch, whose operations are sorted by path
		// within each object.
		expected string
		// patched is current with expected applied, and is what the caller
		// actually asked for.
		patched string
	}{
		{
			name:       "explicit null is assigned rather than removed",
			mergePatch: `{"spec":{"chartValues":{"b":null}}}`,
			modified:   `{"spec":{"chartValues":{"a":"1","b":null}}}`,
			current:    `{"spec":{"chartValues":{"a":"1","b":"2"}}}`,
			expected:   `[{"op":"add","path":"/spec/chartValues/b","value":null}]`,
			patched:    `{"spec":{"chartValues":{"a":"1","b":null}}}`,
		},
		{
			name:       "null for a key absent from modified is a removal",
			mergePatch: `{"spec":{"chartValues":{"b":null}}}`,
			modified:   `{"spec":{"chartValues":{"a":"1"}}}`,
			current:    `{"spec":{"chartValues":{"a":"1","b":"2"}}}`,
			expected:   `[{"op":"remove","path":"/spec/chartValues/b"}]`,
			patched:    `{"spec":{"chartValues":{"a":"1"}}}`,
		},
		{
			name: "removal of a key absent from current emits nothing",
			// RFC 6902 fails the whole patch on a remove of a missing path,
			// where the merge patch it came from was a no-op.
			mergePatch: `{"spec":{"chartValues":{"b":null}}}`,
			modified:   `{"spec":{"chartValues":{"a":"1"}}}`,
			current:    `{"spec":{"chartValues":{"a":"1"}}}`,
			expected:   `[]`,
			patched:    `{"spec":{"chartValues":{"a":"1"}}}`,
		},
		{
			name:       "empty merge patch",
			mergePatch: `{}`,
			modified:   `{"spec":{"chartValues":{"a":"1"}}}`,
			current:    `{"spec":{"chartValues":{"a":"1"}}}`,
			expected:   `[]`,
			patched:    `{"spec":{"chartValues":{"a":"1"}}}`,
		},
		{
			name:       "pointer tokens are escaped",
			mergePatch: `{"metadata":{"annotations":{"a/b":"x","c~d":null}}}`,
			modified:   `{"metadata":{"annotations":{"a/b":"x","c~d":null}}}`,
			current:    `{"metadata":{"annotations":{"a/b":"1","c~d":"2"}}}`,
			expected:   `[{"op":"add","path":"/metadata/annotations/a~1b","value":"x"},{"op":"add","path":"/metadata/annotations/c~0d","value":null}]`,
			patched:    `{"metadata":{"annotations":{"a/b":"x","c~d":null}}}`,
		},
		{
			name:       "arrays are replaced wholesale",
			mergePatch: `{"spec":{"list":[3]}}`,
			modified:   `{"spec":{"list":[3]}}`,
			current:    `{"spec":{"list":[1,2,3]}}`,
			expected:   `[{"op":"add","path":"/spec/list","value":[3]}]`,
			patched:    `{"spec":{"list":[3]}}`,
		},
		{
			name: "subtree missing from current is added wholesale with its nulls",
			// The parent of every emitted path has to exist, so the walk stops
			// at the deepest object current holds.
			mergePatch: `{"spec":{"chartValues":{"a":null}}}`,
			modified:   `{"spec":{"chartValues":{"a":null}}}`,
			current:    `{"spec":{}}`,
			expected:   `[{"op":"add","path":"/spec/chartValues","value":{"a":null}}]`,
			patched:    `{"spec":{"chartValues":{"a":null}}}`,
		},
		{
			name: "stale removal inside a new subtree is dropped",
			// The subtree value comes from modified, not from the merge patch,
			// so a removal of a key that never reaches the server is not
			// written out as a null.
			mergePatch: `{"spec":{"chartValues":{"a":"1","b":null}}}`,
			modified:   `{"spec":{"chartValues":{"a":"1"}}}`,
			current:    `{"spec":{}}`,
			expected:   `[{"op":"add","path":"/spec/chartValues","value":{"a":"1"}}]`,
			patched:    `{"spec":{"chartValues":{"a":"1"}}}`,
		},
		{
			name:       "object replacing a scalar is set wholesale",
			mergePatch: `{"spec":{"x":{"y":1}}}`,
			modified:   `{"spec":{"x":{"y":1}}}`,
			current:    `{"spec":{"x":"str"}}`,
			expected:   `[{"op":"add","path":"/spec/x","value":{"y":1}}]`,
			patched:    `{"spec":{"x":{"y":1}}}`,
		},
		{
			name:       "operations are emitted in a stable order",
			mergePatch: `{"spec":{"c":"3","a":"1","b":null},"metadata":{"name":"n"}}`,
			modified:   `{"metadata":{"name":"n"},"spec":{"a":"1","b":null,"c":"3"}}`,
			current:    `{"metadata":{"name":"o"},"spec":{"a":"0","b":"2","c":"0"}}`,
			expected: `[{"op":"add","path":"/metadata/name","value":"n"},` +
				`{"op":"add","path":"/spec/a","value":"1"},` +
				`{"op":"add","path":"/spec/b","value":null},` +
				`{"op":"add","path":"/spec/c","value":"3"}]`,
			patched: `{"metadata":{"name":"n"},"spec":{"a":"1","b":null,"c":"3"}}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := CreateJSONPatchFromMergePatch([]byte(test.mergePatch), []byte(test.modified), []byte(test.current))
			require.NoError(t, err)
			assert.Equal(t, test.expected, string(got))

			if string(got) == "[]" {
				// An empty patch is never sent, and evanphx rejects it.
				assert.JSONEq(t, test.patched, test.current)
				return
			}

			// Apply routes on the leading "[", so this exercises the same path
			// the apiserver takes.
			patched, err := Apply([]byte(test.current), got)
			require.NoError(t, err)
			assert.JSONEq(t, test.patched, string(patched))

			// JSONEq compares decoded values, where a null member and an absent
			// one are both nil, so the distinction this whole translation
			// exists for needs checking on the bytes.
			assertSameNullKeys(t, test.patched, string(patched))
		})
	}
}

func TestCreateJSONPatchFromMergePatchInvalidInput(t *testing.T) {
	valid := []byte(`{"spec":{}}`)

	_, err := CreateJSONPatchFromMergePatch([]byte(`not json`), valid, valid)
	assert.ErrorContains(t, err, "unmarshalling merge patch")

	_, err = CreateJSONPatchFromMergePatch(valid, []byte(`not json`), valid)
	assert.ErrorContains(t, err, "unmarshalling modified")

	_, err = CreateJSONPatchFromMergePatch(valid, valid, []byte(`not json`))
	assert.ErrorContains(t, err, "unmarshalling current")
}

// assertSameNullKeys checks that the two documents hold explicit nulls at the
// same paths, which is the difference assert.JSONEq cannot see.
func assertSameNullKeys(t *testing.T, want, got string) {
	t.Helper()
	assert.Equal(t, nullPaths(t, want), nullPaths(t, got), "explicit nulls differ")
}

func nullPaths(t *testing.T, doc string) []string {
	t.Helper()

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(doc), &data))

	paths := []string{}
	var walk func(node map[string]interface{}, path string)
	walk = func(node map[string]interface{}, path string) {
		for k, v := range node {
			p := path + "/" + k
			switch child := v.(type) {
			case nil:
				paths = append(paths, p)
			case map[string]interface{}:
				walk(child, p)
			}
		}
	}
	walk(data, "")
	sort.Strings(paths)

	return paths
}
