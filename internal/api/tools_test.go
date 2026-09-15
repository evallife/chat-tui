package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evallife/chat-tui/internal/types"
)

func TestExtraToolsNonEmpty(t *testing.T) {
	cfg := types.Config{}
	cfg.Normalize()
	tools := extraTools(cfg)
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

func TestExtraToolsRespectsDisable(t *testing.T) {
	cfg := types.Config{DisableWriteFile: true, DisableRunCommand: true}
	cfg.Normalize()
	names := ToolNames(cfg)
	for _, n := range names {
		if n == "write_file" || n == "run_command" {
			t.Fatalf("disabled tool still present: %s in %v", n, names)
		}
	}
	cfg2 := types.Config{}
	cfg2.Normalize()
	names2 := ToolNames(cfg2)
	hasWrite, hasRun := false, false
	for _, n := range names2 {
		if n == "write_file" {
			hasWrite = true
		}
		if n == "run_command" {
			hasRun = true
		}
	}
	if !hasWrite || !hasRun {
		t.Fatalf("expected write_file and run_command by default, got %v", names2)
	}
}

func TestReadAndListTools(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello eino"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := &toolEnv{root: dir}
	rt, err := newReadFileTool(env)
	if err != nil {
		t.Fatal(err)
	}
	out, err := rt.InvokableRun(context.Background(), `{"path":"hello.txt"}`)
	if err != nil {
		t.Fatal(err)
	}
	if out == "" || !contains(out, "hello eino") {
		t.Fatalf("unexpected read output: %s", out)
	}
	lt, err := newListDirectoryTool(env)
	if err != nil {
		t.Fatal(err)
	}
	lout, err := lt.InvokableRun(context.Background(), `{"path":""}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(lout, "hello.txt") {
		t.Fatalf("unexpected list output: %s", lout)
	}
}

func TestResolveInWorkspaceAllowDeny(t *testing.T) {
	dir := t.TempDir()
	inside, err := resolveInWorkspace(dir, "sub/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "sub", "file.txt")
	if inside != want {
		t.Fatalf("got %q want %q", inside, want)
	}
	// Absolute path still inside root is OK.
	absInside := filepath.Join(dir, "ok.txt")
	got, err := resolveInWorkspace(dir, absInside)
	if err != nil {
		t.Fatal(err)
	}
	if got != absInside {
		t.Fatalf("abs inside: got %q", got)
	}
	// Escape via ..
	if _, err := resolveInWorkspace(dir, "../outside.txt"); err == nil {
		t.Fatal("expected escape via .. to fail")
	}
	// Absolute outside
	if _, err := resolveInWorkspace(dir, "/tmp/definitely-outside-chat-tui-test"); err == nil {
		// On some systems /tmp might be under a weird root; only fail if clearly outside.
		outside := filepath.Clean("/tmp/definitely-outside-chat-tui-test")
		if !pathInsideRoot(dir, outside) {
			t.Fatal("expected absolute outside path to fail")
		}
	}
	rootOnly, err := resolveInWorkspace(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if rootOnly != filepath.Clean(dir) {
		t.Fatalf("empty path: got %q want %q", rootOnly, dir)
	}
}

func TestEffectiveInstruction(t *testing.T) {
	cfg := types.Config{}
	cfg.Normalize()
	if got := EffectiveInstruction("", cfg); got != DefaultAgentInstruction {
		t.Fatalf("expected default instruction, got %q", got[:min(40, len(got))])
	}
	if got := EffectiveInstruction("  custom  ", cfg); got != "  custom  " {
		t.Fatalf("expected custom prompt preserved, got %q", got)
	}
	cfg.DisableDefaultInstruction = true
	if got := EffectiveInstruction("", cfg); got != "" {
		t.Fatalf("expected empty when disabled, got %q", got)
	}
}

func TestGlobAndSearchBasic(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.go"), []byte("package pkg\nconst Needle = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := &toolEnv{root: dir}

	gt, err := newGlobFilesTool(env)
	if err != nil {
		t.Fatal(err)
	}
	gout, err := gt.InvokableRun(context.Background(), `{"pattern":"**/*.go"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(gout, "a.go") {
		t.Fatalf("glob missing a.go: %s", gout)
	}

	st, err := newSearchTextTool(env)
	if err != nil {
		t.Fatal(err)
	}
	sout, err := st.InvokableRun(context.Background(), `{"query":"Needle","glob":"*.go"}`)
	if err != nil {
		t.Fatal(err)
	}
	var parsed searchTextOutput
	if err := json.Unmarshal([]byte(sout), &parsed); err != nil {
		// InvokableRun may return JSON string already
		if !contains(sout, "Needle") {
			t.Fatalf("search missing Needle: %s", sout)
		}
	} else if len(parsed.Matches) == 0 && !contains(sout, "Needle") {
		t.Fatalf("search no matches: %s", sout)
	}

	mt, err := newMakeDirectoryTool(env)
	if err != nil {
		t.Fatal(err)
	}
	mout, err := mt.InvokableRun(context.Background(), `{"path":"newdir/nested"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(mout, "newdir") {
		t.Fatalf("mkdir output: %s", mout)
	}
	if _, err := os.Stat(filepath.Join(dir, "newdir", "nested")); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeAgentDefaults(t *testing.T) {
	cfg := types.Config{}
	cfg.Normalize()
	if cfg.MaxIterations != 20 {
		t.Fatalf("MaxIterations=%d", cfg.MaxIterations)
	}
	if cfg.AgentName != "chat-tui" {
		t.Fatalf("AgentName=%q", cfg.AgentName)
	}
	cfg.MaxIterations = 500
	cfg.Normalize()
	if cfg.MaxIterations != 100 {
		t.Fatalf("clamp high: %d", cfg.MaxIterations)
	}
	cfg.MaxIterations = -3
	cfg.Normalize()
	if cfg.MaxIterations != 1 {
		t.Fatalf("clamp low: %d", cfg.MaxIterations)
	}
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
