package data

import (
	"fmt"

	"golang.org/x/exp/slices"
)

// A Field represents the path to the current field in an unstructured object
type Field struct {
	// Name is the name of this field in the parent object - only usable for map entries
	Name *string

	// ListIndex is the index of this object in the parent array - only usable for array entries
	ListIndex *int

	// SubField is the "next field" in the path to a leaf node.
	SubField *Field

	// root is the parent object that starts the chain to a leaf field. Should not be directly set by a caller.
	root *Field
}

func NewNameField(name string) *Field {
	return &Field{
		Name: &name,
	}
}

func NewIndexField(index int) *Field {
	return &Field{
		ListIndex: &index,
	}
}

// Child adds a child field (e.x. entry in a map) to the current field and returns the new field. Can return nil
// if it encounters a cycle (f1.SubField == f2, f2.SubField == f1) in f.
func (f *Field) Child(subfield string) *Field {
	if f == nil {
		return nil
	}
	child := &Field{
		Name: &subfield,
	}
	return f.copyTreeToChild(child)
}

// Index adds the index of a child field of an array to the current field and returns the new field. Can return nil
// if it encounters a cycle (f1.SubField == f2, f2.SubField == f1) in f
func (f *Field) Index(index int) *Field {
	if f == nil {
		return nil
	}
	child := &Field{
		ListIndex: &index,
	}
	return f.copyTreeToChild(child)
}

// copyTreeToChild copies the call tree of f into child, with new memory allocated for each entry. This is done
// so that f is not mutated by a child. Returns the mutated child, or nil if it encounters a cycle
func (f *Field) copyTreeToChild(child *Field) *Field {
	var current *Field
	if f.root == nil {
		current = f.Copy()
		child.root = current
	} else {
		current = f.Root().Copy()
		// we may encounter a cycle, but don't want to loop forever, so exit with
		// nil in that case
		seen := map[*Field]struct{}{}
		for current != f && current != nil {
			if _, ok := seen[current]; ok {
				return nil
			}
			seen[current] = struct{}{}
			newCurrent := current.Copy()
			current = newCurrent.SubField
		}
		if current == nil {
			// this inidcates that we found a broken link in the path from
			// root to current. Exit with nil
			return nil
		}
		child.root = current.root
	}
	current.SubField = child
	return child
}

// Copy produces a copy of the current field. Does not recursively copy root or subfield, only a shallow copy
func (f *Field) Copy() *Field {
	if f == nil {
		return nil
	}
	return &Field{
		Name:      f.Name,
		ListIndex: f.ListIndex,
		SubField:  f.SubField,
		root:      f.root,
	}
}

// Root returns the root field, so that a caller can traverse the full path in an object to a given field
func (f *Field) Root() *Field {
	return f.root
}

// GetField gets the value for field from data. Returns the retrieved value, and an error (which may be a dataError).
// Field must be a valid leaf node (no subfield) and data must be a map[string]any or []any
func GetField(data any, field *Field) (any, error) {
	recurser := getRecurser{}
	_, err := runRecurse(data, field, &recurser)
	if err != nil {
		return nil, err
	}
	return recurser.found, nil
}

// RemoveField removes field from data. Returns (in order) the modified data, the removed value, and an error (which
// may be a data error). Field must be a valid leaf node (no subfield) and data must be a map[string]any or []any
func RemoveField(data any, field *Field) (any, any, error) {
	recurser := removeRecurser{}
	newData, err := runRecurse(data, field, &recurser)
	if err != nil {
		return nil, nil, err
	}
	return newData, recurser.removed, nil
}

// PutField puts value into data that the specified field. Returns the modified data and an error (which may be
// a data error). Field must be a valid leaf node (no subfield) and data must be a map[string]any or []any. If
// field (or any parent field of field) is missing, a default value will be initialized.
func PutField(data any, field *Field, value any) (any, error) {
	recurser := putRecurser{
		putValue: value,
	}
	newData, err := runRecurse(data, field, &recurser)
	if err != nil {
		return nil, err
	}
	return newData, nil
}

// runRecurse runs the recurse operation on data for field using recurser. Returns the modified data and
// (optionally) an error
func runRecurse(data any, field *Field, recurser onCaseRecurser) (any, error) {
	// if we aren't a top-level node, recurse to the start of the path
	start := field

	if field.Root() != nil {
		start = field.Root()
	}
	newData, err := recurseInternal(data, start, field, map[*Field]struct{}{}, recurser)
	if err != nil {
		return nil, err
	}
	return newData, nil
}

type getRecurser struct {
	found any
}

func (g *getRecurser) onFound(data any, field *Field, value any) (any, error) {
	g.found = value
	return data, nil
}

func (g *getRecurser) onMissing(data any, field *Field) (any, error) {
	message := fmt.Sprintf("field %v not found in data %v", field, data)
	return nil, newDataError(message, errorFieldValueNotFound)
}

type removeRecurser struct {
	removed any
}

func (r *removeRecurser) onFound(data any, field *Field, value any) (any, error) {
	r.removed = value
	if field.Name != nil {
		mapData := data.(map[string]any)
		delete(mapData, *field.Name)
		return mapData, nil
	}
	sliceData := data.([]any)
	newData := slices.Delete(sliceData, *field.ListIndex, *field.ListIndex+1)
	return newData, nil
}

func (g *removeRecurser) onMissing(data any, field *Field) (any, error) {
	message := fmt.Sprintf("field %v not found in data %v", field, data)
	return nil, newDataError(message, errorFieldValueNotFound)
}

type putRecurser struct {
	putValue any
}

func (p *putRecurser) onFound(data any, field *Field, value any) (any, error) {
	if field.Name != nil {
		mapData := data.(map[string]any)
		mapData[*field.Name] = p.putValue
		return mapData, nil
	}
	sliceData := data.([]any)
	sliceData[*field.ListIndex] = p.putValue
	return sliceData, nil
}

func (p *putRecurser) onMissing(data any, field *Field) (any, error) {
	if field.Name != nil {
		mapData := data.(map[string]any)
		// if we have a subfield, we need to create a matching sub-entry in data
		if field.SubField != nil {
			if err := validateFieldNameIndex(field.SubField); err != nil {
				return nil, err
			}
			// name subFields are for a map, so init the default value as a map; not necessary to add the
			// specific value since this function will be called on the next recursion
			if field.SubField.Name != nil {
				mapData[*field.Name] = map[string]any{}
			} else {
				mapData[*field.Name] = []any{}
			}
		}
		return mapData, nil
	}
	sliceData := data.([]any)
	for len(sliceData) <= *field.ListIndex {
		sliceData = append(sliceData, nil)
	}
	if field.SubField != nil {
		if err := validateFieldNameIndex(field.SubField); err != nil {
			return nil, err
		}
		// name subFields are for a map, so init the default value as a map; not necessary to add the
		// specific value since this function will be called on the next recursion
		if field.SubField.Name != nil {
			sliceData[*field.ListIndex] = map[string]any{}
		} else {
			sliceData[*field.ListIndex] = []any{}
		}
	}
	return sliceData, nil
}

// onCaseRecurser is an interface providing methods to manipulate data when a leaf case (i.e. missing or found
// resource) is identified
type onCaseRecurser interface {
	// onFound identifies what to do with a found value. Must return the modified data as a returned value, and optionally
	// an error if the data/value/field could not be processed
	onFound(data any, field *Field, value any) (any, error)
	// onMissing identifies what to do if a field is determined to be missing. Must return the modified data as a returned
	// value, and optionally an error if the data/field could not be processed
	onMissing(data any, field *Field) (any, error)
}

// recurseInternal recurses through data to field, and calls recurser.OnFound when the value is found. If a field is
// not found at any point, recurser.OnMissing is called. When onFound is called, it will return the result immediately.
// When onMissing is called, it will return an error result, but will continue on if no error was returned, after
// updating data with the returned value. Returns the modified data, and optionally, an error.
func recurseInternal(data any, current *Field, want *Field, seen map[*Field]struct{}, recurser onCaseRecurser) (any, error) {
	if err := validateFieldNameIndex(current); err != nil {
		return nil, err
	}
	_, isSeen := seen[current]
	if isSeen {
		message := fmt.Sprintf("cycle detected, %v was already processed", current)
		return nil, newDataError(message, errorInvalidField)
	}
	seen[current] = struct{}{}
	mapData, ok := data.(map[string]any)
	if ok {
		return recurseInternalMap(mapData, current, want, seen, recurser)
	}
	sliceData, ok := data.([]any)
	if !ok {
		message := fmt.Sprintf("data %v was not a map[string]any or a []any, cannot process", data)
		return nil, newDataError(message, errorInvalidData)
	}
	return recurseInternalSlice(sliceData, current, want, seen, recurser)
}

func recurseInternalMap(data map[string]any, current *Field, want *Field, seen map[*Field]struct{}, recurser onCaseRecurser) (map[string]any, error) {
	if current.ListIndex != nil {
		// this field is for a list index, but this is a map entry
		message := fmt.Sprintf("field %v for data %v was a ListIndex field, but data was a map", current, data)
		return nil, newDataError(message, errorInvalidField)
	}
	currentValue, ok := data[*current.Name]
	if !ok {
		// we only break if onMissing returns an error, some cases may want to keep processing if
		// the item is not found
		newData, err := recurser.onMissing(data, current)
		if err != nil {
			return nil, err
		}
		newDataMap := newData.(map[string]any)
		data = newDataMap
		// refetch the currentValue which may have been initialized by the onMissing function
		currentValue = data[*current.Name]
	}
	if current == want {
		// base case, this field is a leaf node, call onFound
		newData, err := recurser.onFound(data, current, currentValue)
		if err != nil {
			return nil, err
		}
		mapData, ok := newData.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("recurser did not produce expected map[string]any onFound, got %v", data)
		}
		return mapData, nil

	} else {
		// this field is not a leaf, recurse to the next level
		if current.SubField == nil {
			message := fmt.Sprintf("unable to find target field %v; reached leaf node %v", want, current)
			return nil, newDataError(message, errorInvalidField)
		}
		newData, err := recurseInternal(currentValue, current.SubField, want, seen, recurser)
		if err != nil {
			return nil, err
		}
		data[*current.Name] = newData
		return data, nil
	}
}

func recurseInternalSlice(data []any, current *Field, want *Field, seen map[*Field]struct{}, recurser onCaseRecurser) ([]any, error) {
	if current.Name != nil {
		// this field is for a map index, but this is a list entry
		message := fmt.Sprintf("field %v for data %v was a Name field, but data was a slice", current, data)
		return nil, newDataError(message, errorInvalidField)
	}
	if len(data) <= *current.ListIndex {
		// we only break if onMissing returns an error, some cases may want to keep processing if
		// the item is not found
		newData, err := recurser.onMissing(data, current)
		if err != nil {
			return nil, err
		}
		newDataSlice := newData.([]any)
		data = newDataSlice
	}
	currentValue := data[*current.ListIndex]
	if current == want {
		// base case, this field is a leaf node, extract and return
		newData, err := recurser.onFound(data, current, currentValue)
		if err != nil {
			return nil, err
		}
		sliceData, ok := newData.([]any)
		if !ok {
			return nil, fmt.Errorf("recurser did not produce expected []any onFound, got %v", data)
		}
		return sliceData, nil
	} else {
		// this field is not a leaf node, recurse to the next level
		if current.SubField == nil {
			message := fmt.Sprintf("unable to find target field %v; reached leaf node %v", want, current)
			return nil, newDataError(message, errorInvalidField)
		}
		newData, err := recurseInternal(currentValue, current.SubField, want, seen, recurser)
		if err != nil {
			return nil, err
		}
		data[*current.ListIndex] = newData
		return data, nil
	}
}

// validateFieldNameIndex valides that field has only one of Name/ListIndex set. Will return an error if this is not the case.
func validateFieldNameIndex(field *Field) error {
	if field.Name == nil && field.ListIndex == nil || field.Name != nil && field.ListIndex != nil {
		message := fmt.Sprintf("field must have exactly one of name or index set, had name: %v, index: %v", field.Name, field.ListIndex)
		return newDataError(message, errorInvalidField)
	}
	return nil
}
