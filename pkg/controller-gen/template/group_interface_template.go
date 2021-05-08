package generator

var GroupInterfaceTemplate = `package {{.Version}}

import (
	"github.com/rancher/lasso/pkg/controller"
	{{.Version}} "github.com/rancher/rancher/pkg/apis/{{.Group}}/{{.Version}}"
	"github.com/rancher/wrangler/pkg/schemes"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func init() {
	schemes.Register({{.Version}}.AddToScheme)
}

type Interface interface {
    {{range .Names}}
	{{.Name}}() {{.Name}}Controller{{end}}
}

func New(controllerFactory controller.SharedControllerFactory) Interface {
	return &version{
		controllerFactory: controllerFactory,
	}
}

type version struct {
	controllerFactory controller.SharedControllerFactory
}

{{range .Names}}
func (c *version) {{.Name}}() {{.Name}}Controller {
	return New{{.Name}}Controller(schema.GroupVersionKind{Group: "{{$.Group}}", Version: "{{$.Version}}", Kind: "{{.Name}}"}, "{{.Name | toLower | toPlural}}", {{.Namespaced}}, c.controllerFactory)
}
{{end}}
`
