package args

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/gengo/v2/types"
)

type CustomArgs struct {
	// Package is the directory path where generated code output will be located
	Package       string
	ImportPackage string
	TypesByGroup  map[schema.GroupVersion][]*types.Name
	Options       Options
	OutputBase    string
	// BoilerplateContent is the actual boilerplate content that has been
	// read. It will be added to every generated files.
	BoilerplateContent []byte
}

type Options struct {
	ImportPackage string
	// OutputPackage is the directory path where generated code output will be located
	OutputPackage string
	// ModulePath is the GOPATH style repo root,
	// used when the repos module path is longer than 3 segments (e.g. "source.com/org/sub/repo").
	// If empty, the first 3 segments of OutputPackage are used.
	ModulePath string
	Groups     map[string]Group
	// Boilerplate is the filepath to a boilerplate file whose content will
	// be added to every generated files.
	Boilerplate string
	// ExternalApplyConfigurations appends to https://pkg.go.dev/k8s.io/code-generator@v0.37.2/cmd/applyconfiguration-gen/args
	// additional reference documentation from controller-gen generators
	// https://book.kubebuilder.io/reference/controller-gen.html#generators
	// by default args.New() prepopulates common ExternalApplyConfigurations
	//		{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "Condition"}:                "k8s.io/client-go/applyconfigurations/meta/v1",
	//		{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "DeleteOptions"}:            "k8s.io/client-go/applyconfigurations/meta/v1",
	//		{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "LabelSelector"}:            "k8s.io/client-go/applyconfigurations/meta/v1",
	//		{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "LabelSelectorRequirement"}: "k8s.io/client-go/applyconfigurations/meta/v1",
	//		{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "ManagedFieldsEntry"}:       "k8s.io/client-go/applyconfigurations/meta/v1",
	//		{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "ObjectMeta"}:               "k8s.io/client-go/applyconfigurations/meta/v1",
	//		{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "OwnerReference"}:           "k8s.io/client-go/applyconfigurations/meta/v1",
	//		{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "TypeMeta"}:                 "k8s.io/client-go/applyconfigurations/meta/v1",
	//	},
	// Use this to add more mappings to the append / replace existing mapping
	ExternalApplyConfigurations map[types.Name]string
}

type Type struct {
	Version string
	Package string
	Name    string
}

type Group struct {
	// Types is a slice of the following types
	// Instance of any struct: used for reflection to describe the type
	// string: a directory that will be listed (non-recursively) for types
	// Type: a description of a type
	Types         []interface{}
	GenerateTypes bool
	// Generate clientsets
	GenerateClients             bool
	OutputControllerPackageName string
	// Generate listers
	GenerateListers bool
	// Generate informers
	GenerateInformers bool
	// Generate openapi
	GenerateOpenAPI bool
	// Open API model package name
	OpenAPIModelPackageName string
	// OpenAPI extra dependencies
	OpenAPIDependencies []string
	// The package name of the API types
	PackageName string
}
