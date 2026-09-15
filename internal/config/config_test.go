package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/evallife/chat-tui/internal/paths"
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
	cfgPath := GetConfigPath()
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
	if cfg.WorkspaceRoot != DefaultWorkspacePath() {
		t.Fatalf("workspace=%q", cfg.WorkspaceRoot)
	}

	raw, _ := json.Marshal(cfg)
	if len(raw) == 0 {
		t.Fatal("empty marshal")
	}
}

func TestDefaultConfigHasEmptyAPIKeyAndWorkspace(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := DefaultConfig()
	if cfg.APIKey != "" {
		t.Fatalf("default API key should be empty, got %q", cfg.APIKey)
	}
	if cfg.Provider != types.ProviderOpenAI {
		t.Fatalf("provider=%q", cfg.Provider)
	}
	if cfg.WorkspaceRoot != filepath.Join(home, paths.WorkspaceDirName) {
		t.Fatalf("workspace=%q", cfg.WorkspaceRoot)
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
	if _, err := os.Stat(GetConfigPath()); err != nil {
		t.Fatal(err)
	}
	if cfg.WorkspaceRoot != DefaultWorkspacePath() {
		t.Fatalf("workspace=%q", cfg.WorkspaceRoot)
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
	if err := os.WriteFile(GetConfigPath(), []byte("{not-json"), 0644); err != nil {
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

func TestMigratesLegacyConfigAndRemovesOldFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	old := []byte(`{"api_key":"sk-legacy","model":"gpt-4o-mini","theme":"nord"}`)
	if err := os.WriteFile(LegacyConfigPath(), old, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, created, err := LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("migrated config is not a first-run create")
	}
	if cfg.APIKey != "sk-legacy" || cfg.Model != "gpt-4o-mini" {
		t.Fatalf("migrated cfg: %+v", cfg)
	}
	if cfg.WorkspaceRoot != DefaultWorkspacePath() {
		t.Fatalf("workspace not filled: %q", cfg.WorkspaceRoot)
	}
	if _, err := os.Stat(GetConfigPath()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(LegacyConfigPath()); !os.IsNotExist(err) {
		t.Fatalf("legacy config should be removed, err=%v", err)
	}
}

func TestPrefersNewConfigOverLegacy(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(LegacyConfigPath(), []byte(`{"api_key":"old","model":"old"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(GetConfigPath(), []byte(`{"api_key":"new","model":"new","workspace_root":"/tmp/ws"}`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "new" || cfg.Model != "new" {
		t.Fatalf("should keep new file: %+v", cfg)
	}
	if cfg.WorkspaceRoot != "/tmp/ws" {
		t.Fatalf("should keep explicit workspace: %q", cfg.WorkspaceRoot)
	}
	if _, err := os.Stat(LegacyConfigPath()); err != nil {
		t.Fatal("legacy file should remain when new exists")
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

func TestEnsureWorkspaceDir(t *testing.T) {
	dir := t.TempDir()
	ws := filepath.Join(dir, "workspace")
	if err := EnsureWorkspaceDir(types.Config{WorkspaceRoot: ws}); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(ws)
	if err != nil || !st.IsDir() {
		t.Fatalf("workspace dir: %v", err)
	}
}
