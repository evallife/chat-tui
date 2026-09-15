package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/evallife/chat-tui/internal/types"
)

func TestCanonicalProviderDefaults(t *testing.T) {
	cases := map[string]string{
		"":                  types.ProviderOpenAI,
		"openai-compatible": types.ProviderOpenAI,
		"anthropic":         types.ProviderClaude,
		"google":            types.ProviderGemini,
		"doubao":            types.ProviderArk,
		"dashscope":         types.ProviderQwen,
		"OLLAMA":            types.ProviderOllama,
	}
	for in, want := range cases {
		if got := types.CanonicalProvider(in); got != want {
			t.Fatalf("CanonicalProvider(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestLoadConfigMissingProviderDefaultsToOpenAI(t *testing.T) {
	legacy := []byte(`{"base_url":"https://api.openai.com/v1","api_key":"sk-test","model":"gpt-4","theme":"nord"}`)
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgPath := filepath.Join(home, ".xftui.json")
	if err := os.WriteFile(cfgPath, legacy, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != types.ProviderOpenAI {
		t.Fatalf("provider=%q, want openai", cfg.Provider)
	}
	if cfg.Model != "gpt-4" {
		t.Fatalf("model=%q", cfg.Model)
	}

	raw, _ := json.Marshal(cfg)
	if len(raw) == 0 {
		t.Fatal("empty marshal")
	}
}

func TestSaveConfigNormalizesProvider(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := SaveConfig(types.Config{
		Provider: "anthropic",
		APIKey:   "k",
		Model:    "claude-3-haiku",
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != types.ProviderClaude {
		t.Fatalf("provider=%q, want claude", cfg.Provider)
	}
}
