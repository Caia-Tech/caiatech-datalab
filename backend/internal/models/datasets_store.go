package models

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type ListDatasetsParams struct {
	Query  string
	Limit  int
	Offset int
}

func ListDatasets(ctx context.Context, db *sql.DB, p ListDatasetsParams) ([]Dataset, error) {
	q := strings.TrimSpace(p.Query)
	if q == "" {
		rows, err := db.QueryContext(ctx, `
SELECT d.id, d.name, d.description, d.kind,
       COALESCE(di.cnt, 0) AS item_count,
       COALESCE(cc.cnt, 0) AS conversation_count,
       d.created_at, d.updated_at
FROM datasets d
LEFT JOIN (
  SELECT dataset_id, COUNT(*) AS cnt
  FROM dataset_items
  GROUP BY dataset_id
) di ON di.dataset_id = d.id
LEFT JOIN (
  SELECT dataset_id, COUNT(*) AS cnt
  FROM conversations
  GROUP BY dataset_id
) cc ON cc.dataset_id = d.id
ORDER BY d.id DESC
LIMIT $1 OFFSET $2
`, p.Limit, p.Offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanDatasets(rows)
	}

	pattern := "%" + q + "%"
	rows, err := db.QueryContext(ctx, `
SELECT d.id, d.name, d.description, d.kind,
       COALESCE(di.cnt, 0) AS item_count,
       COALESCE(cc.cnt, 0) AS conversation_count,
       d.created_at, d.updated_at
FROM datasets d
LEFT JOIN (
  SELECT dataset_id, COUNT(*) AS cnt
  FROM dataset_items
  GROUP BY dataset_id
) di ON di.dataset_id = d.id
LEFT JOIN (
  SELECT dataset_id, COUNT(*) AS cnt
  FROM conversations
  GROUP BY dataset_id
) cc ON cc.dataset_id = d.id
WHERE d.name ILIKE $1 OR d.description ILIKE $1
ORDER BY d.id DESC
LIMIT $2 OFFSET $3
`, pattern, p.Limit, p.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDatasets(rows)
}

func GetDataset(ctx context.Context, db *sql.DB, id int64) (Dataset, error) {
	var d Dataset
	err := db.QueryRowContext(ctx, `
SELECT d.id, d.name, d.description, d.kind,
       COALESCE(di.cnt, 0) AS item_count,
       COALESCE(cc.cnt, 0) AS conversation_count,
       d.created_at, d.updated_at
FROM datasets d
LEFT JOIN (
  SELECT dataset_id, COUNT(*) AS cnt
  FROM dataset_items
  WHERE dataset_id = $1
  GROUP BY dataset_id
) di ON di.dataset_id = d.id
LEFT JOIN (
  SELECT dataset_id, COUNT(*) AS cnt
  FROM conversations
  WHERE dataset_id = $1
  GROUP BY dataset_id
) cc ON cc.dataset_id = d.id
WHERE d.id = $1
`, id).Scan(&d.ID, &d.Name, &d.Description, &d.Kind, &d.ItemCount, &d.ConversationCount, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return Dataset{}, ErrNotFound
		}
		return Dataset{}, err
	}
	return d, nil
}

func CreateDataset(ctx context.Context, db *sql.DB, name string, description string, kind string) (Dataset, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	kind = strings.TrimSpace(strings.ToLower(kind))
	if name == "" {
		return Dataset{}, ErrInvalidInput
	}
	if kind == "" {
		kind = "items"
	}
	row := db.QueryRowContext(ctx, `
INSERT INTO datasets (name, description, kind)
VALUES ($1, $2, $3)
RETURNING id, name, description, kind, created_at, updated_at
`, name, description, kind)

	var d Dataset
	if err := row.Scan(&d.ID, &d.Name, &d.Description, &d.Kind, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return Dataset{}, err
	}
	return d, nil
}

func UpdateDataset(ctx context.Context, db *sql.DB, id int64, name string, description string, kind string) (Dataset, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	kind = strings.TrimSpace(strings.ToLower(kind))

	now := time.Now().UTC()
	res, err := db.ExecContext(ctx, `
UPDATE datasets
SET name = COALESCE(NULLIF($2, ''), name),
    description = COALESCE($3, description),
    kind = COALESCE(NULLIF($4, ''), kind),
    updated_at = $5
WHERE id = $1
`, id, name, description, kind, now)
	if err != nil {
		return Dataset{}, err
	}
	a, err := res.RowsAffected()
	if err != nil {
		return Dataset{}, err
	}
	if a == 0 {
		return Dataset{}, ErrNotFound
	}
	return GetDataset(ctx, db, id)
}

func DeleteDataset(ctx context.Context, db *sql.DB, id int64) error {
	res, err := db.ExecContext(ctx, `DELETE FROM datasets WHERE id = $1`, id)
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

func EnsureDataset(ctx context.Context, db *sql.DB, name string) (Dataset, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "default"
	}

	var d Dataset
	err := db.QueryRowContext(ctx, `
SELECT id, name, description, kind, created_at, updated_at
FROM datasets
WHERE name = $1
`, name).Scan(&d.ID, &d.Name, &d.Description, &d.Kind, &d.CreatedAt, &d.UpdatedAt)
	if err == nil {
		return d, nil
	}
	if err != sql.ErrNoRows {
		return Dataset{}, err
	}

	row := db.QueryRowContext(ctx, `
INSERT INTO datasets (name)
VALUES ($1)
RETURNING id, name, description, kind, created_at, updated_at
`, name)
	if err := row.Scan(&d.ID, &d.Name, &d.Description, &d.Kind, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return Dataset{}, err
	}
	return d, nil
}

func scanDatasets(rows *sql.Rows) ([]Dataset, error) {
	var out []Dataset
	for rows.Next() {
		var d Dataset
		if err := rows.Scan(
			&d.ID,
			&d.Name,
			&d.Description,
			&d.Kind,
			&d.ItemCount,
			&d.ConversationCount,
			&d.CreatedAt,
			&d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
