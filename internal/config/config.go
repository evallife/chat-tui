package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"github.com/evallife/chat-tui/internal/types"
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
