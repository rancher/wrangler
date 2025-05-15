package condition

import (
	"bytes"
	"errors"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"testing"
)

func TestFluentConditionSetStatus(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	assert.Equal(t, "True", TestCondtion.ToK8sCondition().GetStatus(testObj))
	TestCondtion.ToFluentBuilder(testObj).SetStatus("False").Apply(testObj)
	assert.Equal(t, "False", TestCondtion.ToK8sCondition().GetStatus(testObj))

	assert.Equal(t, "", AnotherTestCondtion.GetStatus(testObj))
	AnotherTestCondtion.ToFluentBuilder(testObj).SetStatus("Unknown").Apply(testObj)
	assert.Equal(t, "Unknown", AnotherTestCondtion.GetStatus(testObj))
}

func TestFluentBoolHelpers(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)

	assert.Equal(t, "True", TestCondtion.ToK8sCondition().GetStatus(testObj))
	TestCondtion.ToFluentBuilder(testObj).False().Apply(testObj)
	assert.Equal(t, "False", TestCondtion.ToK8sCondition().GetStatus(testObj))

	assert.Equal(t, "", AnotherTestCondtion.ToK8sCondition().GetStatus(testObj))
	AnotherTestCondtion.ToFluentBuilder(testObj).True().Apply(testObj)
	assert.Equal(t, "True", AnotherTestCondtion.ToK8sCondition().GetStatus(testObj))
	AnotherTestCondtion.ToFluentBuilder(nil).Target(testObj).False().Apply(testObj)
	assert.Equal(t, "False", AnotherTestCondtion.ToK8sCondition().GetStatus(testObj))

	TestCondtion.ToFluentBuilder(testObj).Unknown().Apply(testObj)
	assert.Equal(t, "Unknown", TestCondtion.ToK8sCondition().GetStatus(testObj))
}

func TestFluentConditionSetStatusBool(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	assert.Equal(t, "True", TestCondtion.ToK8sCondition().GetStatus(testObj))
	TestCondtion.ToFluentBuilder(testObj).SetStatusBool(false).Apply(testObj)
	assert.Equal(t, "False", TestCondtion.ToK8sCondition().GetStatus(testObj))

	assert.Equal(t, "", AnotherTestCondtion.ToK8sCondition().GetStatus(testObj))
	AnotherTestCondtion.ToFluentBuilder(testObj).SetStatusBool(true).Apply(testObj)
	assert.Equal(t, "True", AnotherTestCondtion.ToK8sCondition().GetStatus(testObj))
}

func TestSetError(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	testErr := errors.New("test error")

	TestCondtion.ToFluentBuilder(testObj).SetError("it all broke", testErr).Apply(testObj)
	assert.Equal(t, "False", TestCondtion.ToK8sCondition().GetStatus(testObj))
	assert.Equal(t, "it all broke", TestCondtion.ToK8sCondition().GetReason(testObj))
	assert.Equal(t, "test error", TestCondtion.ToK8sCondition().GetMessage(testObj))

	testErr = errors.New("another test error")
	TestCondtion.ToFluentBuilder(testObj).SetError("", testErr).Apply(testObj)
	assert.Equal(t, "False", TestCondtion.ToK8sCondition().GetStatus(testObj))
	assert.Equal(t, "Error", TestCondtion.ToK8sCondition().GetReason(testObj))
	assert.Equal(t, "another test error", TestCondtion.ToK8sCondition().GetMessage(testObj))

	TestCondtion.ToFluentBuilder(testObj).SetError("Skip-itty Skip", generic.ErrSkip).Apply(testObj)
	assert.Equal(t, "True", TestCondtion.ToK8sCondition().GetStatus(testObj))
	assert.Equal(t, "Skip-itty Skip", TestCondtion.ToK8sCondition().GetReason(testObj))
	assert.Equal(t, "", TestCondtion.ToK8sCondition().GetMessage(testObj))
}

func TestFluentConditionSetReasonMethods(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	TestCondtion.ToFluentBuilder(testObj).SetReason("Because I Said So").Apply(testObj)
	assert.Equal(t, "Because I Said So", TestCondtion.ToK8sCondition().GetReason(testObj))

	assert.Equal(t, "", AnotherTestCondtion.ToK8sCondition().GetReason(testObj))
	AnotherTestCondtion.ToFluentBuilder(testObj).SetReason("Because Tom Said So").Apply(testObj)
	assert.Equal(t, "Because Tom Said So", AnotherTestCondtion.ToK8sCondition().GetReason(testObj))
}

func TestFluentConditionSetMessage(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	TestCondtion.ToFluentBuilder(testObj).SetMessage("This will NOT be ignored").Apply(testObj)
	assert.Equal(t, "This will NOT be ignored", TestCondtion.GetMessage(testObj))

	assert.Equal(t, "", AnotherTestCondtion.GetMessage(testObj))
	AnotherTestCondtion.ToFluentBuilder(testObj).SetMessage("This will be updated").Apply(testObj)
	assert.Equal(t, "This will be updated", AnotherTestCondtion.GetMessage(testObj))
}

func TestFluentConditionSetMessageIfBlank(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	TestCondtion.ToFluentBuilder(testObj).SetMessageIfBlank("This will be ignored").Apply(testObj)
	assert.NotEqual(t, "This will be ignored", TestCondtion.GetMessage(testObj))

	assert.Equal(t, "", AnotherTestCondtion.GetMessage(testObj))
	AnotherTestCondtion.ToFluentBuilder(testObj).SetMessageIfBlank("This will be updated").Apply(testObj)
	assert.Equal(t, "This will be updated", AnotherTestCondtion.GetMessage(testObj))
	AnotherTestCondtion.ToFluentBuilder(testObj).SetMessageIfBlank("This will NOT be updated").Apply(testObj)
	assert.Equal(t, "This will be updated", AnotherTestCondtion.GetMessage(testObj))
}

func TestIncorrectBuilder(t *testing.T) {
	testObj := newTestObjStd(TestCondtion)
	fluentBuilder := MetaV1ConditionFluentBuilder{
		RootCondition: TestCondtion,
	}

	// Base control expectations
	assert.Equal(t, "True", TestCondtion.ToK8sCondition().GetStatus(testObj))
	assert.False(t, fluentBuilder.initedTarget)
	assert.Equal(t, metav1.Condition{}, fluentBuilder.workingCondition)

	// Capture logs
	var logBuf bytes.Buffer
	logrus.SetOutput(&logBuf)
	defer logrus.SetOutput(os.Stderr) // reset after test

	// Verify SetStatus
	fluentBuilder.SetStatus("Unknown")
	assert.NotEqual(t, "Unknown", TestCondtion.ToK8sCondition().GetStatus(testObj))
	assert.Contains(t, logBuf.String(), "fluent condition handler not initialized")

	// Verify SetError
	logBuf = bytes.Buffer{}
	assert.Empty(t, logBuf.String())
	testErr := errors.New("test error")
	fluentBuilder.SetError("", testErr)
	assert.Contains(t, logBuf.String(), "fluent condition handler not initialized")

	// Verify SetReason
	logBuf = bytes.Buffer{}
	assert.Empty(t, logBuf.String())
	fluentBuilder.SetReason("for fun")
	assert.NotEqual(t, "for fun", TestCondtion.ToK8sCondition().GetReason(testObj))
	assert.Contains(t, logBuf.String(), "fluent condition handler not initialized")

	// Verify SetMessage
	logBuf = bytes.Buffer{}
	assert.Empty(t, logBuf.String())
	fluentBuilder.SetMessage("some message")
	assert.NotEqual(t, "some message", TestCondtion.ToK8sCondition().GetReason(testObj))
	assert.Contains(t, logBuf.String(), "fluent condition handler not initialized")

	logBuf = bytes.Buffer{}
	assert.Empty(t, logBuf.String())
	fluentBuilder.SetMessageIfBlank("some message")
	assert.NotEqual(t, "some message", TestCondtion.ToK8sCondition().GetReason(testObj))
	assert.Contains(t, logBuf.String(), "fluent condition handler not initialized")

	// Verify Apply
	logBuf = bytes.Buffer{}
	assert.Empty(t, logBuf.String())
	res := fluentBuilder.Apply(testObj)
	assert.False(t, res)
	assert.Contains(t, logBuf.String(), "fluent condition handler not initialized")
}
