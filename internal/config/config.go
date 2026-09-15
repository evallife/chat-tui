package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/evallife/chat-tui/internal/types"
)

func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".xftui.json")
}

func DefaultConfig() types.Config {
	cfg := types.Config{
		Provider: types.ProviderOpenAI,
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-3.5-turbo",
		Theme:    "night",
	}
	cfg.Normalize()
	return cfg
}

func LoadConfig() (types.Config, error) {
	path := GetConfigPath()
	file, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig(), err
	}
	var cfg types.Config
	if err := json.Unmarshal(file, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	cfg.Normalize()
	return cfg, nil
}

// LoadOrCreate loads ~/.xftui.json, or writes DefaultConfig and returns it.
// created is true when the file did not exist. A non-nil error with created=true
// means the default was returned in-memory but could not be persisted.
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
