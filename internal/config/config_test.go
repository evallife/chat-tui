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

func TestDefaultConfigHasEmptyAPIKey(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.APIKey != "" {
		t.Fatalf("default API key should be empty, got %q", cfg.APIKey)
	}
	if cfg.Provider != types.ProviderOpenAI {
		t.Fatalf("provider=%q", cfg.Provider)
	}
}

func TestLoadOrCreateWritesDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, created, err := LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected created=true on first run")
	}
	if cfg.APIKey != "" {
		t.Fatalf("API key=%q, want empty", cfg.APIKey)
	}
	if _, err := os.Stat(filepath.Join(home, ".xftui.json")); err != nil {
		t.Fatal(err)
	}

	cfg2, created2, err := LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	if created2 {
		t.Fatal("second load should not create")
	}
	if cfg2.Model != cfg.Model {
		t.Fatalf("model=%q want %q", cfg2.Model, cfg.Model)
	}
}

func TestLoadOrCreateParseError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".xftui.json"), []byte("{not-json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, created, err := LoadOrCreate()
	if err == nil {
		t.Fatal("expected parse error")
	}
	if created {
		t.Fatal("parse error must not report created")
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
