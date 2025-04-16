package condition

import (
	"errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

type testObjStatusStd struct {
	Conditions []metav1.Condition `json:"conditions"`
}

type testResourceObjStd struct {
	Status testObjStatusStd `json:"status"`
}

func newKatesCondition(condition Cond, message string) metav1.Condition {
	return metav1.Condition{
		Type:    string(condition),
		Status:  metav1.ConditionTrue,
		Message: message,
	}
}

func newTestObjStd(conditions ...Cond) testResourceObjStd {
	newObj := testResourceObjStd{
		Status: testObjStatusStd{
			Conditions: []metav1.Condition{},
		},
	}
	newConditions := make([]metav1.Condition, 0, len(conditions))
	if len(conditions) > 0 {
		for _, condition := range conditions {
			newConditions = append(
				newConditions,
				newKatesCondition(condition, "Hello World"),
			)
		}
	}
	newObj.Status.Conditions = newConditions

	return newObj
}

func TestConditionToKatesCondition(t *testing.T) {
	katesCondition := TestCondtion.ToKatesCondition()
	expected := MetaV1ConditionHandler{
		RootCondition: TestCondtion,
	}
	assert.Equal(t, &expected, katesCondition)
}

func TestStandardConditionHasCondition(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	assert.Equal(t, true, TestCondtion.ToKatesCondition().HasCondition(&testObj))
	assert.Equal(t, true, TestCondtion.ToKatesCondition().HasCondition(&testObj.Status))

	assert.Equal(t, false, AnotherTestCondtion.ToKatesCondition().HasCondition(&testObj))
	assert.Equal(t, false, AnotherTestCondtion.ToKatesCondition().HasCondition(&testObj.Status))
}

func TestStandardConditionGetStatus(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	assert.Equal(t, "True", TestCondtion.ToKatesCondition().GetStatus(&testObj))
	assert.Equal(t, "True", TestCondtion.ToKatesCondition().GetStatus(&testObj.Status))

	assert.Equal(t, "", AnotherTestCondtion.ToKatesCondition().GetStatus(&testObj))
	assert.Equal(t, "", AnotherTestCondtion.ToKatesCondition().GetStatus(&testObj.Status))
}

func TestStandardConditionSetStatus(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	assert.Equal(t, "True", TestCondtion.ToKatesCondition().GetStatus(&testObj))
	assert.Equal(t, "True", TestCondtion.ToKatesCondition().GetStatus(&testObj.Status))
	TestCondtion.ToKatesCondition().SetStatus(&testObj, "False")
	assert.Equal(t, "False", TestCondtion.ToKatesCondition().GetStatus(&testObj))
	assert.Equal(t, "False", TestCondtion.ToKatesCondition().GetStatus(&testObj.Status))

	assert.Equal(t, "", AnotherTestCondtion.GetStatus(&testObj))
	assert.Equal(t, "", AnotherTestCondtion.GetStatus(&testObj.Status))
	AnotherTestCondtion.SetStatus(&testObj, "Unknown")
	assert.Equal(t, "Unknown", AnotherTestCondtion.GetStatus(&testObj))
	assert.Equal(t, "Unknown", AnotherTestCondtion.GetStatus(&testObj.Status))
}

func TestStandardBoolHelpers(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	assert.Equal(t, "True", TestCondtion.ToKatesCondition().GetStatus(&testObj))
	assert.Equal(t, "True", TestCondtion.ToKatesCondition().GetStatus(&testObj.Status))
	TestCondtion.ToKatesCondition().False(&testObj)
	assert.Equal(t, "False", TestCondtion.ToKatesCondition().GetStatus(&testObj))
	assert.Equal(t, "False", TestCondtion.ToKatesCondition().GetStatus(&testObj.Status))

	assert.Equal(t, "", AnotherTestCondtion.ToKatesCondition().GetStatus(&testObj))
	assert.Equal(t, "", AnotherTestCondtion.ToKatesCondition().GetStatus(&testObj.Status))
	AnotherTestCondtion.ToKatesCondition().True(&testObj)
	assert.Equal(t, "True", AnotherTestCondtion.ToKatesCondition().GetStatus(&testObj))
	assert.Equal(t, "True", AnotherTestCondtion.ToKatesCondition().GetStatus(&testObj.Status))
	AnotherTestCondtion.ToKatesCondition().False(&testObj)
	assert.Equal(t, "False", AnotherTestCondtion.ToKatesCondition().GetStatus(&testObj))
	assert.Equal(t, "False", AnotherTestCondtion.ToKatesCondition().GetStatus(&testObj.Status))
}

func TestStandardConditionSetStatusBool(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	assert.Equal(t, "True", TestCondtion.ToKatesCondition().GetStatus(&testObj))
	assert.Equal(t, "True", TestCondtion.ToKatesCondition().GetStatus(&testObj.Status))
	TestCondtion.ToKatesCondition().SetStatusBool(&testObj, false)
	assert.Equal(t, "False", TestCondtion.ToKatesCondition().GetStatus(&testObj))
	assert.Equal(t, "False", TestCondtion.ToKatesCondition().GetStatus(&testObj.Status))

	assert.Equal(t, "", AnotherTestCondtion.ToKatesCondition().GetStatus(&testObj))
	assert.Equal(t, "", AnotherTestCondtion.ToKatesCondition().GetStatus(&testObj.Status))
	AnotherTestCondtion.ToKatesCondition().SetStatusBool(&testObj, true)
	assert.Equal(t, "True", AnotherTestCondtion.ToKatesCondition().GetStatus(&testObj))
	assert.Equal(t, "True", AnotherTestCondtion.ToKatesCondition().GetStatus(&testObj.Status))
}

func TestStandardConditionReasonMethods(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	TestCondtion.ToKatesCondition().SetReason(&testObj, "Because I Said So")
	assert.Equal(t, "Because I Said So", TestCondtion.ToKatesCondition().GetReason(&testObj))

	assert.Equal(t, "", AnotherTestCondtion.ToKatesCondition().GetReason(&testObj))
	AnotherTestCondtion.ToKatesCondition().SetReason(&testObj, "Because Tom Said So")
	assert.Equal(t, "Because Tom Said So", AnotherTestCondtion.ToKatesCondition().GetReason(&testObj))
}

func TestStandardConditionSetMessageIfBlank(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	TestCondtion.ToKatesCondition().SetMessageIfBlank(&testObj, "This will be ignored")
	assert.NotEqual(t, "This will be ignored", TestCondtion.GetMessage(&testObj))

	assert.Equal(t, "", AnotherTestCondtion.GetMessage(&testObj))
	AnotherTestCondtion.ToKatesCondition().SetMessageIfBlank(&testObj, "This will be updated")
	assert.Equal(t, "This will be updated", AnotherTestCondtion.GetMessage(&testObj))
	AnotherTestCondtion.ToKatesCondition().SetMessageIfBlank(&testObj, "This will NOT be updated")
	assert.Equal(t, "This will be updated", AnotherTestCondtion.GetMessage(&testObj))
}

func TestStandardConditionMessageMethods(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	assert.Equal(t, "Hello World", TestCondtion.ToKatesCondition().GetMessage(&testObj))
	TestCondtion.ToKatesCondition().SetMessage(&testObj, "")
	assert.Equal(t, "", TestCondtion.ToKatesCondition().GetMessage(&testObj))

	AnotherTestCondtion.ToKatesCondition().SetMessage(&testObj, "This will be updated")
	assert.Equal(t, "This will be updated", AnotherTestCondtion.ToKatesCondition().GetMessage(&testObj))
}

func TestStandardConditionErrorMethods(t *testing.T) {
	const SubStatusCondition Cond = "SomeStepSpecificState"

	testError := errors.New("some test error")

	testObj := newTestObjStd(TestCondtion)
	TestCondtion.ToKatesCondition().False(&testObj)
	SubStatusCondition.ToKatesCondition().False(&testObj)
	SubStatusCondition.ToKatesCondition().SetError(&testObj, "", testError)

	assert.Equal(t, "Error", SubStatusCondition.ToKatesCondition().GetReason(&testObj))
	assert.Equal(t, "some test error", SubStatusCondition.ToKatesCondition().GetMessage(&testObj))
	assert.True(t, SubStatusCondition.ToKatesCondition().MatchesError(&testObj, "Error", testError))
	assert.True(t, SubStatusCondition.ToKatesCondition().MatchesError(&testObj, "", testError))

	SubStatusCondition.ToKatesCondition().SetError(&testObj, "Because it Broke", testError)
	assert.False(t, SubStatusCondition.ToKatesCondition().MatchesError(&testObj, "", testError))
	assert.True(t, SubStatusCondition.ToKatesCondition().MatchesError(&testObj, "Because it Broke", testError))
	assert.False(t, SubStatusCondition.ToKatesCondition().MatchesError(&testObj, "Because something else Broke", testError))

	SubStatusCondition.ToKatesCondition().SetError(&testObj, "Because something else Broke", nil)
	assert.False(t, SubStatusCondition.ToKatesCondition().MatchesError(&testObj, "Because something else Broke", testError))
	assert.True(t, SubStatusCondition.ToKatesCondition().MatchesError(&testObj, "Because something else Broke", nil))

}
