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
	katesCondition := TestCondtion.ToK8sCondition()
	expected := MetaV1ConditionHandler{
		RootCondition: TestCondtion,
	}
	assert.Equal(t, &expected, katesCondition)
}

func TestStandardConditionHasCondition(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	assert.Equal(t, true, TestCondtion.ToK8sCondition().HasCondition(&testObj))
	assert.Equal(t, true, TestCondtion.ToK8sCondition().HasCondition(&testObj.Status))

	assert.Equal(t, false, AnotherTestCondtion.ToK8sCondition().HasCondition(&testObj))
	assert.Equal(t, false, AnotherTestCondtion.ToK8sCondition().HasCondition(&testObj.Status))
}

func TestStandardConditionGetStatus(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	assert.Equal(t, "True", TestCondtion.ToK8sCondition().GetStatus(&testObj))
	assert.Equal(t, "True", TestCondtion.ToK8sCondition().GetStatus(&testObj.Status))

	assert.Equal(t, "", AnotherTestCondtion.ToK8sCondition().GetStatus(&testObj))
	assert.Equal(t, "", AnotherTestCondtion.ToK8sCondition().GetStatus(&testObj.Status))
}

func TestStandardConditionSetStatus(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	assert.Equal(t, "True", TestCondtion.ToK8sCondition().GetStatus(&testObj))
	assert.Equal(t, "True", TestCondtion.ToK8sCondition().GetStatus(&testObj.Status))
	TestCondtion.ToK8sCondition().SetStatus(&testObj, "False")
	assert.Equal(t, "False", TestCondtion.ToK8sCondition().GetStatus(&testObj))
	assert.Equal(t, "False", TestCondtion.ToK8sCondition().GetStatus(&testObj.Status))

	assert.Equal(t, "", AnotherTestCondtion.GetStatus(&testObj))
	assert.Equal(t, "", AnotherTestCondtion.GetStatus(&testObj.Status))
	AnotherTestCondtion.SetStatus(&testObj, "Unknown")
	assert.Equal(t, "Unknown", AnotherTestCondtion.GetStatus(&testObj))
	assert.Equal(t, "Unknown", AnotherTestCondtion.GetStatus(&testObj.Status))
}

func TestStandardBoolHelpers(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	assert.Equal(t, "True", TestCondtion.ToK8sCondition().GetStatus(&testObj))
	assert.Equal(t, "True", TestCondtion.ToK8sCondition().GetStatus(&testObj.Status))
	TestCondtion.ToK8sCondition().False(&testObj)
	assert.Equal(t, "False", TestCondtion.ToK8sCondition().GetStatus(&testObj))
	assert.Equal(t, "False", TestCondtion.ToK8sCondition().GetStatus(&testObj.Status))

	assert.Equal(t, "", AnotherTestCondtion.ToK8sCondition().GetStatus(&testObj))
	assert.Equal(t, "", AnotherTestCondtion.ToK8sCondition().GetStatus(&testObj.Status))
	AnotherTestCondtion.ToK8sCondition().True(&testObj)
	assert.Equal(t, "True", AnotherTestCondtion.ToK8sCondition().GetStatus(&testObj))
	assert.Equal(t, "True", AnotherTestCondtion.ToK8sCondition().GetStatus(&testObj.Status))
	AnotherTestCondtion.ToK8sCondition().False(&testObj)
	assert.Equal(t, "False", AnotherTestCondtion.ToK8sCondition().GetStatus(&testObj))
	assert.Equal(t, "False", AnotherTestCondtion.ToK8sCondition().GetStatus(&testObj.Status))
}

func TestStandardConditionSetStatusBool(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	assert.Equal(t, "True", TestCondtion.ToK8sCondition().GetStatus(&testObj))
	assert.Equal(t, "True", TestCondtion.ToK8sCondition().GetStatus(&testObj.Status))
	TestCondtion.ToK8sCondition().SetStatusBool(&testObj, false)
	assert.Equal(t, "False", TestCondtion.ToK8sCondition().GetStatus(&testObj))
	assert.Equal(t, "False", TestCondtion.ToK8sCondition().GetStatus(&testObj.Status))

	assert.Equal(t, "", AnotherTestCondtion.ToK8sCondition().GetStatus(&testObj))
	assert.Equal(t, "", AnotherTestCondtion.ToK8sCondition().GetStatus(&testObj.Status))
	AnotherTestCondtion.ToK8sCondition().SetStatusBool(&testObj, true)
	assert.Equal(t, "True", AnotherTestCondtion.ToK8sCondition().GetStatus(&testObj))
	assert.Equal(t, "True", AnotherTestCondtion.ToK8sCondition().GetStatus(&testObj.Status))
}

func TestStandardConditionSetReasonMethods(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	TestCondtion.ToK8sCondition().SetReason(&testObj, "Because I Said So")
	assert.Equal(t, "Because I Said So", TestCondtion.ToK8sCondition().GetReason(&testObj))

	assert.Equal(t, "", AnotherTestCondtion.ToK8sCondition().GetReason(&testObj))
	AnotherTestCondtion.ToK8sCondition().SetReason(&testObj, "Because Tom Said So")
	assert.Equal(t, "Because Tom Said So", AnotherTestCondtion.ToK8sCondition().GetReason(&testObj))
}

func TestStandardConditionSetMessageIfBlank(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	TestCondtion.ToK8sCondition().SetMessageIfBlank(&testObj, "This will be ignored")
	assert.NotEqual(t, "This will be ignored", TestCondtion.GetMessage(&testObj))

	assert.Equal(t, "", AnotherTestCondtion.GetMessage(&testObj))
	AnotherTestCondtion.ToK8sCondition().SetMessageIfBlank(&testObj, "This will be updated")
	assert.Equal(t, "This will be updated", AnotherTestCondtion.GetMessage(&testObj))
	AnotherTestCondtion.ToK8sCondition().SetMessageIfBlank(&testObj, "This will NOT be updated")
	assert.Equal(t, "This will be updated", AnotherTestCondtion.GetMessage(&testObj))
}

func TestStandardConditionMessageMethods(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	assert.Equal(t, "Hello World", TestCondtion.ToK8sCondition().GetMessage(&testObj))
	TestCondtion.ToK8sCondition().SetMessage(&testObj, "")
	assert.Equal(t, "", TestCondtion.ToK8sCondition().GetMessage(&testObj))

	AnotherTestCondtion.ToK8sCondition().SetMessage(&testObj, "This will be updated")
	assert.Equal(t, "This will be updated", AnotherTestCondtion.ToK8sCondition().GetMessage(&testObj))
}

func TestStandardConditionErrorMethods(t *testing.T) {
	const SubStatusCondition Cond = "SomeStepSpecificState"

	testError := errors.New("some test error")

	testObj := newTestObjStd(TestCondtion)
	TestCondtion.ToK8sCondition().False(&testObj)
	SubStatusCondition.ToK8sCondition().False(&testObj)
	SubStatusCondition.ToK8sCondition().SetError(&testObj, "", testError)

	assert.Equal(t, "Error", SubStatusCondition.ToK8sCondition().GetReason(&testObj))
	assert.Equal(t, "some test error", SubStatusCondition.ToK8sCondition().GetMessage(&testObj))
	assert.True(t, SubStatusCondition.ToK8sCondition().MatchesError(&testObj, "Error", testError))
	assert.True(t, SubStatusCondition.ToK8sCondition().MatchesError(&testObj, "", testError))

	SubStatusCondition.ToK8sCondition().SetError(&testObj, "Because it Broke", testError)
	assert.False(t, SubStatusCondition.ToK8sCondition().MatchesError(&testObj, "", testError))
	assert.True(t, SubStatusCondition.ToK8sCondition().MatchesError(&testObj, "Because it Broke", testError))
	assert.False(t, SubStatusCondition.ToK8sCondition().MatchesError(&testObj, "Because something else Broke", testError))

	SubStatusCondition.ToK8sCondition().SetError(&testObj, "Because something else Broke", nil)
	assert.False(t, SubStatusCondition.ToK8sCondition().MatchesError(&testObj, "Because something else Broke", testError))
	assert.True(t, SubStatusCondition.ToK8sCondition().MatchesError(&testObj, "Because something else Broke", nil))

}
