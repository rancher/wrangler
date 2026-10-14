package summary

import (
	"fmt"
	"testing"
	"time"

	"github.com/rancher/wrangler/v3/pkg/data"
	"github.com/stretchr/testify/assert"
)

func TestCheckCertificate(t *testing.T) {
	type testCase struct {
		name        string
		certName    string
		inputStatus data.Object
		expected    Summary
	}
	readyCondition := NewCondition("Ready", "True", "", "")
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)

	cases := []testCase{
		{
			name:     "no conditions",
			certName: "joe",
			inputStatus: map[string]interface{}{
				"conditions": []Condition{},
			},
			expected: Summary{},
		},
		{
			name:     "failed issuance-attempts > 0",
			certName: "pat",
			inputStatus: map[string]interface{}{
				"failedIssuanceAttempts": 1,
				"conditions": []Condition{
					{
						Object: map[string]interface{}{
							"type":   "Ready",
							"status": "True",
						},
					},
				},
			},
			expected: Summary{
				State:   "error",
				Error:   true,
				Message: []string{"Too many failed issuance-attempts"},
			},
		},
		{
			name:     "issuing certificate",
			certName: "bob",
			inputStatus: map[string]interface{}{
				"conditions": []Condition{
					NewCondition("Ready", "True", "", "We're busy issuing certificates"),
				},
			},
			expected: Summary{
				State:   "in-progress",
				Error:   false,
				Message: []string{"We're busy issuing certificates"},
			},
		},
		{
			name:     "has expired",
			certName: "roger",
			inputStatus: map[string]interface{}{
				"notAfter":   yesterday,
				"conditions": []Condition{readyCondition},
			},
			expected: Summary{
				State:   "expired",
				Error:   true,
				Message: []string{"certificate has expired"},
			},
		},
		{
			name:     "will expire soon",
			certName: "tony",
			inputStatus: map[string]interface{}{
				"renewalTime": tomorrow,
				"conditions":  []Condition{readyCondition},
			},
			expected: Summary{
				State:   "expiring",
				Error:   false,
				Message: nil,
			},
		},
		{
			name:     "is ready",
			certName: "fred",
			inputStatus: map[string]interface{}{
				"conditions": []Condition{readyCondition},
			},
			expected: Summary{
				State:   "active",
				Error:   false,
				Message: []string{},
			},
		},
		{
			name:     "is an error",
			certName: "dave_m",
			inputStatus: map[string]interface{}{
				"conditions": []Condition{
					NewCondition("Ready", "False", "", ""),
				},
			},
			expected: Summary{
				State:   "error",
				Error:   true,
				Message: []string{},
			},
		},
		{
			name:     "is pending",
			certName: "bob_m",
			inputStatus: map[string]interface{}{
				"conditions": []Condition{
					NewCondition("Ready", "Unknown", "", ""),
				},
			},
			expected: Summary{
				State:   "pending",
				Error:   false,
				Message: []string{},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			obj := data.Object{
				"id":         "default/failedissuances",
				"apiVersion": "cert-manager.io/v1",
				"kind":       "Certificate",
				"metadata": map[string]interface{}{
					"name":      c.name,
					"namespace": "default",
				},
				"status": c.inputStatus,
			}
			status, ok := obj["status"].(data.Object)
			if !ok {
				t.Fatalf("status was not a data.Object: %#v", obj["status"])
			}
			conditions, ok := status["conditions"].([]Condition)
			if !ok {
				t.Fatalf("conditions were not []Condition: %#v", status["conditions"])
			}
			if c.name == "failed issuance-attempts > 0" {
				fmt.Println("QQQ: stop here")
			}
			actual := checkCertificate(obj, conditions, Summary{})
			assert.Equal(t, c.expected, actual)
		})
	}
}

// CertificateRequests are similar to Certs but don't have
// status.failedIssuanceAttempts nor expiration times
func TestCheckCertificateRequest(t *testing.T) {
	type testCase struct {
		name        string
		certName    string
		inputStatus data.Object
		expected    Summary
	}
	readyCondition := NewCondition("Ready", "True", "", "")

	cases := []testCase{
		{
			name:     "no conditions",
			certName: "joe",
			inputStatus: map[string]interface{}{
				"conditions": []Condition{},
			},
			expected: Summary{},
		},
		{
			name:     "is ready",
			certName: "fred",
			inputStatus: map[string]interface{}{
				"conditions": []Condition{readyCondition},
			},
			expected: Summary{
				State:   "active",
				Error:   false,
				Message: []string{},
			},
		},
		{
			name:     "is an error",
			certName: "dave_m",
			inputStatus: map[string]interface{}{
				"conditions": []Condition{
					NewCondition("Ready", "False", "", ""),
				},
			},
			expected: Summary{
				State:   "error",
				Error:   true,
				Message: []string{},
			},
		},
		{
			name:     "is pending",
			certName: "bob_m",
			inputStatus: map[string]interface{}{
				"conditions": []Condition{
					NewCondition("Ready", "Unknown", "", ""),
				},
			},
			expected: Summary{
				State:   "pending",
				Error:   false,
				Message: []string{},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			obj := data.Object{
				"apiVersion": "cert-manager.io/v1",
				"kind":       "CertificateRequest",
				"metadata": map[string]interface{}{
					"name":      c.name,
					"namespace": "default",
				},
				"status": c.inputStatus,
			}
			status, ok := obj["status"].(data.Object)
			if !ok {
				t.Fatalf("status was not a data.Object: %#v", obj["status"])
			}
			conditions, ok := status["conditions"].([]Condition)
			if !ok {
				t.Fatalf("conditions were not []Condition: %#v", status["conditions"])
			}
			actual := checkCertificate(obj, conditions, Summary{})
			assert.Equal(t, c.expected, actual)
		})
	}
}

// Verify non-supported GVKs don't go through summary analysis
func TestCheckNonCertificate(t *testing.T) {
	type testCase struct {
		name        string
		certName    string
		inputStatus data.Object
		expected    Summary
	}
	readyCondition := NewCondition("Ready", "True", "", "")
	cases := []testCase{
		{
			name:     "no conditions",
			certName: "joe",
			inputStatus: map[string]interface{}{
				"conditions": []Condition{readyCondition},
			},
			expected: Summary{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, kind := range []string{"Issuer", "SuedeMohairJacket"} {
				obj := data.Object{
					"apiVersion": "cert-manager.io/v1",
					"kind":       kind,
					"metadata": map[string]interface{}{
						"name":      c.name,
						"namespace": "default",
					},
					"status": c.inputStatus,
				}
				status, ok := obj["status"].(data.Object)
				if !ok {
					t.Fatalf("status was not a data.Object: %#v", obj["status"])
				}
				conditions, ok := status["conditions"].([]Condition)
				if !ok {
					t.Fatalf("conditions were not []Condition: %#v", status["conditions"])
				}
				actual := checkCertificate(obj, conditions, Summary{})
				assert.Equal(t, c.expected, actual)
			}
		})
	}
}
