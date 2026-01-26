package types

import (
	"time"
	"github.com/sashabaranov/go-openai"
)

type Config struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}

type Conversation struct {
	ID           string                         `json:"id"`
	Title        string                         `json:"title"`
	Model        string                         `json:"model"`
	SystemPrompt string                         `json:"system_prompt"`
	Messages     []openai.ChatCompletionMessage `json:"messages"`
	CreatedAt    time.Time                      `json:"created_at"`
}

type SystemPrompt struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
}
