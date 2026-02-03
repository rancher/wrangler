package summary

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/rancher/wrangler/v3/pkg/data"
)

func GetUnstructuredConditions(obj map[string]interface{}) []Condition {
	return getConditions(obj)
}

func getRawConditions(obj data.Object) []data.Object {
	// Extract conditions from status field
	conditions := getStatusConditions(obj)

	// Append conditions from cattle.io/status annotation
	conditions = append(conditions, getAnnotationConditions(obj)...)

	return conditions
}

// getStatusConditions extracts conditions from the status field.
// It prioritizes deprecated CAPI v1beta1 conditions if they exist.
func getStatusConditions(obj data.Object) []data.Object {
	// CAPI - use the deprecated v1beta1 conditions if they exist
	deprecatedConditions := obj.Slice("status", "deprecated", "v1beta1", "conditions")
	if len(deprecatedConditions) > 0 {
		return deprecatedConditions
	}

	return obj.Slice("status", "conditions")
}

// getAnnotationConditions extracts conditions from the cattle.io/status annotation.
// Returns an empty slice if the annotation doesn't exist or cannot be parsed.
func getAnnotationConditions(obj data.Object) []data.Object {
	statusAnn := obj.String("metadata", "annotations", "cattle.io/status")
	if statusAnn == "" {
		return []data.Object{}
	}

	var status data.Object
	if err := json.Unmarshal([]byte(statusAnn), &status); err != nil {
		return []data.Object{}
	}

	conditions := status.Slice("conditions")
	if conditions == nil {
		return []data.Object{}
	}
	return conditions
}

func getConditions(obj data.Object) (result []Condition) {
	for _, condition := range getRawConditions(obj) {
		result = append(result, Condition{Object: condition})
	}
	return
}

type Condition struct {
	data.Object
}

func NewCondition(conditionType, status, reason, message string) Condition {
	return Condition{
		Object: map[string]interface{}{
			"type":    conditionType,
			"status":  status,
			"reason":  reason,
			"message": message,
		},
	}
}

func (c Condition) Type() string {
	return c.String("type")
}

func (c Condition) Status() string {
	return c.String("status")
}

func (c Condition) Reason() string {
	return c.String("reason")
}

func (c Condition) Message() string {
	return c.String("message")
}

func (c Condition) Equals(other Condition) bool {
	return c.Type() == other.Type() &&
		c.Status() == other.Status() &&
		c.Reason() == other.Reason() &&
		c.Message() == other.Message()
}

func NormalizeConditions(runtimeObj runtime.Object) {
	var (
		obj           data.Object
		newConditions []map[string]interface{}
	)

	unstr, ok := runtimeObj.(*unstructured.Unstructured)
	if !ok {
		return
	}

	obj = unstr.Object
	for _, condition := range obj.Slice("status", "conditions") {
		var summary Summary
		for _, summarizer := range ConditionSummarizers {
			summary = summarizer(obj, []Condition{{Object: condition}}, summary)
		}
		condition.Set("error", summary.Error)
		condition.Set("transitioning", summary.Transitioning)

		if condition.String("lastUpdateTime") == "" {
			condition.Set("lastUpdateTime", condition.String("lastTransitionTime"))
		}
		newConditions = append(newConditions, condition)
	}

	if len(newConditions) > 0 {
		obj.SetNested(newConditions, "status", "conditions")
	}

}
