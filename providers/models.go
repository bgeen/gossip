package provider

var AvailableModels = map[string]string{
	"gpt-4o":                   "openai",
	"gpt-4o-mini":              "openai",
	"o1-mini":                  "openai",
	"claude-3-5-sonnet-latest": "anthropic",
	"claude-3-7-sonnet-latest": "anthropic",
	"llama-3.3-70b-versatile":  "groq",
}
