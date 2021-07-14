package crd

import (
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/rancher/wrangler/pkg/data/convert"
	"github.com/rancher/wrangler/pkg/kv"
	"github.com/rancher/wrangler/pkg/name"
	"github.com/rancher/wrangler/pkg/schemas/openapi"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	// Ensure the gvks are loaded so that apply works correctly
	_ "github.com/rancher/wrangler/pkg/generated/controllers/apiextensions.k8s.io/v1"
)

type CRD struct {
	GVK            schema.GroupVersionKind
	PluralName     string
	SingularName   string
	NonNamespace   bool
	Schema         *apiextv1.JSONSchemaProps
	SchemaV1Beta1  *apiextv1beta1.JSONSchemaProps
	SchemaObject   interface{}
	Columns        []apiextv1.CustomResourceColumnDefinition
	ColumnsV1Beta1 []apiextv1beta1.CustomResourceColumnDefinition
	Status         bool
	Scale          bool
	Categories     []string
	ShortNames     []string
	Labels         map[string]string
	Annotations    map[string]string

	Override runtime.Object
}

func (c CRD) WithSchema(schema *apiextv1.JSONSchemaProps) CRD {
	c.Schema = schema
	return c
}

func (c CRD) WithSchemaV1Beta1(schema *apiextv1beta1.JSONSchemaProps) CRD {
	c.SchemaV1Beta1 = schema
	return c
}

func (c CRD) WithSchemaFromStruct(obj interface{}) CRD {
	c.SchemaObject = obj
	return c
}

func (c CRD) WithColumn(name, path string) CRD {
	c.Columns = append(c.Columns, apiextv1.CustomResourceColumnDefinition{
		Name:     name,
		Type:     "string",
		Priority: 0,
		JSONPath: path,
	})
	return c
}

func (c CRD) WithColumnV1Beta1(name, path string) CRD {
	c.ColumnsV1Beta1 = append(c.ColumnsV1Beta1, apiextv1beta1.CustomResourceColumnDefinition{
		Name:     name,
		Type:     "string",
		Priority: 0,
		JSONPath: path,
	})
	return c
}

func getType(obj interface{}) reflect.Type {
	if t, ok := obj.(reflect.Type); ok {
		return t
	}

	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func (c CRD) WithColumnsFromStruct(obj interface{}) CRD {
	c.Columns = append(c.Columns, readCustomColumns(getType(obj), ".")...)
	return c
}

func (c CRD) WithColumnsFromStructV1Beta1(obj interface{}) CRD {
	c.ColumnsV1Beta1 = append(c.ColumnsV1Beta1, readCustomColumnsV1Beta1(getType(obj), ".")...)
	return c
}

func fieldName(f reflect.StructField) string {
	jsonTag := f.Tag.Get("json")
	if jsonTag == "-" {
		return ""
	}
	name := strings.Split(jsonTag, ",")[0]
	if name == "" {
		return f.Name
	}
	return name
}

func tagToColumn(f reflect.StructField) (apiextv1.CustomResourceColumnDefinition, bool) {
	c := apiextv1.CustomResourceColumnDefinition{
		Name: f.Name,
		Type: "string",
	}

	columnDef, ok := f.Tag.Lookup("column")
	if !ok {
		return c, false
	}

	for k, v := range kv.SplitMap(columnDef, ",") {
		switch k {
		case "name":
			c.Name = v
		case "type":
			c.Type = v
		case "format":
			c.Format = v
		case "description":
			c.Description = v
		case "priority":
			p, _ := strconv.Atoi(v)
			c.Priority = int32(p)
		case "jsonpath":
			c.JSONPath = v
		}
	}

	return c, true
}

func tagToColumnV1Beta1(f reflect.StructField) (apiextv1beta1.CustomResourceColumnDefinition, bool) {
	c := apiextv1beta1.CustomResourceColumnDefinition{
		Name: f.Name,
		Type: "string",
	}

	columnDef, ok := f.Tag.Lookup("column")
	if !ok {
		return c, false
	}

	for k, v := range kv.SplitMap(columnDef, ",") {
		switch k {
		case "name":
			c.Name = v
		case "type":
			c.Type = v
		case "format":
			c.Format = v
		case "description":
			c.Description = v
		case "priority":
			p, _ := strconv.Atoi(v)
			c.Priority = int32(p)
		case "jsonpath":
			c.JSONPath = v
		}
	}

	return c, true
}

func readCustomColumns(t reflect.Type, path string) (result []apiextv1.CustomResourceColumnDefinition) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fieldName := fieldName(f)
		if fieldName == "" {
			continue
		}

		t := f.Type
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() == reflect.Struct {
			if f.Anonymous {
				result = append(result, readCustomColumns(t, path)...)
			} else {
				result = append(result, readCustomColumns(t, path+"."+fieldName)...)
			}
		} else {
			if col, ok := tagToColumn(f); ok {
				result = append(result, col)
			}
		}
	}

	return result
}

func readCustomColumnsV1Beta1(t reflect.Type, path string) (result []apiextv1beta1.CustomResourceColumnDefinition) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fieldName := fieldName(f)
		if fieldName == "" {
			continue
		}

		t := f.Type
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() == reflect.Struct {
			if f.Anonymous {
				result = append(result, readCustomColumnsV1Beta1(t, path)...)
			} else {
				result = append(result, readCustomColumnsV1Beta1(t, path+"."+fieldName)...)
			}
		} else {
			if col, ok := tagToColumnV1Beta1(f); ok {
				result = append(result, col)
			}
		}
	}

	return result
}

func (c CRD) WithCustomColumn(columns ...apiextv1.CustomResourceColumnDefinition) CRD {
	c.Columns = append(c.Columns, columns...)
	return c
}

func (c CRD) WithCustomColumnV1Beta1(columns ...apiextv1beta1.CustomResourceColumnDefinition) CRD {
	c.ColumnsV1Beta1 = append(c.ColumnsV1Beta1, columns...)
	return c
}

func (c CRD) WithStatus() CRD {
	c.Status = true
	return c
}

func (c CRD) WithScale() CRD {
	c.Scale = true
	return c
}

func (c CRD) WithCategories(categories ...string) CRD {
	c.Categories = categories
	return c
}

func (c CRD) WithGroup(group string) CRD {
	c.GVK.Group = group
	return c
}

func (c CRD) WithShortNames(shortNames ...string) CRD {
	c.ShortNames = shortNames
	return c
}

func (c CRD) ToCustomResourceDefinition() (runtime.Object, error) {
	if c.Override != nil {
		return c.Override, nil
	}

	c, name, singular, plural := c.prepareCustomResourceDefinition()

	crd := apiextv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Group: c.GVK.Group,
			Versions: []apiextv1.CustomResourceDefinitionVersion{
				{
					Name:                     c.GVK.Version,
					Storage:                  true,
					Served:                   true,
					AdditionalPrinterColumns: c.Columns,
				},
			},
			Names: apiextv1.CustomResourceDefinitionNames{
				Plural:     plural,
				Singular:   singular,
				Kind:       c.GVK.Kind,
				Categories: c.Categories,
				ShortNames: c.ShortNames,
			},
			PreserveUnknownFields: false,
		},
	}

	if c.Schema != nil {
		crd.Spec.Versions[0].Schema = &apiextv1.CustomResourceValidation{
			OpenAPIV3Schema: c.Schema,
		}
	}

	if c.SchemaObject != nil {
		schema, err := openapi.ToOpenAPIFromStruct(c.SchemaObject)
		if err != nil {
			return nil, err
		}
		crd.Spec.Versions[0].Schema = &apiextv1.CustomResourceValidation{
			OpenAPIV3Schema: schema,
		}
	}

	// add a dummy schema because v1 requires OpenAPIV3Schema to be set
	if crd.Spec.Versions[0].Schema == nil {
		crd.Spec.Versions[0].Schema = &apiextv1.CustomResourceValidation{
			OpenAPIV3Schema: &apiextv1.JSONSchemaProps{
				Type: "object",
				Properties: map[string]apiextv1.JSONSchemaProps{
					"spec": {
						XPreserveUnknownFields: &[]bool{true}[0],
					},
					"status": {
						XPreserveUnknownFields: &[]bool{true}[0],
					},
				},
			},
		}
	}

	if c.Status {
		crd.Spec.Versions[0].Subresources = &apiextv1.CustomResourceSubresources{
			Status: &apiextv1.CustomResourceSubresourceStatus{},
		}
		if c.Scale {
			sel := "Spec.Selector"
			crd.Spec.Versions[0].Subresources.Scale = &apiextv1.CustomResourceSubresourceScale{
				SpecReplicasPath:   "Spec.Replicas",
				StatusReplicasPath: "Status.Replicas",
				LabelSelectorPath:  &sel,
			}
		}
	}

	if c.NonNamespace {
		crd.Spec.Scope = apiextv1.ClusterScoped
	} else {
		crd.Spec.Scope = apiextv1.NamespaceScoped
	}

	crd.Labels = c.Labels
	crd.Annotations = c.Annotations

	// Convert to unstructured to ensure that PreserveUnknownFields=false is set because the struct will omit false
	mapData, err := convert.EncodeToMap(crd)
	if err != nil {
		return nil, err
	}
	mapData["kind"] = "CustomResourceDefinition"
	mapData["apiVersion"] = apiextv1.SchemeGroupVersion.String()

	return &unstructured.Unstructured{
		Object: mapData,
	}, unstructured.SetNestedField(mapData, false, "spec", "preserveUnknownFields")
}

func (c CRD) ToCustomResourceDefinitionV1Beta1() (runtime.Object, error) {
	if c.Override != nil {
		return c.Override, nil
	}

	c, name, singular, plural := c.prepareCustomResourceDefinition()

	crd := apiextv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apiextv1beta1.CustomResourceDefinitionSpec{
			Group:                    c.GVK.Group,
			AdditionalPrinterColumns: c.ColumnsV1Beta1,
			Versions: []apiextv1beta1.CustomResourceDefinitionVersion{
				{
					Name:    c.GVK.Version,
					Storage: true,
					Served:  true,
				},
			},
			Names: apiextv1beta1.CustomResourceDefinitionNames{
				Plural:     plural,
				Singular:   singular,
				Kind:       c.GVK.Kind,
				Categories: c.Categories,
				ShortNames: c.ShortNames,
			},
		},
	}

	if c.SchemaV1Beta1 != nil {
		crd.Spec.Validation = &apiextv1beta1.CustomResourceValidation{
			OpenAPIV3Schema: c.SchemaV1Beta1,
		}
	}

	if c.SchemaObject != nil {
		schema, err := openapi.ToOpenAPIFromStructV1Beta1(c.SchemaObject)
		if err != nil {
			return nil, err
		}
		crd.Spec.Validation = &apiextv1beta1.CustomResourceValidation{
			OpenAPIV3Schema: schema,
		}
	}

	if c.Status {
		crd.Spec.Subresources = &apiextv1beta1.CustomResourceSubresources{
			Status: &apiextv1beta1.CustomResourceSubresourceStatus{},
		}
		if c.Scale {
			sel := "Spec.Selector"
			crd.Spec.Subresources.Scale = &apiextv1beta1.CustomResourceSubresourceScale{
				SpecReplicasPath:   "Spec.Replicas",
				StatusReplicasPath: "Status.Replicas",
				LabelSelectorPath:  &sel,
			}
		}
	}

	if c.NonNamespace {
		crd.Spec.Scope = apiextv1beta1.ClusterScoped
	} else {
		crd.Spec.Scope = apiextv1beta1.NamespaceScoped
	}

	crd.Labels = c.Labels
	return &crd, nil
}

func (c CRD) prepareCustomResourceDefinition() (CRD, string, string, string) {
	if c.SchemaObject != nil && c.GVK.Kind == "" {
		t := getType(c.SchemaObject)
		c.GVK.Kind = t.Name()
	}

	if c.SchemaObject != nil && c.GVK.Version == "" {
		t := getType(c.SchemaObject)
		c.GVK.Version = filepath.Base(t.PkgPath())
	}

	if c.SchemaObject != nil && c.GVK.Group == "" {
		t := getType(c.SchemaObject)
		c.GVK.Group = filepath.Base(filepath.Dir(t.PkgPath()))
	}

	plural := c.PluralName
	if plural == "" {
		plural = strings.ToLower(name.GuessPluralName(c.GVK.Kind))
	}

	singular := c.SingularName
	if singular == "" {
		singular = strings.ToLower(c.GVK.Kind)
	}

	return c, strings.ToLower(plural + "." + c.GVK.Group), singular, plural
}

func NamespacedType(name string) CRD {
	kindGroup, version := kv.Split(name, "/")
	kind, group := kv.Split(kindGroup, ".")
	kind = convert.Capitalize(kind)
	group = strings.ToLower(group)

	return FromGV(schema.GroupVersion{
		Group:   group,
		Version: version,
	}, kind)
}

func New(group, version string) CRD {
	return CRD{
		GVK: schema.GroupVersionKind{
			Group:   group,
			Version: version,
		},
		PluralName:   "",
		NonNamespace: false,
		Schema:       nil,
		SchemaObject: nil,
		Columns:      nil,
		Status:       false,
		Scale:        false,
		Categories:   nil,
		ShortNames:   nil,
	}
}

func NamespacedTypes(names ...string) (ret []CRD) {
	for _, name := range names {
		ret = append(ret, NamespacedType(name))
	}
	return
}

func NonNamespacedType(name string) CRD {
	crd := NamespacedType(name)
	crd.NonNamespace = true
	return crd
}

func NonNamespacedTypes(names ...string) (ret []CRD) {
	for _, name := range names {
		ret = append(ret, NonNamespacedType(name))
	}
	return
}

func FromGV(gv schema.GroupVersion, kind string) CRD {
	return CRD{
		GVK: gv.WithKind(kind),
	}
}
