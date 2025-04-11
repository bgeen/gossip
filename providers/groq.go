package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
)

const GroqEndpoint = "https://api.groq.com/openai/v1/chat/completions"

type Groq struct {
	AgentConfig
	Tools []GroqTool
}

type GroqMessage struct { // or InputItem
	Role       string         `json:"role,omitempty"` // developer | user | assistant | tool
	Content    string         `json:"content,omitempty"`
	ToolCalls  []GroqToolCall `json:"tool_calls,omitempty"`
	ToolCallId string         `json:"tool_call_id,omitempty"`
}

type GroqRequest struct {
	Model           string        `json:"model"`
	Messages        []GroqMessage `json:"messages"`
	ReasoningEffort string        `json:"reasoning_effort,omitempty"`
	Temperature     float32       `json:"temperature,omitempty"`
	Tools           []GroqTool    `json:"tools,omitempty"`
}

type GroqTool struct {
	Type     string       `json:"type"` // type = "function"
	Function GroqFunction `json:"function"`
}

type GroqFunction struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  Parameters `json:"parameters"`
	Strict      bool       `json:"strict"`
}

type GroqResponse struct {
	ID      string       `json:"id"`
	Choices []GroqChoice `json:"choices"`
	Usage   GroqUsage    `json:"usage"`
}

type GroqChoice struct {
	Index        int         `json:"index"`
	Message      GroqMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type GroqUsage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	TotalTime        float64 `json:"total_time"`
}

type GroqToolCall struct {
	Type     string           `json:"type,omitempty"`
	Id       string           `json:"id,omitempty"`
	Function GroqFunctionResp `json:"function"`
}

type GroqFunctionResp struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func (provider Groq) FormatMessages(messages []Message) []GroqMessage {
	var groqMessages []GroqMessage

	for _, msg := range messages {
		var groqMsg GroqMessage

		if msg.ToolIntent != nil {
			groqMsg.Role = "assistant"
			toolCall := GroqToolCall{
				Type: "function",
				Id:   msg.ToolIntent.Id,
				Function: GroqFunctionResp{
					Name:      msg.ToolIntent.Name,
					Arguments: msg.ToolIntent.Arguments,
				},
			}
			groqMsg.ToolCalls = append(groqMsg.ToolCalls, toolCall)
		} else if msg.ToolResult != nil {
			groqMsg.Role = "tool"
			groqMsg.ToolCallId = msg.ToolResult.Id
			groqMsg.Content = msg.ToolResult.Output

		} else {
			groqMsg.Role = msg.Role
			groqMsg.Content = msg.Text
		}
		groqMessages = append(groqMessages, groqMsg)
	}
	return groqMessages
}

func (provider Groq) Run(prompt string, messageHistory ...[]Message) (*AgentResult, error) {

	log.Println("provider groq called")
	apiKey := provider.ApiKey

	var groqMessages []GroqMessage

	if len(messageHistory) > 0 {
		groqMessages = provider.FormatMessages(messageHistory[0])
	}
	if prompt != "" {
		newMessage := GroqMessage{
			Role:    "user",
			Content: prompt,
		}
		groqMessages = append(groqMessages, newMessage)
	}
	if provider.SystemPrompt != "" {
		systemPrompt := GroqMessage{
			Role:    "developer",
			Content: provider.SystemPrompt,
		}
		groqMessages = append(groqMessages, systemPrompt)
	}

	reqBody := GroqRequest{
		Model:    provider.ModelName,
		Messages: groqMessages,
	}
	if provider.ReasoningEffort != "" {
		reqBody.ReasoningEffort = provider.ReasoningEffort
	}
	if provider.Temperature != 0 {
		reqBody.Temperature = provider.Temperature
	}

	var tools []GroqTool
	if len(provider.ToolStore.functions) > 0 {
		for fn, _ := range provider.ToolStore.functions {
			fnName := fn
			properties, required := ConvertToProperties(reflect.New(provider.ToolStore.paramTypes[fnName]).Interface())
			tool := GroqTool{
				Type: "function",
				Function: GroqFunction{
					Name:        fnName,
					Description: provider.ToolStore.descriptions[fnName],
					Parameters: Parameters{
						Type:                 "object",
						Required:             required,
						Properties:           properties,
						AdditionalProperties: false,
					},
					Strict: true,
				},
			}
			tools = append(tools, tool)
		}
		reqBody.Tools = tools
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	fmt.Print("request\n", string(jsonData), "\n")

	// Create HTTP request
	req, err := http.NewRequest("POST", GroqEndpoint, bytes.NewBuffer(jsonData))
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

	fmt.Print("response\n", string(body), "\n")

	// Parse JSON response
	var response GroqResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	var msgHistory []Message
	var newMessages []Message
	var finalText string
	var toolIntent ToolIntent

	if len(messageHistory) > 0 {
		msgHistory = messageHistory[0]
	}
	if prompt != "" {
		newMessages = append(newMessages, Message{Role: "user", Text: prompt})
	}
	for _, choice := range response.Choices {
		msg := choice.Message

		if msg.Content != "" {
			responseMessage := Message{
				Role: "assistant",
				Text: msg.Content,
			}
			newMessages = append(newMessages, responseMessage)
			finalText = msg.Content
		} else if len(msg.ToolCalls) > 0 {
			toolCall := msg.ToolCalls[0]
			toolIntent = ToolIntent{
				Id:        toolCall.Id,
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			}
			newMessages = append(newMessages, Message{
				Type:       "tool_intent",
				ToolIntent: &toolIntent,
			})
		} else {
			return nil, fmt.Errorf("(groq.go, Run) unexpected response")
		}
	}

	if toolIntent.Id != "" {
		tempAgentResult := &AgentResult{
			AllMessages:   append(msgHistory, newMessages...),
			NewMessages:   newMessages,
			Text:          finalText,
			ToolArguments: toolIntent.Arguments,
			ToolIntent:    &toolIntent,
		}
		toolResult, err := provider.ExecuteToolIntent(toolIntent)
		if err != nil {
			return tempAgentResult, err
		}
		newMessages = append(newMessages, Message{ToolResult: toolResult})
		internalAgentResult, err := provider.Run("", append(msgHistory, newMessages...))
		if err != nil {
			return tempAgentResult, err
		}
		newMessages = append(newMessages, internalAgentResult.NewMessages...)
	}

	return &AgentResult{
		AllMessages:   append(msgHistory, newMessages...),
		NewMessages:   newMessages,
		Text:          finalText,
		ToolIntent:    &toolIntent,
		ToolArguments: toolIntent.Arguments,
	}, nil
}

func (provider *Groq) RegisterTool(fn any, paramType any, desctiption string) error {
	provider.AgentConfig.RegisterTool(fn, paramType, desctiption)
	return nil
}
