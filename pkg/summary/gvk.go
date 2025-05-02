package summary

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	conditionStatusSep = "/"
	gvkFormat          = "%s" + conditionStatusSep + "%s" + conditionStatusSep + "%s"
)

// conditionTypeStatusJSON is a custom JSON to map a complex object into a standard JSON object. It maps Groups, Versions and Kinds to
// Conditions Types and Status, indicating with a flag if a certain condition with specific status represents an error or not.
// Is is expected to be something like:
//
//	{
//			"gvk": 			"helm.cattle.io/v1/HelmChart",
//			"conditionMapping": [
//				{
//					"type": "JobCreated"	// This means JobCreated is mostly informational and True or False doesn't mean error
//				},
//				{
//					"type": "Failed",	// This means Failed is considered error if it's status is True
//					"status": ["True"]
//				},
//			}
//	}
//
// IMPORTANT: Please pay attention to the the conditionStatusSep char, in this case it is a '/'. It separates Groups, Versions and Kinds.
type conditionTypeStatusJSON struct {
	GVK              string                     `json:"gvk"`
	ConditionMapping []conditionStatusErrorJSON `json:"conditionMapping"`
}

type conditionStatusErrorJSON struct {
	Type   string   `json:"type"`
	Status []string `json:"status"`
}

type ConditionTypeStatusErrorMapping map[schema.GroupVersionKind]map[string]sets.Set[string]

func (m ConditionTypeStatusErrorMapping) MarshalJSON() ([]byte, error) {
	output := []conditionTypeStatusJSON{}
	for gvk, mapping := range m {
		output = append(output, conditionTypeStatusJSON{
			GVK: fmt.Sprintf(gvkFormat, gvk.Group, gvk.Version, gvk.Kind),
			ConditionMapping: func() []conditionStatusErrorJSON {
				conditions := make([]conditionStatusErrorJSON, 0)
				for condition, statuses := range mapping {
					conditions = append(conditions, conditionStatusErrorJSON{
						Type: condition,
						Status: func() []string {
							output := []string{}
							for status := range statuses {
								output = append(output, status)
							}
							return output
						}(),
					})
				}
				return conditions
			}(),
		})
	}
	return json.Marshal(output)
}

func (m ConditionTypeStatusErrorMapping) UnmarshalJSON(data []byte) error {
	var GVKConditionMapping []conditionTypeStatusJSON
	err := json.Unmarshal(data, &GVKConditionMapping)
	if err != nil {
		return err
	}

	for _, mapping := range GVKConditionMapping {
		// parsing Group, Version and Kind in a format of group/version/kind
		// eg: helm.cattle.io/v1/HelmChart
		gvk := strings.Split(mapping.GVK, conditionStatusSep)
		if len(gvk) != 3 {
			return errors.New("gvk parsing failed: wrong GVK format")
		}
		conditionMapping := map[string]sets.Set[string]{}
		for _, condition := range mapping.ConditionMapping {
			conditionMapping[condition.Type] = sets.New[string](condition.Status...)
		}

		m[schema.GroupVersionKind{
			Group:   gvk[0],
			Version: gvk[1],
			Kind:    gvk[2],
		}] = conditionMapping
	}
	return nil
}
