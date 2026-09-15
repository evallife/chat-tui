package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExtraToolsNonEmpty(t *testing.T) {
	tools := extraTools()
	if len(tools) < 5 {
		t.Fatalf("expected several tools, got %d", len(tools))
	}
	for _, tl := range tools {
		info, err := tl.Info(context.Background())
		if err != nil {
			t.Fatalf("Info: %v", err)
		}
		if info.Name == "" {
			t.Fatal("empty tool name")
		}
	}
}

func TestReadAndListTools(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello eino"), 0o644); err != nil {
		t.Fatal(err)
	}
	rt, err := newReadFileTool()
	if err != nil {
		t.Fatal(err)
	}
	out, err := rt.InvokableRun(context.Background(), `{"path":"`+filepath.ToSlash(path)+`"}`)
	if err != nil {
		t.Fatal(err)
	}
	if out == "" || !contains(out, "hello eino") {
		t.Fatalf("unexpected read output: %s", out)
	}
	lt, err := newListDirectoryTool()
	if err != nil {
		t.Fatal(err)
	}
	lout, err := lt.InvokableRun(context.Background(), `{"path":"`+filepath.ToSlash(dir)+`"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(lout, "hello.txt") {
		t.Fatalf("unexpected list output: %s", lout)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
