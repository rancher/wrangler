package condition

import (
	"errors"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/wrangler/v3/pkg/generic"
)

type MetaV1ConditionHandler struct {
	RootCondition Cond
}

// getConditionSlice attempts to retrieve and validate the slice of conditions
// from the given object. It returns the slice of metav1.Condition if successful,
// and nil otherwise.
func getConditionSlice(obj interface{}) ([]metav1.Condition, bool) {
	condSliceValue := getValue(obj, "Status", "Conditions")
	if !condSliceValue.IsValid() {
		condSliceValue = getValue(obj, "Conditions")
	}

	if !condSliceValue.IsValid() || condSliceValue.Kind() != reflect.Slice {
		return nil, false
	}

	condSliceInterface := condSliceValue.Interface()
	conditions, ok := condSliceInterface.([]metav1.Condition)
	return conditions, ok
}

func setConditionSlice(obj interface{}, conditions []metav1.Condition) {
	condSliceValue := getValue(obj, "Status", "Conditions")
	if !condSliceValue.IsValid() {
		condSliceValue = getValue(obj, "Conditions")
		if !condSliceValue.IsValid() {
			panic("obj doesn't have conditions")
		}
	}

	condSliceValue.Set(reflect.ValueOf(conditions))
}

func findCondition(obj interface{}, name string) *metav1.Condition {
	conditionsSlice, ok := getConditionSlice(obj)
	if !ok {
		return nil
	}
	return meta.FindStatusCondition(conditionsSlice, name)
}

func (ch *MetaV1ConditionHandler) HasCondition(obj interface{}) bool {
	foundCondition := findCondition(obj, ch.RootCondition.Name())
	return foundCondition != nil
}

func (ch *MetaV1ConditionHandler) GetStatus(obj interface{}) string {
	foundCondition := findCondition(obj, ch.RootCondition.Name())
	if foundCondition == nil {
		return ""
	}

	return string(foundCondition.Status)
}

func (ch *MetaV1ConditionHandler) SetStatus(obj interface{}, status string) {
	ch.setStatus(obj, status)
}

func (ch *MetaV1ConditionHandler) SetStatusBool(obj interface{}, val bool) {
	if val {
		ch.setStatus(obj, "True")
	} else {
		ch.setStatus(obj, "False")
	}
}

func (ch *MetaV1ConditionHandler) False(obj interface{}) {
	ch.setStatus(obj, "False")
}
func (ch *MetaV1ConditionHandler) IsFalse(obj interface{}) bool {
	foundCondition := findCondition(obj, ch.RootCondition.Name())
	if foundCondition == nil {
		return false
	}

	return foundCondition.Status == metav1.ConditionFalse
}
func (ch *MetaV1ConditionHandler) True(obj interface{}) {
	ch.setStatus(obj, "True")
}
func (ch *MetaV1ConditionHandler) IsTrue(obj interface{}) bool {
	foundCondition := findCondition(obj, ch.RootCondition.Name())
	if foundCondition == nil {
		return false
	}

	return foundCondition.Status == metav1.ConditionTrue
}
func (ch *MetaV1ConditionHandler) Unknown(obj interface{}) {
	ch.setStatus(obj, "Unknown")
}
func (ch *MetaV1ConditionHandler) IsUnknown(obj interface{}) bool {
	foundCondition := findCondition(obj, ch.RootCondition.Name())
	if foundCondition == nil {
		return false
	}

	return foundCondition.Status == metav1.ConditionUnknown
}

func (ch *MetaV1ConditionHandler) SetError(obj interface{}, reason string, err error) {
	if err == nil || errors.Is(err, generic.ErrSkip) {
		ch.True(obj)
		ch.SetMessage(obj, "")
		ch.SetReason(obj, reason)
		return
	}

	if reason == "" {
		reason = "Error"
	}

	ch.False(obj)
	ch.SetMessage(obj, err.Error())
	ch.SetReason(obj, reason)
}
func (ch *MetaV1ConditionHandler) MatchesError(obj interface{}, reason string, err error) bool {
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

func (ch *MetaV1ConditionHandler) GetReason(obj interface{}) string {
	foundCondition := findCondition(obj, ch.RootCondition.Name())
	if foundCondition == nil {
		return ""
	}

	return foundCondition.Reason
}

func (ch *MetaV1ConditionHandler) SetReason(obj interface{}, reason string) {
	cond := ch.findOrCreateCondition(obj)
	cond.Reason = reason
}

func (ch *MetaV1ConditionHandler) GetMessage(obj interface{}) string {
	foundCondition := findCondition(obj, ch.RootCondition.Name())
	if foundCondition == nil {
		return ""
	}

	return foundCondition.Message
}

func (ch *MetaV1ConditionHandler) SetMessage(obj interface{}, msg string) {
	cond := ch.findOrCreateCondition(obj)
	cond.Message = msg
}

func (ch *MetaV1ConditionHandler) SetMessageIfBlank(obj interface{}, msg string) {
	cond := ch.findOrCreateCondition(obj)
	if cond.Message == "" {
		cond.Message = msg
	}
}

func (ch *MetaV1ConditionHandler) touchTransitionTS(condition *metav1.Condition) {
	now := metav1.NewTime(time.Now())
	condition.LastTransitionTime = now
}

func (ch *MetaV1ConditionHandler) setStatus(obj interface{}, status string) {
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
	if cond.Status != statusParsed {
		ch.touchTransitionTS(cond)
	}
	cond.Status = statusParsed
}

func (ch *MetaV1ConditionHandler) findOrCreateCondition(obj interface{}) *metav1.Condition {
	foundCondition := findCondition(obj, ch.RootCondition.Name())
	if foundCondition != nil {
		return foundCondition
	}

	newCond := metav1.Condition{
		Type:               ch.RootCondition.Name(),
		Status:             metav1.ConditionUnknown,
		LastTransitionTime: metav1.NewTime(time.Now()),
		Reason:             "Created",
		Message:            "",
	}

	conditionsSlice, ok := getConditionSlice(obj)
	if !ok {
		panic("conditions slice does not exist")
	}

	conditionsSlice = append(conditionsSlice, newCond)
	setConditionSlice(obj, conditionsSlice)
	return findCondition(obj, ch.RootCondition.Name())
}
