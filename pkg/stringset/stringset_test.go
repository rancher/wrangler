package stringset

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func stringPtr(s string) *string {
	return &s
}

func Test_Set(t *testing.T) {
	tests := []struct {
		name          string
		addStrings    [][]string
		deleteStrings [][]string
		hasString     *string
		missingString *string
		finalStrings  []string
		wantLen       int
	}{
		{
			name:          "test 1",
			addStrings:    [][]string{},
			deleteStrings: [][]string{},
			hasString:     nil,
			missingString: stringPtr("bar"),
			finalStrings:  []string{},
			wantLen:       0,
		},
		{
			name:          "test 2",
			addStrings:    [][]string{{"foo"}},
			deleteStrings: [][]string{},
			hasString:     stringPtr("foo"),
			missingString: stringPtr("bar"),
			finalStrings:  []string{"foo"},
			wantLen:       1,
		},
		{
			name:          "test 3",
			addStrings:    [][]string{{"foo", "bar", "baz"}, {"bar", "baz"}, {"bop"}},
			deleteStrings: [][]string{{"foo", "baz"}},
			hasString:     stringPtr("bar"),
			missingString: stringPtr("foo"),
			finalStrings:  []string{"bar", "bop"},
			wantLen:       2,
		},
		{
			name:          "test 4",
			addStrings:    [][]string{{"foo"}, {""}, {"bar"}},
			deleteStrings: [][]string{{"bar"}},
			hasString:     stringPtr(""),
			missingString: stringPtr("bar"),
			finalStrings:  []string{"foo", ""},
			wantLen:       2,
		},
		{
			name:          "test 5",
			addStrings:    [][]string{{"foo"}, {"foo", "bar"}, {"foo", "bar", "baz"}},
			deleteStrings: [][]string{{"foo"}, {"bar", "baz"}},
			hasString:     nil,
			missingString: stringPtr("foo"),
			finalStrings:  []string{},
			wantLen:       0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := Set{}
			for _, ss := range tt.addStrings {
				set.Add(ss...)
			}
			for _, s := range tt.deleteStrings {
				set.Delete(s...)
			}

			if tt.hasString != nil {
				hasString := set.Has(*tt.hasString)
				assert.True(t, hasString, "HasString(%#v)", tt.hasString)
			}

			if tt.missingString != nil {
				missingString := set.Has(*tt.missingString)
				assert.False(t, missingString, "HasString(%#v)", tt.missingString)
			}

			gotStrings := set.Values()
			assert.ElementsMatchf(t, tt.finalStrings, gotStrings, "Values() = %v, want %v", gotStrings, tt.finalStrings)

			gotLen := set.Len()
			assert.Equal(t, tt.wantLen, gotLen, "Len() = %v, want %v", gotLen, tt.wantLen)
		})
	}
}
