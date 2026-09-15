package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evallife/chat-tui/internal/types"
)

// toolEnv carries workspace sandbox settings for FS/shell tools.
type toolEnv struct {
	root string
}

func newToolEnv(cfg types.Config) *toolEnv {
	return &toolEnv{root: workspaceRoot(cfg)}
}

// workspaceRoot returns the absolute sandbox root. Empty WorkspaceRoot → cwd.
func workspaceRoot(cfg types.Config) string {
	root := strings.TrimSpace(cfg.WorkspaceRoot)
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "."
		}
		abs, err := filepath.Abs(wd)
		if err != nil {
			return wd
		}
		return abs
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return filepath.Clean(root)
	}
	return abs
}

// resolveInWorkspace joins path under root (if relative), cleans it, and rejects escapes.
// Empty path means the workspace root itself.
func resolveInWorkspace(root, path string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("workspace root: %w", err)
	}
	rootAbs = filepath.Clean(rootAbs)

	path = strings.TrimSpace(path)
	var candidate string
	if path == "" || path == "." {
		candidate = rootAbs
	} else if filepath.IsAbs(path) {
		candidate = filepath.Clean(path)
	} else {
		candidate = filepath.Clean(filepath.Join(rootAbs, path))
	}
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)

	if !pathInsideRoot(rootAbs, abs) {
		return "", fmt.Errorf("path %q escapes workspace root %q", abs, rootAbs)
	}
	return abs, nil
}

func pathInsideRoot(root, abs string) bool {
	if abs == root {
		return true
	}
	sep := string(filepath.Separator)
	prefix := root
	if !strings.HasSuffix(prefix, sep) {
		prefix += sep
	}
	return strings.HasPrefix(abs, prefix)
}
