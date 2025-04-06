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
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float32            `json:"temperature,omitempty"`
	System      string             `json:"system,omitempty"`
	Messages    []AnthropicMessage `json:"messages"`
	Tools       []AnthropicTool    `json:"tools,omitempty"`
}

type AnthropicMessage struct {
	Role    string             `json:"role"`
	Content []AnthropicContent `json:"content"`
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

type AnthropicContent struct {
	Type      string         `json:"type"` // text, tool_use, tool_result
	Text      string         `json:"text,omitempty"`
	Id        string         `json:"id,omitempty"`          // 'tool_use' id
	Name      string         `json:"name,omitempty"`        // function name
	Input     map[string]any `json:"input,omitempty"`       // json object containing parameters returned by tool_use
	ToolUseId string         `json:"tool_use_id,omitempty"` // tool_use_id is used to return tool call result. value is same as 'id' in type 'tool_use'
	Content   string         `json:"content,omitempty"`     //	tool result value
	// Source    AnthropicImageSource `json:"source,omitempty"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type AnthropicResponse struct {
	Content      []AnthropicContent `json:"content"`
	Model        string             `json:"model"`
	Role         string             `json:"role"`
	StopReason   string             `json:"stop_reason"`
	StopSequence any                `json:"stop_sequence"`
	Type         string             `json:"type"`
	Usage        AnthropicUsage     `json:"usage"`
}

func (provider Anthropic) FormatMessages(messages []Message) []AnthropicMessage {
	var anthropicMessages []AnthropicMessage

	for _, msg := range messages {
		var content AnthropicContent
		var role string

		if msg.ToolIntent != nil {
			role = "assistant"
			content.Type = "tool_use"
			content.Id = msg.ToolIntent.Id
			content.Name = msg.ToolIntent.Name
			if msg.ToolIntent.Arguments != "" {
				var input map[string]any
				err := json.Unmarshal([]byte(msg.ToolIntent.Arguments), &input)
				if err != nil {
					log.Fatal("failed to unmarshal arguments string to map[string]any")
				}
				content.Input = input
			}
		} else if msg.ToolResult != nil {
			role = "user"
			content.Type = "tool_result"
			content.ToolUseId = msg.ToolResult.Id
			content.Content = msg.ToolResult.Output

		} else {
			role = "user"
			content.Type = "text"
			content.Text = msg.Text
		}

		anthropicMessages = append(anthropicMessages, AnthropicMessage{
			Role:    role,
			Content: []AnthropicContent{content},
		})
	}
	return anthropicMessages
}

func (provider Anthropic) Run(prompt string, messageHistory ...[]Message) *AgentResult {
	apiKey := provider.ApiKey

	var finalPrompt []AnthropicMessage
	if len(messageHistory) > 0 {
		finalPrompt = provider.FormatMessages(messageHistory[0])
	}

	if prompt != "" {
		newMessage := AnthropicMessage{
			Role: "user",
			Content: []AnthropicContent{
				{
					Type: "text",
					Text: prompt,
				},
			},
		}
		finalPrompt = append(finalPrompt, newMessage)
	}

	reqBody := AnthropicRequest{
		Model:     provider.ModelName,
		MaxTokens: 1024,
		Messages:  finalPrompt,
	}

	if provider.SystemPrompt != "" {
		reqBody.System = provider.SystemPrompt
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

	var allMessages []Message
	var responseMessage Message
	var toolIntent ToolIntent

	if len(messageHistory) > 0 {
		allMessages = append(allMessages, messageHistory[0]...)
	}
	if prompt != "" {
		allMessages = append(allMessages, Message{Role: "user", Text: prompt})
	}

	for _, item := range response.Content { // assuming there will be only one element in response.content list
		switch item.Type {
		case "text":
			responseMessage = Message{
				Role: response.Role,
				Text: item.Text,
			}
			allMessages = append(allMessages, responseMessage)
		case "tool_use":
			argumentsString, err := json.Marshal(item.Input)
			if err != nil {
				log.Fatal("failed to convert arguments json object to string")
			}
			intent := ToolIntent{
				Id:        item.Id,
				Name:      item.Name,
				Arguments: string(argumentsString),
			}
			allMessages = append(allMessages, Message{
				Type:       "tool_intent",
				ToolIntent: &intent,
			})
			toolIntent = intent
		default:
			log.Fatal("Unexpected message type")
		}
	}

	return &AgentResult{
		AllMessages:   allMessages,
		NewMessage:    responseMessage,
		ToolIntent:    &toolIntent,
		Data:          responseMessage.Text,
		ToolArguments: toolIntent.Arguments,
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
