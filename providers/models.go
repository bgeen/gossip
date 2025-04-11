package provider

var AvailableModels = map[string]bool{
	"openai:gpt-4o":                      true,
	"openai:gpt-4o-mini":                 true,
	"openai:o1-mini":                     true,
	"anthropic:claude-3-5-sonnet-latest": true,
	"anthropic:claude-3-7-sonnet-latest": true,
	"groq:llama-3.3-70b-versatile":       true,
}
