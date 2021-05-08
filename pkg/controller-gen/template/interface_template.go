package generator

var InterfaceTemplate = `package {{.Package}}

import (
	"github.com/rancher/lasso/pkg/controller"
	{{.Version}} "github.com/rancher/rancher/pkg/generated/controllers/{{.Group}}/{{.Version}}"
)

type Interface interface {
	{{.Version | upper}}() {{.Version}}.Interface
}

type group struct {
	controllerFactory controller.SharedControllerFactory
}

// New returns a new Interface.
func New(controllerFactory controller.SharedControllerFactory) Interface {
	return &group{
		controllerFactory: controllerFactory,
	}
}

func (g *group) {{.Version | upper}}() {{.Version}}.Interface {
	return {{.Version}}.New(g.controllerFactory)
}
`
