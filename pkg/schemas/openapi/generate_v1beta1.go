package openapi

import (
	"encoding/json"
	"fmt"
	"sort"

	types "github.com/rancher/wrangler/pkg/schemas"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func ToOpenAPIFromStructV1Beta1(obj interface{}) (*v1beta1.JSONSchemaProps, error) {
	schemas := types.EmptySchemas()
	schema, err := schemas.Import(obj)
	if err != nil {
		return nil, err
	}

	return ToOpenAPIV1Beta1(schema.ID, schemas)
}

func ToOpenAPIV1Beta1(name string, schemas *types.Schemas) (*v1beta1.JSONSchemaProps, error) {
	schema := schemas.Schema(name)
	if schema == nil {
		return nil, fmt.Errorf("failed to find schema: %s", name)
	}

	newSchema := schema.DeepCopy()
	if newSchema.InternalSchema != nil {
		newSchema = newSchema.InternalSchema.DeepCopy()
	}
	delete(newSchema.ResourceFields, "kind")
	delete(newSchema.ResourceFields, "apiVersion")
	delete(newSchema.ResourceFields, "metadata")

	return schemaToPropsV1Beta1(newSchema, schemas, map[string]bool{})
}

func populateFieldV1Beta1(fieldJSP *v1beta1.JSONSchemaProps, f *types.Field) error {
	fieldJSP.Description = f.Description
	// don't reset this to not nullable
	if f.Nullable {
		fieldJSP.Nullable = f.Nullable
	}
	fieldJSP.MinLength = f.MinLength
	fieldJSP.MaxLength = f.MaxLength

	if f.Type == "string" && len(f.Options) > 0 {
		for _, opt := range append(f.Options, "") {
			bytes, err := json.Marshal(&opt)
			if err != nil {
				return err
			}
			fieldJSP.Enum = append(fieldJSP.Enum, v1beta1.JSON{
				Raw: bytes,
			})
		}
	}

	if len(f.InvalidChars) > 0 {
		fieldJSP.Pattern = fmt.Sprintf("^[^%s]*$", f.InvalidChars)
	}

	if len(f.ValidChars) > 0 {
		fieldJSP.Pattern = fmt.Sprintf("^[%s]*$", f.ValidChars)
	}

	if f.Min != nil {
		fl := float64(*f.Min)
		fieldJSP.Minimum = &fl
	}

	if f.Max != nil {
		fl := float64(*f.Max)
		fieldJSP.Maximum = &fl
	}

	if f.Default != nil {
		bytes, err := json.Marshal(f.Default)
		if err != nil {
			return err
		}
		fieldJSP.Default = &v1beta1.JSON{
			Raw: bytes,
		}
	}

	return nil
}

func typeToPropsV1Beta1(typeName string, schemas *types.Schemas, inflight map[string]bool) (*v1beta1.JSONSchemaProps, error) {
	t, subType, schema, err := typeAndSchema(typeName, schemas)
	if err != nil {
		return nil, err
	}

	if schema != nil {
		return schemaToPropsV1Beta1(schema, schemas, inflight)
	}

	jsp := &v1beta1.JSONSchemaProps{}

	switch t {
	case "map":
		additionalProps, err := typeToPropsV1Beta1(subType, schemas, inflight)
		if err != nil {
			return nil, err
		}
		jsp.Type = "object"
		jsp.Nullable = true
		if additionalProps.Type != "object" {
			jsp.AdditionalProperties = &v1beta1.JSONSchemaPropsOrBool{
				Allows: true,
				Schema: additionalProps,
			}
		}
	case "array":
		items, err := typeToPropsV1Beta1(subType, schemas, inflight)
		if err != nil {
			return nil, err
		}
		jsp.Type = "array"
		jsp.Nullable = true
		jsp.Items = &v1beta1.JSONSchemaPropsOrArray{
			Schema: items,
		}
	case "string":
		jsp.Type = t
		jsp.Nullable = true
	default:
		jsp.Type = t
	}
	return jsp, nil
}

func schemaToPropsV1Beta1(schema *types.Schema, schemas *types.Schemas, inflight map[string]bool) (*v1beta1.JSONSchemaProps, error) {
	jsp := &v1beta1.JSONSchemaProps{
		Description: schema.Description,
		Type:        "object",
	}

	if inflight[schema.ID] {
		return jsp, nil
	}

	inflight[schema.ID] = true
	defer delete(inflight, schema.ID)

	jsp.Properties = map[string]v1beta1.JSONSchemaProps{}

	for name, f := range schema.ResourceFields {
		fieldJSP, err := typeToPropsV1Beta1(f.Type, schemas, inflight)
		if err != nil {
			return nil, err
		}
		if err := populateFieldV1Beta1(fieldJSP, &f); err != nil {
			return nil, err
		}
		if f.Required {
			jsp.Required = append(jsp.Required, name)
		}
		jsp.Properties[name] = *fieldJSP
	}
	sort.Strings(jsp.Required)
	return jsp, nil
}
