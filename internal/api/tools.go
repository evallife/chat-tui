package api

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/evallife/chat-tui/internal/types"
)

const (
	maxReadBytes     = 64 << 10 // 64 KiB
	maxHTTPBytes     = 64 << 10
	maxListEntries   = 200
	maxGlobEntries   = 200
	maxSearchMatches = 50
	maxCmdOutput     = 32 << 10
	httpTimeout      = 15 * time.Second
	cmdTimeout       = 20 * time.Second
	searchTimeout    = 10 * time.Second
)

// extraTools is the ReAct tool set for ChatModelAgent, filtered by cfg policy.
func extraTools(cfg types.Config) []tool.BaseTool {
	env := newToolEnv(cfg)
	builders := []func(*toolEnv) (tool.InvokableTool, error){
		newGetCurrentTimeTool,
		newGetWorkingDirectoryTool,
		newListDirectoryTool,
		newReadFileTool,
		newHTTPGetTool,
		newGlobFilesTool,
		newSearchTextTool,
		newMakeDirectoryTool,
	}
	tools := make([]tool.BaseTool, 0, 12)
	for _, build := range builders {
		t, err := build(env)
		if err != nil {
			continue
		}
		tools = append(tools, t)
	}
	if cfg.WriteFileEnabled() {
		if t, err := newWriteFileTool(env); err == nil {
			tools = append(tools, t)
		}
	}
	if cfg.RunCommandEnabled() {
		if t, err := newRunCommandTool(env); err == nil {
			tools = append(tools, t)
		}
	}
	return tools
}

// ToolNames returns sorted names of tools registered for cfg (excluding ADK exit).
func ToolNames(cfg types.Config) []string {
	tools := extraTools(cfg)
	names := make([]string, 0, len(tools))
	for _, tl := range tools {
		info, err := tl.Info(context.Background())
		if err != nil || info.Name == "" {
			continue
		}
		names = append(names, info.Name)
	}
	sort.Strings(names)
	return names
}

type emptyInput struct{}

type timeOutput struct {
	RFC3339  string `json:"rfc3339"`
	Unix     int64  `json:"unix"`
	Timezone string `json:"timezone"`
	Local    string `json:"local"`
}

func newGetCurrentTimeTool(_ *toolEnv) (tool.InvokableTool, error) {
	return utils.InferTool("get_current_time", "Get the current local date and time.",
		func(ctx context.Context, _ emptyInput) (timeOutput, error) {
			now := time.Now()
			name, offset := now.Zone()
			return timeOutput{
				RFC3339:  now.Format(time.RFC3339),
				Unix:     now.Unix(),
				Timezone: fmt.Sprintf("%s (%+d)", name, offset),
				Local:    now.Format("2006-01-02 15:04:05"),
			}, nil
		})
}

type cwdOutput struct {
	Path          string `json:"path"`
	WorkspaceRoot string `json:"workspace_root,omitempty"`
}

func newGetWorkingDirectoryTool(env *toolEnv) (tool.InvokableTool, error) {
	root := env.root
	return utils.InferTool("get_working_directory", "Get the agent workspace root used as the tool working directory.",
		func(ctx context.Context, _ emptyInput) (cwdOutput, error) {
			return cwdOutput{Path: root, WorkspaceRoot: root}, nil
		})
}

type listDirInput struct {
	Path string `json:"path" jsonschema:"description=Directory path under workspace. Empty means workspace root."`
}

type listDirOutput struct {
	Path      string   `json:"path"`
	Entries   []string `json:"entries"`
	Truncated bool     `json:"truncated,omitempty"`
}

func newListDirectoryTool(env *toolEnv) (tool.InvokableTool, error) {
	root := env.root
	return utils.InferTool("list_directory", "List files and directories at a path under the workspace (non-recursive).",
		func(ctx context.Context, in listDirInput) (listDirOutput, error) {
			abs, err := resolveInWorkspace(root, in.Path)
			if err != nil {
				return listDirOutput{}, err
			}
			ents, err := os.ReadDir(abs)
			if err != nil {
				return listDirOutput{}, err
			}
			out := listDirOutput{Path: abs, Entries: make([]string, 0, len(ents))}
			for i, e := range ents {
				if i >= maxListEntries {
					out.Truncated = true
					break
				}
				name := e.Name()
				if e.IsDir() {
					name += "/"
				}
				out.Entries = append(out.Entries, name)
			}
			return out, nil
		})
}

type readFileInput struct {
	Path string `json:"path" jsonschema:"required,description=File path under workspace to read as UTF-8 text."`
}

type readFileOutput struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Bytes     int    `json:"bytes"`
	Truncated bool   `json:"truncated,omitempty"`
}

func newReadFileTool(env *toolEnv) (tool.InvokableTool, error) {
	root := env.root
	return utils.InferTool("read_file", "Read a text file under the workspace (capped size). Prefer for source/config/logs.",
		func(ctx context.Context, in readFileInput) (readFileOutput, error) {
			path := strings.TrimSpace(in.Path)
			if path == "" {
				return readFileOutput{}, fmt.Errorf("path is required")
			}
			abs, err := resolveInWorkspace(root, path)
			if err != nil {
				return readFileOutput{}, err
			}
			f, err := os.Open(abs)
			if err != nil {
				return readFileOutput{}, err
			}
			defer f.Close()
			limited := io.LimitReader(f, maxReadBytes+1)
			data, err := io.ReadAll(limited)
			if err != nil {
				return readFileOutput{}, err
			}
			truncated := len(data) > maxReadBytes
			if truncated {
				data = data[:maxReadBytes]
			}
			if !utf8.Valid(data) {
				return readFileOutput{}, fmt.Errorf("file is not valid UTF-8 text")
			}
			return readFileOutput{
				Path:      abs,
				Content:   string(data),
				Bytes:     len(data),
				Truncated: truncated,
			}, nil
		})
}

type writeFileInput struct {
	Path    string `json:"path" jsonschema:"required,description=File path under workspace to write."`
	Content string `json:"content" jsonschema:"required,description=UTF-8 text content to write."`
	Append  bool   `json:"append,omitempty" jsonschema:"description=If true, append instead of overwrite."`
}

type writeFileOutput struct {
	Path  string `json:"path"`
	Bytes int    `json:"bytes"`
}

func newWriteFileTool(env *toolEnv) (tool.InvokableTool, error) {
	root := env.root
	return utils.InferTool("write_file", "Write UTF-8 text to a file under the workspace (create or overwrite; optional append).",
		func(ctx context.Context, in writeFileInput) (writeFileOutput, error) {
			path := strings.TrimSpace(in.Path)
			if path == "" {
				return writeFileOutput{}, fmt.Errorf("path is required")
			}
			if len(in.Content) > maxReadBytes {
				return writeFileOutput{}, fmt.Errorf("content exceeds %d bytes", maxReadBytes)
			}
			abs, err := resolveInWorkspace(root, path)
			if err != nil {
				return writeFileOutput{}, err
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return writeFileOutput{}, err
			}
			flag := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
			if in.Append {
				flag = os.O_CREATE | os.O_WRONLY | os.O_APPEND
			}
			f, err := os.OpenFile(abs, flag, 0o644)
			if err != nil {
				return writeFileOutput{}, err
			}
			defer f.Close()
			n, err := f.WriteString(in.Content)
			if err != nil {
				return writeFileOutput{}, err
			}
			return writeFileOutput{Path: abs, Bytes: n}, nil
		})
}

type httpGetInput struct {
	URL string `json:"url" jsonschema:"required,description=HTTP or HTTPS URL to GET."`
}

type httpGetOutput struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body"`
	Truncated  bool              `json:"truncated,omitempty"`
}

func newHTTPGetTool(_ *toolEnv) (tool.InvokableTool, error) {
	return utils.InferTool("http_get", "HTTP GET a URL and return status, selected headers, and text body (capped).",
		func(ctx context.Context, in httpGetInput) (httpGetOutput, error) {
			u := strings.TrimSpace(in.URL)
			if u == "" {
				return httpGetOutput{}, fmt.Errorf("url is required")
			}
			if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
				return httpGetOutput{}, fmt.Errorf("url must start with http:// or https://")
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
			if err != nil {
				return httpGetOutput{}, err
			}
			req.Header.Set("User-Agent", "chat-tui-eino-agent/0.9")
			client := &http.Client{Timeout: httpTimeout}
			resp, err := client.Do(req)
			if err != nil {
				return httpGetOutput{}, err
			}
			defer resp.Body.Close()
			limited := io.LimitReader(resp.Body, maxHTTPBytes+1)
			data, err := io.ReadAll(limited)
			if err != nil {
				return httpGetOutput{}, err
			}
			truncated := len(data) > maxHTTPBytes
			if truncated {
				data = data[:maxHTTPBytes]
			}
			headers := map[string]string{}
			for _, k := range []string{"Content-Type", "Content-Length", "Location"} {
				if v := resp.Header.Get(k); v != "" {
					headers[k] = v
				}
			}
			body := string(data)
			if !utf8.Valid(data) {
				body = fmt.Sprintf("<binary %d bytes>", len(data))
			}
			return httpGetOutput{
				StatusCode: resp.StatusCode,
				Headers:    headers,
				Body:       body,
				Truncated:  truncated,
			}, nil
		})
}

type runCommandInput struct {
	Command string `json:"command" jsonschema:"required,description=Shell command to run via /bin/sh -c (or cmd on Windows). Keep it short and non-interactive. Runs with cwd=workspace root."`
}

type runCommandOutput struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Dir      string `json:"dir,omitempty"`
	TimedOut bool   `json:"timed_out,omitempty"`
}

func newRunCommandTool(env *toolEnv) (tool.InvokableTool, error) {
	root := env.root
	return utils.InferTool("run_command", "Run a short non-interactive shell command in the workspace root with timeout; returns stdout/stderr (capped).",
		func(ctx context.Context, in runCommandInput) (runCommandOutput, error) {
			cmdLine := strings.TrimSpace(in.Command)
			if cmdLine == "" {
				return runCommandOutput{}, fmt.Errorf("command is required")
			}
			cctx, cancel := context.WithTimeout(ctx, cmdTimeout)
			defer cancel()
			var cmd *exec.Cmd
			if filepath.Separator == '\\' {
				cmd = exec.CommandContext(cctx, "cmd", "/C", cmdLine)
			} else {
				cmd = exec.CommandContext(cctx, "/bin/sh", "-c", cmdLine)
			}
			cmd.Dir = root
			var stdout, stderr strings.Builder
			cmd.Stdout = &capWriter{w: &stdout, n: maxCmdOutput}
			cmd.Stderr = &capWriter{w: &stderr, n: maxCmdOutput}
			err := cmd.Run()
			out := runCommandOutput{
				Stdout: stdout.String(),
				Stderr: stderr.String(),
				Dir:    root,
			}
			if cctx.Err() == context.DeadlineExceeded {
				out.TimedOut = true
				out.ExitCode = -1
				return out, nil
			}
			if err == nil {
				out.ExitCode = 0
				return out, nil
			}
			if ee, ok := err.(*exec.ExitError); ok {
				out.ExitCode = ee.ExitCode()
				return out, nil
			}
			return out, err
		})
}

type globFilesInput struct {
	Pattern string `json:"pattern" jsonschema:"required,description=Glob pattern relative to path (e.g. *.go or **/*.md)."`
	Path    string `json:"path,omitempty" jsonschema:"description=Optional subdirectory under workspace to search from."`
}

type globFilesOutput struct {
	Root      string   `json:"root"`
	Matches   []string `json:"matches"`
	Truncated bool     `json:"truncated,omitempty"`
}

func newGlobFilesTool(env *toolEnv) (tool.InvokableTool, error) {
	root := env.root
	return utils.InferTool("glob_files", "List files under the workspace matching a glob pattern (capped).",
		func(ctx context.Context, in globFilesInput) (globFilesOutput, error) {
			pattern := strings.TrimSpace(in.Pattern)
			if pattern == "" {
				return globFilesOutput{}, fmt.Errorf("pattern is required")
			}
			base, err := resolveInWorkspace(root, in.Path)
			if err != nil {
				return globFilesOutput{}, err
			}
			matches, truncated, err := globUnder(base, pattern, maxGlobEntries)
			if err != nil {
				return globFilesOutput{}, err
			}
			relMatches := make([]string, 0, len(matches))
			for _, m := range matches {
				if rel, err := filepath.Rel(root, m); err == nil && !strings.HasPrefix(rel, "..") {
					relMatches = append(relMatches, filepath.ToSlash(rel))
				} else {
					relMatches = append(relMatches, m)
				}
			}
			return globFilesOutput{Root: base, Matches: relMatches, Truncated: truncated}, nil
		})
}

func globUnder(base, pattern string, limit int) ([]string, bool, error) {
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	if strings.Contains(pattern, "**") {
		return walkGlob(base, pattern, limit)
	}
	full := filepath.Join(base, filepath.FromSlash(pattern))
	hits, err := filepath.Glob(full)
	if err != nil {
		return nil, false, err
	}
	sort.Strings(hits)
	truncated := false
	if len(hits) > limit {
		hits = hits[:limit]
		truncated = true
	}
	return hits, truncated, nil
}

func walkGlob(base, pattern string, limit int) ([]string, bool, error) {
	out := make([]string, 0, 32)
	truncated := false
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return nil
		}
		relSlash := filepath.ToSlash(rel)
		if !matchDoublestar(pattern, relSlash) {
			return nil
		}
		if len(out) >= limit {
			truncated = true
			return filepath.SkipAll
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	sort.Strings(out)
	return out, truncated, nil
}

// matchDoublestar matches patterns that may contain ** (any directory depth).
func matchDoublestar(pattern, name string) bool {
	pattern = filepath.ToSlash(pattern)
	name = filepath.ToSlash(name)
	if !strings.Contains(pattern, "**") {
		ok, _ := filepath.Match(pattern, name)
		return ok
	}
	parts := strings.Split(pattern, "**")
	for i := range parts {
		parts[i] = strings.Trim(parts[i], "/")
	}
	// Leading **: name may start anywhere relative to first non-empty part.
	rest := name
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i == len(parts)-1 && !strings.Contains(pattern[strings.LastIndex(pattern, "**"):], "/") {
			// Final part after ** often is a file pattern like *.go
			okBase, _ := filepath.Match(part, filepath.Base(rest))
			okFull, _ := filepath.Match(part, rest)
			if okBase || okFull {
				return true
			}
			// Also allow part to match a trailing suffix path
			segs := strings.Split(rest, "/")
			for j := 0; j < len(segs); j++ {
				suffix := strings.Join(segs[j:], "/")
				if ok, _ := filepath.Match(part, suffix); ok {
					return true
				}
			}
			return false
		}
		found := false
		segs := strings.Split(rest, "/")
		for j := 0; j <= len(segs); j++ {
			candidate := strings.Join(segs[:j], "/")
			if ok, _ := filepath.Match(part, candidate); ok {
				rest = strings.Join(segs[j:], "/")
				found = true
				break
			}
			if j < len(segs) {
				// multi-segment part
				for k := j + 1; k <= len(segs); k++ {
					candidate = strings.Join(segs[j:k], "/")
					if ok, _ := filepath.Match(part, candidate); ok {
						rest = strings.Join(segs[k:], "/")
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
		if !found {
			return false
		}
	}
	return true
}

type searchTextInput struct {
	Query string `json:"query" jsonschema:"required,description=Substring to search for in text files."`
	Path  string `json:"path,omitempty" jsonschema:"description=Optional subdirectory under workspace."`
	Glob  string `json:"glob,omitempty" jsonschema:"description=Optional file glob filter (e.g. *.go)."`
}

type searchMatch struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

type searchTextOutput struct {
	Matches   []searchMatch `json:"matches"`
	Truncated bool          `json:"truncated,omitempty"`
	Engine    string        `json:"engine,omitempty"`
}

func newSearchTextTool(env *toolEnv) (tool.InvokableTool, error) {
	root := env.root
	return utils.InferTool("search_text", "Search for a text substring under the workspace (prefers rg; capped matches).",
		func(ctx context.Context, in searchTextInput) (searchTextOutput, error) {
			q := in.Query
			if q == "" {
				return searchTextOutput{}, fmt.Errorf("query is required")
			}
			base, err := resolveInWorkspace(root, in.Path)
			if err != nil {
				return searchTextOutput{}, err
			}
			cctx, cancel := context.WithTimeout(ctx, searchTimeout)
			defer cancel()

			if rgPath, err := exec.LookPath("rg"); err == nil {
				out, truncated, err := searchWithRg(cctx, rgPath, base, q, in.Glob, root)
				if err == nil {
					return searchTextOutput{Matches: out, Truncated: truncated, Engine: "rg"}, nil
				}
				if cctx.Err() != nil {
					return searchTextOutput{}, cctx.Err()
				}
			}
			out, truncated, err := searchWithWalk(cctx, base, q, in.Glob, root)
			if err != nil {
				return searchTextOutput{}, err
			}
			return searchTextOutput{Matches: out, Truncated: truncated, Engine: "walk"}, nil
		})
}

func searchWithRg(ctx context.Context, rgPath, base, query, glob, workspaceRoot string) ([]searchMatch, bool, error) {
	args := []string{"--line-number", "--with-filename", "--no-heading", "--color", "never",
		"--max-count", "20", "-F", query}
	if g := strings.TrimSpace(glob); g != "" {
		args = append(args, "--glob", g)
	}
	args = append(args, ".")
	cmd := exec.CommandContext(ctx, rgPath, args...)
	cmd.Dir = base
	var stdout strings.Builder
	cmd.Stdout = &capWriter{w: &stdout, n: maxCmdOutput}
	cmd.Stderr = io.Discard
	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			return nil, false, ctx.Err()
		}
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return nil, false, nil
		}
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() > 1 {
			return nil, false, err
		}
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, false, err
		}
	}
	return parseRgOutput(stdout.String(), workspaceRoot, base)
}

func parseRgOutput(raw, workspaceRoot, base string) ([]searchMatch, bool, error) {
	matches := make([]searchMatch, 0, 16)
	truncated := false
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		path, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		linenoStr, content, ok := strings.Cut(rest, ":")
		if !ok {
			continue
		}
		var lineno int
		if _, err := fmt.Sscanf(linenoStr, "%d", &lineno); err != nil {
			continue
		}
		p := path
		abs := path
		if !filepath.IsAbs(path) {
			abs = filepath.Join(base, path)
		}
		if rel, err := filepath.Rel(workspaceRoot, abs); err == nil && !strings.HasPrefix(rel, "..") {
			p = filepath.ToSlash(rel)
		}
		matches = append(matches, searchMatch{Path: p, Line: lineno, Content: strings.TrimRight(content, "\r")})
		if len(matches) >= maxSearchMatches {
			truncated = true
			break
		}
	}
	return matches, truncated, nil
}

func searchWithWalk(ctx context.Context, base, query, glob, workspaceRoot string) ([]searchMatch, bool, error) {
	matches := make([]searchMatch, 0, 16)
	truncated := false
	glob = strings.TrimSpace(glob)
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if truncated {
			return filepath.SkipAll
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if glob != "" {
			okName, _ := filepath.Match(glob, d.Name())
			rel := filepath.ToSlash(mustRel(base, path))
			okRel := matchDoublestar(glob, rel)
			if !okName && !okRel {
				return nil
			}
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		var head [512]byte
		n, _ := f.Read(head[:])
		if n > 0 && (bytesContainNull(head[:n]) || !utf8.Valid(head[:n])) {
			return nil
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return nil
		}
		sc := bufio.NewScanner(f)
		buf := make([]byte, 0, 64*1024)
		sc.Buffer(buf, 1024*1024)
		lineNo := 0
		for sc.Scan() {
			lineNo++
			line := sc.Text()
			if !strings.Contains(line, query) {
				continue
			}
			rel := path
			if r, err := filepath.Rel(workspaceRoot, path); err == nil {
				rel = filepath.ToSlash(r)
			}
			matches = append(matches, searchMatch{Path: rel, Line: lineNo, Content: line})
			if len(matches) >= maxSearchMatches {
				truncated = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	if err != nil && err != filepath.SkipAll {
		return matches, truncated, err
	}
	return matches, truncated, nil
}

func mustRel(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return rel
}

func bytesContainNull(b []byte) bool {
	for _, c := range b {
		if c == 0 {
			return true
		}
	}
	return false
}

type makeDirInput struct {
	Path string `json:"path" jsonschema:"required,description=Directory path under workspace to create (mkdir -p)."`
}

type makeDirOutput struct {
	Path string `json:"path"`
}

func newMakeDirectoryTool(env *toolEnv) (tool.InvokableTool, error) {
	root := env.root
	return utils.InferTool("make_directory", "Create a directory under the workspace (including parents).",
		func(ctx context.Context, in makeDirInput) (makeDirOutput, error) {
			path := strings.TrimSpace(in.Path)
			if path == "" {
				return makeDirOutput{}, fmt.Errorf("path is required")
			}
			abs, err := resolveInWorkspace(root, path)
			if err != nil {
				return makeDirOutput{}, err
			}
			if err := os.MkdirAll(abs, 0o755); err != nil {
				return makeDirOutput{}, err
			}
			return makeDirOutput{Path: abs}, nil
		})
}

type capWriter struct {
	w *strings.Builder
	n int
}

func (c *capWriter) Write(p []byte) (int, error) {
	remain := c.n - c.w.Len()
	if remain <= 0 {
		return len(p), nil
	}
	if len(p) > remain {
		_, _ = c.w.Write(p[:remain])
		return len(p), nil
	}
	return c.w.Write(p)
}
