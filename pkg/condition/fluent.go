package condition

import (
	"errors"
	"github.com/rancher/wrangler/v3/pkg/condition/types"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MetaV1ConditionFluentBuilder struct {
	RootCondition    Cond
	initedTarget     bool
	workingCondition metav1.Condition
}

func (m *MetaV1ConditionFluentBuilder) Target(obj interface{}) types.FluentCondition {
	cond := m.findOrInitCond(obj)
	m.workingCondition = *cond.DeepCopy()
	m.initedTarget = true

	return m
}

func (m *MetaV1ConditionFluentBuilder) findOrInitCond(obj interface{}) metav1.Condition {
	foundCondition := findCondition(obj, m.RootCondition.Name())
	if foundCondition != nil {
		return *foundCondition
	}

	return metav1.Condition{
		Type:    m.RootCondition.Name(),
		Status:  metav1.ConditionUnknown,
		Reason:  "Created",
		Message: "",
	}
}

func (m *MetaV1ConditionFluentBuilder) SetStatus(status string) types.FluentCondition {
	if !m.initedTarget {
		logrus.Warnf("fluent condition handler not initialized")
		return m
	}

	if status != string(metav1.ConditionTrue) &&
		status != string(metav1.ConditionFalse) &&
		status != string(metav1.ConditionUnknown) {
		panic("unknown condition status: " + status)
	}

	m.workingCondition.Status = metav1.ConditionStatus(status)

	return m
}

func (m *MetaV1ConditionFluentBuilder) True() types.FluentCondition {
	m.SetStatus("True")
	return m
}

func (m *MetaV1ConditionFluentBuilder) False() types.FluentCondition {
	m.SetStatus("False")
	return m
}

func (m *MetaV1ConditionFluentBuilder) Unknown() types.FluentCondition {
	m.SetStatus("Unknown")
	return m
}

func (m *MetaV1ConditionFluentBuilder) SetStatusBool(val bool) types.FluentCondition {
	if val {
		m.SetStatus("True")
	} else {
		m.SetStatus("False")
	}

	return m
}

func (m *MetaV1ConditionFluentBuilder) SetError(reason string, err error) types.FluentCondition {
	if !m.initedTarget {
		logrus.Warnf("fluent condition handler not initialized")
		return m
	}

	if err == nil || errors.Is(err, generic.ErrSkip) {
		m.workingCondition.Status = metav1.ConditionTrue
		m.workingCondition.Message = ""
		m.workingCondition.Reason = reason

		return m
	}

	if reason == "" {
		reason = "Error"
	}

	m.workingCondition.Status = metav1.ConditionFalse
	m.workingCondition.Message = err.Error()
	m.workingCondition.Reason = reason

	return m
}

func (m *MetaV1ConditionFluentBuilder) SetReason(reason string) types.FluentCondition {
	if !m.initedTarget {
		logrus.Warnf("fluent condition handler not initialized")
		return m
	}

	m.workingCondition.Reason = reason
	return m
}

func (m *MetaV1ConditionFluentBuilder) SetMessage(message string) types.FluentCondition {
	if !m.initedTarget {
		logrus.Warnf("fluent condition handler not initialized")
		return m
	}

	m.workingCondition.Message = message
	return m
}

func (m *MetaV1ConditionFluentBuilder) SetMessageIfBlank(message string) types.FluentCondition {
	if !m.initedTarget {
		logrus.Warnf("fluent condition handler not initialized")
		return m
	}

	if m.workingCondition.Message != "" {
		return m
	}

	m.workingCondition.Message = message
	return m
}

func (m *MetaV1ConditionFluentBuilder) Apply(obj interface{}) bool {
	if !m.initedTarget {
		logrus.Warnf("fluent condition handler not initialized")
		return false
	}

	conditionsSlice, ok := getConditionsSlice(obj)
	if ok {
		changed := meta.SetStatusCondition(&conditionsSlice, m.workingCondition)
		m.initedTarget = false
		m.workingCondition = metav1.Condition{}
		if changed {
			setConditionsSlice(obj, conditionsSlice)
		}
		return changed
	}

	return false
}
