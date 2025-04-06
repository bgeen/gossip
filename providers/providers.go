package provider

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

func CheckModel(provider string, model string) bool {
	var found bool
	if provider == "anthropic" {
		found = AnthropicModels[model]
	}

	if provider == "openai" {
		found = OpenaiModels[model]
	}
	return found
}

type Agent interface {
	Run(string, ...[]Message) *AgentResult
	RegisterTool(any, any, string) error
}

type AgentConfig struct {
	ModelName       string
	ApiKey          string
	SystemPrompt    string
	ReasoningEffort string
	Temperature     float32
	ToolStore
}

type AgentResult struct {
	AllMessages   []Message
	NewMessage    Message
	Data          string
	ToolArguments string
	ToolIntent    *ToolIntent
	ToolResult    ToolResult
}

type Message struct { // or InputItem
	Role       string      `json:"role,omitempty"` // developer | user | assistant
	Text       string      `json:"text,omitempty"`
	Type       string      `json:"type,omitempty"`
	ToolIntent *ToolIntent `json:"tool_intent,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

type AgentOption func(*AgentConfig)

func WithSystemPrompt(prompt string) AgentOption {
	return func(a *AgentConfig) {
		a.SystemPrompt = prompt
	}
}

func WithReasoningEffort(reasoningEffort string) AgentOption {
	return func(a *AgentConfig) {
		a.ReasoningEffort = reasoningEffort
	}
}

func WithTemperature(temperature float32) AgentOption {
	return func(a *AgentConfig) {
		a.Temperature = temperature
	}
}

func NewAgent(modelName string, opts ...AgentOption) (Agent, error) {
	provider, model, found := strings.Cut(modelName, ":")
	if !found {
		return nil, fmt.Errorf("seperator not found in model name")
	}
	if !CheckModel(provider, model) {
		return nil, fmt.Errorf("model not available")
	}
	keyName := strings.ToUpper(provider) + "_API_KEY"
	apiKey, keyFound := os.LookupEnv(keyName)
	if !keyFound {
		return nil, fmt.Errorf("api key not found")
	}
	toolStore := ToolStore{
		functions:    make(map[string]any),
		paramTypes:   make(map[string]reflect.Type),
		descriptions: make(map[string]string),
	}
	config := AgentConfig{ModelName: model, ApiKey: apiKey, ToolStore: toolStore}

	for _, opt := range opts {
		opt(&config)
	}

	switch provider {
	case "anthropic":
		return &Anthropic{config, nil}, nil
	case "openai":
		return &Openai{config, nil}, nil
	default:
		return nil, fmt.Errorf("unknown provider!")
	}
}
