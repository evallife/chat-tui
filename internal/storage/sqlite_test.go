package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/evallife/chat-tui/internal/types"
	"github.com/sashabaranov/go-openai"
	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *Manager {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	m, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })
	return m
}

func TestOpenMigratesToCurrentVersion(t *testing.T) {
	m := openTestDB(t)
	v, err := m.SchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema version=%d, want %d", v, currentSchemaVersion)
	}
}

func TestConversationMessageCRUD(t *testing.T) {
	m := openTestDB(t)
	id, err := m.CreateConversation("hello", "gpt-4o-mini", "be brief")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("empty conversation id")
	}
	if err := m.SaveMessage(id, openai.ChatMessageRoleUser, "hi"); err != nil {
		t.Fatal(err)
	}
	if err := m.SaveMessage(id, openai.ChatMessageRoleAssistant, "hello"); err != nil {
		t.Fatal(err)
	}
	msgs, err := m.GetMessages(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("messages=%d, want 2", len(msgs))
	}
	if msgs[0].Role != openai.ChatMessageRoleUser || msgs[0].Content != "hi" {
		t.Fatalf("first message: %+v", msgs[0])
	}
	conv, err := m.GetConversation(id)
	if err != nil {
		t.Fatal(err)
	}
	if conv.Title != "hello" || conv.SystemPrompt != "be brief" || conv.Model != "gpt-4o-mini" {
		t.Fatalf("conversation: %+v", conv)
	}
	list, err := m.ListConversations()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != id {
		t.Fatalf("list: %+v", list)
	}
	if err := m.DeleteConversation(id); err != nil {
		t.Fatal(err)
	}
	msgs, err = m.GetMessages(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected no messages after delete, got %d", len(msgs))
	}
	list, err = m.ListConversations()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty conversation list, got %+v", list)
	}
}

func TestSystemPromptCRUDAndSeed(t *testing.T) {
	m := openTestDB(t)
	prompts, err := m.ListSystemPrompts()
	if err != nil {
		t.Fatal(err)
	}
	if len(prompts) != 3 {
		t.Fatalf("seeded prompts=%d, want 3", len(prompts))
	}
	again, err := m.ListSystemPrompts()
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 3 {
		t.Fatalf("re-list seeded twice: %d", len(again))
	}
	p := types.SystemPrompt{Name: "Reviewer", Content: "Review this diff."}
	if err := m.SaveSystemPrompt(p); err != nil {
		t.Fatal(err)
	}
	all, err := m.ListSystemPrompts()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 4 {
		t.Fatalf("after save: %d", len(all))
	}
	var saved types.SystemPrompt
	for _, item := range all {
		if item.Name == "Reviewer" {
			saved = item
		}
	}
	if saved.ID == "" {
		t.Fatal("saved prompt missing id")
	}
	if err := m.DeleteSystemPrompt(saved.ID); err != nil {
		t.Fatal(err)
	}
	all, err = m.ListSystemPrompts()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("after delete: %d", len(all))
	}
}

func TestLegacyDBMigratesAndKeepsData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE conversations (
			id TEXT PRIMARY KEY,
			title TEXT,
			model TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id TEXT,
			role TEXT,
			content TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO conversations (id, title, model) VALUES ('c1', 'old', 'gpt-3.5-turbo');
		INSERT INTO messages (conversation_id, role, content) VALUES ('c1', 'user', 'ping');
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	m, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	v, err := m.SchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("migrated version=%d, want %d", v, currentSchemaVersion)
	}
	conv, err := m.GetConversation("c1")
	if err != nil {
		t.Fatal(err)
	}
	if conv.Title != "old" {
		t.Fatalf("title=%q", conv.Title)
	}
	if conv.SystemPrompt != "" {
		t.Fatalf("system_prompt should default empty, got %q", conv.SystemPrompt)
	}
	msgs, err := m.GetMessages("c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Content != "ping" {
		t.Fatalf("messages: %+v", msgs)
	}
}

func TestNewManagerUsesHomeDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	m, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	want := filepath.Join(home, ".chat-tui.db")
	if m.Path() != want {
		t.Fatalf("path=%q want %q", m.Path(), want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatal(err)
	}
}

func TestMigratesLegacyDatabase(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	oldPath := filepath.Join(home, ".xftui.db")
	old, err := Open(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	id, err := old.CreateConversation("legacy chat", "gpt-4o-mini", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := old.SaveMessage(id, openai.ChatMessageRoleUser, "hello from old db"); err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	m, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if m.Path() != DefaultDBPath() {
		t.Fatalf("path=%q", m.Path())
	}
	conv, err := m.GetConversation(id)
	if err != nil {
		t.Fatal(err)
	}
	if conv.Title != "legacy chat" {
		t.Fatalf("title=%q", conv.Title)
	}
	msgs, err := m.GetMessages(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Content != "hello from old db" {
		t.Fatalf("messages: %+v", msgs)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("legacy db should be removed, err=%v", err)
	}
}
