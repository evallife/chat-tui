package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/lrstanley/bubblezone"
	"github.com/sashabaranov/go-openai"
	"github.com/evallife/chat-tui/internal/api"
	"github.com/evallife/chat-tui/internal/config"
	"github.com/evallife/chat-tui/internal/storage"
	"github.com/evallife/chat-tui/internal/types"
)

type sessionState uint

const (
	chatView sessionState = iota
	historyView
	settingsView
)

type streamResult struct {
	content string
	done    bool
	err     error
}

type item struct {
	id, title string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.id }
func (i item) FilterValue() string { return i.title }

type Model struct {
	config        types.Config
	state         sessionState
	viewport      viewport.Model
	textarea      textarea.Model
	list          list.Model
	inputs        []textinput.Model // For settings: 0: APIKey, 1: BaseURL, 2: Model
	focusIndex    int               // Which input is focused in settings
	messages      []openai.ChatCompletionMessage
	apiClient     *api.Client
	storage       *storage.Manager
	convID        string
	renderer      *glamour.TermRenderer
	err           error
	width         int
	height        int
	isThinking    bool
	currResponse  string
	currentStream *openai.ChatCompletionStream
	zoneManager   *zone.Manager
}

func NewModel(cfg types.Config, store *storage.Manager) Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message, /read <file>, or use Ctrl+N for new..."
	ta.Focus()
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.KeyMap.InsertNewline.SetEnabled(false) // Enter to send

	vp := viewport.New(80, 20)

	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "History Conversations"

	// Settings inputs
	inputs := make([]textinput.Model, 3)
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "API Key"
	inputs[0].SetValue(cfg.APIKey)

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Base URL"
	inputs[1].SetValue(cfg.BaseURL)

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "Model (e.g. gpt-4o)"
	inputs[2].SetValue(cfg.Model)

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	return Model{
		config:      cfg,
		state:       chatView,
		textarea:    ta,
		viewport:    vp,
		list:        l,
		inputs:      inputs,
		renderer:    renderer,
		apiClient:   api.NewClient(cfg),
		storage:     store,
		messages:    []openai.ChatCompletionMessage{},
		zoneManager: zone.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	// Settings view update logic
	if m.state == settingsView {
		switch msg := msg.(type) {
		case tea.MouseMsg:
			if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
				for i := range m.inputs {
					if m.zoneManager.Get(fmt.Sprintf("input-%d", i)).InBounds(msg) {
						m.focusIndex = i
						for j := range m.inputs {
							if i == j {
								m.inputs[j].Focus()
							} else {
								m.inputs[j].Blur()
							}
						}
						return m, nil
					}
				}
				if m.zoneManager.Get("save-btn").InBounds(msg) {
					m.focusIndex = len(m.inputs)
					// Trigger save
					m.config.APIKey = m.inputs[0].Value()
					m.config.BaseURL = m.inputs[1].Value()
					m.config.Model = m.inputs[2].Value()
					config.SaveConfig(m.config)
					m.apiClient = api.NewClient(m.config)
					m.state = chatView
					return m, nil
				}
			}
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.state = chatView
				return m, nil
			case "tab", "shift+tab", "up", "down":
				s := msg.String()
				if s == "up" || s == "shift+tab" {
					m.focusIndex--
				} else {
					m.focusIndex++
				}

				if m.focusIndex > len(m.inputs) {
					m.focusIndex = 0
				} else if m.focusIndex < 0 {
					m.focusIndex = len(m.inputs)
				}

				cmds := make([]tea.Cmd, len(m.inputs))
				for i := 0; i < len(m.inputs); i++ {
					if i == m.focusIndex {
						cmds[i] = m.inputs[i].Focus()
						continue
					}
					m.inputs[i].Blur()
				}
				return m, tea.Batch(cmds...)

			case "enter":
				if m.focusIndex == len(m.inputs) { // Save button logic
					m.config.APIKey = m.inputs[0].Value()
					m.config.BaseURL = m.inputs[1].Value()
					m.config.Model = m.inputs[2].Value()
					config.SaveConfig(m.config)
					m.apiClient = api.NewClient(m.config) // Re-init client
					m.state = chatView
					return m, nil
				}
			}
		}

		// Update all inputs
		cmds := make([]tea.Cmd, len(m.inputs))
		for i := range m.inputs {
			m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
		}
		return m, tea.Batch(cmds...)
	}

	if m.state == historyView {
		m.list, tiCmd = m.list.Update(msg) // Overuse tiCmd but it's fine for now
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "enter" {
				selected := m.list.SelectedItem().(item)
				m.convID = selected.id
				m.messages, _ = m.storage.GetMessages(selected.id)
				m.state = chatView
				m.renderMessages()
				m.viewport.GotoBottom()
			} else if msg.String() == "esc" || msg.String() == "ctrl+h" {
				m.state = chatView
			}
		}
		return m, tiCmd
	}

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if m.zoneManager.Get("textarea").InBounds(msg) {
				m.textarea.Focus()
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 6
		m.textarea.SetWidth(msg.Width)
		m.list.SetSize(msg.Width, msg.Height)
		m.renderer, _ = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(msg.Width-4),
		)
		m.renderMessages()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if m.isThinking {
				return m, nil
			}
			input := m.textarea.Value()
			if input == "" {
				return m, nil
			}
			m.textarea.Reset()
			return m.handleInput(input)
		case "ctrl+n":
			m.messages = []openai.ChatCompletionMessage{}
			m.convID = ""
			m.renderMessages()
			m.viewport.GotoTop()
			return m, nil
		case "ctrl+h":
			convs, _ := m.storage.ListConversations()
			items := make([]list.Item, len(convs))
			for i, c := range convs {
				items[i] = item{id: c.ID, title: c.Title}
			}
			m.list.SetItems(items)
			m.state = historyView
			return m, nil
		case "ctrl+s":
			m.state = settingsView
			m.focusIndex = 0
			m.inputs[0].Focus()
			return m, nil
		case "ctrl+e":
			return m, m.exportHistory()
		}

	case *openai.ChatCompletionStream:
		m.currentStream = msg
		return m, m.listenToStream()

	case streamResult:
		if msg.err != nil {
			m.err = msg.err
			m.isThinking = false
			return m, nil
		}
		if msg.done {
			m.isThinking = false
			m.messages = append(m.messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: m.currResponse,
			})
			// Save to DB
			if m.convID != "" {
				m.storage.SaveMessage(m.convID, openai.ChatMessageRoleAssistant, m.currResponse)
			}
			m.currResponse = ""
			m.renderMessages()
			m.viewport.GotoBottom()
			if m.currentStream != nil {
				m.currentStream.Close()
			}
			return m, nil
		}
		m.currResponse += msg.content
		m.renderMessages()
		m.viewport.GotoBottom()
		return m, m.listenToStream()

	case error:
		m.err = msg
		m.isThinking = false
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m *Model) renderMessages() {
	var sb strings.Builder
	for _, msg := range m.messages {
		roleColor := "5" // Purple for User
		if msg.Role == openai.ChatMessageRoleAssistant {
			roleColor = "2" // Green for AI
		}
		role := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(roleColor)).Render(strings.ToUpper(msg.Role))
		content, _ := m.renderer.Render(msg.Content)
		sb.WriteString(fmt.Sprintf("%s\n%s\n", role, content))
	}
	if m.currResponse != "" {
		role := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")).Render("ASSISTANT")
		content, _ := m.renderer.Render(m.currResponse)
		sb.WriteString(fmt.Sprintf("%s\n%s\n", role, content))
	}
	m.viewport.SetContent(sb.String())
}

func (m Model) View() string {
	if m.state == settingsView {
		var b strings.Builder
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5")).Render("SETTINGS\n\n"))

		for i := range m.inputs {
			b.WriteString(fmt.Sprintf("%s\n%s\n\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(m.inputs[i].Placeholder),
				m.zoneManager.Mark(fmt.Sprintf("input-%d", i), m.inputs[i].View())))
		}

		buttonStyle := lipgloss.NewStyle().Padding(0, 3).MarginTop(1)
		if m.focusIndex == len(m.inputs) {
			buttonStyle = buttonStyle.Background(lipgloss.Color("5")).Foreground(lipgloss.Color("15"))
		} else {
			buttonStyle = buttonStyle.Background(lipgloss.Color("240")).Foreground(lipgloss.Color("250"))
		}
		b.WriteString(m.zoneManager.Mark("save-btn", buttonStyle.Render(" SAVE ")))
		b.WriteString("\n\n[Tab: Switch | Enter: Save/Action | Esc: Back]")

		return m.zoneManager.Scan(lipgloss.NewStyle().Padding(1, 4).Render(b.String()))
	}

	if m.state == historyView {
		return m.list.View()
	}

	var body string
	if m.err != nil {
		body = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Error: %v", m.err))
	} else {
		body = m.viewport.View()
	}

	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("\n[Enter: Send | Ctrl+N: New | Ctrl+H: History | Ctrl+S: Settings | Ctrl+E: Export | Esc: Quit]")
	return m.zoneManager.Scan(lipgloss.JoinVertical(
		lipgloss.Left,
		body,
		"\n",
		m.zoneManager.Mark("textarea", m.textarea.View()),
		footer,
	))
}

func (m Model) handleInput(input string) (Model, tea.Cmd) {
	if strings.HasPrefix(input, "/read ") {
		filePath := strings.TrimSpace(input[6:])
		content, err := os.ReadFile(filePath)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.messages = append(m.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("Content of file %s:\n\n%s", filePath, string(content)),
		})
		m.renderMessages()
		return m, nil
	}

	m.messages = append(m.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: input,
	})

	// Persistence logic
	if m.convID == "" {
		title := input
		if len(title) > 30 {
			title = title[:27] + "..."
		}
		id, _ := m.storage.CreateConversation(title, m.config.Model)
		m.convID = id
	}
	m.storage.SaveMessage(m.convID, openai.ChatMessageRoleUser, input)

	m.renderMessages()
	m.isThinking = true
	m.currResponse = ""

	return m, m.sendToOpenAI()
}

func (m Model) sendToOpenAI() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		stream, err := m.apiClient.StreamChat(ctx, m.messages)
		if err != nil {
			return err
		}
		return stream
	}
}

func (m Model) listenToStream() tea.Cmd {
	return func() tea.Msg {
		response, err := m.currentStream.Recv()
		if err != nil {
			if err.Error() == "EOF" || strings.Contains(err.Error(), "EOF") {
				return streamResult{done: true}
			}
			return streamResult{err: err}
		}
		return streamResult{content: response.Choices[0].Delta.Content}
	}
}

func (m Model) exportHistory() tea.Cmd {
	return func() tea.Msg {
		filename := fmt.Sprintf("chat_export_%d.md", time.Now().Unix())
		var sb strings.Builder
		for _, msg := range m.messages {
			sb.WriteString(fmt.Sprintf("## %s\n\n%s\n\n---\n\n", strings.ToUpper(msg.Role), msg.Content))
		}
		err := os.WriteFile(filename, []byte(sb.String()), 0644)
		if err != nil {
			return err
		}
		return nil
	}
}
