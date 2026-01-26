package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
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
			// Create a default template if not exists
			defaultCfg := types.Config{
				BaseURL: "https://api.openai.com/v1",
				Model:   "gpt-3.5-turbo",
				APIKey:  "YOUR_API_KEY_HERE",
			}
			config.SaveConfig(defaultCfg)
			fmt.Printf("Created default config at: %s\n", config.GetConfigPath())
			fmt.Println("Please edit the file and add your API Key.")
			os.Exit(0)
		}
		fmt.Printf("Warning: Could not load config: %v\n", err)
	}

	if cfg.APIKey == "" || cfg.APIKey == "YOUR_API_KEY_HERE" {
		fmt.Printf("Error: API Key is required in %s\n", config.GetConfigPath())
		os.Exit(1)
	}

	p := tea.NewProgram(ui.NewModel(cfg, store), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
