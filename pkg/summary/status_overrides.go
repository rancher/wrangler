package summary

import (
	"strings"

	"github.com/rancher/wrangler/v3/pkg/data"
)

// StatusOverrideAction defines what to do when a StatusOverride matches.
type StatusOverrideAction int

const (
	// OverrideSkip causes the condition to be skipped entirely in checkTransitioning.
	OverrideSkip StatusOverrideAction = iota
	// OverrideTransitioning marks the summary as transitioning.
	OverrideTransitioning
	// OverrideError marks the summary as error.
	OverrideError
)

// StatusOverride defines a data-driven rule that overrides the default behavior
// of a condition in checkTransitioning. Overrides are evaluated in order;
// the first match wins. More specific overrides (with more matchers) should be
// listed before general ones.
type StatusOverride struct {
	// ConditionType is the condition type to match (required).
	ConditionType string
	// Status is the condition status to match ("True", "False", "Unknown").
	// Empty string matches any status.
	Status string
	// Reason restricts the override to conditions with this exact reason.
	Reason string
	// MessageContains restricts the override to conditions whose message contains this substring.
	MessageContains string

	// Object matchers (all non-empty fields must match):

	// Kind restricts the override to objects of this kind.
	// Example: "Machine"
	Kind string
	// APIVersion is a more specific version matcher that requires an exact match.
	// Example: "cluster.x-k8s.io/v1beta2"
	APIVersion string
	// APIGroup is a more general version matcher that matches any version in this API group prefix.
	// Example: "cluster.x-k8s.io"
	APIGroup string

	// Action to take when matched.
	Action StatusOverrideAction

	// StateOverride, when non-empty, replaces the computed summary.state.
	// This is orthogonal to Action and can be combined with any action type,
	// it is ignored when Action is OverrideSkip since the condition is skipped entirely.
	StateOverride string
}

// matches returns true if this override applies to the given object and condition.
func (o StatusOverride) matches(obj data.Object, c Condition) bool {
	if o.ConditionType != c.Type() {
		return false
	}
	if o.Status != "" && o.Status != c.Status() {
		return false
	}
	if o.Kind != "" && obj.String("kind") != o.Kind {
		return false
	}
	apiVersion := obj.String("apiVersion")
	if o.APIVersion != "" && o.APIVersion != apiVersion {
		return false
	}
	if o.APIGroup != "" && !strings.HasPrefix(apiVersion, o.APIGroup+"/") {
		return false
	}
	if o.Reason != "" && c.Reason() != o.Reason {
		return false
	}
	if o.MessageContains != "" && !strings.Contains(c.Message(), o.MessageContains) {
		return false
	}
	return true
}

// findOverride returns the first matching StatusOverride for the given object and condition,
// or nil if no override matches.
func findOverride(overrides []StatusOverride, obj data.Object, c Condition) *StatusOverride {
	for i := range overrides {
		if overrides[i].matches(obj, c) {
			return &overrides[i]
		}
	}
	return nil
}

// StatusOverrides defines data-driven overrides for conditions that don't perfectly
// fit the TransitioningUnknown / TransitioningFalse / TransitioningTrue categories.
//
// Overrides are evaluated in order (first match wins). More specific entries
// (with more matchers) should appear before general ones.
var StatusOverrides = []StatusOverride{
	// CAPI Machine: "Updating" is an in-place update tracker, skip it.
	{
		ConditionType: "Updating",
		Kind:          "Machine",
		APIGroup:      "cluster.x-k8s.io",
		Action:        OverrideSkip,
	},
	// CAPI Machine: "Available" is False until Ready is true and has no meaningful message, skip it.
	{
		ConditionType: "Available",
		Kind:          "Machine",
		APIGroup:      "cluster.x-k8s.io",
		Action:        OverrideSkip,
	},
	// CAPI Machine: NodeHealthy/NodeReady with InspectionFailed reason usually happens during provisioning,
	// treat as transitioning with state overridden to "provisioning".
	{
		ConditionType: "NodeHealthy",
		Reason:        "InspectionFailed",
		Kind:          "Machine",
		APIGroup:      "cluster.x-k8s.io",
		Action:        OverrideTransitioning,
		StateOverride: "provisioning",
	},
	{
		ConditionType: "NodeReady",
		Reason:        "InspectionFailed",
		Kind:          "Machine",
		APIGroup:      "cluster.x-k8s.io",
		Action:        OverrideTransitioning,
		StateOverride: "provisioning",
	},
	// CAPI MachineDeployment, MachineSet: MachinesReady with Unknown status and message contains
	// "NodeHealthy: Waiting for", treat as transitioning instead of error.
	{
		ConditionType:   "MachinesReady",
		Status:          "Unknown",
		APIGroup:        "cluster.x-k8s.io",
		MessageContains: "NodeHealthy: Waiting for",
		Action:          OverrideTransitioning,
	},
	// MGMT Cluster: change the state to "paused" when the CAPI cluster is paused,
	// which is more descriptive than "updating".
	{
		ConditionType:   "Updated",
		Reason:          "Waiting",
		Status:          "Unknown",
		MessageContains: "CAPI cluster or RKEControlPlane is paused",
		Kind:            "Cluster",
		APIVersion:      "management.cattle.io/v3",
		Action:          OverrideTransitioning,
		StateOverride:   "paused",
	},
	// Provisioning Cluster: change the state to "paused" when the CAPI cluster is paused,
	// which is more descriptive than "updating" and "provisioning".
	{
		ConditionType:   "Updated",
		Status:          "Unknown",
		Reason:          "Waiting",
		MessageContains: "CAPI cluster or RKEControlPlane is paused",
		Kind:            "Cluster",
		APIGroup:        "provisioning.cattle.io",
		Action:          OverrideTransitioning,
		StateOverride:   "paused",
	},
	{
		ConditionType:   "Provisioned",
		Status:          "Unknown",
		Reason:          "Waiting",
		MessageContains: "CAPI cluster or RKEControlPlane is paused",
		Kind:            "Cluster",
		APIGroup:        "provisioning.cattle.io",
		Action:          OverrideTransitioning,
		StateOverride:   "paused",
	},
}
