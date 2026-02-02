package config

import (
	"encoding/json"
	"github.com/evallife/chat-tui/internal/types"
	"os"
	"path/filepath"
)

func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".xftui.json")
}

func LoadConfig() (types.Config, error) {
	path := GetConfigPath()
	file, err := os.ReadFile(path)
	if err != nil {
		return types.Config{
			BaseURL: "https://api.openai.com/v1",
			Model:   "gpt-3.5-turbo",
			Theme:   "night",
		}, err
	}
	var cfg types.Config
	err = json.Unmarshal(file, &cfg)
	return cfg, err
}

func SaveConfig(cfg types.Config) error {
	path := GetConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
