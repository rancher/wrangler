package summary

import (
	"github.com/rancher/wrangler/pkg/data"
)

func checkCattleTypes(obj data.Object, condition []Condition, summary Summary) Summary {
	return checkRelease(obj, condition, summary)
}

func checkRelease(obj data.Object, _ []Condition, summary Summary) Summary {
	if !isKind(obj, "Release", "catalog.cattle.io") {
		return summary
	}
	if obj.String("status", "summary", "state") != "deployed" {
		return summary
	}
	for _, resources := range obj.Slice("spec", "resources") {
		summary.Relationships = append(summary.Relationships, Relationship{
			Name:       resources.String("name"),
			Kind:       resources.String("kind"),
			APIVersion: resources.String("apiVersion"),
			Type:       "manages",
		})
	}
	return summary
}
