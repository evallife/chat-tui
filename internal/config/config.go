package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/evallife/chat-tui/internal/paths"
	"github.com/evallife/chat-tui/internal/types"
)

func GetConfigPath() string {
	return paths.ConfigFile()
}

func LegacyConfigPath() string {
	return paths.LegacyConfigFile()
}

func DefaultWorkspacePath() string {
	return paths.DefaultWorkspace()
}

func DefaultConfig() types.Config {
	cfg := types.Config{
		Provider:      types.ProviderOpenAI,
		BaseURL:       "https://api.openai.com/v1",
		Model:         "gpt-3.5-turbo",
		Theme:         "night",
		WorkspaceRoot: DefaultWorkspacePath(),
	}
	cfg.Normalize()
	return cfg
}

func LoadConfig() (types.Config, error) {
	rel, err := paths.CopyIfNeeded(LegacyConfigPath(), GetConfigPath())
	if err != nil {
		return DefaultConfig(), fmt.Errorf("migrate legacy config: %w", err)
	}

	file, err := os.ReadFile(GetConfigPath())
	if err != nil {
		return DefaultConfig(), err
	}
	var cfg types.Config
	if err := json.Unmarshal(file, &cfg); err != nil {
		if rel.Did {
			_ = os.Remove(GetConfigPath())
		}
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	cfg.Normalize()
	if applyDefaultWorkspace(&cfg) {
		if err := SaveConfig(cfg); err != nil {
			return cfg, err
		}
	}
	if rel.Did {
		_ = os.Remove(rel.From)
	}
	return cfg, nil
}

// LoadOrCreate loads ~/.chat-tui.json (migrating ~/.xftui.json if needed),
// or writes DefaultConfig and returns it.
// created is true when neither current nor legacy config existed.
// A non-nil error with created=true means the default was returned in-memory
// but could not be persisted.
func LoadOrCreate() (cfg types.Config, created bool, err error) {
	cfg, err = LoadConfig()
	if err == nil {
		return cfg, false, nil
	}
	if !os.IsNotExist(err) {
		return types.Config{}, false, err
	}
	cfg = DefaultConfig()
	if err := SaveConfig(cfg); err != nil {
		return cfg, true, err
	}
	return cfg, true, nil
}

func SaveConfig(cfg types.Config) error {
	cfg.Normalize()
	path := GetConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config to %s: %w", path, err)
	}
	return nil
}

func applyDefaultWorkspace(cfg *types.Config) bool {
	if cfg == nil || strings.TrimSpace(cfg.WorkspaceRoot) != "" {
		return false
	}
	cfg.WorkspaceRoot = DefaultWorkspacePath()
	return true
}

// EnsureWorkspaceDir creates WorkspaceRoot when it is set.
func EnsureWorkspaceDir(cfg types.Config) error {
	root := strings.TrimSpace(cfg.WorkspaceRoot)
	if root == "" {
		return nil
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("create workspace %s: %w", root, err)
	}
	return nil
}
