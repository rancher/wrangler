package name

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	string32 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	string63 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	string64 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

// TODO: Improve GuessPluralNames. The rules used do not accurately pluralize nouns.
// For now this unit test covers the existing functionality to ensure backwards compatibility.
func TestGuessPluralName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		nouns map[string]string
	}{
		{
			name:  "Empty string",
			nouns: map[string]string{"": ""},
		},
		{
			name:  "Special cases",
			nouns: map[string]string{"Endpoints": "Endpoints"},
		},
		{
			name: "Strings ending in s ch x and sh",
			nouns: map[string]string{
				"iris":  "irises",
				"leech": "leeches",
				"tax":   "taxes",
				"fish":  "fishes",
			},
		},
		{
			name: "Strings ending in f and fe",
			nouns: map[string]string{
				"elf":  "elfves",
				"safe": "safeves",
			},
		},
		{
			name: "Strings ending in y",
			nouns: map[string]string{
				"candy":    "candies",
				"birthday": "birthdays",
				"turkey":   "turkeys",
				"toy":      "toys",
				"guy":      "guys",
			},
		},
		{
			name: "Non-special strings",
			nouns: map[string]string{
				"friend": "friends",
				"cat":    "cats",
				"dog":    "dogs",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for k, v := range tt.nouns {
				got := GuessPluralName(k)
				assert.Equal(t, v, got, "GuessPluralName(%v) = %v, want %v", k, got, v)
			}
		})
	}
}

func TestLimit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		s         string
		count     int
		want      string
		wantPanic bool
	}{
		{
			name:  "length of string is less than count",
			s:     "aaaaaa",
			count: 7,
			want:  "aaaaaa",
		},
		{
			name:  "length of string is equal to count",
			s:     "aaaaaaa",
			count: 7,
			want:  "a-5d793",
		},
		{
			name:  "length of string is greater than count",
			s:     "aaaaaaaaaaaaaaaaaa",
			count: 8,
			want:  "aa-2c60c",
		},
		{
			name:  "only hash exists when length of string and count are 6",
			s:     "aaaaaa",
			count: 6,
			want:  "-0b4e7",
		},
		{
			name:  "empty string",
			s:     "",
			count: 7,
			want:  "",
		},
		{
			name:      "panic when count <= 5",
			s:         "aaaaaaa",
			count:     5,
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantPanic {
				assert.Panics(t, func() { Limit(tt.s, tt.count) })
				return
			}
			if got := Limit(tt.s, tt.count); got != tt.want {
				t.Errorf("Limit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		s         string
		n         int
		want      string
		wantPanic bool
	}{
		{
			name: "basic test",
			s:    "aaaaaa",
			n:    4,
			want: "0b4e",
		},
		{
			name: "full checksum",
			s:    "aaaaaa",
			n:    32,
			want: "0b4e7a0e5fe84ad35fb5f95b9ceeac79",
		},
		{
			name:      "get more characters than full checksum",
			s:         "aaaaaa",
			n:         33,
			wantPanic: true,
		},
		{
			name:      "get negative characters",
			s:         "aaaaaa",
			n:         -1,
			wantPanic: true,
		},
		{
			name: "get 0 characters",
			s:    "aaaaaa",
			n:    0,
			want: "",
		},
		{
			name: "empty string",
			s:    "",
			n:    4,
			want: "d41d",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantPanic {
				assert.Panics(t, func() { Hex(tt.s, tt.n) })
				return
			}
			if got := Hex(tt.s, tt.n); got != tt.want {
				t.Errorf("Hex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeConcatName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  []string
		output string
	}{
		{
			name:   "empty input",
			output: "",
		},
		{
			name:   "single string",
			input:  []string{string63},
			output: string63,
		},
		{
			name:   "single long string",
			input:  []string{string64},
			output: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-ffe05",
		},
		{
			name:   "concatenate strings",
			input:  []string{"first", "second", "third"},
			output: "first-second-third",
		},
		{
			name:   "concatenate past 64 characters",
			input:  []string{string32, string32},
			output: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-aaaaaaaaaaaaaaaaaaaaaaaa-da5ed",
		},
		{
			name:   "last character after truncation is not alphanumeric",
			input:  []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-aaaaaaa"},
			output: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-768c62",
		},
		{
			name:   "last characters after truncation aren't alphanumeric",
			input:  []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa--aaaaaaa"},
			output: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa--9e8cfe",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := SafeConcatName(tt.input...); got != tt.output {
				t.Errorf("SafeConcatName() = %v, want %v", got, tt.output)
			}
		})
	}
}
