package generator

var ListTemplate = `
// +k8s:deepcopy-gen=package
// +groupName={{.Group}}
package {{.Version}}

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

{{range .Names}}
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// {{.Name}}List is a list of {{.Name}} resources
type {{.Name}}List struct {
	metav1.TypeMeta %BACK%json:",inline"%BACK%
	metav1.ListMeta %BACK%json:"metadata"%BACK%

	Items []{{.Name}} %BACK%json:"items"%BACK%
}

func New{{.Name}}(namespace, name string, obj {{.Name}}) *{{.Name}} {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("{{.Name}}").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}
{{end}}
`
