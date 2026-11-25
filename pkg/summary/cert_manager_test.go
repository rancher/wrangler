package summary

import (
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
				Message: []string{"1 failed issuance attempt"},
			},
		},
		{
			name:     "failed issuance-attempts > 1",
			certName: "pat",
			inputStatus: map[string]interface{}{
				"failedIssuanceAttempts": 77,
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
				Message: []string{"77 failed issuance attempts"},
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
			name:     "will expire in less than 7 days",
			certName: "tony",
			inputStatus: map[string]interface{}{
				"notAfter":   tomorrow,
				"conditions": []Condition{readyCondition},
			},
			expected: Summary{
				State:   "expiring",
				Error:   false,
				Message: nil,
			},
		},
		{
			name:     "active as the cert will be good for > 10% of its age",
			certName: "tony",
			inputStatus: map[string]interface{}{
				"notBefore":  yesterday,
				"notAfter":   tomorrow,
				"conditions": []Condition{readyCondition},
			},
			expected: Summary{
				State:   "active",
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
				Message: []string(nil),
			},
		},
		{
			name:     "is an error",
			certName: "dave_m",
			inputStatus: map[string]interface{}{
				"conditions": []Condition{
					NewCondition("Ready", "False", "", "oops I did it again"),
				},
			},
			expected: Summary{
				State:   "error",
				Error:   true,
				Message: []string{"oops I did it again"},
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
				Message: []string(nil),
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
			actual := checkCertificate(obj, conditions, Summary{})
			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestCheckCertificateRequest(t *testing.T) {
	type testCase struct {
		name            string
		inputStatus     data.Object
		inputConditions []Condition
		expected        Summary
	}
	readyCondition := NewCondition("Ready", "True", "", "")

	cases := []testCase{
		{
			name:            "no conditions",
			inputStatus:     map[string]interface{}{},
			inputConditions: []Condition{},
			expected:        Summary{},
		},
		{
			name:            "ready true is active",
			inputStatus:     map[string]interface{}{},
			inputConditions: []Condition{readyCondition},
			expected: Summary{
				State:   "active",
				Error:   false,
				Message: []string(nil),
			},
		},
		{
			name:        "denied true is denied",
			inputStatus: map[string]interface{}{},
			inputConditions: []Condition{
				NewCondition("Denied", "True", "", "request denied"),
			},
			expected: Summary{
				State:   "denied",
				Error:   true,
				Message: []string{"request denied"},
			},
		},
		{
			name:        "denied/false is pending",
			inputStatus: map[string]interface{}{},
			inputConditions: []Condition{
				NewCondition("Denied", "False", "", "can't even successfully deny a request"),
			},
			expected: Summary{
				State:   "pending",
				Error:   false,
				Message: []string(nil),
			},
		},
		{
			name:        "invalid request is error",
			inputStatus: map[string]interface{}{},
			inputConditions: []Condition{
				NewCondition("InvalidRequest", "True", "", "bad request"),
			},
			expected: Summary{
				State:   "error",
				Error:   true,
				Message: []string{"bad request"},
			},
		},
		{
			name:        "ready failed reason is error",
			inputStatus: map[string]interface{}{},
			inputConditions: []Condition{
				NewCondition("Ready", "False", "Failed", "issuance failed"),
			},
			expected: Summary{
				State:   "error",
				Error:   true,
				Message: []string{"issuance failed"},
			},
		},
		{
			name:        "approved is in progress",
			inputStatus: map[string]interface{}{},
			inputConditions: []Condition{
				NewCondition("Approved", "True", "", "approved"),
			},
			expected: Summary{
				State:   "in-progress",
				Error:   false,
				Message: []string(nil),
			},
		},
		{
			name:        "anything else is pending",
			inputStatus: map[string]interface{}{},
			inputConditions: []Condition{
				NewCondition("0-wheel wheelbarrow", "", "", ""),
			},
			expected: Summary{
				State:   "pending",
				Error:   false,
				Message: []string(nil),
			},
		},
		{
			name:        "not approved is pending",
			inputStatus: map[string]interface{}{},
			inputConditions: []Condition{
				NewCondition("Approved", "False", "", "waiting for approval"),
			},
			expected: Summary{
				State:   "pending",
				Error:   false,
				Message: []string{"waiting for approval"},
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
				"status": func() data.Object {
					status := c.inputStatus
					status["conditions"] = c.inputConditions
					return status
				}(),
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

func TestCheckIssuerKind(t *testing.T) {
	type testCase struct {
		name            string
		kind            string
		inputConditions []Condition
		expected        Summary
	}
	readyCondition := NewCondition("Ready", "True", "", "")
	cases := []testCase{
		{
			name:            "no conditions",
			inputConditions: []Condition{},
			expected:        Summary{},
		},
		{
			name:            "ready false is error",
			inputConditions: []Condition{NewCondition("Ready", "False", "", "issuer failed")},
			expected: Summary{
				State:   "error",
				Error:   true,
				Message: []string{"issuer failed"},
			},
		},
		{
			name:            "ready true is active",
			inputConditions: []Condition{readyCondition},
			expected: Summary{
				State:   "active",
				Error:   false,
				Message: []string(nil),
			},
		},
		{
			name:            "anything else for a ready type is pending",
			inputConditions: []Condition{NewCondition("Ready", "Unknown", "", "waiting for a miracle")},
			expected: Summary{
				State:   "pending",
				Error:   false,
				Message: []string(nil),
			},
		},
		{
			name:            "other types end up as errors",
			inputConditions: []Condition{NewCondition("NanoNano", "", "", "got a problem here")},
			expected: Summary{
				State:   "NanoNano",
				Error:   true,
				Message: []string{"got a problem here"},
			},
		},
	}
	for _, kind := range []string{"Issuer", "ClusterIssuer"} {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				obj := data.Object{
					"apiVersion": "cert-manager.io/v1",
					"kind":       kind,
					"metadata": map[string]interface{}{
						"name":      c.name,
						"namespace": "default",
					},
					"status": data.Object{
						"conditions": c.inputConditions,
					},
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
}

// Verify non-supported GVKs don't go through summary analysis
func TestCheckNonCertificate(t *testing.T) {
	obj := data.Object{
		"apiVersion": "cert-manager.io/v1",
		"kind":       "SuedeMohairJacket",
		"metadata": map[string]interface{}{
			"name":      "joe",
			"namespace": "default",
		},
		"status": data.Object{
			"conditions": []Condition{NewCondition("Ready", "True", "", "")},
		},
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
	assert.Equal(t, Summary{}, actual)
}
