package condition

import (
	"errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rancher/wrangler/v3/pkg/generic"
)

type MetaV1ConditionHandler struct {
	RootCondition Cond
}

// getConditionsSlice attempts to retrieve and validate the slice of conditions
// from the given object. It returns the slice of metav1.Condition if successful,
// and nil otherwise.
func getConditionsSlice(obj client.Object) ([]metav1.Condition, bool) {
	condSliceValue := getValue(obj, "Status", "Conditions")

	if !condSliceValue.IsValid() || condSliceValue.Kind() != reflect.Slice {
		return nil, false
	}

	condSliceInterface := condSliceValue.Interface()
	conditions, ok := condSliceInterface.([]metav1.Condition)
	return conditions, ok
}

func findCondition(obj client.Object, name string) *metav1.Condition {
	conditionsSlice, ok := getConditionsSlice(obj)
	if !ok {
		return nil
	}
	return meta.FindStatusCondition(conditionsSlice, name)
}

func (ch *MetaV1ConditionHandler) HasCondition(obj client.Object) bool {
	conditionsSlice, ok := getConditionsSlice(obj)
	if !ok || len(conditionsSlice) == 0 {
		return false
	}

	for i := range conditionsSlice {
		if conditionsSlice[i].Type == ch.RootCondition.Name() {
			return true
		}
	}

	return false
}

func (ch *MetaV1ConditionHandler) GetStatus(obj client.Object) string {
	foundCondition := findCondition(obj, ch.RootCondition.Name())
	if foundCondition == nil {
		return ""
	}

	return string(foundCondition.Status)
}

func (ch *MetaV1ConditionHandler) SetStatus(obj client.Object, status string) {
	ch.setStatus(obj, status)
}

func (ch *MetaV1ConditionHandler) SetStatusBool(obj client.Object, val bool) {
	if val {
		ch.setStatus(obj, "True")
	} else {
		ch.setStatus(obj, "False")
	}
}

func (ch *MetaV1ConditionHandler) False(obj client.Object) {
	ch.setStatus(obj, "False")
}
func (ch *MetaV1ConditionHandler) IsFalse(obj client.Object) bool {
	conditionsSlice, ok := getConditionsSlice(obj)
	if !ok {
		return false
	}

	return meta.IsStatusConditionFalse(conditionsSlice, ch.RootCondition.Name())
}

func (ch *MetaV1ConditionHandler) True(obj client.Object) {
	ch.setStatus(obj, "True")
}

func (ch *MetaV1ConditionHandler) IsTrue(obj client.Object) bool {
	conditionsSlice, ok := getConditionsSlice(obj)
	if !ok {
		return false
	}

	return meta.IsStatusConditionTrue(conditionsSlice, ch.RootCondition.Name())
}
func (ch *MetaV1ConditionHandler) Unknown(obj client.Object) {
	ch.setStatus(obj, "Unknown")
}
func (ch *MetaV1ConditionHandler) IsUnknown(obj client.Object) bool {
	foundCondition := findCondition(obj, ch.RootCondition.Name())
	if foundCondition == nil {
		return false
	}

	return foundCondition.Status == metav1.ConditionUnknown
}

func (ch *MetaV1ConditionHandler) SetError(obj client.Object, reason string, err error) {
	cond := ch.findOrCreateCondition(obj)

	if err == nil || errors.Is(err, generic.ErrSkip) {
		cond.Status = metav1.ConditionTrue
		cond.Message = ""
		cond.Reason = reason
		cond.ObservedGeneration = getResourceGeneration(obj)
		return
	}

	if reason == "" {
		reason = "Error"
	}

	cond.Status = metav1.ConditionFalse
	cond.Message = err.Error()
	cond.Reason = reason
	cond.ObservedGeneration = getResourceGeneration(obj)
}

func (ch *MetaV1ConditionHandler) MatchesError(obj client.Object, reason string, err error) bool {
	if err == nil {
		return ch.IsTrue(obj) &&
			ch.GetMessage(obj) == "" &&
			ch.GetReason(obj) == reason
	}
	if reason == "" {
		reason = "Error"
	}
	return ch.IsFalse(obj) &&
		ch.GetMessage(obj) == err.Error() &&
		ch.GetReason(obj) == reason
}

func (ch *MetaV1ConditionHandler) GetReason(obj client.Object) string {
	foundCondition := findCondition(obj, ch.RootCondition.Name())
	if foundCondition == nil {
		return ""
	}

	return foundCondition.Reason
}

func (ch *MetaV1ConditionHandler) SetReason(obj client.Object, reason string) {
	cond := ch.findOrCreateCondition(obj)
	updatedCond := cond.DeepCopy()
	updatedCond.Reason = reason
	updatedCond.ObservedGeneration = getResourceGeneration(obj)

	conditionsSlice, ok := getConditionsSlice(obj)
	if !ok {
		panic("conditions slice does not exist")
	}

	_ = meta.SetStatusCondition(&conditionsSlice, *updatedCond)
}

func (ch *MetaV1ConditionHandler) GetMessage(obj client.Object) string {
	foundCondition := findCondition(obj, ch.RootCondition.Name())
	if foundCondition == nil {
		return ""
	}

	return foundCondition.Message
}

func (ch *MetaV1ConditionHandler) SetMessage(obj client.Object, msg string) {
	cond := ch.findOrCreateCondition(obj)
	updatedCond := cond.DeepCopy()
	updatedCond.Message = msg
	updatedCond.ObservedGeneration = getResourceGeneration(obj)

	conditionsSlice, ok := getConditionsSlice(obj)
	if !ok {
		panic("conditions slice does not exist")
	}

	_ = meta.SetStatusCondition(&conditionsSlice, *updatedCond)
}

func (ch *MetaV1ConditionHandler) SetMessageIfBlank(obj client.Object, msg string) {
	cond := ch.findOrCreateCondition(obj)
	updatedCond := cond.DeepCopy()
	if cond.Message == "" {
		updatedCond.Message = msg
	}

	if updatedCond.Message == cond.Message {
		return
	}
	updatedCond.ObservedGeneration = getResourceGeneration(obj)

	conditionsSlice, ok := getConditionsSlice(obj)
	if !ok {
		panic("conditions slice does not exist")
	}

	_ = meta.SetStatusCondition(&conditionsSlice, *updatedCond)
}

func (ch *MetaV1ConditionHandler) setStatus(obj client.Object, status string) {
	if reflect.TypeOf(obj).Kind() != reflect.Ptr {
		panic("obj passed must be a pointer")
	}

	if status != string(metav1.ConditionTrue) &&
		status != string(metav1.ConditionFalse) &&
		status != string(metav1.ConditionUnknown) {
		panic("unknown condition status: " + status)
	}

	statusParsed := metav1.ConditionStatus(status)
	cond := ch.findOrCreateCondition(obj)

	cond.Status = statusParsed
	cond.ObservedGeneration = getResourceGeneration(obj)
}

func (ch *MetaV1ConditionHandler) findOrCreateCondition(obj client.Object) *metav1.Condition {
	foundCondition := findCondition(obj, ch.RootCondition.Name())
	if foundCondition != nil {
		return foundCondition
	}

	newCond := metav1.Condition{
		Type:               ch.RootCondition.Name(),
		Status:             metav1.ConditionUnknown,
		ObservedGeneration: getResourceGeneration(obj),
		Reason:             "Created",
		Message:            "",
	}

	conditionsSlice, ok := getConditionsSlice(obj)
	if !ok {
		panic("conditions slice does not exist")
	}

	changed := meta.SetStatusCondition(&conditionsSlice, newCond)
	if changed {
		setConditionsSlice(obj, conditionsSlice)
	}

	return findCondition(obj, ch.RootCondition.Name())
}

// getResourceGeneration attempts to retrieve and validate the slice of conditions
// from the given object. It returns the slice of metav1.Condition if successful,
// and nil otherwise.
func getResourceGeneration(obj client.Object) int64 {
	return obj.GetGeneration()
}

func setConditionsSlice(obj client.Object, conditions []metav1.Condition) {
	statusValue := getValue(obj, "Status")
	if !statusValue.IsValid() {
		panic("object does not have a Status field")
	}

	condSliceValue := getValue(obj, "Status", "Conditions")
	if !condSliceValue.IsValid() {
		panic("Status does not have a Conditions field")
	}

	if condSliceValue.Kind() != reflect.Slice {
		panic("Conditions field must be a slice")
	}

	condSliceValue.Set(reflect.ValueOf(conditions))
}
