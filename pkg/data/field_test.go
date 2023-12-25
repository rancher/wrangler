package data

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type FieldTestSuite struct {
	suite.Suite
	brokenMapField          *Field
	brokenSliceField        *Field
	loopField               *Field
	invalidNameIndexField   *Field
	invalidNoNameIndexField *Field
}

func TestField(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(FieldTestSuite))
}

func (f *FieldTestSuite) SetupSuite() {
	rootField := NewNameField("test")
	rootField.SubField = nil
	brokenMapField := NewNameField("nested")
	brokenMapField.root = rootField

	rootSlice := NewIndexField(0)
	rootField.SubField = nil
	brokenSliceField := NewIndexField(0)
	brokenSliceField.root = rootSlice

	rootCopy := rootField.Copy()
	child := NewNameField("child")
	rootCopy.SubField = child
	child.SubField = rootCopy
	loopField := NewNameField("nested")
	loopField.root = rootCopy

	invalidIndex := 1
	invalidFieldNameAndIndex := NewNameField("test")
	invalidFieldNameAndIndex.ListIndex = &invalidIndex

	invalidFieldNoNameAndIndex := &Field{}

	f.brokenMapField = brokenMapField
	f.brokenSliceField = brokenSliceField
	f.loopField = loopField
	f.invalidNameIndexField = invalidFieldNameAndIndex
	f.invalidNoNameIndexField = invalidFieldNoNameAndIndex
}

func (f *FieldTestSuite) TestChild() {
	f.T().Parallel()
	metadata := NewNameField("metadata")
	annotations := metadata.Child("annotations")
	f.Require().Nil(metadata.SubField)
	f.Require().Equal("metadata", *annotations.Root().Name)
	// circular field
	metadata.SubField = annotations
	annotations.SubField = metadata
	loop := NewNameField("loop")
	loop.root = metadata
	f.Require().Nil(loop.Child("test"))
	// test nil values can be chained
	f.Require().Nil(loop.Child("test").Child("next"))
	// test root value
	nested := NewNameField("metadata").Child("annotations").Child("nested")
	f.Require().NotNil(nested)
	f.Require().NotNil(nested.Root())
	f.Require().Equal("metadata", *nested.Root().Name)
	// test broken chain from root
	metadata.SubField = nil
	annotations.root = metadata
	f.Require().Nil(annotations.Child("test"))
}

func (f *FieldTestSuite) TestIndex() {
	f.T().Parallel()
	start := NewIndexField(0)
	next := start.Index(1)
	f.Require().Nil(start.SubField)
	f.Require().Equal(0, *next.Root().ListIndex)
	// circular field
	start.SubField = next
	next.SubField = start
	loop := NewIndexField(2)
	loop.root = start
	f.Require().Nil(loop.Index(1))
	// test nil values can be chained
	f.Require().Nil(loop.Index(1).Index(0))
	// test root value
	nested := NewIndexField(0).Index(1).Index(2)
	f.Require().NotNil(nested)
	f.Require().NotNil(nested.Root())
	f.Require().Equal(0, *nested.Root().ListIndex)
	// test broken chain from root
	start.SubField = nil
	next.root = start
	f.Require().Nil(next.Index(1))
}

func (f *FieldTestSuite) TestCopy() {
	f.T().Parallel()
	start := NewNameField("test").Index(0).Root()
	startCopy := start.Copy()
	// the values should be equal, but they should now refer to different memory
	f.Require().Equal(start, startCopy)
	f.Require().False(start == startCopy)
	var nilField *Field
	f.Require().Nil(nilField.Copy())
}

func (f *FieldTestSuite) TestGetField() {
	tests := []struct {
		name          string
		data          any
		field         *Field
		want          any
		wantError     bool
		wantErrorCode errorCode //ignored if wantError == false
	}{
		{
			name: "basic map",
			data: map[string]any{
				"test": "val",
			},
			field: NewNameField("test"),
			want:  "val",
		},
		{
			name: "nested map",
			data: map[string]any{
				"test": map[string]any{
					"val": "nested",
				},
			},
			field: NewNameField("test").Child("val"),
			want:  "nested",
		},
		{
			name:  "basic slice",
			data:  []any{"value", "other"},
			field: NewIndexField(1),
			want:  "other",
		},
		{
			name:  "nested slice",
			data:  []any{"value", []any{"other"}},
			field: NewIndexField(1).Index(0),
			want:  "other",
		},
		{
			name: "nested slice in a map",
			data: map[string]any{
				"test": []any{"nested"},
			},
			field: NewNameField("test").Index(0),
			want:  "nested",
		},
		{
			name: "nested map in a slice",
			data: []any{"some", map[string]any{
				"test": "nested",
			}},
			field: NewIndexField(1).Child("test"),
			want:  "nested",
		},
		{
			name: "deeply nested",
			data: map[string]any{
				"test": []any{
					map[string]any{
						"nested": "value",
					},
				},
			},
			field: NewNameField("test").Index(0).Child("nested"),
			want:  "value",
		},
		{
			name: "missing value map",
			data: map[string]any{
				"test": map[string]any{
					"nested": "value",
				},
			},
			field:         NewNameField("test").Child("notFound"),
			wantError:     true,
			wantErrorCode: errorFieldValueNotFound,
		},
		{
			name:          "missing value slice",
			data:          []any{[]any{"value"}},
			field:         NewIndexField(0).Index(1),
			wantError:     true,
			wantErrorCode: errorFieldValueNotFound,
		},
		{
			name: "invalid value",
			data: map[string]any{
				"test": []string{"hello"},
			},
			field:         NewNameField("test").Index(0),
			wantError:     true,
			wantErrorCode: errorInvalidData,
		},
		{
			name: "listIndex given for a map",
			data: map[string]any{
				"test": map[string]any{
					"nested": "true",
				},
			},
			field:         NewNameField("test").Index(0),
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name: "name given for a slice",
			data: map[string]any{
				"test": []any{"nested"},
			},
			field:         NewNameField("test").Child("nested"),
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name: "broken link map",
			data: map[string]any{
				"test": map[string]any{
					"nested": "value",
				},
			},
			field:         f.brokenMapField,
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name:          "broken link slice",
			data:          []any{[]any{"nested"}},
			field:         f.brokenSliceField,
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name: "nested loop",
			data: map[string]any{
				"test": map[string]any{
					"child": map[string]any{
						"nested": "val",
					},
				},
			},
			field:         f.loopField,
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name: "invalid field name + index",
			data: map[string]any{
				"test": map[string]any{
					"nested": "val",
				},
			},
			field:         f.invalidNameIndexField,
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name: "invalid field no name or index",
			data: map[string]any{
				"test": map[string]any{
					"nested": "val",
				},
			},
			field:         f.invalidNoNameIndexField,
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
	}
	for _, test := range tests {
		test := test
		f.Run(test.name, func() {
			f.T().Parallel()
			got, gotError := GetField(test.data, test.field)
			if !test.wantError {
				f.Require().Nil(gotError)
				f.Require().Equal(test.want, got)
				return
			}
			f.Require().NotNil(gotError)
			switch test.wantErrorCode {
			// don't validate error code if its not a known code
			case errorInvalidData:
				f.Require().True(IsInvalidDataError(gotError))
			case errorInvalidField:
				f.Require().True(IsInvalidFieldError(gotError))
			case errorFieldValueNotFound:
				f.Require().True(IsFieldValueNotFoundError(gotError))
			}
		})
	}
}

func (f *FieldTestSuite) TestRemoveField() {
	tests := []struct {
		name          string
		data          any
		field         *Field
		wantData      any
		want          any
		wantError     bool
		wantErrorCode errorCode //ignored if wantError == false
	}{
		{
			name: "basic map",
			data: map[string]any{
				"test": "val",
			},
			wantData: map[string]any{},
			field:    NewNameField("test"),
			want:     "val",
		},
		{
			name: "nested map",
			data: map[string]any{
				"test": map[string]any{
					"val": "nested",
				},
			},
			wantData: map[string]any{
				"test": map[string]any{},
			},
			field: NewNameField("test").Child("val"),
			want:  "nested",
		},
		{
			name:     "basic slice",
			data:     []any{"value", "other"},
			wantData: []any{"value"},
			field:    NewIndexField(1),
			want:     "other",
		},
		{
			name:     "nested slice",
			data:     []any{"value", []any{"other"}},
			wantData: []any{"value", []any{}},
			field:    NewIndexField(1).Index(0),
			want:     "other",
		},
		{
			name: "nested slice in a map",
			data: map[string]any{
				"test": []any{"nested"},
			},
			wantData: map[string]any{
				"test": []any{},
			},
			field: NewNameField("test").Index(0),
			want:  "nested",
		},
		{
			name: "nested map in a slice",
			data: []any{"some", map[string]any{
				"test": "nested",
			}},
			wantData: []any{"some", map[string]any{}},
			field:    NewIndexField(1).Child("test"),
			want:     "nested",
		},
		{
			name: "deeply nested",
			data: map[string]any{
				"test": []any{
					map[string]any{
						"nested": "value",
					},
				},
			},
			wantData: map[string]any{
				"test": []any{
					map[string]any{},
				},
			},
			field: NewNameField("test").Index(0).Child("nested"),
			want:  "value",
		},
		{
			name: "missing value map",
			data: map[string]any{
				"test": map[string]any{
					"nested": "value",
				},
			},
			field:         NewNameField("test").Child("notFound"),
			wantError:     true,
			wantErrorCode: errorFieldValueNotFound,
		},
		{
			name:          "missing value slice",
			data:          []any{[]any{"value"}},
			field:         NewIndexField(0).Index(1),
			wantError:     true,
			wantErrorCode: errorFieldValueNotFound,
		},
		{
			name: "invalid value",
			data: map[string]any{
				"test": []string{"hello"},
			},
			field:         NewNameField("test").Index(0),
			wantError:     true,
			wantErrorCode: errorInvalidData,
		},
		{
			name: "listIndex given for a map",
			data: map[string]any{
				"test": map[string]any{
					"nested": "true",
				},
			},
			field:         NewNameField("test").Index(0),
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name: "name given for a slice",
			data: map[string]any{
				"test": []any{"nested"},
			},
			field:         NewNameField("test").Child("nested"),
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name: "broken link map",
			data: map[string]any{
				"test": map[string]any{
					"nested": "value",
				},
			},
			field:         f.brokenMapField,
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name:          "broken link slice",
			data:          []any{[]any{"nested"}},
			field:         f.brokenSliceField,
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name: "nested loop",
			data: map[string]any{
				"test": map[string]any{
					"child": map[string]any{
						"nested": "val",
					},
				},
			},
			field:         f.loopField,
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name: "invalid field name + index",
			data: map[string]any{
				"test": map[string]any{
					"nested": "val",
				},
			},
			field:         f.invalidNameIndexField,
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name: "invalid field no name or index",
			data: map[string]any{
				"test": map[string]any{
					"nested": "val",
				},
			},
			field:         f.invalidNoNameIndexField,
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
	}
	for _, test := range tests {
		test := test
		f.Run(test.name, func() {
			f.T().Parallel()
			gotData, gotRemoved, gotError := RemoveField(test.data, test.field)
			if !test.wantError {
				f.Require().Nil(gotError)
				f.Require().Equal(test.want, gotRemoved)
				f.Require().Equal(test.wantData, gotData)
				return
			}
			f.Require().NotNil(gotError)
			switch test.wantErrorCode {
			// don't validate error code if its not a known code
			case errorInvalidData:
				f.Require().True(IsInvalidDataError(gotError))
			case errorInvalidField:
				f.Require().True(IsInvalidFieldError(gotError))
			case errorFieldValueNotFound:
				f.Require().True(IsFieldValueNotFoundError(gotError))
			}
		})
	}
}

func (f *FieldTestSuite) TestPutField() {
	invalidIdx := 1
	invalidName := "invalid"

	invalidMapEntry := NewNameField("test").Child("notFound").Child("notFoundNext")
	invalidMapEntry.ListIndex = &invalidIdx

	invalidSliceEntry := NewIndexField(0).Index(1).Index(1)
	invalidSliceEntry.Name = &invalidName

	tests := []struct {
		name          string
		data          any
		field         *Field
		value         any
		wantData      any
		wantError     bool
		wantErrorCode errorCode //ignored if wantError == false
	}{
		{
			name: "basic map",
			data: map[string]any{
				"test": "val",
			},
			field: NewNameField("key"),
			value: "value",
			wantData: map[string]any{
				"test": "val",
				"key":  "value",
			},
		},
		{
			name: "nested map",
			data: map[string]any{
				"test": map[string]any{
					"val": "nested",
				},
			},
			field: NewNameField("test").Child("key"),
			value: "value",
			wantData: map[string]any{
				"test": map[string]any{
					"val": "nested",
					"key": "value",
				},
			},
		},
		{
			name:     "basic slice",
			data:     []any{"value"},
			field:    NewIndexField(1),
			value:    "other",
			wantData: []any{"value", "other"},
		},
		{
			name:     "nested slice",
			data:     []any{"value"},
			field:    NewIndexField(1).Index(0),
			value:    "other",
			wantData: []any{"value", []any{"other"}},
		},
		{
			name: "nested slice in a map",
			data: map[string]any{
				"test": []any{},
			},
			field: NewNameField("test").Index(0),
			value: "nested",
			wantData: map[string]any{
				"test": []any{"nested"},
			},
		},
		{
			name:  "nested map in a slice",
			data:  []any{"some", map[string]any{}},
			field: NewIndexField(1).Child("test"),
			value: "nested",
			wantData: []any{"some", map[string]any{
				"test": "nested",
			}},
		},
		{
			name: "deeply nested",
			data: map[string]any{
				"test": []any{
					map[string]any{
						"nested": "value",
					},
				},
			},
			wantData: map[string]any{
				"test": []any{
					map[string]any{
						"nested":  "value",
						"nested2": "value2",
					},
				},
			},
			field: NewNameField("test").Index(0).Child("nested2"),
			value: "value2",
		},
		{
			name: "missing value map",
			data: map[string]any{
				"test": map[string]any{
					"nested": "value",
				},
			},
			field: NewNameField("test").Child("notFound"),
			value: "newValue",
			wantData: map[string]any{
				"test": map[string]any{
					"nested":   "value",
					"notFound": "newValue",
				},
			},
		},
		{
			name: "missing value map, need submap",
			data: map[string]any{
				"test": map[string]any{
					"nested": "value",
				},
			},
			field: NewNameField("test").Child("notFound").Child("notFoundNext"),
			value: "newValue",
			wantData: map[string]any{
				"test": map[string]any{
					"nested": "value",
					"notFound": map[string]any{
						"notFoundNext": "newValue",
					},
				},
			},
		},
		{
			name: "missing value map, need subslice",
			data: map[string]any{
				"test": map[string]any{
					"nested": "value",
				},
			},
			field: NewNameField("test").Child("notFound").Index(1),
			value: "newValue",
			wantData: map[string]any{
				"test": map[string]any{
					"nested":   "value",
					"notFound": []any{nil, "newValue"},
				},
			},
		},
		{
			name: "missing value map, invalid sub-entry",
			data: map[string]any{
				"test": map[string]any{
					"nested": "value",
				},
			},
			field:         invalidMapEntry,
			value:         "newValue",
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name:     "missing value slice",
			data:     []any{[]any{"value"}},
			field:    NewIndexField(0).Index(1),
			value:    "newValue",
			wantData: []any{[]any{"value", "newValue"}},
		},
		{
			name:     "missing value slice, need submap",
			data:     []any{[]any{"value"}},
			field:    NewIndexField(0).Index(1).Child("nested"),
			value:    "newValue",
			wantData: []any{[]any{"value", map[string]any{"nested": "newValue"}}},
		},
		{
			name:     "missing value slice, need subslice",
			data:     []any{[]any{"value"}},
			field:    NewIndexField(0).Index(1).Index(1),
			value:    "newValue",
			wantData: []any{[]any{"value", []any{nil, "newValue"}}},
		},
		{
			name:          "missing value slice, invalid sub-entry",
			data:          []any{[]any{"value"}},
			field:         invalidSliceEntry,
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name: "invalid value",
			data: map[string]any{
				"test": []string{"hello"},
			},
			field:         NewNameField("test").Index(0),
			value:         "value",
			wantError:     true,
			wantErrorCode: errorInvalidData,
		},
		{
			name: "listIndex given for a map",
			data: map[string]any{
				"test": map[string]any{
					"nested": "true",
				},
			},
			field:         NewNameField("test").Index(0),
			value:         "value",
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name: "name given for a slice",
			data: map[string]any{
				"test": []any{"nested"},
			},
			field:         NewNameField("test").Child("nested"),
			value:         "value",
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name: "broken link map",
			data: map[string]any{
				"test": map[string]any{
					"nested": "value",
				},
			},
			field:         f.brokenMapField,
			value:         "value",
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name:          "broken link slice",
			data:          []any{[]any{"nested"}},
			field:         f.brokenSliceField,
			value:         "value",
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name: "nested loop",
			data: map[string]any{
				"test": map[string]any{
					"child": map[string]any{
						"nested": "val",
					},
				},
			},
			field:         f.loopField,
			value:         "value",
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name: "invalid field name + index",
			data: map[string]any{
				"test": map[string]any{
					"nested": "val",
				},
			},
			field:         f.invalidNameIndexField,
			value:         "value",
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
		{
			name: "invalid field no name or index",
			data: map[string]any{
				"test": map[string]any{
					"nested": "val",
				},
			},
			field:         f.invalidNoNameIndexField,
			value:         "value",
			wantError:     true,
			wantErrorCode: errorInvalidField,
		},
	}
	for _, test := range tests {
		test := test
		f.Run(test.name, func() {
			f.T().Parallel()
			gotData, gotError := PutField(test.data, test.field, test.value)
			if !test.wantError {
				f.Require().Nil(gotError)
				f.Require().Equal(test.wantData, gotData)
				return
			}
			f.Require().NotNil(gotError)
			switch test.wantErrorCode {
			// don't validate error code if its not a known code
			case errorInvalidData:
				f.Require().True(IsInvalidDataError(gotError))
			case errorInvalidField:
				f.Require().True(IsInvalidFieldError(gotError))
			}
		})
	}
}
