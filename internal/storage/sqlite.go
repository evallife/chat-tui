package storage

import (
	"database/sql"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"github.com/evallife/chat-tui/internal/types"
	_ "modernc.org/sqlite"
)

type Manager struct {
	db *sql.DB
}

func NewManager() (*Manager, error) {
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".xftui.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Create tables
	query := `
	CREATE TABLE IF NOT EXISTS conversations (
		id TEXT PRIMARY KEY,
		title TEXT,
		model TEXT,
		system_prompt TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		conversation_id TEXT,
		role TEXT,
		content TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(conversation_id) REFERENCES conversations(id)
	);
	CREATE TABLE IF NOT EXISTS system_prompts (
		id TEXT PRIMARY KEY,
		name TEXT,
		content TEXT
	);`
	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	// Migrate if needed
	_, _ = db.Exec("ALTER TABLE conversations ADD COLUMN system_prompt TEXT")

	return &Manager{db: db}, nil
}

func (m *Manager) CreateConversation(title, modelName, systemPrompt string) (string, error) {
	id := uuid.New().String()
	_, err := m.db.Exec("INSERT INTO conversations (id, title, model, system_prompt) VALUES (?, ?, ?, ?)", id, title, modelName, systemPrompt)
	return id, err
}

func (m *Manager) SaveMessage(convID, role, content string) error {
	_, err := m.db.Exec("INSERT INTO messages (conversation_id, role, content) VALUES (?, ?, ?)", convID, role, content)
	return err
}

func (m *Manager) GetMessages(convID string) ([]openai.ChatCompletionMessage, error) {
	rows, err := m.db.Query("SELECT role, content FROM messages WHERE conversation_id = ? ORDER BY id ASC", convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []openai.ChatCompletionMessage
	for rows.Next() {
		var msg openai.ChatCompletionMessage
		if err := rows.Scan(&msg.Role, &msg.Content); err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

type ConvSummary struct {
	ID    string
	Title string
}

func (m *Manager) ListConversations() ([]ConvSummary, error) {
	rows, err := m.db.Query("SELECT id, title FROM conversations ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []ConvSummary
	for rows.Next() {
		var c ConvSummary
		if err := rows.Scan(&c.ID, &c.Title); err != nil {
			return nil, err
		}
		convs = append(convs, c)
	}
	return convs, nil
}

func (m *Manager) GetConversation(id string) (types.Conversation, error) {
	var c types.Conversation
	err := m.db.QueryRow("SELECT id, title, model, system_prompt FROM conversations WHERE id = ?", id).
		Scan(&c.ID, &c.Title, &c.Model, &c.SystemPrompt)
	return c, err
}

func (m *Manager) ListSystemPrompts() ([]types.SystemPrompt, error) {
	rows, err := m.db.Query("SELECT id, name, content FROM system_prompts")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var prompts []types.SystemPrompt
	for rows.Next() {
		var p types.SystemPrompt
		if err := rows.Scan(&p.ID, &p.Name, &p.Content); err != nil {
			return nil, err
		}
		prompts = append(prompts, p)
	}
	// Add default ones if empty
	if len(prompts) == 0 {
		defaults := []types.SystemPrompt{
			{ID: "default", Name: "Default Chat", Content: ""},
			{ID: "translator", Name: "Translator (ZH-EN)", Content: "You are a professional translator. Translate between Chinese and English."},
			{ID: "coder", Name: "Code Expert", Content: "You are an expert software engineer. Provide concise and accurate code solutions."},
		}
		for _, p := range defaults {
			_, _ = m.db.Exec("INSERT INTO system_prompts (id, name, content) VALUES (?, ?, ?)", p.ID, p.Name, p.Content)
		}
		return defaults, nil
	}
	return prompts, nil
}


func (m *Manager) DeleteConversation(convID string) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM messages WHERE conversation_id = ?", convID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec("DELETE FROM conversations WHERE id = ?", convID); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
