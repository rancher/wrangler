package yaml

import (
	"bytes"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// TestExport checks Export's cleanup: status and server-managed kubectl
// annotations are removed.
func TestExport(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":        "test-deploy",
			"annotations": map[string]interface{}{"kubectl.kubernetes.io/last-applied-configuration": "{}"},
		},
		"status": map[string]interface{}{"readyReplicas": int64(3)},
	}}

	b, err := Export(obj)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if s := string(b); strings.Contains(s, "status") || strings.Contains(s, "kubectl.kubernetes.io") {
		t.Errorf("Export() did not clean object: %q", s)
	}
}

// TestToBytes checks wrangler's own framing: multiple objects are joined with a
// "---" separator.
func TestToBytes(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]interface{}{"name": "d"},
	}}

	b, err := ToBytes([]runtime.Object{obj, obj})
	if err != nil {
		t.Fatalf("ToBytes() error = %v", err)
	}
	if !bytes.Contains(b, []byte("\n---\n")) {
		t.Errorf("ToBytes() missing document separator: %q", string(b))
	}
}
