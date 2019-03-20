package generators

import (
	"strings"

	"k8s.io/gengo/namer"
)

const (
	GenericPackage = "github.com/rancher/wrangler/pkg/generic"
)

func groupPath(group string) string {
	g := strings.Replace(strings.Split(group, ".")[0], "-", "", -1)
	return handleCore(g)
}

func handleCore(group string) string {
	if group == "" {
		return "core"
	}
	return group
}

func upperLowercase(name string) string {
	return namer.IC(strings.ToLower(groupPath(name)))
}
