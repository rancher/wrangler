package generators

import (
	"path"

	"github.com/rancher/wrangler/v3/pkg/controller-gen/args"
	"k8s.io/gengo/v2/generator"
)

func ModelsSchemaCmdGo(customArgs *args.CustomArgs) generator.Generator {
	return &modelsSchemaCmdGo{
		customArgs: customArgs,
		GoGenerator: generator.GoGenerator{
			OutputFilename: "main.go",
			OptionalBody:   []byte(modelsSchemaCmdBody),
		},
	}
}

type modelsSchemaCmdGo struct {
	generator.GoGenerator

	customArgs *args.CustomArgs
}

func (m *modelsSchemaCmdGo) Imports(*generator.Context) []string {
	return []string{
		"encoding/json",
		"fmt",
		"os",
		path.Join(m.customArgs.Package, "openapi"),
		"k8s.io/kube-openapi/pkg/common",
		"k8s.io/kube-openapi/pkg/validation/spec",
	}
}

var modelsSchemaCmdBody = `
// Outputs openAPI schema JSON containing the schema definitions in zz_generated.openapi.go.
func main() {
	err := output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed: %v", err) //nolint:errcheck
		os.Exit(1)
	}
}

func output() error {
	refFunc := func(name string) spec.Ref {
		return spec.MustCreateRef(fmt.Sprintf("#/definitions/%s", name))
	}
	defs := openapi.GetOpenAPIDefinitions(refFunc)
	schemaDefs := make(map[string]spec.Schema, len(defs))
	for k, v := range defs {
		// Replace top-level schema with v2 if a v2 schema is embedded
		// so that the output of this program is always in OpenAPI v2.
		// This is done by looking up an extension that marks the embedded v2
		// schema, and, if the v2 schema is found, make it the resulting schema for
		// the type.
		if schema, ok := v.Schema.Extensions[common.ExtensionV2Schema]; ok {
			if v2Schema, isOpenAPISchema := schema.(spec.Schema); isOpenAPISchema {
				schemaDefs[k] = v2Schema
				continue
			}
		}

		schemaDefs[k] = v.Schema
	}
	data, err := json.Marshal(&spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Definitions: schemaDefs,
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Title:   "Kubernetes",
					Version: "unversioned",
				},
			},
			Swagger: "2.0",
		},
	})
	if err != nil {
		return fmt.Errorf("error serializing api definitions: %w", err)
	}
	os.Stdout.Write(data) // nolint:errcheck
	return nil
}
`
