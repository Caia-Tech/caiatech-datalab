package models

import (
	"context"
	"database/sql"
)

func loadMessages(ctx context.Context, db *sql.DB, conversationID int64) ([]Message, error) {
	rows, err := db.QueryContext(ctx, `
SELECT role, name, content, meta
FROM conversation_messages
WHERE conversation_id = $1
ORDER BY idx ASC
`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var role string
		var name string
		var content string
		var meta []byte
		if err := rows.Scan(&role, &name, &content, &meta); err != nil {
			return nil, err
		}
		out = append(out, Message{Role: Role(role), Name: name, Content: content, Meta: meta})
	}
	return out, rows.Err()
}
