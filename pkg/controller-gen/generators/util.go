package generators

import (
	"strings"
)

const (
	GenericPackage = "github.com/rancher/wrangler/pkg/generic"
)

func groupPath(group string) string {
	g := strings.Split(group, ".")[0]
	if g == "" {
		return "core"
	}
	return g
}
