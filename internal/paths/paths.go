package paths

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	ConfigFileName       = ".chat-tui.json"
	LegacyConfigFileName = ".xftui.json"
	DBFileName           = ".chat-tui.db"
	LegacyDBFileName     = ".xftui.db"
	WorkspaceDirName     = "chat-tui-workspace"
)

// Relocation describes a one-time move from a legacy path to the current path.
type Relocation struct {
	From string
	To   string
	Did  bool
}

func HomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

func ConfigFile() string {
	return filepath.Join(HomeDir(), ConfigFileName)
}

func LegacyConfigFile() string {
	return filepath.Join(HomeDir(), LegacyConfigFileName)
}

func DBFile() string {
	return filepath.Join(HomeDir(), DBFileName)
}

func LegacyDBFile() string {
	return filepath.Join(HomeDir(), LegacyDBFileName)
}

func DefaultWorkspace() string {
	return filepath.Join(HomeDir(), WorkspaceDirName)
}

// CopyIfNeeded copies from → to when to is missing and from exists.
func CopyIfNeeded(from, to string) (Relocation, error) {
	r := Relocation{From: from, To: to}
	if from == "" || to == "" || from == to {
		return r, nil
	}
	if _, err := os.Stat(to); err == nil {
		return r, nil
	} else if err != nil && !os.IsNotExist(err) {
		return r, err
	}
	if _, err := os.Stat(from); err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return r, err
	}
	if err := copyFile(from, to); err != nil {
		return r, err
	}
	r.Did = true
	return r, nil
}

func copyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(to, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		_ = os.Remove(to)
		return fmt.Errorf("copy %s → %s: %w", from, to, err)
	}
	if err := out.Sync(); err != nil {
		_ = os.Remove(to)
		return err
	}
	info, err := os.Stat(from)
	if err == nil {
		_ = os.Chmod(to, info.Mode())
	}
	return nil
}
