package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyIfNeededMigratesOnce(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "old.json")
	to := filepath.Join(dir, "new.json")
	if err := os.WriteFile(from, []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := CopyIfNeeded(from, to)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Did {
		t.Fatal("expected copy")
	}
	got, err := os.ReadFile(to)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"ok":true}` {
		t.Fatalf("copied content: %s", got)
	}

	if err := os.WriteFile(to, []byte(`{"ok":"new"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r2, err := CopyIfNeeded(from, to)
	if err != nil {
		t.Fatal(err)
	}
	if r2.Did {
		t.Fatal("must not overwrite existing dest")
	}
	got, _ = os.ReadFile(to)
	if string(got) != `{"ok":"new"}` {
		t.Fatalf("dest overwritten: %s", got)
	}
}

func TestCopyIfNeededMissingSource(t *testing.T) {
	dir := t.TempDir()
	r, err := CopyIfNeeded(filepath.Join(dir, "nope"), filepath.Join(dir, "out"))
	if err != nil {
		t.Fatal(err)
	}
	if r.Did {
		t.Fatal("expected no copy")
	}
}

func TestHomePathsUseHOME(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if ConfigFile() != filepath.Join(home, ConfigFileName) {
		t.Fatalf("config: %s", ConfigFile())
	}
	if LegacyConfigFile() != filepath.Join(home, LegacyConfigFileName) {
		t.Fatalf("legacy config: %s", LegacyConfigFile())
	}
	if DBFile() != filepath.Join(home, DBFileName) {
		t.Fatalf("db: %s", DBFile())
	}
	if DefaultWorkspace() != filepath.Join(home, WorkspaceDirName) {
		t.Fatalf("workspace: %s", DefaultWorkspace())
	}
}
