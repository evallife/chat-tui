package main

import (
	"fmt"
	"os"

	"github.com/user/xftui/internal/config"
	"github.com/user/xftui/internal/storage"
	"github.com/user/xftui/internal/types"
	"github.com/user/xftui/internal/ui"
)

func main() {
	store, err := storage.NewManager()
	if err != nil {
		fmt.Printf("Error initializing storage: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		if os.IsNotExist(err) {
			defaultCfg := types.Config{
				BaseURL: "https://api.openai.com/v1",
				Model:   "gpt-3.5-turbo",
				APIKey:  "YOUR_API_KEY_HERE",
			}
			config.SaveConfig(defaultCfg)
			fmt.Printf("Created default config at: %s\n", config.GetConfigPath())
			os.Exit(0)
		}
	}

	app := ui.NewTViewUI(cfg, store)
	if err := app.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
