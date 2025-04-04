package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

const OpenaiEndpoint = "https://api.openai.com/v1/chat/completions"

var OpenaiModels map[string]bool = map[string]bool{
	"o3-mini":     true,
	"o1-mini":     true,
	"gpt-4o-mini": true,
	"gpt-4o":      true,
}

type Openai struct {
	AgentConfig
	Tools []Tool
}

type OpenaiRequest struct {
	Model           string    `json:"model"`
	Messages        []Message `json:"messages"`
	ReasoningEffort string    `json:"reasoning_effort,omitempty"`
	Temperature     float32   `json:"temperature,omitempty"`
	Tools           []Tool    `json:"tools,omitzero"`
}

type OpenaiChoice struct {
	Index        *int    `json:"index,omitempty"`
	Message      Message `json:"message,omitempty"`
	FinishReason *string `json:"finish_reason,omitempty"`
}
type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}
type OpenaiUsage struct {
	PromptTokens        int                 `json:"prompt_tokens"`
	CompletionTokens    int                 `json:"completion_tokens"`
	TotalTokens         int                 `json:"total_tokens"`
	PromptTokensDetails PromptTokensDetails `json:"prompt_tokens_details"`
}

type OpenaiChatCompletionObject struct {
	ID      string         `json:"id"`
	Model   string         `json:"model"`
	Choices []OpenaiChoice `json:"choices"`
	Usage   OpenaiUsage    `json:"usage"`
}

func (provider Openai) Run(prompt string, messageHistory ...[]Message) *AgentResult {
	apiKey := provider.ApiKey

	newMessage := Message{
		Role:    "user",
		Content: prompt,
	}
	var finalPrompt []Message
	if len(messageHistory) > 0 {
		finalPrompt = messageHistory[0]
	}
	if provider.SystemPrompt != "" {
		systemPrompt := Message{
			Role:    "developer",
			Content: provider.SystemPrompt,
		}
		finalPrompt = append(finalPrompt, systemPrompt)
	}
	finalPrompt = append(finalPrompt, newMessage)

	reqBody := OpenaiRequest{
		Model:    provider.ModelName,
		Messages: finalPrompt,
	}
	if provider.ReasoningEffort != "" {
		reqBody.ReasoningEffort = provider.ReasoningEffort
	}
	if provider.Temperature != 0 {
		reqBody.Temperature = provider.Temperature
	}
	fmt.Println(provider.SystemPrompt)
	data, err := json.Marshal(reqBody)
	fmt.Println(string(data))

	// Convert request body to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatal(err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", OpenaiEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal(err)
	}

	// Add headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Parse JSON response
	var response OpenaiChatCompletionObject
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatal(err)
	}
	data, err = json.Marshal(response)
	allMessages := finalPrompt
	var assistantMessage Message
	for _, choice := range response.Choices {
		assistantMessage = Message{
			Role:    choice.Message.Role,
			Content: choice.Message.Content,
		}
		allMessages = append(allMessages, assistantMessage)
	}

	return &AgentResult{
		AllMessages: allMessages,
		NewMessage:  assistantMessage,
	}

}

func (provider Openai) RegisterTool(name string, desctiption string, schema any) {
	tool := Tool{
		Name:        name,
		Description: desctiption,
		Parameters: Parameters{
			Type:       "object",
			Required:   []string{},
			Properties: ConvertToProperties(schema),
		},
	}
	provider.Tools = append(provider.Tools, tool)
}
