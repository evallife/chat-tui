package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/evallife/chat-tui/internal/api"
	"github.com/evallife/chat-tui/internal/storage"
	"github.com/evallife/chat-tui/internal/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sashabaranov/go-openai"
)

type TViewUI struct {
	App            *tview.Application
	Pages          *tview.Pages
	ChatView       *tview.TextView
	InputField     *tview.TextArea
	HistoryList    *tview.List
	HistoryPreview *tview.TextView
	SettingsForm   *tview.Form
	CopyView       *tview.TextArea

	Sidebar        *tview.List
	MainFlex       *tview.Flex
	chatColumnFlex *tview.Flex
	StatusBar      *tview.TextView
	sidebarVisible bool

	config       types.Config
	storage      *storage.Manager
	apiClient    *api.Client
	messages     []openai.ChatCompletionMessage
	convID       string
	systemPrompt string
	renderer     *glamour.TermRenderer

	lastClickedIdx  int
	lastClickedTime time.Time

	isStreaming  bool
	streamCancel context.CancelFunc

	isProcessingInput bool

	inputHistory []string
	historyIndex int
	draftInput   string

	searchInput  *tview.InputField
	searchActive bool
	searchQuery  string
	searchHits   []string
	searchIdx    int

	confirmMu  sync.Mutex
	confirmGen int
}

func NewTViewUI(cfg types.Config, store *storage.Manager) *TViewUI {
	cfg.Normalize()
	if cfg.Theme == "" {
		cfg.Theme = "night"
	}
	ui := &TViewUI{
		App:            tview.NewApplication(),
		Pages:          tview.NewPages(),
		config:         cfg,
		storage:        store,
		apiClient:      api.NewClient(cfg),
		lastClickedIdx: -1,
		historyIndex:   -1,
	}
	ui.apiClient.SetConfirmer(ui)

	ui.applyTheme(cfg.Theme)

	if r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	); err == nil {
		ui.renderer = r
	}

	ui.setupSidebar()
	ui.setupChatView()
	ui.setupHistoryView()
	ui.setupSettingsView()
	ui.setupSearch()
	ui.setupStatusBar()

	ui.mountChat(false)
	ui.App.SetRoot(ui.Pages, true).EnableMouse(true).EnablePaste(true)
	ui.App.SetInputCapture(ui.globalKeys)

	ui.startupNotices()
	return ui
}

func (ui *TViewUI) Run() error {
	return ui.App.Run()
}

func (ui *TViewUI) globalKeys(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyCtrlC {
		if page, _ := ui.Pages.GetFrontPage(); page == "copy" {
			ui.copySelectionToClipboard()
			return nil
		}
		ui.confirmQuit()
		return nil
	}
	switch event.Key() {
	case tcell.KeyCtrlN:
		ui.newConversation()
		return nil
	case tcell.KeyCtrlH:
		ui.showHistory()
		return nil
	case tcell.KeyCtrlS:
		ui.showSettings()
		return nil
	case tcell.KeyCtrlE:
		if event.Modifiers()&tcell.ModShift != 0 {
			ui.showExportDialog()
		} else {
			ui.exportHistory()
		}
		return nil
	case tcell.KeyCtrlB:
		ui.toggleSidebar()
		return nil
	case tcell.KeyCtrlY:
		ui.showCopyMode()
		return nil
	case tcell.KeyCtrlF:
		ui.toggleSearch()
		return nil
	case tcell.KeyEsc:
		return ui.handleEsc()
	}
	return event
}

func (ui *TViewUI) handleEsc() *tcell.EventKey {
	page, _ := ui.Pages.GetFrontPage()
	switch escActionFor(page, ui.searchActive, ui.isStreaming) {
	case escCancelStream:
		ui.stopStreaming()
	case escCloseSearch:
		ui.clearSearch()
	case escBackChat:
		ui.dismissToChat(page)
	case escBackPrompts:
		ui.dismissPromptOverlay(page)
	case escCloseOverlay:
		if ui.Pages.HasPage(page) {
			ui.Pages.RemovePage(page)
		}
		// confirm-quit / export-dialog return to chat input when chat is front.
		// confirm-delete stays on history (focusChatIfFront is a no-op).
		ui.focusChatIfFront()
	case escNone:
		// Idle chat: consume Esc so it cannot quit.
		// HITL confirm-tool: consume Esc so the modal cannot auto-deny.
	}
	return nil
}

func (ui *TViewUI) dismissToChat(page string) {
	switch page {
	case "copy":
		ui.hideCopyMode()
		return
	case "system_prompts_mgr":
		if ui.Pages.HasPage("system_prompts_mgr") {
			ui.Pages.RemovePage("system_prompts_mgr")
		}
	}
	ui.focusChatInput()
}

func (ui *TViewUI) dismissPromptOverlay(page string) {
	switch page {
	case "prompt_editor":
		if ui.Pages.HasPage("prompt_editor") {
			ui.Pages.RemovePage("prompt_editor")
		}
		ui.showSystemPrompts()
	case "confirm_delete_prompt":
		if ui.Pages.HasPage("confirm_delete_prompt") {
			ui.Pages.RemovePage("confirm_delete_prompt")
		}
		if ui.Pages.HasPage("system_prompts_mgr") {
			ui.Pages.SwitchToPage("system_prompts_mgr")
		}
	}
}

func (ui *TViewUI) focusChatInput() {
	ui.Pages.SwitchToPage("chat")
	ui.App.SetFocus(ui.InputField)
}

func (ui *TViewUI) focusChatIfFront() {
	front, _ := ui.Pages.GetFrontPage()
	if front == "chat" || front == "" {
		ui.focusChatInput()
	}
}

func (ui *TViewUI) startupNotices() {
	ws := strings.TrimSpace(ui.config.WorkspaceRoot)
	if ws == "" {
		ui.appendSystemMsg("Workspace is unset; file/shell tools use the process current directory. Set Workspace Root in Settings to sandbox them.")
	} else {
		ui.appendSystemMsg(fmt.Sprintf("Workspace sandbox: %s\nFile and shell tools stay inside this folder. write_file / run_command still require Allow/Deny.", ws))
	}
	ui.warnIfUnconfigured()
}

func (ui *TViewUI) warnIfUnconfigured() {
	p := ui.config.CanonicalProvider()
	if p == types.ProviderOllama {
		return
	}
	if strings.TrimSpace(ui.config.APIKey) != "" {
		return
	}
	if p == types.ProviderArk && strings.TrimSpace(ui.config.AccessKey) != "" {
		return
	}
	ui.appendSystemMsg("No API credentials set. Press Ctrl+S to open Settings, then Save.")
}

func (ui *TViewUI) reportError(what string, err error) {
	if err == nil {
		return
	}
	ui.appendSystemMsg(fmt.Sprintf("%s: %v", what, err))
}
