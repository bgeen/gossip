package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"
)

const OpenaiEndpoint = "https://api.openai.com/v1/responses"

var OpenaiModels map[string]bool = map[string]bool{
	"o3-mini":     true,
	"o1-mini":     true,
	"gpt-4o-mini": true,
	"gpt-4o":      true,
}

type Openai struct {
	AgentConfig
	Tools []OpenaiTool
}

type OpenaiMessage struct { // or InputItem
	Role      string `json:"role,omitempty"` // developer | user | assistant
	Content   string `json:"content,omitempty"`
	Type      string `json:"type,omitempty"`
	Id        string `json:"id,omitempty"`
	CallId    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Output    string `json:"output,omitempty"`
}

type OpenaiTool struct {
	Type        string     `json:"type"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  Parameters `json:"parameters"`
	Strict      bool       `json:"strict"`
}

type OpenaiRequest struct {
	Model           string          `json:"model"`
	Input           []OpenaiMessage `json:"input"`
	ReasoningEffort string          `json:"reasoning_effort,omitempty"`
	Temperature     float32         `json:"temperature,omitempty"`
	Tools           []OpenaiTool    `json:"tools,omitempty"`
}

type OpenaiContent struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

type OpenaiOutputItem struct {
	Type      string          `json:"type"`                // tool use + chat
	Id        string          `json:"id,omitempty"`        // tool use + chat
	Status    string          `json:"status,omitempty"`    // tool use + chat
	Role      string          `json:"role,omitempty"`      // chat
	Content   []OpenaiContent `json:"content,omitempty"`   // chat
	CallId    string          `json:"call_id,omitempty"`   // tool use
	Name      string          `json:"name,omitempty"`      // tool use
	Arguments string          `json:"arguments,omitempty"` // tool use
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

type OpenaiResponse struct {
	ID          string             `json:"id"`
	Status      string             `json:"status"`
	Store       bool               `json:"store"`
	Temperature float32            `json:"temperature,omitempty"`
	ToolChoice  string             `json:"tool_choice,omitempty"`
	Model       string             `json:"model"`
	Output      []OpenaiOutputItem `json:"output"`
	Usage       OpenaiUsage        `json:"usage"`
}

func (provider Openai) FormatMessages(messages []Message) []OpenaiMessage {
	var openaiMessages []OpenaiMessage

	for _, msg := range messages {
		var openaiMsg OpenaiMessage

		if msg.ToolIntent != nil {
			openaiMsg.Type = "function_call"
			openaiMsg.CallId = msg.ToolIntent.Id
			openaiMsg.Name = msg.ToolIntent.Name
			if msg.ToolIntent.Arguments != "" {
				openaiMsg.Arguments = msg.ToolIntent.Arguments
			}
		} else if msg.ToolResult != nil {
			openaiMsg.Type = "function_call_output"
			openaiMsg.CallId = msg.ToolResult.Id
			openaiMsg.Output = msg.ToolResult.Output

		} else {
			openaiMsg.Role = "user"
			openaiMsg.Content = msg.Text
		}
		openaiMessages = append(openaiMessages, openaiMsg)
	}
	return openaiMessages
}

func (provider Openai) Run(prompt string, messageHistory ...[]Message) (*AgentResult, error) {
	fmt.Printf("[%s] Provider openai called\n", time.Now().Format(time.RFC3339))
	apiKey := provider.ApiKey

	var requestInput []OpenaiMessage

	if len(messageHistory) > 0 {
		requestInput = provider.FormatMessages(messageHistory[0])
	}
	if prompt != "" {
		newMessage := OpenaiMessage{
			Role:    "user",
			Content: prompt,
		}
		requestInput = append(requestInput, newMessage)
	}
	if provider.SystemPrompt != "" {
		systemPrompt := OpenaiMessage{
			Role:    "developer",
			Content: provider.SystemPrompt,
		}
		requestInput = append(requestInput, systemPrompt)
	}

	reqBody := OpenaiRequest{
		Model: provider.ModelName,
		Input: requestInput,
	}
	if provider.ReasoningEffort != "" {
		reqBody.ReasoningEffort = provider.ReasoningEffort
	}
	if provider.Temperature != 0 {
		reqBody.Temperature = provider.Temperature
	}

	var tools []OpenaiTool
	if len(provider.ToolStore.functions) > 0 {
		for fn, _ := range provider.ToolStore.functions {
			fnName := fn
			properties, required := ConvertToProperties(reflect.New(provider.ToolStore.paramTypes[fnName]).Interface())
			tool := OpenaiTool{
				Type:        "function",
				Name:        fnName,
				Description: provider.ToolStore.descriptions[fnName],
				Parameters: Parameters{
					Type:                 "object",
					Required:             required,
					Properties:           properties,
					AdditionalProperties: false,
				},
				Strict: true,
			}
			tools = append(tools, tool)
		}
		reqBody.Tools = tools
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", OpenaiEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	// headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse JSON response
	var response OpenaiResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	var allMessages []Message
	var responseMessage Message
	var toolIntent ToolIntent

	if len(messageHistory) > 0 {
		allMessages = append(allMessages, messageHistory[0]...)
	}
	if prompt != "" {
		allMessages = append(allMessages, Message{Role: "user", Text: prompt})
	}
	for _, output := range response.Output {
		switch output.Type {
		case "message":
			for _, content := range output.Content {
				if content.Type == "output_text" {
					responseMessage = Message{
						Role: output.Role,
						Text: content.Text,
					}
					allMessages = append(allMessages, responseMessage)
				}
			}
		case "function_call":
			intent := ToolIntent{
				Id:        output.CallId,
				Name:      output.Name,
				Arguments: output.Arguments,
			}
			allMessages = append(allMessages, Message{
				Type:       "tool_intent",
				ToolIntent: &intent,
			})
			toolIntent = intent
		default:
			return nil, fmt.Errorf("(openai.go, Run) unexpected message type")
		}
	}

	if toolIntent.Id != "" {
		toolResult, err := provider.ExecuteToolIntent(toolIntent)
		if err != nil {
			return nil, err
		}
		allMessages = append(allMessages, Message{ToolResult: toolResult})
		internalAgentCall, err := provider.Run("", allMessages)
		if err != nil {
			return nil, err
		}
		responseMessage = internalAgentCall.NewMessage
		allMessages = append(allMessages, responseMessage)
	}

	return &AgentResult{
		AllMessages:   allMessages,
		NewMessage:    responseMessage,
		ToolIntent:    &toolIntent,
		Data:          responseMessage.Text,
		ToolArguments: toolIntent.Arguments,
	}, nil
}

func (provider *Openai) RegisterTool(fn any, paramType any, desctiption string) error {
	provider.AgentConfig.RegisterTool(fn, paramType, desctiption)
	return nil
}
