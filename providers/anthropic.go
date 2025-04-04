package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

const AnthropicEndpoint = "https://api.anthropic.com/v1/messages"

var AnthropicModels map[string]bool = map[string]bool{
	"claude-3-5-sonnet-latest": true,
	"claude-3-5-haiku-latest":  true,
}

type Anthropic struct {
	AgentConfig
	Tools []AnthropicTool
}

type AnthropicRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float32         `json:"temperature"`
	Messages    []Message       `json:"messages"`
	Tools       []AnthropicTool `json:"tools,omitzero"`
}

type AnthropicTool struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  Parameters `json:"input_schema"`
}

type AnthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type AnthropicContentItem struct {
	Type   string                `json:"type"`
	Text   *string               `json:"text,omitzero"`
	Source *AnthropicImageSource `json:"source,omitzero"`
	Id     string                `json:"id,omitzero"`
	Name   string                `json:"name,omitzero"`
	Input  map[string]any        `json:"input,omitzero"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type AnthropicResponse struct {
	Content      []AnthropicContentItem `json:"content"`
	ID           string                 `json:"id"`
	Model        string                 `json:"model"`
	Role         string                 `json:"role"`
	StopReason   string                 `json:"stop_reason"`
	StopSequence any                    `json:"stop_sequence"`
	Type         string                 `json:"type"`
	Usage        AnthropicUsage         `json:"usage"`
}

func (provider Anthropic) Run(prompt string, messageHistory ...[]Message) *AgentResult {
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
			Role:    "user",
			Content: provider.SystemPrompt,
		}
		finalPrompt = append(finalPrompt, systemPrompt)
	}
	finalPrompt = append(finalPrompt, newMessage)

	reqBody := AnthropicRequest{
		Model:     provider.ModelName,
		MaxTokens: 1024,
		Messages:  finalPrompt,
	}

	if provider.Temperature != 0 {
		reqBody.Temperature = provider.Temperature
	}

	if len(provider.Tools) > 0 {
		reqBody.Tools = provider.Tools
	}

	data, err := json.MarshalIndent(reqBody, "", "    ")
	fmt.Println(string(data))
	// return nil

	// Convert request body to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatal(err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", AnthropicEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal(err)
	}

	// Add headers
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

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

	fmt.Println("resp body: ", string(body))
	// return nil

	// Parse JSON response
	var response AnthropicResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatal(err)
	}

	data, _ = json.MarshalIndent(response, "", "    ")
	fmt.Println("response data \n", string(data))

	var responseMessage Message
	for _, item := range response.Content {
		switch item.Type {
		case "text":
			responseMessage = Message{
				Role:    response.Role,
				Content: *item.Text,
			}
		default:
			log.Fatal("Unexpected message type")
		}
	}

	allMessages := finalPrompt
	allMessages = append(allMessages, responseMessage)

	return &AgentResult{
		AllMessages: allMessages,
		NewMessage:  responseMessage,
	}
}

func (provider *Anthropic) RegisterTool(name string, desctiption string, schema any) {
	tool := AnthropicTool{
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
