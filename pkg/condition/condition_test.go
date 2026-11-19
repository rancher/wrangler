package condition

import (
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type genericCondition struct {
	Type           string
	Status         string
	LastUpdateTime string
	Reason         string
	Message        string
}

type objWithGenericConditions struct {
	Status struct {
		Conditions []genericCondition
	}
}

type objWithMetav1Conditions struct {
	Status struct {
		Conditions []metav1.Condition
	}
}

func TestCondGenericConditionSetsLastUpdateTime(t *testing.T) {
	obj := &objWithGenericConditions{}
	cond := Cond("Ready")

	cond.True(obj)
	cond.Message(obj, "all good")

	if !cond.IsTrue(obj) {
		t.Fatalf("expected condition to be true")
	}
	if got := cond.GetMessage(obj); got != "all good" {
		t.Fatalf("expected message %q, got %q", "all good", got)
	}
	if cond.GetLastUpdated(obj) == "" {
		t.Fatalf("expected LastUpdateTime to be set")
	}

	cond.LastUpdated(obj, "2024-01-01T00:00:00Z")
	if got := cond.GetLastUpdated(obj); got != "2024-01-01T00:00:00Z" {
		t.Fatalf("expected LastUpdateTime %q, got %q", "2024-01-01T00:00:00Z", got)
	}
}

func TestCondMetav1ConditionWithoutLastUpdateTime(t *testing.T) {
	obj := &objWithMetav1Conditions{}
	cond := Cond("Ready")

	cond.True(obj)
	cond.Message(obj, "all good")
	cond.Reason(obj, "Reconciled")
	cond.SetError(obj, "Failed", errors.New("boom"))
	cond.LastUpdated(obj, "2024-01-01T00:00:00Z")

	if !cond.IsFalse(obj) {
		t.Fatalf("expected condition to be false after SetError")
	}
	if got := cond.GetMessage(obj); got != "boom" {
		t.Fatalf("expected message %q, got %q", "boom", got)
	}
	if got := cond.GetReason(obj); got != "Failed" {
		t.Fatalf("expected reason %q, got %q", "Failed", got)
	}
	if got := cond.GetLastUpdated(obj); got != "" {
		t.Fatalf("expected empty LastUpdateTime, got %q", got)
	}
}
