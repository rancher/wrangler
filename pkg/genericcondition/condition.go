package genericcondition

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type GenericCondition struct {
	// Type of cluster condition.
	Type string `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime string `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// Human-readable message indicating details about last transition
	Message string `json:"message,omitempty"`
}

// ToK8sCondition will translate an existing condition into a k8s condition (essentially drops LastUpdatedTime)
func (g GenericCondition) ToK8sCondition() metav1.Condition {
	lastTransitionTime, _ := time.Parse(time.RFC3339, g.LastTransitionTime)

	return metav1.Condition{
		Type:   g.Type,
		Status: metav1.ConditionStatus(g.Status),
		// ObservedGeneration: nil, // Ideally set this to the current generation at time of conversion
		LastTransitionTime: metav1.NewTime(lastTransitionTime),
		Reason:             g.Reason,
		Message:            g.Message,
	}
}
