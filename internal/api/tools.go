package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

const (
	maxReadBytes   = 64 << 10 // 64 KiB
	maxHTTPBytes   = 64 << 10
	maxListEntries = 200
	maxCmdOutput   = 32 << 10
	httpTimeout    = 15 * time.Second
	cmdTimeout     = 20 * time.Second
)

// extraTools is the ReAct tool set for ChatModelAgent.
func extraTools() []tool.BaseTool {
	tools := make([]tool.BaseTool, 0, 8)
	for _, build := range []func() (tool.InvokableTool, error){
		newGetCurrentTimeTool,
		newGetWorkingDirectoryTool,
		newListDirectoryTool,
		newReadFileTool,
		newWriteFileTool,
		newHTTPGetTool,
		newRunCommandTool,
	} {
		t, err := build()
		if err != nil {
			// Skip a broken tool definition rather than failing agent startup.
			continue
		}
		tools = append(tools, t)
	}
	return tools
}

type emptyInput struct{}

type timeOutput struct {
	RFC3339  string `json:"rfc3339"`
	Unix     int64  `json:"unix"`
	Timezone string `json:"timezone"`
	Local    string `json:"local"`
}

func newGetCurrentTimeTool() (tool.InvokableTool, error) {
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
	Path string `json:"path"`
}

func newGetWorkingDirectoryTool() (tool.InvokableTool, error) {
	return utils.InferTool("get_working_directory", "Get the process current working directory.",
		func(ctx context.Context, _ emptyInput) (cwdOutput, error) {
			wd, err := os.Getwd()
			if err != nil {
				return cwdOutput{}, err
			}
			return cwdOutput{Path: wd}, nil
		})
}

type listDirInput struct {
	Path string `json:"path" jsonschema:"description=Directory path to list. Empty means current working directory."`
}

type listDirOutput struct {
	Path    string   `json:"path"`
	Entries []string `json:"entries"`
	Truncated bool   `json:"truncated,omitempty"`
}

func newListDirectoryTool() (tool.InvokableTool, error) {
	return utils.InferTool("list_directory", "List files and directories at a path (non-recursive).",
		func(ctx context.Context, in listDirInput) (listDirOutput, error) {
			path := strings.TrimSpace(in.Path)
			if path == "" {
				path = "."
			}
			abs, err := filepath.Abs(path)
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
	Path string `json:"path" jsonschema:"required,description=File path to read as UTF-8 text."`
}

type readFileOutput struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Bytes     int    `json:"bytes"`
	Truncated bool   `json:"truncated,omitempty"`
}

func newReadFileTool() (tool.InvokableTool, error) {
	return utils.InferTool("read_file", "Read a text file (capped size). Prefer for source/config/logs.",
		func(ctx context.Context, in readFileInput) (readFileOutput, error) {
			path := strings.TrimSpace(in.Path)
			if path == "" {
				return readFileOutput{}, fmt.Errorf("path is required")
			}
			abs, err := filepath.Abs(path)
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
	Path    string `json:"path" jsonschema:"required,description=File path to write."`
	Content string `json:"content" jsonschema:"required,description=UTF-8 text content to write."`
	Append  bool   `json:"append,omitempty" jsonschema:"description=If true, append instead of overwrite."`
}

type writeFileOutput struct {
	Path  string `json:"path"`
	Bytes int    `json:"bytes"`
}

func newWriteFileTool() (tool.InvokableTool, error) {
	return utils.InferTool("write_file", "Write UTF-8 text to a file (create or overwrite; optional append).",
		func(ctx context.Context, in writeFileInput) (writeFileOutput, error) {
			path := strings.TrimSpace(in.Path)
			if path == "" {
				return writeFileOutput{}, fmt.Errorf("path is required")
			}
			if len(in.Content) > maxReadBytes {
				return writeFileOutput{}, fmt.Errorf("content exceeds %d bytes", maxReadBytes)
			}
			abs, err := filepath.Abs(path)
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

func newHTTPGetTool() (tool.InvokableTool, error) {
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
			req.Header.Set("User-Agent", "chat-tui-eino-agent/0.8")
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
	Command string `json:"command" jsonschema:"required,description=Shell command to run via /bin/sh -c (or cmd on Windows). Keep it short and non-interactive."`
}

type runCommandOutput struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	TimedOut bool   `json:"timed_out,omitempty"`
}

func newRunCommandTool() (tool.InvokableTool, error) {
	return utils.InferTool("run_command", "Run a short non-interactive shell command with timeout; returns stdout/stderr (capped).",
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
			var stdout, stderr strings.Builder
			cmd.Stdout = &capWriter{w: &stdout, n: maxCmdOutput}
			cmd.Stderr = &capWriter{w: &stderr, n: maxCmdOutput}
			err := cmd.Run()
			out := runCommandOutput{
				Stdout: stdout.String(),
				Stderr: stderr.String(),
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
