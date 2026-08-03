package summary

import (
	"strings"

	"github.com/rancher/wrangler/v3/pkg/data"
)

// operationKinds are the operation.cattle.io kinds that embed the shared
// OperationStatus contract.
var operationKinds = map[string]bool{
	"EncryptionKeyRotation": true,
	"ETCDSnapshotRestore":   true,
	"ETCDSnapshotSave":      true,
	"CertificateRotation":   true,
}

// isOperation returns true if the object is a Rancher operation
// (operation.cattle.io/*/{EncryptionKeyRotation,ETCDSnapshotRestore,ETCDSnapshotSave,CertificateRotation}).
func isOperation(obj data.Object) bool {
	return strings.HasPrefix(obj.String("apiVersion"), "operation.cattle.io/") &&
		operationKinds[obj.String("kind")]
}

// operationStates maps each operation condition to the summary it implies, in
// priority order.
var operationStates = []struct {
	condition     string
	state         string
	err           bool
	transitioning bool
}{
	{condition: "Failed", state: "failed", err: true},
	{condition: "Canceled", state: "canceled", err: true},
	{condition: "Succeeded", state: "succeeded"},
	{condition: "Paused", state: "paused", transitioning: true},
	{condition: "InProgress", state: "in-progress", transitioning: true},
	{condition: "Pending", state: "pending", transitioning: true},
}

// checkOperationTransitioning computes summary state for Rancher operations.
//
// Operations carry one condition per lifecycle phase (Pending, InProgress,
// Succeeded, Failed, Canceled) plus Paused. These mirror status.phase rather
// than expressing readiness, so True is the normal state for every one of them
// and the generic TransitioningUnknown/False/True tables do not apply.
//
// The first condition in operationStates whose status is True wins. Priority
// matters because Rancher does not clear the conditions of prior phases: a
// canceled operation retains InProgress=True, and terminal conditions must
// outrank Paused so that a paused-but-finished operation reads as finished.
//
// When conditions exist but none are True the summary passes through untouched,
// leaving checkPhase to name the state.
func checkOperationTransitioning(conditions []Condition, summary Summary) Summary {
	for _, state := range operationStates {
		for _, c := range conditions {
			if c.Type() != state.condition || c.Status() != "True" {
				continue
			}

			summary.State = state.state
			if state.err {
				summary.Error = true
			}
			if state.transitioning {
				summary.Transitioning = true
			}
			if msg := c.Message(); msg != "" {
				summary.Message = append(summary.Message, msg)
			}

			return summary
		}
	}

	// Rancher writes a condition alongside every phase, so an operation with no
	// conditions has not been reconciled yet. Without this, the kstatus fallback
	// in checkStandard reports a brand-new operation as active.
	if len(conditions) == 0 {
		summary.State = "pending"
		summary.Transitioning = true
	}

	return summary
}
