package types

import (
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

type Config struct {
	// Provider selects an eino-ext ChatModel implementation.
	// Empty values default to "openai" (OpenAI-compatible) for backward compatibility.
	Provider string `json:"provider,omitempty"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	Theme    string `json:"theme"`
	// Region is used by Ark (Volcengine) and optional Claude Bedrock setups.
	Region string `json:"region,omitempty"`
	// AccessKey / SecretKey are Ark alternatives to api_key.
	AccessKey string `json:"access_key,omitempty"`
	SecretKey string `json:"secret_key,omitempty"`

	// MaxIterations caps ChatModelAgent generation cycles (default 20, clamped 1–100).
	MaxIterations int `json:"max_iterations,omitempty"`
	// WorkspaceRoot sandboxes file/shell tools. Empty means process cwd.
	// New and migrated configs default to ~/chat-tui-workspace.
	WorkspaceRoot string `json:"workspace_root,omitempty"`
	// AgentName is the ADK ChatModelAgent Name (default "chat-tui").
	AgentName string `json:"agent_name,omitempty"`
	// DisableWriteFile skips registering write_file when true (default false = enabled).
	DisableWriteFile bool `json:"disable_write_file,omitempty"`
	// DisableRunCommand skips registering run_command when true (default false = enabled).
	DisableRunCommand bool `json:"disable_run_command,omitempty"`
	// DisableDefaultInstruction skips injecting the built-in agent instruction when system prompt is empty.
	DisableDefaultInstruction bool `json:"disable_default_instruction,omitempty"`
}

const (
	ProviderOpenAI   = "openai"
	ProviderArk      = "ark"
	ProviderOllama   = "ollama"
	ProviderClaude   = "claude"
	ProviderGemini   = "gemini"
	ProviderQwen     = "qwen"
	ProviderDeepSeek = "deepseek"
)

// CanonicalProvider maps aliases and empty values to a supported provider id.
func CanonicalProvider(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "", "openai-compatible", "openai_compatible", "openai":
		return ProviderOpenAI
	case "ark", "volcengine", "doubao", "volc":
		return ProviderArk
	case "ollama":
		return ProviderOllama
	case "claude", "anthropic":
		return ProviderClaude
	case "gemini", "google":
		return ProviderGemini
	case "qwen", "dashscope", "tongyi":
		return ProviderQwen
	case "deepseek":
		return ProviderDeepSeek
	default:
		return strings.ToLower(strings.TrimSpace(p))
	}
}

// Normalize fills backward-compatible defaults in place.
func (c *Config) Normalize() {
	c.Provider = CanonicalProvider(c.Provider)
	if c.Theme == "" {
		c.Theme = "night"
	}
	if c.MaxIterations == 0 {
		c.MaxIterations = 20
	}
	if c.MaxIterations < 1 {
		c.MaxIterations = 1
	}
	if c.MaxIterations > 100 {
		c.MaxIterations = 100
	}
	if c.AgentName == "" {
		c.AgentName = "chat-tui"
	}
}

func (c Config) CanonicalProvider() string {
	return CanonicalProvider(c.Provider)
}

// WriteFileEnabled reports whether write_file should be registered.
func (c Config) WriteFileEnabled() bool {
	return !c.DisableWriteFile
}

// RunCommandEnabled reports whether run_command should be registered.
func (c Config) RunCommandEnabled() bool {
	return !c.DisableRunCommand
}

// DefaultInstructionEnabled reports whether the built-in instruction may be used.
func (c Config) DefaultInstructionEnabled() bool {
	return !c.DisableDefaultInstruction
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
