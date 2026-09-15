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

	Sidebar  *tview.List
	MainFlex *tview.Flex

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

	ui.mountChat(false)
	ui.App.SetRoot(ui.Pages, true).EnableMouse(true).EnablePaste(true)
	ui.App.SetInputCapture(ui.globalKeys)

	ui.warnIfUnconfigured()
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
		if page, _ := ui.Pages.GetFrontPage(); page == "copy" {
			ui.hideCopyMode()
			return nil
		}
		if ui.isStreaming && ui.streamCancel != nil {
			ui.streamCancel()
			return nil
		}
		ui.confirmQuit()
		return nil
	}
	return event
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
	ui.appendSystemMsg("No API credentials set. Press Ctrl+S to open Settings, then Save. write_file / run_command require confirmation before they run.")
}

func (ui *TViewUI) reportError(what string, err error) {
	if err == nil {
		return
	}
	ui.appendSystemMsg(fmt.Sprintf("%s: %v", what, err))
}
