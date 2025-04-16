package condition

import (
	"github.com/rancher/wrangler/v3/pkg/genericcondition"
	"github.com/stretchr/testify/assert"
	"testing"
)

type testObjStatus struct {
	Conditions []genericcondition.GenericCondition `json:"conditions"`
}

type testResourceObj struct {
	Status testObjStatus `json:"status"`
}

func newTestObj(conditions ...Cond) testResourceObj {
	newObj := testResourceObj{
		Status: testObjStatus{
			Conditions: []genericcondition.GenericCondition{},
		},
	}
	if len(conditions) > 0 {
		for _, condition := range conditions {
			condition.SetStatusBool(&newObj, true)
			condition.Message(&newObj, "Hello World")
		}
	}

	return newObj
}

const (
	TestCondtion        Cond = "Test"
	AnotherTestCondtion Cond = "SecondTest"
)

func TestGetStatus(t *testing.T) {
	testObj := newTestObj(TestCondtion)
	assert.Equal(t, "True", TestCondtion.GetStatus(&testObj))
	assert.Equal(t, "True", TestCondtion.GetStatus(&testObj.Status))

	assert.Equal(t, "", AnotherTestCondtion.GetStatus(&testObj))
	assert.Equal(t, "", AnotherTestCondtion.GetStatus(&testObj.Status))
}

func TestSetStatus(t *testing.T) {
	testObj := newTestObj(TestCondtion)
	assert.Equal(t, "True", TestCondtion.GetStatus(&testObj))
	assert.Equal(t, "True", TestCondtion.GetStatus(&testObj.Status))
	TestCondtion.SetStatus(&testObj, "False")
	assert.Equal(t, "False", TestCondtion.GetStatus(&testObj))
	assert.Equal(t, "False", TestCondtion.GetStatus(&testObj.Status))

	assert.Equal(t, "", AnotherTestCondtion.GetStatus(&testObj))
	assert.Equal(t, "", AnotherTestCondtion.GetStatus(&testObj.Status))
	AnotherTestCondtion.SetStatus(&testObj, "Unknown")
	assert.Equal(t, "Unknown", AnotherTestCondtion.GetStatus(&testObj))
	assert.Equal(t, "Unknown", AnotherTestCondtion.GetStatus(&testObj.Status))
}

func TestSetStatusBool(t *testing.T) {
	testObj := newTestObj(TestCondtion)
	assert.Equal(t, "True", TestCondtion.GetStatus(&testObj))
	assert.Equal(t, "True", TestCondtion.GetStatus(&testObj.Status))
	TestCondtion.SetStatusBool(&testObj, false)
	assert.Equal(t, "False", TestCondtion.GetStatus(&testObj))
	assert.Equal(t, "False", TestCondtion.GetStatus(&testObj.Status))

	assert.Equal(t, "", AnotherTestCondtion.GetStatus(&testObj))
	assert.Equal(t, "", AnotherTestCondtion.GetStatus(&testObj.Status))
	AnotherTestCondtion.SetStatusBool(&testObj, true)
	assert.Equal(t, "True", AnotherTestCondtion.GetStatus(&testObj))
	assert.Equal(t, "True", AnotherTestCondtion.GetStatus(&testObj.Status))
}

func TestBoolHelpers(t *testing.T) {
	testObj := newTestObj(TestCondtion)
	assert.Equal(t, "True", TestCondtion.GetStatus(&testObj))
	assert.Equal(t, "True", TestCondtion.GetStatus(&testObj.Status))
	TestCondtion.False(&testObj)
	assert.Equal(t, "False", TestCondtion.GetStatus(&testObj))
	assert.Equal(t, "False", TestCondtion.GetStatus(&testObj.Status))

	assert.Equal(t, "", AnotherTestCondtion.GetStatus(&testObj))
	assert.Equal(t, "", AnotherTestCondtion.GetStatus(&testObj.Status))
	AnotherTestCondtion.True(&testObj)
	assert.Equal(t, "True", AnotherTestCondtion.GetStatus(&testObj))
	assert.Equal(t, "True", AnotherTestCondtion.GetStatus(&testObj.Status))
	AnotherTestCondtion.False(&testObj)
	assert.Equal(t, "False", AnotherTestCondtion.GetStatus(&testObj))
	assert.Equal(t, "False", AnotherTestCondtion.GetStatus(&testObj.Status))
}

func TestBoolConditoins(t *testing.T) {
	testObj := newTestObj(TestCondtion)
	assert.True(t, TestCondtion.IsTrue(&testObj))
	assert.True(t, TestCondtion.IsTrue(&testObj.Status))
	TestCondtion.False(&testObj)
	assert.True(t, TestCondtion.IsFalse(&testObj))
	assert.True(t, TestCondtion.IsFalse(&testObj.Status))

	assert.Equal(t, "", AnotherTestCondtion.GetStatus(&testObj))
	assert.Equal(t, "", AnotherTestCondtion.GetStatus(&testObj.Status))
	AnotherTestCondtion.True(&testObj)
	assert.True(t, AnotherTestCondtion.IsTrue(&testObj))
	assert.True(t, AnotherTestCondtion.IsTrue(&testObj.Status))
	AnotherTestCondtion.False(&testObj)
	assert.True(t, AnotherTestCondtion.IsFalse(&testObj))
	assert.True(t, AnotherTestCondtion.IsFalse(&testObj.Status))
}

func TestUnknownHelpers(t *testing.T) {
	testObj := newTestObj(TestCondtion)
	assert.False(t, TestCondtion.IsUnknown(&testObj))
	assert.False(t, AnotherTestCondtion.IsUnknown(&testObj))
	AnotherTestCondtion.Message(&testObj, "Test Message, will default status to unknown")
	assert.True(t, AnotherTestCondtion.IsUnknown(&testObj))

	TestCondtion.Unknown(&testObj)
	assert.True(t, TestCondtion.IsUnknown(&testObj))
}

func TestCreateUnknownIfNotExists(t *testing.T) {
	testObj := newTestObj(TestCondtion)
	assert.False(t, TestCondtion.IsUnknown(&testObj))
	assert.False(t, AnotherTestCondtion.IsUnknown(&testObj))
	AnotherTestCondtion.CreateUnknownIfNotExists(&testObj)
	assert.True(t, AnotherTestCondtion.IsUnknown(&testObj))
	TestCondtion.CreateUnknownIfNotExists(&testObj)
	assert.False(t, TestCondtion.IsUnknown(&testObj))
}

func TestReasonMethods(t *testing.T) {
	testObj := newTestObj(TestCondtion)
	TestCondtion.Reason(&testObj, "Because I Said So")
	assert.Equal(t, "Because I Said So", TestCondtion.GetReason(&testObj))

	assert.Equal(t, "", AnotherTestCondtion.GetReason(&testObj))
	AnotherTestCondtion.Reason(&testObj, "Because Tom Said So")
	assert.Equal(t, "Because Tom Said So", AnotherTestCondtion.GetReason(&testObj))
}

func TestSetMessageIfBlank(t *testing.T) {
	testObj := newTestObj(TestCondtion)
	TestCondtion.SetMessageIfBlank(&testObj, "This will be ignored")
	assert.NotEqual(t, "This will be ignored", TestCondtion.GetMessage(&testObj))

	assert.Equal(t, "", AnotherTestCondtion.GetMessage(&testObj))
	AnotherTestCondtion.SetMessageIfBlank(&testObj, "This will be updated")
	assert.Equal(t, "This will be updated", AnotherTestCondtion.GetMessage(&testObj))
	AnotherTestCondtion.SetMessageIfBlank(&testObj, "This will NOT be updated")
	assert.Equal(t, "This will be updated", AnotherTestCondtion.GetMessage(&testObj))
}

func TestMessageMethods(t *testing.T) {
	testObj := newTestObj(TestCondtion)
	assert.Equal(t, "Hello World", TestCondtion.GetMessage(&testObj))
	TestCondtion.Message(&testObj, "")
	assert.Equal(t, "", TestCondtion.GetMessage(&testObj))

	AnotherTestCondtion.Message(&testObj, "This will be updated")
	assert.Equal(t, "This will be updated", AnotherTestCondtion.GetMessage(&testObj))
}
