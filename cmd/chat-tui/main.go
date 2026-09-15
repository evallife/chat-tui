package main

import (
	"fmt"
	"os"

	"github.com/evallife/chat-tui/internal/config"
	"github.com/evallife/chat-tui/internal/storage"
	"github.com/evallife/chat-tui/internal/ui"
)

func main() {
	willMigrateCfg := missing(config.GetConfigPath()) && exists(config.LegacyConfigPath())
	willMigrateDB := missing(storage.DefaultDBPath()) && exists(storage.LegacyDBPath())

	store, err := storage.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing storage: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	if willMigrateDB {
		fmt.Fprintf(os.Stderr, "Migrated database:\n  %s\n  → %s\n", storage.LegacyDBPath(), store.Path())
	}

	cfg, created, err := config.LoadOrCreate()
	if err != nil && !created {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	if willMigrateCfg {
		fmt.Fprintf(os.Stderr, "Migrated config:\n  %s\n  → %s\n", config.LegacyConfigPath(), config.GetConfigPath())
	}
	if created {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save default config: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Created default config at: %s\n", config.GetConfigPath())
		}
	}

	cfg.Normalize()
	if err := config.EnsureWorkspaceDir(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	} else if root := cfg.WorkspaceRoot; root != "" {
		if created || willMigrateCfg {
			fmt.Fprintf(os.Stderr, "Workspace: %s\n", root)
		}
	}

	app := ui.NewTViewUI(cfg, store)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func missing(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}
