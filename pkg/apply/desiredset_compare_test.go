package apply

import (
	"bytes"
	"testing"
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
