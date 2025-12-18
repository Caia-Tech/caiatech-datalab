package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

type ListConversationsParams struct {
	DatasetID int64
	Split     Split
	Status    ConversationStatus
	Query     string
	Limit     int
	Offset    int
}

func ListConversations(ctx context.Context, db *sql.DB, p ListConversationsParams) ([]Conversation, error) {
	q := strings.TrimSpace(p.Query)
	if q == "" {
		rows, err := db.QueryContext(ctx, `
SELECT
  c.id, c.dataset_id, c.split, c.status, c.tags, c.source, c.notes, c.created_at, c.updated_at,
  (SELECT COUNT(*) FROM conversation_messages m WHERE m.conversation_id = c.id) AS message_count,
  COALESCE((SELECT LEFT(m.content, 160) FROM conversation_messages m WHERE m.conversation_id = c.id AND m.role = 'user' ORDER BY m.idx ASC LIMIT 1), '') AS preview_user,
  COALESCE((SELECT LEFT(m.content, 160) FROM conversation_messages m WHERE m.conversation_id = c.id AND m.role = 'assistant' ORDER BY m.idx ASC LIMIT 1), '') AS preview_assistant
FROM conversations c
WHERE c.dataset_id = $1 AND c.split = $2 AND c.status = $3
ORDER BY c.id DESC
LIMIT $4 OFFSET $5
`, p.DatasetID, p.Split, p.Status, p.Limit, p.Offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanConversations(rows)
	}

	pattern := "%" + q + "%"
	rows, err := db.QueryContext(ctx, `
SELECT DISTINCT
  c.id, c.dataset_id, c.split, c.status, c.tags, c.source, c.notes, c.created_at, c.updated_at,
  (SELECT COUNT(*) FROM conversation_messages m WHERE m.conversation_id = c.id) AS message_count,
  COALESCE((SELECT LEFT(m.content, 160) FROM conversation_messages m WHERE m.conversation_id = c.id AND m.role = 'user' ORDER BY m.idx ASC LIMIT 1), '') AS preview_user,
  COALESCE((SELECT LEFT(m.content, 160) FROM conversation_messages m WHERE m.conversation_id = c.id AND m.role = 'assistant' ORDER BY m.idx ASC LIMIT 1), '') AS preview_assistant
FROM conversations c
JOIN conversation_messages mm ON mm.conversation_id = c.id
WHERE c.dataset_id = $1 AND c.split = $2 AND c.status = $3 AND mm.content ILIKE $4
ORDER BY c.id DESC
LIMIT $5 OFFSET $6
`, p.DatasetID, p.Split, p.Status, pattern, p.Limit, p.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConversations(rows)
}

func GetConversation(ctx context.Context, db *sql.DB, id int64) (Conversation, error) {
	var c Conversation
	var tagsRaw []byte
	err := db.QueryRowContext(ctx, `
SELECT id, dataset_id, split, status, tags, source, notes, created_at, updated_at
FROM conversations
WHERE id = $1
`, id).Scan(&c.ID, &c.DatasetID, &c.Split, &c.Status, &tagsRaw, &c.Source, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return Conversation{}, ErrNotFound
		}
		return Conversation{}, err
	}
	_ = json.Unmarshal(tagsRaw, &c.Tags)

	msgs, err := loadMessages(ctx, db, id)
	if err != nil {
		return Conversation{}, err
	}
	c.Messages = msgs
	c.MessageCount = len(msgs)
	for _, m := range msgs {
		if c.PreviewUser == "" && m.Role == RoleUser {
			c.PreviewUser = strings.TrimSpace(m.Content)
		}
		if c.PreviewAssistant == "" && m.Role == RoleAssistant {
			c.PreviewAssistant = strings.TrimSpace(m.Content)
		}
	}
	if len(c.PreviewUser) > 160 {
		c.PreviewUser = c.PreviewUser[:160]
	}
	if len(c.PreviewAssistant) > 160 {
		c.PreviewAssistant = c.PreviewAssistant[:160]
	}
	return c, nil
}

func InsertConversationWithMessages(ctx context.Context, tx *sql.Tx, c Conversation) (Conversation, error) {
	if c.Status == "" {
		c.Status = ConversationStatusApproved
	}
	if c.Split == "" {
		c.Split = SplitTrain
	}
	if c.DatasetID == 0 {
		return Conversation{}, ErrInvalidInput
	}

	tagsJSON, _ := json.Marshal(c.Tags)

	row := tx.QueryRowContext(ctx, `
INSERT INTO conversations (dataset_id, split, status, tags, source, notes)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, dataset_id, split, status, tags, source, notes, created_at, updated_at
`, c.DatasetID, c.Split, c.Status, tagsJSON, c.Source, c.Notes)

	var out Conversation
	var tagsRaw []byte
	if err := row.Scan(&out.ID, &out.DatasetID, &out.Split, &out.Status, &tagsRaw, &out.Source, &out.Notes, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return Conversation{}, err
	}
	_ = json.Unmarshal(tagsRaw, &out.Tags)

	for idx, m := range c.Messages {
		name := strings.TrimSpace(m.Name)
		meta := m.Meta
		if len(meta) == 0 {
			meta = json.RawMessage("{}")
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO conversation_messages (conversation_id, idx, role, name, content, meta)
VALUES ($1, $2, $3, $4, $5, $6)
`, out.ID, idx, m.Role, name, strings.TrimSpace(m.Content), meta); err != nil {
			return Conversation{}, err
		}
	}

	out.Messages = c.Messages
	out.MessageCount = len(c.Messages)
	return out, nil
}

func UpdateConversation(ctx context.Context, db *sql.DB, c Conversation) (Conversation, error) {
	if c.ID == 0 {
		return Conversation{}, ErrNotFound
	}
	if c.DatasetID == 0 {
		return Conversation{}, ErrInvalidInput
	}

	now := time.Now().UTC()
	tagsJSON, _ := json.Marshal(c.Tags)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Conversation{}, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
UPDATE conversations
SET dataset_id = $2,
    split = $3,
    status = $4,
    tags = $5,
    source = $6,
    notes = $7,
    updated_at = $8
WHERE id = $1
`, c.ID, c.DatasetID, c.Split, c.Status, tagsJSON, c.Source, c.Notes, now)
	if err != nil {
		return Conversation{}, err
	}
	a, err := res.RowsAffected()
	if err != nil {
		return Conversation{}, err
	}
	if a == 0 {
		return Conversation{}, ErrNotFound
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM conversation_messages WHERE conversation_id = $1`, c.ID); err != nil {
		return Conversation{}, err
	}
	for idx, m := range c.Messages {
		name := strings.TrimSpace(m.Name)
		meta := m.Meta
		if len(meta) == 0 {
			meta = json.RawMessage("{}")
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO conversation_messages (conversation_id, idx, role, name, content, meta)
VALUES ($1, $2, $3, $4, $5, $6)
`, c.ID, idx, m.Role, name, strings.TrimSpace(m.Content), meta); err != nil {
			return Conversation{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Conversation{}, err
	}

	return GetConversation(ctx, db, c.ID)
}

func DeleteConversation(ctx context.Context, db *sql.DB, id int64) error {
	res, err := db.ExecContext(ctx, `DELETE FROM conversations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	a, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if a == 0 {
		return ErrNotFound
	}
	return nil
}

func scanConversations(rows *sql.Rows) ([]Conversation, error) {
	var out []Conversation
	for rows.Next() {
		var c Conversation
		var tagsRaw []byte
		if err := rows.Scan(
			&c.ID,
			&c.DatasetID,
			&c.Split,
			&c.Status,
			&tagsRaw,
			&c.Source,
			&c.Notes,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.MessageCount,
			&c.PreviewUser,
			&c.PreviewAssistant,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(tagsRaw, &c.Tags)
		out = append(out, c)
	}
	return out, rows.Err()
}
