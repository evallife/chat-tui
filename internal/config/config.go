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

func defaultConfig() types.Config {
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
		return defaultConfig(), err
	}
	var cfg types.Config
	if err := json.Unmarshal(file, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	cfg.Normalize()
	return cfg, nil
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
