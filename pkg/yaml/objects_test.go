package yaml

import (
	"encoding/json"
	"fmt"
	"strings"
)

// JSONstruct is a test struct for verifying Unmarshal.
type JSONstruct struct {
	EmbeddedStruct
	CustomField   CustomStruct
	MismatchField bool `json:"newFieldName"`
	NestedField   EmbeddedStruct
	NormalField   int
}

type EmbeddedStruct struct {
	EmbeddedField string
}
type CustomStruct struct {
	Name      string
	Namespace string
}

func (c *CustomStruct) UnmarshalJSON(data []byte) error {
	var tmp string
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	parts := strings.Split(tmp, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid test string")
	}
	c.Name = parts[0]
	c.Namespace = parts[1]
	return nil
}

var (
	emptyDoc     = []byte("")
	singleString = []byte("string")
	unknownDoc   = []byte("unknown: value")
	invalidYAML  = []byte(`umm:
		21:`)
	singleDeployment = []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
    name: singleDeployment
spec:
    replicas: 3
`)
	multipleDeployments = []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
    name: dep1
spec:
    replicas: 3
---
apiVersion: apps/v1
kind: Deployment
metadata:
    name: dep2
    namespace: dep2-ns
spec:
    paused: true
---
metadata:
    name: dep3
    namespace: dep3-ns
    labels:
       app: testapp
status:
    readyReplicas: 4
`)
	jsonYAML = []byte(`normalField: 28
embeddedField: "embeddedValue"
customField: "testName/testNamespace"
newFieldName: true
nestedField:
    embeddedField: "nestedValue"
`)
)
