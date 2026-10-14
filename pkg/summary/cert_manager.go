package summary

import (
	"errors"
	"strings"
	"time"

	"github.com/rancher/wrangler/v3/pkg/data"
	"github.com/rancher/wrangler/v3/pkg/data/convert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var certKindWhitelist = map[string]bool{"Certificate": true, "CertificateRequest": true}

func checkCertificate(obj data.Object, conditions []Condition, summary Summary) Summary {
	if !strings.Contains(obj.String("apiVersion"), "cert-manager.io/") {
		return summary
	}
	if !certKindWhitelist[obj.String("kind")] {
		return summary
	}
	status := getStatusData(obj)
	failedIssuanceAttemptsValue, ok := status["failedIssuanceAttempts"]
	if ok {
		failedIssuanceAttempts, err := convert.ToNumber(failedIssuanceAttemptsValue)
		if err == nil && failedIssuanceAttempts > 0 {
			summary.State = "error"
			summary.Error = true
			summary.Message = []string{"Too many failed issuance-attempts"}
			return summary
		}
	}
	if len(conditions) == 0 {
		// wait for later runs when conditions have been added
		return summary
	}

	if !summary.Error {
		for _, condition := range conditions {
			message := condition.Message()
			if strings.Contains(strings.ToLower(message), "issuing certificate") {
				summary.State = "in-progress"
				summary.Message = append(summary.Message, message)
				return summary
			}
		}

		statusObj := getStatusData(obj)
		if len(statusObj) != 0 {
			if notAfterTimestamp, ok := statusObj["notAfter"]; ok {
				if notAfterTime, err := toTime(notAfterTimestamp); err == nil && time.Now().After(notAfterTime) {
					summary.State = "expired"
					summary.Error = true
					summary.Message = append(summary.Message, "certificate has expired")
					return summary
				}
			}
			if summary.State == "" {
				if renewalTimestamp, ok := statusObj["renewalTime"]; ok {
					if renewalTime, err := toTime(renewalTimestamp); err == nil && !renewalTime.IsZero() {
						summary.State = "expiring"
						return summary
					}
				}
			}
		}
	}

	if !summary.Error {
		foundNonReady := false
		for _, condition := range conditions {
			if condition.Type() != "Ready" {
				foundNonReady = true
				continue
			}

			switch metav1.ConditionStatus(condition.Status()) {
			case metav1.ConditionTrue:
				if summary.State == "" {
					summary.State = "active"
				}
			case metav1.ConditionFalse:
				summary.State = "error"
				summary.Error = true
				if message := condition.Message(); message != "" {
					summary.Message = append(summary.Message, message)
				}
			case metav1.ConditionUnknown:
				if summary.State == "" {
					summary.State = "pending"
				}
			}
			if summary.Error {
				break
			}
		}
		if !foundNonReady && summary.State == "" {
			summary.State = "pending"
		}
	}

	summary.Message = dedupMessage(summary.Message)
	if summary.Message == nil {
		summary.Message = []string{}
	}
	return summary
}

func getStatusData(obj data.Object) data.Object {
	switch v := obj["status"].(type) {
	case data.Object:
		return v
	case map[string]interface{}:
		return data.Object(v)
	default:
		return data.Object{}
	}
}

func toTime(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case *time.Time:
		if v == nil {
			return time.Time{}, errors.New("invalid time")
		}
		return *v, nil
	case string:
		return time.Parse(time.RFC3339, v)
	default:
		return time.Time{}, errors.New("invalid time")
	}
}
