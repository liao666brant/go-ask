package data

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

type Conversation struct {
	ID    int64
	Title string
}

type Message struct {
	ID      int64
	Role    string
	Content string
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`
PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS conversations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    updated INTEGER NOT NULL DEFAULT (unixepoch('subsec') * 1000)
);
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK(role IN ('user','assistant')),
    content TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS messages_by_conversation ON messages(conversation,id);
CREATE TRIGGER IF NOT EXISTS touch_conversation AFTER INSERT ON messages
BEGIN
    UPDATE conversations SET updated = unixepoch('subsec')*1000 WHERE id=NEW.conversation;
END;`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) NewConversation(question string) (int64, error) {
	title := []rune(strings.Join(strings.Fields(question), " "))
	if len(title) > 60 {
		title = append(title[:60], '…')
	}
	res, err := s.db.Exec(`INSERT INTO conversations(title) VALUES (?)`, string(title))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) AddMessage(id int64, role, content string) (int64, error) {
	res, err := s.db.Exec(`INSERT INTO messages(conversation,role,content) VALUES (?,?,?)`, id, role, content)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) Conversations(query string) ([]Conversation, error) {
	query = strings.TrimSpace(query)
	rows, err := s.db.Query(`SELECT c.id,c.title FROM conversations c WHERE ? = ''
        OR instr(lower(c.title),lower(?))>0
        OR EXISTS (SELECT 1 FROM messages m WHERE m.conversation=c.id AND instr(lower(m.content),lower(?))>0)
        ORDER BY c.updated DESC, c.id DESC`, query, query, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.Title); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (s *Store) Messages(id int64) ([]Message, error) {
	rows, err := s.db.Query(`SELECT id,role,content FROM messages WHERE conversation=? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.Role, &m.Content); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (s *Store) DeleteConversation(id int64) error {
	_, err := s.db.Exec(`DELETE FROM conversations WHERE id=?`, id)
	return err
}
