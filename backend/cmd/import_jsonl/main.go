package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"caiatech-datalab/backend/internal/db"
	"caiatech-datalab/backend/internal/models"
)

type importConversation struct {
	Split    string           `json:"split"`
	Status   string           `json:"status"`
	Tags     []string         `json:"tags"`
	Source   string           `json:"source"`
	Notes    string           `json:"notes"`
	Messages []models.Message `json:"messages"`

	User      string `json:"user"`
	Assistant string `json:"assistant"`
	System    string `json:"system"`
}

func main() {
	var (
		inputPath     = flag.String("input", "", "Input JSONL path")
		databaseURL   = flag.String("database-url", os.Getenv("DATALAB_DATABASE_URL"), "Postgres URL (or set DATALAB_DATABASE_URL)")
		into          = flag.String("into", "items", "Import into: items|conversations")
		defaultSplit  = flag.String("split", "train", "Default split if missing (train|valid|test)")
		defaultStatus = flag.String("status", "approved", "Default status if missing (draft|pending|approved|rejected|archived)")
		defaultSource = flag.String("source", "", "Default source if missing")
		datasetName   = flag.String("dataset", "", "Dataset name to import into (default: source or 'default')")
		replace       = flag.Bool("replace", false, "Delete existing rows in the dataset before import")
		defaultNotes  = flag.String("notes", "", "Default notes if missing")
		defaultTags   = flag.String("tags", "", "Comma-separated tags to apply if missing")
		max           = flag.Int("max", 0, "Max rows to import (0 = unlimited)")
		batch         = flag.Int("batch", 200, "Commit every N rows")
		skipBad       = flag.Bool("skip-bad", true, "Skip invalid lines instead of failing")
		badOut        = flag.String("bad-out", "", "Write invalid lines to this file (optional)")
	)
	flag.Parse()

	if *inputPath == "" {
		log.Fatalf("--input is required")
	}
	if *databaseURL == "" {
		log.Fatalf("--database-url or DATALAB_DATABASE_URL is required")
	}

	in, err := os.Open(*inputPath)
	if err != nil {
		log.Fatalf("open input: %v", err)
	}
	defer in.Close()

	var badFile *os.File
	if *badOut != "" {
		badFile, err = os.Create(*badOut)
		if err != nil {
			log.Fatalf("open bad-out: %v", err)
		}
		defer badFile.Close()
	}

	parsedDefaultTags := parseTags(*defaultTags)
	if *defaultSource == "" {
		*defaultSource = fmt.Sprintf("import:%s", filepathBase(*inputPath))
	}

	if *datasetName == "" {
		// Default dataset name: source (when provided), otherwise file base, otherwise "default".
		if strings.TrimSpace(*defaultSource) != "" {
			*datasetName = *defaultSource
		} else if b := strings.TrimSpace(filepathBase(*inputPath)); b != "" {
			*datasetName = b
		} else {
			*datasetName = "default"
		}
	}
	database, err := db.Open(*databaseURL)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer database.Close()
	ctx := context.Background()

	// Ensure dataset exists
	ds, err := models.EnsureDataset(ctx, database, *datasetName)
	if err != nil {
		log.Fatalf("ensure dataset: %v", err)
	}

	if *replace {
		mode := strings.ToLower(strings.TrimSpace(*into))
		switch mode {
		case "conversations":
			if _, err := database.ExecContext(ctx, "DELETE FROM conversations WHERE dataset_id = $1", ds.ID); err != nil {
				log.Fatalf("replace delete conversations: %v", err)
			}
		default:
			if err := models.DeleteDatasetItemsByDataset(ctx, database, ds.ID); err != nil {
				log.Fatalf("replace delete items: %v", err)
			}
		}
	}

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 1024*1024), 50*1024*1024)

	imported := 0
	bad := 0
	lineNo := 0

	commitBatch := func(tx *sql.Tx) error {
		return tx.Commit()
	}

	newTx := func() *sql.Tx {
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			log.Fatalf("begin tx: %v", err)
		}
		return tx
	}

	tx := newTx()
	started := time.Now()

	mode := strings.ToLower(strings.TrimSpace(*into))
	if mode == "" {
		mode = "items"
	}
	itemSourcePrefix := filepathBase(*inputPath)

	for scanner.Scan() {
		lineNo++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}

		switch mode {
		case "conversations":
			var rec importConversation
			if err := json.Unmarshal([]byte(raw), &rec); err != nil {
				bad++
				if badFile != nil {
					_, _ = badFile.WriteString(raw + "\n")
				}
				if !*skipBad {
					log.Fatalf("line %d: invalid json: %v", lineNo, err)
				}
				continue
			}

			conv, err := normalizeImport(rec, ds.ID, *defaultSplit, *defaultStatus, parsedDefaultTags, *defaultSource, *defaultNotes)
			if err != nil {
				bad++
				if badFile != nil {
					_, _ = badFile.WriteString(raw + "\n")
				}
				if !*skipBad {
					log.Fatalf("line %d: invalid record: %v", lineNo, err)
				}
				continue
			}

			if _, err := models.InsertConversationWithMessages(ctx, tx, conv); err != nil {
				_ = tx.Rollback()
				log.Fatalf("line %d: insert: %v", lineNo, err)
			}

		default:
			// Generic items: store each JSON object as-is in dataset_items.data.
			if !json.Valid([]byte(raw)) {
				bad++
				if badFile != nil {
					_, _ = badFile.WriteString(raw + "\n")
				}
				if !*skipBad {
					log.Fatalf("line %d: invalid json", lineNo)
				}
				continue
			}

			sourceRef := fmt.Sprintf("%s:%d", itemSourcePrefix, lineNo)
			if _, err := tx.ExecContext(ctx, `
INSERT INTO dataset_items (dataset_id, data, source_ref)
VALUES ($1, $2, $3)
`, ds.ID, json.RawMessage(raw), sourceRef); err != nil {
				_ = tx.Rollback()
				log.Fatalf("line %d: insert item: %v", lineNo, err)
			}
		}

		imported++
		if imported%*batch == 0 {
			if err := commitBatch(tx); err != nil {
				log.Fatalf("commit: %v", err)
			}
			tx = newTx()
			log.Printf("imported=%d bad=%d elapsed=%s", imported, bad, time.Since(started).Truncate(time.Second))
		}

		if *max > 0 && imported >= *max {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		_ = tx.Rollback()
		log.Fatalf("scan: %v", err)
	}
	if err := commitBatch(tx); err != nil {
		log.Fatalf("final commit: %v", err)
	}

	log.Printf("done imported=%d bad=%d elapsed=%s", imported, bad, time.Since(started).Truncate(time.Second))
}

func normalizeImport(
	rec importConversation,
	datasetID int64,
	defaultSplit string,
	defaultStatus string,
	defaultTags []string,
	defaultSource string,
	defaultNotes string,
) (models.Conversation, error) {
	splitText := strings.TrimSpace(rec.Split)
	if splitText == "" {
		splitText = defaultSplit
	}
	split, ok := models.NormalizeSplit(splitText)
	if !ok {
		return models.Conversation{}, fmt.Errorf("invalid split: %q", splitText)
	}

	statusText := strings.TrimSpace(rec.Status)
	if statusText == "" {
		statusText = defaultStatus
	}
	status, ok := models.NormalizeConversationStatus(statusText)
	if !ok {
		return models.Conversation{}, fmt.Errorf("invalid status: %q", statusText)
	}

	tags := rec.Tags
	if len(tags) == 0 {
		tags = defaultTags
	}

	source := strings.TrimSpace(rec.Source)
	if source == "" {
		source = defaultSource
	}

	notes := strings.TrimSpace(rec.Notes)
	if notes == "" {
		notes = defaultNotes
	}

	msgs := rec.Messages
	if len(msgs) == 0 {
		user := strings.TrimSpace(rec.User)
		assistant := strings.TrimSpace(rec.Assistant)
		system := strings.TrimSpace(rec.System)
		if user == "" || assistant == "" {
			return models.Conversation{}, fmt.Errorf("missing messages and missing user/assistant")
		}
		if system != "" {
			msgs = append(msgs, models.Message{Role: models.RoleSystem, Content: system})
		}
		msgs = append(msgs,
			models.Message{Role: models.RoleUser, Content: user},
			models.Message{Role: models.RoleAssistant, Content: assistant},
		)
	}

	for i := range msgs {
		msgs[i].Content = strings.TrimSpace(msgs[i].Content)
		msgs[i].Name = strings.TrimSpace(msgs[i].Name)
		if msgs[i].Content == "" {
			return models.Conversation{}, fmt.Errorf("empty content at message %d", i)
		}
		switch msgs[i].Role {
		case models.RoleSystem, models.RoleUser, models.RoleAssistant:
		default:
			return models.Conversation{}, fmt.Errorf("invalid role at message %d", i)
		}
		if len(msgs[i].Meta) == 0 {
			msgs[i].Meta = json.RawMessage("{}")
		}
	}

	return models.Conversation{
		DatasetID: datasetID,
		Split:     split,
		Status:    status,
		Tags:      tags,
		Source:    source,
		Notes:     notes,
		Messages:  msgs,
	}, nil
}

func parseTags(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, t)
	}
	return out
}

func filepathBase(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}
