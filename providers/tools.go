package providers

import (
	"reflect"
	"strings"
)

type Property struct {
	Type        string              `json:"type"`
	Description string              `json:"description,omitempty"`
	Items       *Property           `json:"items,omitempty"`
	Properties  map[string]Property `json:"properties,omitempty"` // For nested objects
}

type Properties map[string]Property

type Parameters struct {
	Type       string     `json:"type"`
	Required   []string   `json:"required"`
	Properties Properties `json:"properties"`
}

type Tool struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  Parameters `json:"parameters"`
}

type ToolRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
	Tools     []Tool    `json:"tools"`
}

func ConvertToProperties(v any) Properties {
	schema := make(Properties)
	t := reflect.TypeOf(v)

	for i := range t.NumField() {
		field := t.Field(i)
		schemaField := processField(field)

		// Use the JSON tag name if present, otherwise use the field name
		fieldName := field.Tag.Get("json")
		if fieldName == "" {
			fieldName = strings.ToLower(field.Name)
		}

		schema[fieldName] = schemaField
	}

	return schema
}

func processField(field reflect.StructField) Property {
	property := Property{
		Description: field.Tag.Get("description"),
	}

	switch field.Type.Kind() {
	case reflect.Struct:
		// Process nested struct
		property.Type = "object"
		property.Properties = make(map[string]Property)

		// Recursively process each field in the nested struct
		for i := range field.Type.NumField() {
			nestedField := field.Type.Field(i)
			fieldName := nestedField.Tag.Get("json")
			if fieldName == "" {
				fieldName = strings.ToLower(nestedField.Name)
			}
			property.Properties[fieldName] = processField(nestedField)
		}

	case reflect.Slice, reflect.Array:
		property.Type = "array"
		// Handle array element type
		property.Items = &Property{
			Type: getBasicType(field.Type.Elem()),
		}

	default:
		property.Type = getBasicType(field.Type)
	}

	return property
}

func getBasicType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Struct:
		return "object"
	case reflect.Map:
		return "object"
	case reflect.Interface:
		return "any"
	default:
		return "object"
	}
}

// type Properties struct {
// 	Type string `json:"type"`
// }
