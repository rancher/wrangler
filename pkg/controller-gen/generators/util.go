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
	if g == "" {
		return "core"
	}
	return g
}

func upperLowercase(name string) string {
	return namer.IC(strings.ToLower(groupPath(name)))
}
