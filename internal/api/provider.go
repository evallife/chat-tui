package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/components/model"
	"github.com/evallife/chat-tui/internal/types"
	"google.golang.org/genai"
)

// ProviderInfo describes a ChatModel backend shown in Settings.
type ProviderInfo struct {
	Key         string
	Label       string
	NeedsAPIKey bool
	NeedsRegion bool
	NeedsAKSK   bool
	Hint        string
}

// SupportedProviders is the set of eino-ext ChatModel packages wired in this build.
func SupportedProviders() []ProviderInfo {
	return []ProviderInfo{
		{Key: types.ProviderOpenAI, Label: "openai (OpenAI-compatible)", NeedsAPIKey: true, Hint: "Any OpenAI-compatible API"},
		{Key: types.ProviderArk, Label: "ark (Volcengine)", NeedsAPIKey: true, NeedsRegion: true, NeedsAKSK: true, Hint: "Doubao / Ark endpoint id"},
		{Key: types.ProviderOllama, Label: "ollama (local)", Hint: "Default http://localhost:11434"},
		{Key: types.ProviderClaude, Label: "claude (Anthropic)", NeedsAPIKey: true, NeedsRegion: true, Hint: "Anthropic Messages API"},
		{Key: types.ProviderGemini, Label: "gemini (Google)", NeedsAPIKey: true, Hint: "Gemini Developer API"},
		{Key: types.ProviderQwen, Label: "qwen (DashScope)", NeedsAPIKey: true, Hint: "Alibaba DashScope compatible-mode"},
		{Key: types.ProviderDeepSeek, Label: "deepseek", NeedsAPIKey: true, Hint: "DeepSeek chat API"},
	}
}

func ProviderKeys() []string {
	infos := SupportedProviders()
	keys := make([]string, len(infos))
	for i, p := range infos {
		keys[i] = p.Key
	}
	return keys
}

func DefaultBaseURL(provider string) string {
	switch types.CanonicalProvider(provider) {
	case types.ProviderOpenAI:
		return "https://api.openai.com/v1"
	case types.ProviderOllama:
		return "http://localhost:11434"
	case types.ProviderQwen:
		return "https://dashscope.aliyuncs.com/compatible-mode/v1"
	case types.ProviderDeepSeek:
		return "https://api.deepseek.com/"
	default:
		return ""
	}
}

func DefaultModel(provider string) string {
	switch types.CanonicalProvider(provider) {
	case types.ProviderArk:
		return "doubao-pro-32k"
	case types.ProviderOllama:
		return "llama3.2"
	case types.ProviderClaude:
		return "claude-sonnet-4-20250514"
	case types.ProviderGemini:
		return "gemini-2.0-flash"
	case types.ProviderQwen:
		return "qwen-plus"
	case types.ProviderDeepSeek:
		return "deepseek-chat"
	default:
		return "gpt-4o-mini"
	}
}

func IsKnownDefaultBaseURL(url string) bool {
	url = strings.TrimRight(strings.TrimSpace(url), "/")
	if url == "" {
		return true
	}
	for _, p := range ProviderKeys() {
		def := strings.TrimRight(DefaultBaseURL(p), "/")
		if def != "" && def == url {
			return true
		}
	}
	return false
}

// NewChatModel builds an eino ToolCallingChatModel for cfg.Provider.
func NewChatModel(ctx context.Context, cfg types.Config) (model.ToolCallingChatModel, error) {
	cfg.Normalize()
	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required")
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL(cfg.Provider)
	}

	switch cfg.Provider {
	case types.ProviderOpenAI:
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:  cfg.APIKey,
			BaseURL: baseURL,
			Model:   cfg.Model,
		})
	case types.ProviderArk:
		return ark.NewChatModel(ctx, &ark.ChatModelConfig{
			APIKey:    cfg.APIKey,
			BaseURL:   baseURL,
			Region:    cfg.Region,
			Model:     cfg.Model,
			AccessKey: cfg.AccessKey,
			SecretKey: cfg.SecretKey,
		})
	case types.ProviderOllama:
		if baseURL == "" {
			baseURL = DefaultBaseURL(types.ProviderOllama)
		}
		return ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
			BaseURL: baseURL,
			Model:   cfg.Model,
		})
	case types.ProviderClaude:
		var base *string
		if baseURL != "" {
			u := baseURL
			base = &u
		}
		return claude.NewChatModel(ctx, &claude.Config{
			APIKey:    cfg.APIKey,
			BaseURL:   base,
			Model:     cfg.Model,
			MaxTokens: 4096,
			Region:    cfg.Region,
		})
	case types.ProviderGemini:
		gcfg := &genai.ClientConfig{
			APIKey:  cfg.APIKey,
			Backend: genai.BackendGeminiAPI,
		}
		if baseURL != "" {
			gcfg.HTTPOptions.BaseURL = baseURL
		}
		client, err := genai.NewClient(ctx, gcfg)
		if err != nil {
			return nil, fmt.Errorf("gemini client: %w", err)
		}
		return gemini.NewChatModel(ctx, &gemini.Config{
			Client: client,
			Model:  cfg.Model,
		})
	case types.ProviderQwen:
		if baseURL == "" {
			baseURL = DefaultBaseURL(types.ProviderQwen)
		}
		return qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
			APIKey:  cfg.APIKey,
			BaseURL: baseURL,
			Model:   cfg.Model,
		})
	case types.ProviderDeepSeek:
		return deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
			APIKey:  cfg.APIKey,
			BaseURL: baseURL,
			Model:   cfg.Model,
		})
	default:
		return nil, fmt.Errorf("unsupported provider %q (supported: %s)", cfg.Provider, strings.Join(ProviderKeys(), ", "))
	}
}
