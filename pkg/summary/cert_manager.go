package summary

import (
	"errors"
	"strings"
	"time"

	"github.com/rancher/wrangler/v3/pkg/data"
	"github.com/rancher/wrangler/v3/pkg/data/convert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func checkCertificate(obj data.Object, conditions []Condition, summary Summary) Summary {
	if !strings.Contains(obj.String("apiVersion"), "cert-manager.io/") {
		return summary
	}
	switch obj.String("kind") {
	case "CertificateRequest":
		return checkCertificateRequest(obj, conditions, summary)
	case "Certificate":
		return checkCertificateResource(obj, conditions, summary)
	case "Issuer":
		return checkIssuer(obj, conditions, summary)
	case "ClusterIssuer":
		return checkIssuer(obj, conditions, summary)
	default:
		return summary
	}
}

func checkCertificateResource(obj data.Object, conditions []Condition, summary Summary) Summary {
	statusObj := getStatusData(obj)
	failedIssuanceAttempts, err := convert.ToNumber(data.GetValueN(statusObj, "failedIssuanceAttempts"))
	if err == nil && failedIssuanceAttempts > 0 {
		summary.State = "error"
		summary.Error = true
		summary.Message = []string{"Too many failed issuance-attempts"}
		return summary
	}
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
				if renewalTime, err := toTime(renewalTimestamp); err == nil {
					if time.Now().After(renewalTime) {
						summary.State = "expiring"
						return summary
					}
				}
			}
		}
	}
	if len(conditions) == 0 || summary.Error {
		// wait for later runs when conditions have been added
		return summary
	}

	for _, condition := range conditions {
		message := condition.Message()
		if strings.Contains(strings.ToLower(message), "issuing certificate") {
			summary.State = "in-progress"
			summary.Message = append(summary.Message, message)
			return summary
		}
	}

	processReadyCondition(conditions, &summary, "pending")
	return summary
}

func checkCertificateRequest(obj data.Object, conditions []Condition, summary Summary) Summary {
	if len(conditions) == 0 {
		// wait for later runs when conditions have been added
		return summary
	}
	if summary.Error {
		return summary
	}

	for _, condition := range conditions {
		if condition.Type() == "Denied" && metav1.ConditionStatus(condition.Status()) == metav1.ConditionTrue {
			summary.State = "denied"
			summary.Error = true
			if message := condition.Message(); message != "" {
				summary.Message = append(summary.Message, message)
			}
			return summary
		}
	}
	for _, condition := range conditions {
		if condition.Type() == "InvalidRequest" && metav1.ConditionStatus(condition.Status()) == metav1.ConditionTrue {
			summary.State = "error"
			summary.Error = true
			if message := condition.Message(); message != "" {
				summary.Message = append(summary.Message, message)
			}
			return summary
		}
	}
	for _, condition := range conditions {
		if condition.Type() != "Ready" {
			continue
		}
		switch metav1.ConditionStatus(condition.Status()) {
		case metav1.ConditionTrue:
			summary.State = "active"
			return summary
		case metav1.ConditionFalse:
			if condition.Reason() == "Failed" {
				summary.State = "error"
				summary.Error = true
				if message := condition.Message(); message != "" {
					summary.Message = append(summary.Message, message)
				}
				return summary
			}
		}
	}
	for _, condition := range conditions {
		if condition.Type() == "Approved" {
			if metav1.ConditionStatus(condition.Status()) == metav1.ConditionTrue {
				summary.State = "in-progress"
				return summary
			}
			summary.State = "pending"
			if message := condition.Message(); message != "" {
				summary.Message = append(summary.Message, message)
			}
			return summary
		}
	}
	summary.State = "pending"
	return summary
}

func checkIssuer(obj data.Object, conditions []Condition, summary Summary) Summary {
	if len(conditions) == 0 {
		return summary
	}
	if summary.Error {
		return summary
	}

	for _, condition := range conditions {
		if tp := condition.Type(); tp != "Ready" {
			summary.State = tp
			summary.Error = true
			if message := condition.Message(); message != "" {
				summary.Message = append(summary.Message, message)
			}
			return summary
		}
	}
	for _, condition := range conditions {
		switch metav1.ConditionStatus(condition.Status()) {
		case metav1.ConditionFalse:
			summary.State = "error"
			summary.Error = true
			if message := condition.Message(); message != "" {
				summary.Message = append(summary.Message, message)
			}
			return summary
		case metav1.ConditionTrue:
			summary.State = "active"
			return summary
		case metav1.ConditionUnknown:
			summary.State = "pending"
			return summary
		}
	}
	summary.State = "pending"
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

func processReadyCondition(conditions []Condition, summary *Summary, defaultState string) {
	foundReadyTrue := false
	foundReadyOther := false
	for _, condition := range conditions {
		if condition.Type() != "Ready" {
			summary.State = "error"
			summary.Error = true
			if message := condition.Message(); message != "" {
				summary.Message = append(summary.Message, message)
			}
			return
		}
		switch metav1.ConditionStatus(condition.Status()) {
		case metav1.ConditionTrue:
			foundReadyTrue = true
		case metav1.ConditionFalse:
			summary.State = "error"
			summary.Error = true
			if message := condition.Message(); message != "" {
				summary.Message = append(summary.Message, message)
			}
			return
		case metav1.ConditionUnknown:
			if summary.State == "" {
				summary.State = "pending"
			}
			foundReadyOther = true
		}
	}
	if summary.State == "" {
		if !foundReadyTrue {
			// If we found an error, we returned inside the loop
			summary.State = defaultState
			summary.Error = true
		}
		if !foundReadyOther {
			summary.State = "active"
		}
	}
	return
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
