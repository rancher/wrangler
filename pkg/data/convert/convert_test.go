package convert

import (
	"testing"
)

type data struct {
	TTLMillis int `json:"ttl"`
}

func TestJSON(t *testing.T) {
	d := &data{
		TTLMillis: 57600000,
	}

	m, err := EncodeToMap(d)
	if err != nil {
		t.Fatal(err)
	}

	i, _ := ToNumber(m["ttl"])
	if i != 57600000 {
		t.Fatal("not", 57600000, "got", m["ttl"])
	}
}

func TestArgKey(t *testing.T) {
	data := []struct {
		input  string
		output string
	}{
		{
			input:  "disableOpenAPIValidation",
			output: "--disable-open-api-validation",
		},
		{
			input:  "skipCRDs",
			output: "--skip-crds",
		},
	}

	for _, data := range data {
		if ToArgKey(data.input) != data.output {
			t.Errorf("expected %s, got %s", data.output, ToArgKey(data.input))
		}
	}
}
