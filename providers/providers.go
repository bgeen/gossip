package providers

import (
	"fmt"
	"os"
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
	RegisterTool(string, string, any)
}

type AgentConfig struct {
	ModelName       string
	ApiKey          string
	SystemPrompt    string
	ReasoningEffort string
	Temperature     float32
}

type AgentResult struct {
	AllMessages []Message
	NewMessage  Message
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

func NewAgent(modelName string, opts ...AgentOption) (Agent, bool) {
	provider, model, found := strings.Cut(modelName, ":")
	if !found {
		fmt.Println("seperator not found in model name")
		return nil, false
	}
	if !CheckModel(provider, model) {
		fmt.Println("Model not available")
		return nil, false
	}
	keyName := strings.ToUpper(provider) + "_API_KEY"
	apiKey, keyFound := os.LookupEnv(keyName)
	if !keyFound {
		fmt.Println("api key not found")
		return nil, false
	}
	config := AgentConfig{ModelName: model, ApiKey: apiKey}

	for _, opt := range opts {
		opt(&config)
	}

	switch provider {
	case "anthropic":
		return &Anthropic{config, nil}, true
	case "openai":
		return &Openai{config, nil}, true
	default:
		fmt.Println("unknown provider!")
		return nil, false
	}
}
