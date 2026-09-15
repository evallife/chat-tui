package api

import (
	"context"
	"testing"

	"github.com/evallife/chat-tui/internal/types"
)

func TestDefaultBaseURL(t *testing.T) {
	if DefaultBaseURL(types.ProviderOllama) != "http://localhost:11434" {
		t.Fatal(DefaultBaseURL(types.ProviderOllama))
	}
	if DefaultBaseURL("openai-compatible") != "https://api.openai.com/v1" {
		t.Fatal(DefaultBaseURL("openai-compatible"))
	}
}

func TestNewChatModelUnsupported(t *testing.T) {
	_, err := NewChatModel(context.Background(), types.Config{
		Provider: "nope",
		Model:    "x",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewChatModelOpenAIAndOllama(t *testing.T) {
	ctx := context.Background()
	if _, err := NewChatModel(ctx, types.Config{
		Provider: types.ProviderOpenAI,
		APIKey:   "sk-test",
		Model:    "gpt-4o-mini",
	}); err != nil {
		t.Fatalf("openai: %v", err)
	}
	if _, err := NewChatModel(ctx, types.Config{
		Provider: types.ProviderOllama,
		Model:    "llama3.2",
	}); err != nil {
		t.Fatalf("ollama: %v", err)
	}
}

func TestSupportedProvidersIncludesCoreSet(t *testing.T) {
	keys := map[string]bool{}
	for _, p := range SupportedProviders() {
		keys[p.Key] = true
	}
	for _, want := range []string{
		types.ProviderOpenAI, types.ProviderArk, types.ProviderOllama,
		types.ProviderClaude, types.ProviderGemini,
	} {
		if !keys[want] {
			t.Fatalf("missing provider %s", want)
		}
	}
}
