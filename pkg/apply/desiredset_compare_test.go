package apply

import (
	"bytes"
	"testing"

	data2 "github.com/rancher/wrangler/v3/pkg/data"
	"github.com/rancher/wrangler/v3/pkg/data/convert"
	patch2 "github.com/rancher/wrangler/v3/pkg/patch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
)

func TestCompressAndEncode(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "Empty string",
			input:    []byte(""),
			expected: "H4sIAAAAAAAA/wEAAP//AAAAAAAAAAA",
		},
		{
			name:     "Short string",
			input:    []byte("hello world"),
			expected: "H4sIAAAAAAAA/8pIzcnJVyjPL8pJAQQAAP//hRFKDQsAAAA",
		},
		{
			name:     "JSON payload",
			input:    []byte(`{"id": 123, "status": "active", "message": "hello"}`),
			expected: "H4sIAAAAAAAA/6pWykxRslIwNDLWUVAqLkksKS1WslJQSkwuySxLVdJRUMpNLS5OTE8FCWak5uTkK9UCAgAA//9XG2xwMwAAAA",
		},
		{
			name:     "Longer repeating string",
			input:    bytes.Repeat([]byte("test data "), 10),
			expected: "H4sIAAAAAAAA/ypJLS5RSEksSVSgHQsQAAD//02/IfBkAAAA",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Compare the result against our hardcoded golden value
			if got, want := compressAndEncode(tc.input), tc.expected; got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

var nullSafeGVK = schema.GroupVersionKind{Group: "rke.cattle.io", Version: "v1", Kind: "RKEControlPlane"}

// spec builds an object of a kind absent from the client-go scheme, which is
// what makes apply reach for a merge patch.
func spec(gvk schema.GroupVersionKind, spec map[string]interface{}) *unstructured.Unstructured {
	apiVersion, kind := gvk.ToAPIVersionAndKind()
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name":      "cluster",
				"namespace": "fleet-default",
			},
			"spec": spec,
		},
	}
}

func chartValues(gvk schema.GroupVersionKind, values map[string]interface{}) *unstructured.Unstructured {
	return spec(gvk, map[string]interface{}{"chartValues": values})
}

// applied returns obj as the server would hold it after apply created it,
// carrying the annotation the next patch is generated against.
func applied(t *testing.T, gvk schema.GroupVersionKind, obj *unstructured.Unstructured) runtime.Object {
	t.Helper()
	prepared, err := prepareObjectForCreate(gvk, obj)
	require.NoError(t, err)
	return prepared
}

// patchedChartValues applies patch to current the way the apiserver would and
// returns the resulting spec.chartValues, along with whether "b" is present at
// all -- a removal and an assignment of null both decode to nil.
func patchedChartValues(t *testing.T, current runtime.Object, patch []byte) (map[string]interface{}, bool) {
	t.Helper()

	currentJSON, err := json.Marshal(current)
	require.NoError(t, err)

	result, err := patch2.Apply(currentJSON, patch)
	require.NoError(t, err)

	obj := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(result, &obj))

	values := convert.ToMapInterface(data2.GetValueN(obj, "spec", "chartValues"))
	_, ok := values["b"]

	return values, ok
}

// capturePatch runs applyPatch against a patcher that records what it was
// handed rather than sending it.
func capturePatch(t *testing.T, gvk schema.GroupVersionKind, nullSafe bool, current, desired runtime.Object, diffPatches ...[]byte) (types.PatchType, []byte, bool) {
	t.Helper()

	var (
		gotType  types.PatchType
		gotPatch []byte
	)

	ran, err := applyPatch(gvk, nil, func(namespace, name string, pt types.PatchType, data []byte) (runtime.Object, error) {
		gotType, gotPatch = pt, data
		return nil, nil
	}, "test", false, nullSafe, current, desired, diffPatches)
	require.NoError(t, err)

	return gotType, gotPatch, ran
}

func TestApplyPatchNullSafe(t *testing.T) {
	// The desired state removes a chart default by setting it to null, which is
	// the case a merge patch cannot express.
	current := applied(t, nullSafeGVK, chartValues(nullSafeGVK, map[string]interface{}{"a": "1", "b": "2"}))
	desired := chartValues(nullSafeGVK, map[string]interface{}{"a": "1", "b": nil})

	t.Run("merge patch drops the null", func(t *testing.T) {
		patchType, patch, ran := capturePatch(t, nullSafeGVK, false, current, desired)
		require.True(t, ran)
		assert.Equal(t, types.MergePatchType, patchType)

		values, hasB := patchedChartValues(t, current, patch)
		assert.False(t, hasB, "merge patch removed the key instead of setting it to null")
		assert.Equal(t, "1", values["a"])
	})

	t.Run("null safe patch keeps the null", func(t *testing.T) {
		patchType, patch, ran := capturePatch(t, nullSafeGVK, true, current, desired)
		require.True(t, ran)
		assert.Equal(t, types.JSONPatchType, patchType)

		values, hasB := patchedChartValues(t, current, patch)
		assert.True(t, hasB, "the key is gone, so the null was not applied")
		assert.Nil(t, values["b"])
		assert.Equal(t, "1", values["a"])
	})

	t.Run("null safe patch is equivalent when no null is involved", func(t *testing.T) {
		desired := chartValues(nullSafeGVK, map[string]interface{}{"a": "2", "b": "2"})

		mergeType, mergePatch, ran := capturePatch(t, nullSafeGVK, false, current, desired)
		require.True(t, ran)
		require.Equal(t, types.MergePatchType, mergeType)

		jsonType, jsonPatch, ran := capturePatch(t, nullSafeGVK, true, current, desired)
		require.True(t, ran)
		require.Equal(t, types.JSONPatchType, jsonType)

		merged, err := patch2.Apply(mustMarshal(t, current), mergePatch)
		require.NoError(t, err)
		patched, err := patch2.Apply(mustMarshal(t, current), jsonPatch)
		require.NoError(t, err)

		assert.JSONEq(t, string(merged), string(patched))
	})

	t.Run("no change sends nothing", func(t *testing.T) {
		_, _, ran := capturePatch(t, nullSafeGVK, true, current, chartValues(nullSafeGVK, map[string]interface{}{"a": "1", "b": "2"}))
		assert.False(t, ran)
	})

	t.Run("an ignored field is not reinstated by a wholesale add", func(t *testing.T) {
		// The subtree is missing from current, so it is set from modified in one
		// operation -- which has to be the modified the merge patch was
		// generated from, with the ignored field already taken out of it.
		current := applied(t, nullSafeGVK, spec(nullSafeGVK, map[string]interface{}{}))
		desired := chartValues(nullSafeGVK, map[string]interface{}{"a": "1", "ignored": "x"})
		ignore := []byte(`[{"op":"remove","path":"/spec/chartValues/ignored"}]`)

		_, patch, ran := capturePatch(t, nullSafeGVK, true, current, desired, ignore)
		require.True(t, ran)

		values, _ := patchedChartValues(t, current, patch)
		assert.Equal(t, map[string]interface{}{"a": "1"}, values)
	})

	t.Run("strategic merge is left alone", func(t *testing.T) {
		// Strategic merge has its own directives for a null assignment, and it
		// is only reachable for types in the client-go scheme.
		gvk := schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}
		current := applied(t, gvk, configMap(gvk, map[string]interface{}{"a": "1"}))
		desired := configMap(gvk, map[string]interface{}{"a": "2"})

		patchType, _, ran := capturePatch(t, gvk, true, current, desired)
		require.True(t, ran)
		assert.Equal(t, types.StrategicMergePatchType, patchType)
	})
}

func configMap(gvk schema.GroupVersionKind, data map[string]interface{}) *unstructured.Unstructured {
	apiVersion, kind := gvk.ToAPIVersionAndKind()
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name":      "config",
				"namespace": "default",
			},
			"data": data,
		},
	}
}

func mustMarshal(t *testing.T, obj runtime.Object) []byte {
	t.Helper()
	b, err := json.Marshal(obj)
	require.NoError(t, err)
	return b
}

func TestSanitizePatchJSONPatch(t *testing.T) {
	tests := []struct {
		name                      string
		patch                     string
		removeObjectSetAnnotation bool
		expected                  string
	}{
		{
			name:     "a json patch is passed through untouched",
			patch:    `[{"op":"add","path":"/spec/chartValues/b","value":null}]`,
			expected: `[{"op":"add","path":"/spec/chartValues/b","value":null}]`,
		},
		{
			name:                      "the applied annotation is dropped from a plan",
			patch:                     `[{"op":"add","path":"/metadata/annotations/objectset.rio.cattle.io~1applied","value":"x"},{"op":"add","path":"/spec/a","value":"1"}]`,
			removeObjectSetAnnotation: true,
			expected:                  `[{"op":"add","path":"/spec/a","value":"1"}]`,
		},
		{
			name:                      "a patch carrying only the applied annotation is empty",
			patch:                     `[{"op":"add","path":"/metadata/annotations/objectset.rio.cattle.io~1applied","value":"x"}]`,
			removeObjectSetAnnotation: true,
			expected:                  `[]`,
		},
		{
			name: "a whole annotations map is left alone",
			// Only reachable when apply adopts an object that has no
			// annotations at all, where a plan naming the annotation is a
			// cosmetic wart rather than a wrong plan.
			patch:                     `[{"op":"add","path":"/metadata/annotations","value":{"objectset.rio.cattle.io/applied":"x"}}]`,
			removeObjectSetAnnotation: true,
			expected:                  `[{"op":"add","path":"/metadata/annotations","value":{"objectset.rio.cattle.io/applied":"x"}}]`,
		},
		{
			name:                      "a removal of the applied annotation is dropped",
			patch:                     `[{"op":"remove","path":"/metadata/annotations/objectset.rio.cattle.io~1applied"}]`,
			removeObjectSetAnnotation: true,
			expected:                  `[]`,
		},
		{
			name:                      "an unrelated annotation is kept",
			patch:                     `[{"op":"add","path":"/metadata/annotations/other.io~1thing","value":"x"}]`,
			removeObjectSetAnnotation: true,
			expected:                  `[{"op":"add","path":"/metadata/annotations/other.io~1thing","value":"x"}]`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := sanitizePatch([]byte(test.patch), test.removeObjectSetAnnotation)
			require.NoError(t, err)
			assert.Equal(t, test.expected, string(got))
		})
	}
}
