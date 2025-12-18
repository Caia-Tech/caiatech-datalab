package models

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type ExportOptions struct {
	Type          string // pairs|conversations
	DatasetID     int64  // 0 = any
	Split         string // train|valid|test|all
	Status        string // approved|...
	IncludeSystem bool

	// pairs only
	Context      string // none|window|full
	ContextTurns int
	RoleStyle    string // labels|plain

	MaxExamples int
}

type ExportPair struct {
	User      string `json:"user"`
	Assistant string `json:"assistant"`
}

func StreamExport(ctx context.Context, db *sql.DB, w io.Writer, opts ExportOptions) error {
	if opts.Type == "" {
		opts.Type = "pairs"
	}
	if opts.Split == "" {
		opts.Split = string(SplitTrain)
	}
	if opts.Status == "" {
		opts.Status = string(ConversationStatusApproved)
	}

	if opts.DatasetID > 0 {
		ds, err := GetDataset(ctx, db, opts.DatasetID)
		if err != nil {
			return err
		}
		if strings.EqualFold(ds.Kind, "items") {
			return streamDatasetItems(ctx, db, w, opts)
		}
	}

	switch opts.Type {
	case "pairs":
		return streamPairs(ctx, db, w, opts)
	case "conversations":
		return streamConversations(ctx, db, w, opts)
	default:
		return fmt.Errorf("unknown export type: %s", opts.Type)
	}
}

func streamDatasetItems(ctx context.Context, db *sql.DB, w io.Writer, opts ExportOptions) error {
	switch opts.Type {
	case "pairs":
		return streamPairsFromDatasetItems(ctx, db, w, opts)
	case "items":
		return streamDatasetItemsRaw(ctx, db, w, opts)
	case "items_with_meta":
		return streamDatasetItemsWithMeta(ctx, db, w, opts)
	default:
		return fmt.Errorf("unknown export type for items dataset: %s", opts.Type)
	}
}

func streamConversations(ctx context.Context, db *sql.DB, w io.Writer, opts ExportOptions) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()
	enc := json.NewEncoder(bw)

	query, args := conversationsFilterQuery(opts)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int64
		var split string
		var status string
		var tagsRaw []byte
		var source string
		var notes string
		if err := rows.Scan(&id, &split, &status, &tagsRaw, &source, &notes); err != nil {
			return err
		}

		msgs, err := loadMessages(ctx, db, id)
		if err != nil {
			return err
		}

		var tags []string
		_ = json.Unmarshal(tagsRaw, &tags)

		obj := map[string]any{
			"id":       id,
			"split":    split,
			"status":   status,
			"tags":     tags,
			"source":   source,
			"notes":    notes,
			"messages": msgs,
		}

		if err := enc.Encode(obj); err != nil {
			return err
		}

		count++
		if opts.MaxExamples > 0 && count >= opts.MaxExamples {
			break
		}
	}
	return rows.Err()
}

func streamDatasetItemsRaw(ctx context.Context, db *sql.DB, w io.Writer, opts ExportOptions) error {
	if opts.DatasetID <= 0 {
		return fmt.Errorf("dataset_id is required for items export")
	}

	bw := bufio.NewWriter(w)
	defer bw.Flush()

	rows, err := db.QueryContext(ctx, `
SELECT data
FROM dataset_items
WHERE dataset_id = $1
ORDER BY id ASC
`, opts.DatasetID)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var data json.RawMessage
		if err := rows.Scan(&data); err != nil {
			return err
		}
		if _, err := bw.Write(data); err != nil {
			return err
		}
		if err := bw.WriteByte('\n'); err != nil {
			return err
		}
		count++
		if opts.MaxExamples > 0 && count >= opts.MaxExamples {
			break
		}
	}
	return rows.Err()
}

func streamDatasetItemsWithMeta(ctx context.Context, db *sql.DB, w io.Writer, opts ExportOptions) error {
	if opts.DatasetID <= 0 {
		return fmt.Errorf("dataset_id is required for items export")
	}

	bw := bufio.NewWriter(w)
	defer bw.Flush()
	enc := json.NewEncoder(bw)

	rows, err := db.QueryContext(ctx, `
SELECT id, dataset_id, source_ref, data
FROM dataset_items
WHERE dataset_id = $1
ORDER BY id ASC
`, opts.DatasetID)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int64
		var datasetID int64
		var sourceRef string
		var data json.RawMessage
		if err := rows.Scan(&id, &datasetID, &sourceRef, &data); err != nil {
			return err
		}
		obj := map[string]any{
			"id":         id,
			"dataset_id": datasetID,
			"source_ref": sourceRef,
			"data":       json.RawMessage(data),
		}
		if err := enc.Encode(obj); err != nil {
			return err
		}
		count++
		if opts.MaxExamples > 0 && count >= opts.MaxExamples {
			break
		}
	}
	return rows.Err()
}

func streamPairs(ctx context.Context, db *sql.DB, w io.Writer, opts ExportOptions) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()
	enc := json.NewEncoder(bw)

	query, args := conversationsFilterQuery(opts)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int64
		var split string
		var status string
		var tagsRaw []byte
		var source string
		var notes string
		if err := rows.Scan(&id, &split, &status, &tagsRaw, &source, &notes); err != nil {
			return err
		}

		msgs, err := loadMessages(ctx, db, id)
		if err != nil {
			return err
		}

		pairs := derivePairs(msgs, opts)
		for _, p := range pairs {
			if err := enc.Encode(p); err != nil {
				return err
			}
			count++
			if opts.MaxExamples > 0 && count >= opts.MaxExamples {
				return nil
			}
		}
	}
	return rows.Err()
}

func streamPairsFromDatasetItems(ctx context.Context, db *sql.DB, w io.Writer, opts ExportOptions) error {
	if opts.DatasetID <= 0 {
		return fmt.Errorf("dataset_id is required for items export")
	}

	bw := bufio.NewWriter(w)
	defer bw.Flush()
	enc := json.NewEncoder(bw)

	rows, err := db.QueryContext(ctx, `
SELECT data
FROM dataset_items
WHERE dataset_id = $1
ORDER BY id ASC
`, opts.DatasetID)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var data json.RawMessage
		if err := rows.Scan(&data); err != nil {
			return err
		}

		pairs := derivePairsFromItemData(data, opts)
		for _, p := range pairs {
			if err := enc.Encode(p); err != nil {
				return err
			}
			count++
			if opts.MaxExamples > 0 && count >= opts.MaxExamples {
				return nil
			}
		}
	}
	return rows.Err()
}

func derivePairsFromItemData(data json.RawMessage, opts ExportOptions) []ExportPair {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil
	}

	// Simple single-turn: {"user":"...","assistant":"..."}.
	if uRaw, ok := obj["user"]; ok {
		if aRaw, ok := obj["assistant"]; ok {
			var u, a string
			if err := json.Unmarshal(uRaw, &u); err == nil {
				if err := json.Unmarshal(aRaw, &a); err == nil {
					u = strings.TrimSpace(u)
					a = strings.TrimSpace(a)
					if u != "" && a != "" {
						return []ExportPair{{User: u, Assistant: a}}
					}
				}
			}
		}
	}

	// Multi-turn: {"messages":[{"role":"user","content":"..."}, ...]}.
	if mRaw, ok := obj["messages"]; ok {
		var msgs []Message
		if err := json.Unmarshal(mRaw, &msgs); err != nil {
			return nil
		}
		if len(msgs) == 0 {
			return nil
		}
		return derivePairs(msgs, opts)
	}

	return nil
}

func conversationsFilterQuery(opts ExportOptions) (string, []any) {
	args := []any{}
	where := []string{"status = $1"}
	args = append(args, opts.Status)

	if opts.DatasetID > 0 {
		where = append(where, fmt.Sprintf("dataset_id = $%d", len(args)+1))
		args = append(args, opts.DatasetID)
	}

	if opts.Split != "" && opts.Split != "all" {
		where = append(where, fmt.Sprintf("split = $%d", len(args)+1))
		args = append(args, opts.Split)
	}

	q := `
SELECT id, split, status, tags, source, notes
FROM conversations
WHERE ` + strings.Join(where, " AND ") + `
ORDER BY id ASC
`
	return q, args
}

func derivePairs(msgs []Message, opts ExportOptions) []ExportPair {
	contextMode := opts.Context
	if contextMode == "" {
		contextMode = "none"
	}
	roleStyle := opts.RoleStyle
	if roleStyle == "" {
		roleStyle = "labels"
	}

	var pairs []ExportPair

	for i := 0; i < len(msgs); i++ {
		if msgs[i].Role != RoleAssistant {
			continue
		}

		assistantText := strings.TrimSpace(msgs[i].Content)
		if assistantText == "" {
			continue
		}

		userIdx := findPrevRole(msgs, i-1, RoleUser)
		if userIdx < 0 {
			continue
		}

		var prompt string
		switch contextMode {
		case "none":
			prompt = strings.TrimSpace(msgs[userIdx].Content)
		case "window":
			prompt = renderContext(msgs, userIdx, opts.IncludeSystem, opts.ContextTurns, roleStyle)
		case "full":
			prompt = renderContext(msgs, userIdx, opts.IncludeSystem, 0, roleStyle)
		default:
			prompt = strings.TrimSpace(msgs[userIdx].Content)
		}

		if prompt == "" {
			continue
		}

		pairs = append(pairs, ExportPair{User: prompt, Assistant: assistantText})
	}

	return pairs
}

func findPrevRole(msgs []Message, start int, role Role) int {
	for j := start; j >= 0; j-- {
		if msgs[j].Role == role {
			return j
		}
	}
	return -1
}

func renderContext(msgs []Message, userIdx int, includeSystem bool, contextTurns int, roleStyle string) string {
	// Build context from some number of prior user/assistant turns plus the current user message.
	// contextTurns == 0 => full history.

	start := 0
	if contextTurns > 0 {
		turns := 0
		j := userIdx
		for j >= 0 {
			if msgs[j].Role == RoleUser {
				turns++
				if turns >= contextTurns {
					break
				}
			}
			j--
		}
		if j > 0 {
			start = j
		}
	}

	var b strings.Builder
	for i := start; i <= userIdx; i++ {
		m := msgs[i]
		if m.Role == RoleSystem && !includeSystem {
			continue
		}
		if strings.TrimSpace(m.Content) == "" {
			continue
		}

		if b.Len() > 0 {
			b.WriteString("\n")
		}

		switch roleStyle {
		case "plain":
			b.WriteString(strings.TrimSpace(m.Content))
		default:
			b.WriteString(roleLabel(m.Role))
			b.WriteString(strings.TrimSpace(m.Content))
		}
	}

	return b.String()
}

func roleLabel(r Role) string {
	switch r {
	case RoleSystem:
		return "System: "
	case RoleAssistant:
		return "Assistant: "
	default:
		return "User: "
	}
}
