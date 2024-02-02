package schemas

import (
	"errors"

	"github.com/rancher/wrangler/v2/pkg/data"
	"github.com/rancher/wrangler/v2/pkg/data/convert"
	"github.com/rancher/wrangler/v2/pkg/schemas/definition"
)

type Mapper interface {
	FromInternal(data data.Object)
	ToInternal(data data.Object) error
	ModifySchema(schema *Schema, schemas *Schemas) error
}

type Mappers []Mapper

func (m Mappers) FromInternal(data data.Object) {
	for _, mapper := range m {
		mapper.FromInternal(data)
	}
}

func (m Mappers) ToInternal(data data.Object) error {
	var returnErrors error
	for i := len(m) - 1; i >= 0; i-- {
		returnErrors = errors.Join(returnErrors, m[i].ToInternal(data))
	}
	return returnErrors
}

func (m Mappers) ModifySchema(schema *Schema, schemas *Schemas) error {
	for _, mapper := range m {
		if err := mapper.ModifySchema(schema, schemas); err != nil {
			return err
		}
	}
	return nil
}

type typeMapper struct {
	Mappers         []Mapper
	root            bool
	typeName        string
	subSchemas      map[string]*Schema
	subArraySchemas map[string]*Schema
	subMapSchemas   map[string]*Schema
}

func (t *typeMapper) FromInternal(data data.Object) {
	for fieldName, schema := range t.subSchemas {
		if schema.Mapper == nil {
			continue
		}
		schema.Mapper.FromInternal(data.Map(fieldName))
	}

	for fieldName, schema := range t.subMapSchemas {
		if schema.Mapper == nil {
			continue
		}
		for _, fieldData := range data.Map(fieldName).Values() {
			schema.Mapper.FromInternal(fieldData)
		}
	}

	for fieldName, schema := range t.subArraySchemas {
		if schema.Mapper == nil {
			continue
		}
		for _, fieldData := range data.Slice(fieldName) {
			schema.Mapper.FromInternal(fieldData)
		}
	}

	Mappers(t.Mappers).FromInternal(data)
}

func (t *typeMapper) ToInternal(data data.Object) error {
	var returnErrors error
	returnErrors = errors.Join(returnErrors, Mappers(t.Mappers).ToInternal(data))

	for fieldName, schema := range t.subArraySchemas {
		if schema.Mapper == nil {
			continue
		}
		for _, fieldData := range data.Slice(fieldName) {
			returnErrors = errors.Join(returnErrors, schema.Mapper.ToInternal(fieldData))
		}
	}

	for fieldName, schema := range t.subMapSchemas {
		if schema.Mapper == nil {
			continue
		}
		for _, fieldData := range data.Map(fieldName) {
			returnErrors = errors.Join(returnErrors, schema.Mapper.ToInternal(convert.ToMapInterface(fieldData)))
		}
	}

	for fieldName, schema := range t.subSchemas {
		if schema.Mapper == nil {
			continue
		}
		returnErrors = errors.Join(returnErrors, schema.Mapper.ToInternal(data.Map(fieldName)))
	}

	return returnErrors
}

func (t *typeMapper) ModifySchema(schema *Schema, schemas *Schemas) error {
	t.subSchemas = map[string]*Schema{}
	t.subArraySchemas = map[string]*Schema{}
	t.subMapSchemas = map[string]*Schema{}
	t.typeName = schema.ID

	mapperSchema := schema
	if schema.InternalSchema != nil {
		mapperSchema = schema.InternalSchema
	}
	for name, field := range mapperSchema.ResourceFields {
		fieldType := field.Type
		targetMap := t.subSchemas
		if definition.IsArrayType(fieldType) {
			fieldType = definition.SubType(fieldType)
			targetMap = t.subArraySchemas
		} else if definition.IsMapType(fieldType) {
			fieldType = definition.SubType(fieldType)
			targetMap = t.subMapSchemas
		}

		schema := schemas.doSchema(fieldType, false)
		if schema != nil {
			targetMap[name] = schema
		}
	}

	return Mappers(t.Mappers).ModifySchema(schema, schemas)
}
