package main

import (
	"fmt"
	"os"

	"github.com/evallife/chat-tui/internal/config"
	"github.com/evallife/chat-tui/internal/storage"
	"github.com/evallife/chat-tui/internal/ui"
)

func main() {
	store, err := storage.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing storage: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	cfg, created, err := config.LoadOrCreate()
	if err != nil && !created {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	if created {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save default config: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Created default config at: %s\n", config.GetConfigPath())
		}
	}

	cfg.Normalize()
	app := ui.NewTViewUI(cfg, store)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
