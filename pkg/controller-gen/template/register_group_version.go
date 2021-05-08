package generator

var RegisterGroupVersionTemplate = `// +k8s:deepcopy-gen=package
// +groupName={{.group}}
package {{.version}}

import (
	{{.Package}} "github.com/rancher/rancher/pkg/apis/{{.group}}"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
    {{range .Names}}
    {{.Name}}ResourceName                           = "{{.Name | toLower | toPlural}}"{{end}} 
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: {{.Package}}.GroupName, Version: "{{.version}}"}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion, {{range .Names}}
        &{{.Name}}{},
        &{{.Name}}List{},{{end}}
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}`
