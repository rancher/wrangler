package generators

import (
	"path"
	"path/filepath"

	args "github.com/rancher/wrangler/v3/pkg/controller-gen/args"
	"github.com/sirupsen/logrus"
	"k8s.io/gengo/v2"
	"k8s.io/gengo/v2/generator"
	oaargs "k8s.io/kube-openapi/cmd/openapi-gen/args"
	oa "k8s.io/kube-openapi/pkg/generators"
)

type ModelsGenerator struct {
}

func NewOpenAPIGenerator() *ModelsGenerator {
	return &ModelsGenerator{}
}

func (mg *ModelsGenerator) GetOpenAPITargets(context *generator.Context, customArgs *args.CustomArgs) []generator.Target {
	openAPIArgs := oaargs.New()
	openAPIArgs.OutputDir = filepath.Join(customArgs.OutputBase, customArgs.Options.OutputPackage, "openapi")
	openAPIArgs.OutputFile = "zz_generated_openapi.go"
	openAPIArgs.OutputPkg = filepath.Join(customArgs.Options.OutputPackage, "openapi")
	openAPIArgs.GoHeaderFile = customArgs.Options.Boilerplate
	openAPIArgs.OutputModelNameFile = "zz_generated.model_name.go"

	if err := openAPIArgs.Validate(); err != nil {
		logrus.Fatalf("Failed to validate openapi-gen args: %v", err)
	}

	boilerplate, err := gengo.GoBoilerplate(openAPIArgs.GoHeaderFile, gengo.StdBuildTag, gengo.StdGeneratedBy)
	if err != nil {
		logrus.Fatalf("Failed to generate openapi boilerplate: %v", err)
	}

	return append(oa.GetOpenAPITargets(context, openAPIArgs, boilerplate), generator.SimpleTarget{
		PkgName:       "main",
		PkgPath:       path.Join(customArgs.Package, "openapi", "cmd", "models-schema"),
		PkgDir:        filepath.Join(customArgs.OutputBase, customArgs.Package, "openapi", "cmd", "models-schema"),
		HeaderComment: customArgs.BoilerplateContent,
		GeneratorsFunc: func(context *generator.Context) []generator.Generator {
			return []generator.Generator{
				ModelsSchemaCmdGo(customArgs),
			}
		}})
}

func (mg *ModelsGenerator) GetModelNameTargets(context *generator.Context, customArgs *args.CustomArgs) []generator.Target {
	openAPIArgs := oaargs.New()
	openAPIArgs.OutputDir = filepath.Join(customArgs.OutputBase, customArgs.Options.OutputPackage, "openapi")
	openAPIArgs.OutputFile = "zz_generated_openapi.go"
	openAPIArgs.OutputPkg = filepath.Join(customArgs.Options.OutputPackage, "openapi")
	openAPIArgs.GoHeaderFile = customArgs.Options.Boilerplate
	openAPIArgs.OutputModelNameFile = "zz_generated.model_name.go"

	if err := openAPIArgs.Validate(); err != nil {
		logrus.Fatalf("Failed to validate openapi-gen args: %v", err)
	}

	boilerplate, err := gengo.GoBoilerplate(openAPIArgs.GoHeaderFile, gengo.StdBuildTag, gengo.StdGeneratedBy)
	if err != nil {
		logrus.Fatalf("Failed to generate openapi boilerplate: %v", err)
	}

	return oa.GetModelNameTargets(context, openAPIArgs, boilerplate)
}
