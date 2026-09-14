package patch

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// pointerEscaper implements RFC 6901 JSON Pointer token escaping. "~" is listed
// first so that the "0" it introduces is not itself rewritten.
var pointerEscaper = strings.NewReplacer("~", "~0", "/", "~1")

// EscapePointerToken escapes a single JSON Pointer reference token per RFC 6901.
func EscapePointerToken(token string) string {
	return pointerEscaper.Replace(token)
}

// IsJSONPatch reports whether the given bytes are an RFC 6902 JSON Patch, which
// is a list, as opposed to a merge patch, which is an object.
func IsJSONPatch(patch []byte) bool {
	return isJSONPatch(patch)
}

// Operation is a single RFC 6902 JSON Patch operation.
//
// Value is a pointer so that a literal null can be distinguished from an absent
// value: "remove" carries no value member, while assigning null requires one.
type Operation struct {
	Op    string           `json:"op"`
	Path  string           `json:"path"`
	Value *json.RawMessage `json:"value,omitempty"`
}

func addOp(path string, v interface{}) (Operation, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return Operation{}, fmt.Errorf("marshalling value for %s: %w", path, err)
	}
	raw := json.RawMessage(b)
	// RFC 6902 "add" on an object member creates the member or replaces its
	// value, so it is correct whether or not the key already exists. "replace"
	// would fail on a missing target.
	return Operation{Op: "add", Path: path, Value: &raw}, nil
}

// CreateJSONPatchFromMergePatch translates an RFC 7386 JSON merge patch into an
// equivalent RFC 6902 JSON Patch.
//
// A null in a merge patch is ambiguous. RFC 7386 defines it as an instruction to
// remove a key, so a caller cannot use one to set a value to null -- which makes
// an explicit null inexpressible against any resource patched with
// types.MergePatchType, and that is every CRD. RFC 6902 draws the distinction
// the merge format cannot: "remove" and "add" with a null value are different
// operations.
//
// The ambiguity is resolved against modified, the desired object the merge patch
// was generated from. A null at a path that is explicitly null in modified is an
// assignment; a null at a path absent from modified is a removal.
//
// current, the live object, is used to drop removals of paths that do not exist,
// because RFC 6902 requires "remove" to target an existing path and fails the
// entire patch otherwise.
//
// Everything the merge patch would have done is preserved. Arrays are treated as
// opaque leaves and replaced wholesale, matching merge patch semantics and
// keeping positional index operations -- and the races that come with them --
// out of the result.
func CreateJSONPatchFromMergePatch(mergePatch, modified, current []byte) ([]byte, error) {
	var mp, mod, cur map[string]interface{}
	if err := json.Unmarshal(mergePatch, &mp); err != nil {
		return nil, fmt.Errorf("unmarshalling merge patch: %w", err)
	}
	if err := json.Unmarshal(modified, &mod); err != nil {
		return nil, fmt.Errorf("unmarshalling modified: %w", err)
	}
	if err := json.Unmarshal(current, &cur); err != nil {
		return nil, fmt.Errorf("unmarshalling current: %w", err)
	}

	ops := []Operation{}

	var walk func(patch, modNode, curNode map[string]interface{}, path string) error
	walk = func(patch, modNode, curNode map[string]interface{}, path string) error {
		// Sorted so that the generated patch is deterministic, which keeps debug
		// logs and tests stable. Ordering is not required for correctness: an
		// operation is emitted either for a node or for its descendants, never
		// for both, so no two operations overlap.
		keys := make([]string, 0, len(patch))
		for k := range patch {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := patch[k]
			p := path + "/" + EscapePointerToken(k)
			modChild, modHas := modNode[k]
			curChild, curHas := curNode[k]

			if v == nil {
				switch {
				case modHas && modChild == nil:
					// The desired object asks for a literal null. This is the
					// case a merge patch cannot express.
					op, err := addOp(p, nil)
					if err != nil {
						return err
					}
					ops = append(ops, op)
				case curHas:
					ops = append(ops, Operation{Op: "remove", Path: p})
				}
				// A removal of something that is not in current is a no-op and
				// would fail the patch, so nothing is emitted.
				continue
			}

			if patchChild, ok := v.(map[string]interface{}); ok {
				curMap, curIsMap := curChild.(map[string]interface{})
				modMap, modIsMap := modChild.(map[string]interface{})
				if curHas && curIsMap && modIsMap {
					// Recursing only where current already holds an object
					// guarantees the parent of every emitted path exists, which
					// is what RFC 6902 "add" requires.
					if err := walk(patchChild, modMap, curMap, p); err != nil {
						return err
					}
					continue
				}

				// Nothing mergeable underneath, so the subtree is set wholesale.
				// The value is taken from modified rather than from the patch so
				// that any nulls it carries are assignments, not stale removals
				// against a path that does not exist.
				value := interface{}(patchChild)
				if modHas {
					value = modChild
				}
				op, err := addOp(p, value)
				if err != nil {
					return err
				}
				ops = append(ops, op)
				continue
			}

			op, err := addOp(p, v)
			if err != nil {
				return err
			}
			ops = append(ops, op)
		}
		return nil
	}

	if err := walk(mp, mod, cur, ""); err != nil {
		return nil, err
	}

	return json.Marshal(ops)
}
