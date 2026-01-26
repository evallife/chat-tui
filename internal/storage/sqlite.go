package storage

import (
	"database/sql"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
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
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		conversation_id TEXT,
		role TEXT,
		content TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(conversation_id) REFERENCES conversations(id)
	);`
	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	return &Manager{db: db}, nil
}

func (m *Manager) CreateConversation(title, modelName string) (string, error) {
	id := uuid.New().String()
	_, err := m.db.Exec("INSERT INTO conversations (id, title, model) VALUES (?, ?, ?)", id, title, modelName)
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
