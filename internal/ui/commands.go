package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/evallife/chat-tui/internal/api"
	"github.com/rivo/tview"
	"github.com/sashabaranov/go-openai"
)

func (ui *TViewUI) handleCommand(input string) {
	parts := strings.Fields(input)
	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/read":
		if len(args) == 0 {
			ui.appendSystemMsg("Usage: /read <path>")
			return
		}
		filePath := args[0]
		content, err := os.ReadFile(filePath)
		if err != nil {
			ui.appendSystemMsg(fmt.Sprintf("Error reading file: %v", err))
			return
		}
		ui.messages = append(ui.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("Content of file %s:\n\n%s", filePath, string(content)),
		})
		ui.refreshChat()

	case "/clear":
		ui.messages = []openai.ChatCompletionMessage{}
		ui.ChatView.Clear()
		ui.refreshChat()
		ui.appendSystemMsg("Chat display cleared.")

	case "/config":
		ui.config.Normalize()
		ui.appendSystemMsg(fmt.Sprintf("Current Config:\n- Provider: %s\n- BaseURL: %s\n- Model: %s\n- Region: %s\n- Max Iterations: %d\n- Workspace Root: %s\n- Write File: %v\n- Run Command: %v\n- Agent Name: %s\n- Default Instruction: %v\n- Tool confirm: write_file / run_command\n- System Prompt: %s",
			ui.config.CanonicalProvider(), ui.config.BaseURL, ui.config.Model, ui.config.Region,
			ui.config.MaxIterations, ui.config.WorkspaceRoot,
			ui.config.WriteFileEnabled(), ui.config.RunCommandEnabled(),
			ui.config.AgentName, ui.config.DefaultInstructionEnabled(), ui.systemPrompt))

	case "/save":
		filename := "chat_save.md"
		if len(args) > 0 {
			filename = args[0]
		}
		ui.exportToFile(filename)

	case "/export":
		filename := fmt.Sprintf("qa_export_%d.md", time.Now().Unix())
		if len(args) > 0 {
			filename = args[0]
		}
		ui.exportToFile(filename)

	case "/tools":
		ui.config.Normalize()
		names := api.ToolNames(ui.config)
		var b strings.Builder
		b.WriteString("Enabled Eino agent tools:\n")
		for _, n := range names {
			fmt.Fprintf(&b, "  %s\n", n)
		}
		b.WriteString("  exit                   ADK exit (end agent)\n")
		b.WriteString("\nThe model may call these during a reply; results show as ⚙ tool:… in the chat.\n")
		b.WriteString("write_file and run_command pause for Allow/Deny before executing.\n")
		ui.appendSystemMsg(b.String())

	case "/agent":
		ui.config.Normalize()
		eff := api.EffectiveInstruction(ui.systemPrompt, ui.config)
		summary := eff
		if len(summary) > 160 {
			summary = summary[:160] + "…"
		}
		if summary == "" {
			summary = "(none)"
		}
		ws := ui.config.WorkspaceRoot
		if strings.TrimSpace(ws) == "" {
			ws = "(cwd)"
		}
		names := api.ToolNames(ui.config)
		ui.appendSystemMsg(fmt.Sprintf(`Agent status:
- Name: %s
- Max iterations: %d
- Workspace root: %s
- Write file: %v
- Run command: %v
- Default instruction: %v
- HITL confirm: write_file / run_command
- Instruction summary: %s
- Tools (%d): %s
- Exit tool: enabled
`,
			ui.config.AgentName, ui.config.MaxIterations, ws,
			ui.config.WriteFileEnabled(), ui.config.RunCommandEnabled(),
			ui.config.DefaultInstructionEnabled(), summary,
			len(names), strings.Join(names, ", ")))

	case "/help":
		ui.appendSystemMsg(`Commands:
  /read <path>   Inject file content into chat
  /tools         List enabled Eino agent tools
  /agent         Show agent status (instruction, tools, limits)
  /clear         Clear chat display (history kept)
  /config        Show current settings
  /help          Show this help

Keys:
  Esc            Dismiss overlay / stop stream (idle chat: no-op)
  Ctrl+C         Quit confirm (Copy Mode: copy)
  Ctrl+N/H/S/B   New / History / Settings / Menu
  Enter          Send    Shift+Enter  Newline

While streaming: Esc cancels the agent.
Tool calls appear inline as ⚙ tool:<name>.
write_file and run_command require Allow/Deny before they run.
`)

	default:
		ui.appendSystemMsg(fmt.Sprintf("Unknown command: %s. Type /help for list.", cmd))
	}
}

func (ui *TViewUI) exportToFile(filename string) {
	var sb strings.Builder
	if ui.systemPrompt != "" {
		sb.WriteString(fmt.Sprintf("> System Prompt: %s\n\n", ui.systemPrompt))
	}
	for _, msg := range ui.messages {
		roleLabel := strings.ToUpper(msg.Role)
		if msg.Role == "user" {
			roleLabel = "Q"
		} else if msg.Role == "assistant" {
			roleLabel = "A"
		}
		sb.WriteString(fmt.Sprintf("## %s: %s\n\n---\n\n", roleLabel, msg.Content))
	}
	err := os.WriteFile(filename, []byte(sb.String()), 0644)
	if err != nil {
		ui.appendSystemMsg(fmt.Sprintf("Save failed: %v", err))
	} else {
		ui.appendSystemMsg("History saved to " + filename)
	}
}

func (ui *TViewUI) exportHistory() {
	filename := fmt.Sprintf("chat_export_%d.md", time.Now().Unix())
	ui.exportToFile(filename)
}

func (ui *TViewUI) showExportDialog() {
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(" Export Options ").SetTitleAlign(tview.AlignLeft)

	var filename string
	defaultFilename := fmt.Sprintf("qa_export_%d.md", time.Now().Unix())

	form.AddInputField("Filename:", defaultFilename, 50, nil, func(text string) {
		filename = text
	})

	closeExport := func() {
		ui.Pages.RemovePage("export-dialog")
		ui.focusChatIfFront()
	}

	form.AddButton("Export", func() {
		if filename == "" {
			filename = defaultFilename
		}
		ui.exportToFile(filename)
		closeExport()
	})

	form.AddButton("Cancel", closeExport)
	form.SetCancelFunc(closeExport)

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 60, 1, true).
			AddItem(nil, 0, 1, false), 8, 1, true).
		AddItem(nil, 0, 1, false)

	ui.Pages.AddPage("export-dialog", modal, true, true)
}
