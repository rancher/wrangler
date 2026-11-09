package summary

import (
	"errors"
	"fmt"
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

func pluralize(word string, count int64) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", count, word)
}

// CertExpiringSoonWindow is how close to its notAfter time a Certificate
// must be before it is reported as "expiring".
const CertExpiringSoonWindow = 30 * 24 * time.Hour

// checkCertificateResource summarizes a cert-manager Certificate. The order of
// the checks is significant and matches the order used by the Rancher UI, so
// that the state the backend reports (and sorts/filters on) is the state the
// user sees: expired, expiring, in-progress, error, active, then pending.
func checkCertificateResource(obj data.Object, conditions []Condition, summary Summary) Summary {
	statusObj := getStatusData(obj)
	now := time.Now()

	// Expiry beats everything: a certificate can report Ready while its
	// stored material is stale.
	notAfter, hasNotAfter := statusTime(statusObj, "notAfter")
	if hasNotAfter && notAfter.Before(now) {
		summary.State = "expired"
		summary.Error = true
		summary.Message = append(summary.Message, "certificate has expired")
		return summary
	}
	renewalTime, hasRenewalTime := statusTime(statusObj, "renewalTime")
	if (hasRenewalTime && renewalTime.Before(now)) || (hasNotAfter && notAfter.Before(now.Add(CertExpiringSoonWindow))) {
		summary.State = "expiring"
		summary.Error = false
		return summary
	}

	for _, condition := range conditions {
		if condition.Type() == "Issuing" && metav1.ConditionStatus(condition.Status()) == metav1.ConditionTrue {
			summary.State = "in-progress"
			summary.Error = false
			if message := condition.Message(); message != "" {
				summary.Message = append(summary.Message, message)
			}
			return summary
		}
	}

	failedIssuanceAttempts, err := convert.ToNumber(data.GetValueN(statusObj, "failedIssuanceAttempts"))
	if err == nil && failedIssuanceAttempts > 0 {
		summary.State = "error"
		summary.Error = true
		summary.Message = []string{pluralize("failed issuance attempt", failedIssuanceAttempts)}
		return summary
	}

	if len(conditions) == 0 {
		// Leave it to the other summarizers, and to later runs once
		// conditions have been added.
		return summary
	}

	for _, condition := range conditions {
		if condition.Type() != "Ready" {
			continue
		}
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
			summary.Error = false
			return summary
		}
	}
	summary.State = "pending"
	summary.Error = false
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

// statusTime returns the time stored at status[field], if there is a valid one.
func statusTime(statusObj data.Object, field string) (time.Time, bool) {
	value, ok := statusObj[field]
	if !ok {
		return time.Time{}, false
	}
	t, err := toTime(value)
	return t, err == nil
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
