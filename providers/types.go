package providers

type Message struct {
	Role    string `json:"role"` // developer | user | assistant
	Content string `json:"content"`
}

// type Request struct {
// 	Model     string    `json:"model"`
// 	MaxTokens int       `json:"max_tokens"`
// 	Messages  []Message `json:"messages"`
// }

// type ContentItem struct {
// 	Text string `json:"text"`
// 	Type string `json:"type"`
// }

// type Response struct {
// 	Content      []ContentItem `json:"content"`
// 	ID           string        `json:"id"`
// 	Model        string        `json:"model"`
// 	Role         string        `json:"role"`
// 	StopReason   string        `json:"stop_reason"`
// 	StopSequence any           `json:"stop_sequence"`
// 	Type         string        `json:"type"`
// 	Usage        Usage         `json:"usage"`
// }
