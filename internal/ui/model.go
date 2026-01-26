package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/sashabaranov/go-openai"
	"github.com/user/xftui/internal/api"
	"github.com/user/xftui/internal/storage"
	"github.com/user/xftui/internal/types"
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

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	return Model{
		config:    cfg,
		state:     chatView,
		textarea:  ta,
		viewport:  vp,
		list:      l,
		renderer:  renderer,
		apiClient: api.NewClient(cfg),
		storage:   store,
		messages:  []openai.ChatCompletionMessage{},
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
	if m.state == historyView {
		return m.list.View()
	}

	var body string
	if m.err != nil {
		body = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Error: %v", m.err))
	} else {
		body = m.viewport.View()
	}

		footer := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("\n[Enter: Send | Ctrl+N: New | Ctrl+H: History | Ctrl+E: Export | Esc: Quit]")
	return lipgloss.JoinVertical(
		lipgloss.Left,
		body,
		"\n",
		m.textarea.View(),
		footer,
	)
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
