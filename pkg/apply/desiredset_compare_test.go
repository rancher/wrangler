package apply

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
)

func Test_doPatchJSONMergePatch3way(t *testing.T) {
	type args struct {
		gvk      schema.GroupVersionKind
		original []byte
		modified []byte
		current  []byte
	}
	tests := []struct {
		name      string
		args      args
		patchType types.PatchType
		patch     []byte
		wantErr   assert.ErrorAssertionFunc
	}{
		// 3-way JSON Merge Patch
		// def: a 3-way patch from a "modified" object to a "current" object assuming an "original" common ancestor is:
		//    - a 2-way patch from current to modified without deletions, merged with
		//    - a 2-way patch from original to modified with deletions only
		// expected behavior:
		//  - no change between original, modified and current: nothing to do
		//  - changed non-objects (primitive types and arrays) from modified to current: replace value
		//  - changed objects from modified to current:
		//    - if a key is in both modified and current but the corresponding value changed: replace value
		//    - if a key is in modified but not current: add key
		//    - if a key is not in modified but is in current:
		//		- if the key is in original: remove key
		//		- if the key is not in original: nothing to do
		{
			name: "3wayEmptyMapNoChangesThenDoNothing",
			args: args{
				gvk:      testCRDGVK,
				original: toTestCRDBytes(map[string]any{}, t),
				modified: toTestCRDBytes(map[string]any{}, t),
				current:  toTestCRDBytes(map[string]any{}, t),
			},
			patchType: types.MergePatchType,
			patch:     []byte("{}"),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayFullMapNoChangesThenDoNothing",
			args: args{
				gvk:      testCRDGVK,
				original: toTestCRDBytes(map[string]any{"one": "one"}, t),
				modified: toTestCRDBytes(map[string]any{"one": "one"}, t),
				current:  toTestCRDBytes(map[string]any{"one": "one"}, t),
			},
			patchType: types.MergePatchType,
			patch:     []byte("{}"),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayPrimitiveChangedThenReplaceValue",
			args: args{
				gvk:      testCRDGVK,
				original: toTestCRDBytes(map[string]any{"one": "one"}, t),
				modified: toTestCRDBytes(map[string]any{"one": "one"}, t),
				current:  toTestCRDBytes(map[string]any{"one": "two"}, t),
			},
			patchType: types.MergePatchType,
			patch:     []byte(`{"data":{"one":"one"}}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayArrayChangedThenReplaceValue",
			args: args{
				gvk:      testCRDGVK,
				original: toTestCRDBytes(map[string]any{"one": []string{"one"}}, t),
				modified: toTestCRDBytes(map[string]any{"one": []string{"one"}}, t),
				current:  toTestCRDBytes(map[string]any{"one": []string{"two", "three"}}, t),
			},
			patchType: types.MergePatchType,
			patch:     []byte(`{"data":{"one":["one"]}}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayObjectKeyInModifiedAndInCurrentThenReplaceValue",
			args: args{
				gvk:      testCRDGVK,
				original: toTestCRDBytes(map[string]any{}, t),
				modified: toTestCRDBytes(map[string]any{"one": "one"}, t),
				current:  toTestCRDBytes(map[string]any{"one": "two"}, t),
			},
			patchType: types.MergePatchType,
			patch:     []byte(`{"data":{"one":"one"}}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayObjectKeyInModifiedAndNotInCurrentThenAddKey",
			args: args{
				gvk:      testCRDGVK,
				original: toTestCRDBytes(map[string]any{"one": "one"}, t),
				modified: toTestCRDBytes(map[string]any{"one": "one"}, t),
				current:  toTestCRDBytes(map[string]any{}, t),
			},
			patchType: types.MergePatchType,
			patch:     []byte(`{"data":{"one":"one"}}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayObjectKeyNotInModifiedAndInCurrentAndInOriginalThenRemoveKey",
			args: args{
				gvk:      testCRDGVK,
				original: toTestCRDBytes(map[string]any{"one": "one"}, t),
				modified: toTestCRDBytes(map[string]any{}, t),
				current:  toTestCRDBytes(map[string]any{"one": "one"}, t),
			},
			patchType: types.MergePatchType,
			patch:     []byte(`{"data":{"one":null}}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayObjectKeyNotInModifiedAndInCurrentAndNotInOriginalThenDoNothing",
			args: args{
				gvk:      testCRDGVK,
				original: toTestCRDBytes(map[string]any{}, t),
				modified: toTestCRDBytes(map[string]any{}, t),
				current:  toTestCRDBytes(map[string]any{"one": "one"}, t),
			},
			patchType: types.MergePatchType,
			patch:     []byte(`{}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayNullNotInOriginalNotInModifiedInCurrentThenDoNothing",
			args: args{
				gvk:      testCRDGVK,
				original: toTestCRDBytes(map[string]any{}, t),
				modified: toTestCRDBytes(map[string]any{}, t),
				current:  toTestCRDBytes(map[string]any{"a": nil}, t),
			},
			patchType: types.MergePatchType,
			patch:     []byte(`{}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayNullNotInOriginalInModifiedNotInCurrentThenNeedlesslyDelete",
			args: args{
				gvk:      testCRDGVK,
				original: toTestCRDBytes(map[string]any{}, t),
				modified: toTestCRDBytes(map[string]any{"a": nil}, t),
				current:  toTestCRDBytes(map[string]any{}, t),
			},
			patchType: types.MergePatchType,
			// This is not really wanted, but tolerated.
			// The patch deletes an "a" field that does not actually exist in current, it will do no harm but makes
			// the patch non-empty, which could lead to apply cycles in controllers.
			// OTOH, the JSON Merge Patch RFC explicitly states it
			// is not meant to be used with null values:
			// "This design means that merge patch documents are suitable for
			//   describing modifications to JSON documents that primarily use objects
			//   for their structure and do not make use of explicit null values.  The
			//   merge patch format is not appropriate for all JSON syntaxes."
			// from: https://tools.ietf.org/html/rfc7386#section-1
			// Therefore this is undefined behavior, so it can't be treated as a bug.
			patch:   []byte(`{"data":{"a":null}}`),
			wantErr: assert.NoError,
		},
		{
			name: "3wayNullNotInOriginalInModifiedInCurrentThenNeedlesslyDelete",
			args: args{
				gvk:      testCRDGVK,
				original: toTestCRDBytes(map[string]any{}, t),
				modified: toTestCRDBytes(map[string]any{"a": nil}, t),
				current:  toTestCRDBytes(map[string]any{"a": nil}, t),
			},
			patchType: types.MergePatchType,
			// This is not really wanted, but tolerated.
			// The patch deletes an "a" field that exists in current and should be left alone.
			// Note also that Strategic Merge Patch does not do this, it correctly does nothing.
			// OTOH, the JSON Merge Patch RFC explicitly states it
			// is not meant to be used with null values:
			// "This design means that merge patch documents are suitable for
			//   describing modifications to JSON documents that primarily use objects
			//   for their structure and do not make use of explicit null values.  The
			//   merge patch format is not appropriate for all JSON syntaxes."
			// from: https://tools.ietf.org/html/rfc7386#section-1
			// Therefore this is undefined behavior, so it can't be treated as a bug.
			patch:   []byte(`{"data":{"a":null}}`),
			wantErr: assert.NoError,
		},
		{
			name: "3wayNullInOriginaNotlInModifiedNotInCurrentThenNeedlesslyDelete",
			args: args{
				gvk:      testCRDGVK,
				original: toTestCRDBytes(map[string]any{"a": nil}, t),
				modified: toTestCRDBytes(map[string]any{}, t),
				current:  toTestCRDBytes(map[string]any{}, t),
			},
			patchType: types.MergePatchType,
			// This is not really wanted, but tolerated.
			// The patch deletes an "a" field that does not actually exist in current, it will do no harm but makes
			// the patch non-empty, which could lead to apply cycles in controllers.
			// OTOH, the JSON Merge Patch RFC explicitly states it
			// is not meant to be used with null values:
			// "This design means that merge patch documents are suitable for
			//   describing modifications to JSON documents that primarily use objects
			//   for their structure and do not make use of explicit null values.  The
			//   merge patch format is not appropriate for all JSON syntaxes."
			// from: https://tools.ietf.org/html/rfc7386#section-1
			// Therefore this is undefined behavior, so it can't be treated as a bug.
			patch:   []byte(`{"data":{"a":null}}`),
			wantErr: assert.NoError,
		},
		{
			name: "3wayNullInOriginalNotInModifiedInCurrentThenDelete",
			args: args{
				gvk:      testCRDGVK,
				original: toTestCRDBytes(map[string]any{"a": nil}, t),
				modified: toTestCRDBytes(map[string]any{}, t),
				current:  toTestCRDBytes(map[string]any{"a": nil}, t),
			},
			patchType: types.MergePatchType,
			patch:     []byte(`{"data":{"a":null}}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayNullInOriginalInModifiedNotInCurrentThenDoNothing",
			args: args{
				gvk:      testCRDGVK,
				original: toTestCRDBytes(map[string]any{"a": nil}, t),
				modified: toTestCRDBytes(map[string]any{"a": nil}, t),
				current:  toTestCRDBytes(map[string]any{}, t),
			},
			patchType: types.MergePatchType,
			// This is not really wanted, but tolerated.
			// The patch ignores an "a" field instead of creating one in current.
			// Note also that Strategic Merge Patch does not do this, it deletes the non-existing "a" field instead,
			// which is also incorrect.
			// OTOH, the JSON Merge Patch RFC explicitly states it
			// is not meant to be used with null values:
			// "This design means that merge patch documents are suitable for
			//   describing modifications to JSON documents that primarily use objects
			//   for their structure and do not make use of explicit null values.  The
			//   merge patch format is not appropriate for all JSON syntaxes."
			// from: https://tools.ietf.org/html/rfc7386#section-1
			// Therefore this is undefined behavior, so it can't be treated as a bug.
			patch:   []byte(`{}`),
			wantErr: assert.NoError,
		},
		{
			name: "3wayNullInOriginalInModifiedInCurrentThenDoNothing",
			args: args{
				gvk:      testCRDGVK,
				original: toTestCRDBytes(map[string]any{"a": nil}, t),
				modified: toTestCRDBytes(map[string]any{"a": nil}, t),
				current:  toTestCRDBytes(map[string]any{"a": nil}, t),
			},
			patchType: types.MergePatchType,
			patch:     []byte(`{}`),
			wantErr:   assert.NoError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			patchType, patch, err := doPatch(tt.args.gvk, tt.args.original, tt.args.modified, tt.args.current, [][]byte{})
			if !tt.wantErr(t, err, fmt.Sprintf("doPatch(%v, %v, %v, %v)", tt.args.gvk, tt.args.original, tt.args.modified, tt.args.current)) {
				return
			}
			assert.Equalf(t, tt.patchType, patchType, "doPatch(%v, %v, %v, %v)", tt.args.gvk, tt.args.original, tt.args.modified, tt.args.current)
			assert.Equalf(t, string(tt.patch), string(patch), "doPatch(%v, %v, %v, %v)", tt.args.gvk, tt.args.original, tt.args.modified, tt.args.current)
		})
	}
}

func Test_doPatchStrategicMergePatch3way(t *testing.T) {
	type args struct {
		gvk      schema.GroupVersionKind
		original []byte
		modified []byte
		current  []byte
	}
	tests := []struct {
		name      string
		args      args
		patchType types.PatchType
		patch     []byte
		wantErr   assert.ErrorAssertionFunc
	}{
		// 3-way Strategic Merge Patch
		// def: a 3-way patch from a "modified" object to a "current" object assuming an "original" common ancestor is:
		//    - a 2-way patch from current to modified without deletions, merged with
		//    - a 2-way patch from original to modified with deletions only
		// expected behavior:
		//  - no change between original, modified and current: nothing to do
		//  - changed non-objects (primitive types and arrays) from modified to current: replace value
		//  - changed objects from modified to current:
		//    - if a key is in both modified and current but the corresponding value changed: replace value
		//    - if a key is in modified but not current: add key
		//    - if a key is not in modified but is in current:
		//		- if the key is in original: remove key
		//		- if the key is not in original: nothing to do
		//  - if a patchStrategy tag is defined, it should be honored
		{
			name: "3wayEmptyMapNoChangesThenDoNothing",
			args: args{
				gvk:      configMapGVK,
				original: toConfigMapBytes(map[string]string{}, t),
				modified: toConfigMapBytes(map[string]string{}, t),
				current:  toConfigMapBytes(map[string]string{}, t),
			},
			patchType: types.StrategicMergePatchType,
			patch:     []byte("{}"),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayFullMapNoChangesThenDoNothing",
			args: args{
				gvk:      configMapGVK,
				original: toConfigMapBytes(map[string]string{"one": "one"}, t),
				modified: toConfigMapBytes(map[string]string{"one": "one"}, t),
				current:  toConfigMapBytes(map[string]string{"one": "one"}, t),
			},
			patchType: types.StrategicMergePatchType,
			patch:     []byte("{}"),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayPrimitiveChangedThenReplaceValue",
			args: args{
				gvk:      configMapGVK,
				original: toConfigMapBytes(map[string]string{"one": "one"}, t),
				modified: toConfigMapBytes(map[string]string{"one": "one"}, t),
				current:  toConfigMapBytes(map[string]string{"one": "two"}, t),
			},
			patchType: types.StrategicMergePatchType,
			patch:     []byte(`{"data":{"one":"one"}}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayArrayChangedThenReplaceValue",
			args: args{
				gvk:      configMapGVK,
				original: toConfigMapBinaryDataBytes(map[string][]byte{"one": {1}}, t),
				modified: toConfigMapBinaryDataBytes(map[string][]byte{"one": {1}}, t),
				current:  toConfigMapBinaryDataBytes(map[string][]byte{"one": {2, 3}}, t),
			},
			patchType: types.StrategicMergePatchType,
			patch:     []byte(`{"binaryData":{"one":"AQ=="}}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayObjectKeyInModifiedAndInCurrentThenReplaceValue",
			args: args{
				gvk:      configMapGVK,
				original: toConfigMapBytes(map[string]string{}, t),
				modified: toConfigMapBytes(map[string]string{"one": "one"}, t),
				current:  toConfigMapBytes(map[string]string{"one": "two"}, t),
			},
			patchType: types.StrategicMergePatchType,
			patch:     []byte(`{"data":{"one":"one"}}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayObjectKeyInModifiedAndNotInCurrentThenAddKey",
			args: args{
				gvk:      configMapGVK,
				original: toConfigMapBytes(map[string]string{"one": "one"}, t),
				modified: toConfigMapBytes(map[string]string{"one": "one"}, t),
				current:  toConfigMapBytes(map[string]string{}, t),
			},
			patchType: types.StrategicMergePatchType,
			patch:     []byte(`{"data":{"one":"one"}}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayObjectKeyNotInModifiedAndInCurrentAndInOriginalThenRemoveKey",
			args: args{
				gvk:      configMapGVK,
				original: toConfigMapBytes(map[string]string{"one": "one"}, t),
				modified: toConfigMapBytes(map[string]string{}, t),
				current:  toConfigMapBytes(map[string]string{"one": "one"}, t),
			},
			patchType: types.StrategicMergePatchType,
			patch:     []byte(`{"data":null}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayObjectKeyNotInModifiedAndInCurrentAndNotInOriginalThenDoNothing",
			args: args{
				gvk:      configMapGVK,
				original: toConfigMapBytes(map[string]string{}, t),
				modified: toConfigMapBytes(map[string]string{}, t),
				current:  toConfigMapBytes(map[string]string{"one": "one"}, t),
			},
			patchType: types.StrategicMergePatchType,
			patch:     []byte(`{}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayPatchStrategyDefinedThenHonorIt",
			args: args{
				gvk: podGVK,
				original: toPodBytes([]v1.Volume{
					{Name: "one", VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "can be lost"}}},
				}, t),
				modified: toPodBytes([]v1.Volume{
					{Name: "two"},
					{Name: "three", VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "I am new"}}},
				}, t),
				current: toPodBytes([]v1.Volume{
					{Name: "two", VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "do not lose me"}}},
					{Name: "four"},
				}, t),
			},
			patchType: types.StrategicMergePatchType,
			patch:     []byte(`{"spec":{"$setElementOrder/volumes":[{"name":"two"},{"name":"three"}],"volumes":[{"$retainKeys":["name"],"name":"two"},{"hostPath":{"path":"I am new"},"name":"three"},{"$patch":"delete","name":"one"}]}}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayNullNotInOriginalNotInModifiedInCurrentThenDoNothing",
			args: args{
				gvk:      configMapGVK,
				original: toConfigMapBinaryDataBytes(map[string][]byte{}, t),
				modified: toConfigMapBinaryDataBytes(map[string][]byte{}, t),
				current:  toConfigMapBinaryDataBytes(map[string][]byte{"a": nil}, t),
			},
			patchType: types.StrategicMergePatchType,
			patch:     []byte(`{}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayNullNotInOriginalInModifiedNotInCurrentThenNeedlesslyDelete",
			args: args{
				gvk:      configMapGVK,
				original: toConfigMapBinaryDataBytes(map[string][]byte{}, t),
				modified: toConfigMapBinaryDataBytes(map[string][]byte{"a": nil}, t),
				current:  toConfigMapBinaryDataBytes(map[string][]byte{}, t),
			},
			patchType: types.StrategicMergePatchType,
			// This is not really wanted, but tolerated.
			// The patch deletes an "a" field that does not actually exist in current, it will do no harm but makes
			// the patch non-empty, which could lead to apply cycles in controllers.
			// OTOH, the JSON Merge Patch RFC, which is the basis of Strategic Merge Patch, explicitly states it
			// is not meant to be used with null values:
			// "This design means that merge patch documents are suitable for
			//   describing modifications to JSON documents that primarily use objects
			//   for their structure and do not make use of explicit null values.  The
			//   merge patch format is not appropriate for all JSON syntaxes."
			// from: https://tools.ietf.org/html/rfc7386#section-1
			// Therefore this is undefined behavior, so it can't be treated as a bug.
			patch:   []byte(`{"binaryData":{"a":null}}`),
			wantErr: assert.NoError,
		},
		{
			name: "3wayNullNotInOriginalInModifiedInCurrentThenDoNothing",
			args: args{
				gvk:      configMapGVK,
				original: toConfigMapBinaryDataBytes(map[string][]byte{}, t),
				modified: toConfigMapBinaryDataBytes(map[string][]byte{"a": nil}, t),
				current:  toConfigMapBinaryDataBytes(map[string][]byte{"a": nil}, t),
			},
			patchType: types.StrategicMergePatchType,
			patch:     []byte(`{}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayNullInOriginaNotlInModifiedNotInCurrentThenNeedlesslyDelete",
			args: args{
				gvk:      configMapGVK,
				original: toConfigMapBinaryDataBytes(map[string][]byte{"a": nil}, t),
				modified: toConfigMapBinaryDataBytes(map[string][]byte{}, t),
				current:  toConfigMapBinaryDataBytes(map[string][]byte{}, t),
			},
			patchType: types.StrategicMergePatchType,
			// This is not really wanted, but tolerated.
			// The patch deletes a "binaryData" field that does not actually exist in current, it will do no harm but makes
			// the patch non-empty, which could lead to apply cycles in controllers.
			// OTOH, the JSON Merge Patch RFC, which is the basis of Strategic Merge Patch, explicitly states it
			// is not meant to be used with null values:
			// "This design means that merge patch documents are suitable for
			//   describing modifications to JSON documents that primarily use objects
			//   for their structure and do not make use of explicit null values.  The
			//   merge patch format is not appropriate for all JSON syntaxes."
			// from: https://tools.ietf.org/html/rfc7386#section-1
			// Therefore this is undefined behavior, so it can't be treated as a bug.
			patch:   []byte(`{"binaryData":null}`),
			wantErr: assert.NoError,
		},
		{
			name: "3wayNullInOriginalNotInModifiedInCurrentThenDelete",
			args: args{
				gvk:      configMapGVK,
				original: toConfigMapBinaryDataBytes(map[string][]byte{"a": nil}, t),
				modified: toConfigMapBinaryDataBytes(map[string][]byte{}, t),
				current:  toConfigMapBinaryDataBytes(map[string][]byte{"a": nil}, t),
			},
			patchType: types.StrategicMergePatchType,
			patch:     []byte(`{"binaryData":null}`),
			wantErr:   assert.NoError,
		},
		{
			name: "3wayNullInOriginalInModifiedNotInCurrentThenDelete",
			args: args{
				gvk:      configMapGVK,
				original: toConfigMapBinaryDataBytes(map[string][]byte{"a": nil}, t),
				modified: toConfigMapBinaryDataBytes(map[string][]byte{"a": nil}, t),
				current:  toConfigMapBinaryDataBytes(map[string][]byte{}, t),
			},
			patchType: types.StrategicMergePatchType,
			// This is not really wanted, but tolerated.
			// The patch deletes an "a" field instead of creating one in current.
			// OTOH, the JSON Merge Patch RFC, which is the basis of Strategic Merge Patch, explicitly states it
			// is not meant to be used with null values:
			// "This design means that merge patch documents are suitable for
			//   describing modifications to JSON documents that primarily use objects
			//   for their structure and do not make use of explicit null values.  The
			//   merge patch format is not appropriate for all JSON syntaxes."
			// from: https://tools.ietf.org/html/rfc7386#section-1
			// Therefore this is undefined behavior, so it can't be treated as a bug.
			patch:   []byte(`{"binaryData":{"a":null}}`),
			wantErr: assert.NoError,
		},
		{
			name: "3wayNullInOriginalInModifiedInCurrentThenDoNothing",
			args: args{
				gvk:      configMapGVK,
				original: toConfigMapBinaryDataBytes(map[string][]byte{"a": nil}, t),
				modified: toConfigMapBinaryDataBytes(map[string][]byte{"a": nil}, t),
				current:  toConfigMapBinaryDataBytes(map[string][]byte{"a": nil}, t),
			},
			patchType: types.StrategicMergePatchType,
			patch:     []byte(`{}`),
			wantErr:   assert.NoError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			patchType, patch, err := doPatch(tt.args.gvk, tt.args.original, tt.args.modified, tt.args.current, [][]byte{})
			if !tt.wantErr(t, err, fmt.Sprintf("doPatch(%v, %v, %v, %v)", tt.args.gvk, tt.args.original, tt.args.modified, tt.args.current)) {
				return
			}
			assert.Equalf(t, tt.patchType, patchType, "doPatch(%v, %v, %v, %v)", tt.args.gvk, tt.args.original, tt.args.modified, tt.args.current)
			assert.Equalf(t, string(tt.patch), string(patch), "doPatch(%v, %v, %v, %v)", tt.args.gvk, tt.args.original, tt.args.modified, tt.args.current)
		})
	}
}

func Test_sanitizePatch(t *testing.T) {
	type args struct {
		patch                     []byte
		removeObjectSetAnnotation bool
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "EmptyPatch",
			args: args{
				patch:                     []byte(`{}`),
				removeObjectSetAnnotation: false,
			},
			want:    []byte(`{}`),
			wantErr: assert.NoError,
		},
		{
			name: "UnexpectedType",
			args: args{
				patch:                     []byte(`{1: "one"}`),
				removeObjectSetAnnotation: false,
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "RemoveUnwantedFields",
			args: args{
				patch:                     []byte(`{"kind": "patched", "apiVersion": "patched", "status": "patched", "metadata": {"creationTimestamp": "patched", "preserve": "this"}, "preserve": "this too"}`),
				removeObjectSetAnnotation: false,
			},
			want:    []byte(`{"metadata":{"preserve":"this"},"preserve":"this too"}`),
			wantErr: assert.NoError,
		},
		{
			name: "RemoveObjectSetAnnotation",
			args: args{
				patch:                     []byte(`{"metadata": {"annotations": {"objectset.rio.cattle.io/test": "delete me"}}}`),
				removeObjectSetAnnotation: true,
			},
			want:    []byte(`{}`),
			wantErr: assert.NoError,
		},
		{
			name: "DoNotRemoveObjectSetAnnotation",
			args: args{
				patch:                     []byte(`{"metadata": {"annotations": {"objectset.rio.cattle.io/test": "do not delete me"}}}`),
				removeObjectSetAnnotation: false,
			},
			want:    []byte(`{"metadata": {"annotations": {"objectset.rio.cattle.io/test": "do not delete me"}}}`),
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sanitizePatch(tt.args.patch, tt.args.removeObjectSetAnnotation)
			if !tt.wantErr(t, err, fmt.Sprintf("sanitizePatch(%v, %v)", tt.args.patch, tt.args.removeObjectSetAnnotation)) {
				return
			}
			assert.Equalf(t, string(tt.want), string(got), "sanitizePatch(%v, %v)", tt.args.patch, tt.args.removeObjectSetAnnotation)
		})
	}
}

// Utilities

// testCRDGVK is the GVK of a CustomResourceDefinition, which uses MergePatchType (because it is not registered)
var testCRDGVK = schema.GroupVersionKind{
	Group:   "cattle.io",
	Version: "v1",
	Kind:    "TestCRD",
}

// toTestCRDBytes converts a map to a serialized TestCRD
func toTestCRDBytes(data map[string]any, t *testing.T) []byte {
	t.Helper()
	obj := map[string]any{
		"data": data,
	}
	return toBytes(obj, t)
}

// toBytes converts an object to serialized JSON
func toBytes(obj any, t *testing.T) []byte {
	t.Helper()
	res, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("failed to marshal %v: %v", obj, err)
	}
	return res
}

// configMapGVK is the GVK of a ConfigMap, which uses StrategicMergePatchType
var configMapGVK = schema.GroupVersionKind{
	Group:   "",
	Version: "v1",
	Kind:    "ConfigMap",
}

// toConfigMapBytes converts a map to a serialized ConfigMap using the Data field
func toConfigMapBytes(data map[string]string, t *testing.T) []byte {
	t.Helper()
	obj := v1.ConfigMap{
		Data: data,
	}
	return toBytes(obj, t)
}

// toConfigMapBinaryDataBytes converts a map to a serialized ConfigMap using the BinaryData field
func toConfigMapBinaryDataBytes(data map[string][]byte, t *testing.T) []byte {
	t.Helper()
	obj := v1.ConfigMap{
		BinaryData: data,
	}
	return toBytes(obj, t)
}

// podGVK is the GVK of a Pod, which uses StrategicMergePatchType and has a patchStrategy annotation on Volumes
var podGVK = schema.GroupVersionKind{
	Group:   "",
	Version: "v1",
	Kind:    "Pod",
}

// toPodBytes converts a list of volumes to a serialized Pod
func toPodBytes(volumes []v1.Volume, t *testing.T) []byte {
	t.Helper()
	obj := v1.Pod{
		Spec: v1.PodSpec{
			Volumes: volumes,
		},
	}
	return toBytes(obj, t)
}
