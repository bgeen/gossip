package provider

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"runtime"
	"strings"
	"time"
)

type Property struct {
	Type        string              `json:"type"`
	Description string              `json:"description,omitempty"`
	Items       *Property           `json:"items,omitempty"`
	Properties  map[string]Property `json:"properties,omitempty"` // For nested objects
}

type Properties map[string]Property

type Parameters struct {
	Type                 string     `json:"type"`
	Required             []string   `json:"required"`
	Properties           Properties `json:"properties"`
	AdditionalProperties bool       `json:"additionalProperties"`
}

// Registry to store functions and their parameter types
type ToolStore struct {
	functions  map[string]any
	paramTypes map[string]reflect.Type
	// paramTypes   map[string]any
	descriptions map[string]string
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

type ToolIntent struct {
	Id        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type ToolResult struct {
	Id     string `json:"id,omitempty"`
	Output string `json:"output,omitempty"`
}

func ConvertToProperties(v any) (Properties, []string) {
	schema := make(Properties)
	t := reflect.TypeOf(v)

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Verify it's a struct
	if t.Kind() != reflect.Struct {
		panic("Input must be a struct or pointer to struct")
	}
	var fieldNames []string
	for i := range t.NumField() {
		field := t.Field(i)
		schemaField := processField(field)

		// Use the JSON tag name if present, otherwise use the field name
		fieldName := field.Tag.Get("json")
		if fieldName == "" {
			fieldName = strings.ToLower(field.Name)
		}

		schema[fieldName] = schemaField
		fieldNames = append(fieldNames, fieldName)
	}

	return schema, fieldNames
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

func getToolName(f any) (string, error) {
	fullName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	if fullName == "" {
		return "", fmt.Errorf("function name not found")
	}
	// Split the module name and function name
	parts := strings.Split(fullName, ".")
	return parts[len(parts)-1], nil
}

func (provider *AgentConfig) RegisterTool(fn any, paramType any, desctiption string) error {
	fnName, err := getToolName(fn)
	if err != nil {
		return err
	}
	fnType := reflect.TypeOf(fn)

	// Validate function has exactly one parameter
	if fnType.NumIn() != 1 {
		return fmt.Errorf("function must take exactly one parameter")
	}
	provider.ToolStore.functions[fnName] = fn
	provider.ToolStore.paramTypes[fnName] = reflect.TypeOf(paramType)
	provider.ToolStore.descriptions[fnName] = desctiption
	return nil
}

func (provider *AgentConfig) ExecuteToolIntent(toolIntent ToolIntent) (*ToolResult, error) {
	store := provider.ToolStore
	fnName := toolIntent.Name
	log.Printf("[%s] Tool called: %s\n", time.Now().Format(time.RFC3339), fnName)
	fn, exists := store.functions[fnName]
	if !exists {
		return nil, fmt.Errorf("function %s not found", fnName)
	}
	expectedType, exists := store.paramTypes[fnName]
	if !exists {
		return nil, fmt.Errorf("parameter type for function %s not found", fnName)
	}

	paramInstance := reflect.New(expectedType).Interface()
	err := json.Unmarshal([]byte(toolIntent.Arguments), &paramInstance)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool call")
	}

	actualType := reflect.TypeOf(paramInstance).Elem()
	if actualType != expectedType {
		return nil, fmt.Errorf("invalid parameter type. expected %v, got %v", expectedType, actualType)
	}

	fnValue := reflect.ValueOf(fn)
	paramValue := reflect.ValueOf(paramInstance).Elem()
	toolOutputValues := fnValue.Call([]reflect.Value{paramValue})
	if len(toolOutputValues) == 0 {
		return nil, fmt.Errorf("tool call returned nothing")
	}
	toolResult := ToolResult{
		Id:     toolIntent.Id,
		Output: fmt.Sprintf("%v", toolOutputValues[0].Interface()),
	}
	return &toolResult, nil
}
