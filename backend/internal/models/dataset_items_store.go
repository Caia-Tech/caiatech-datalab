package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

type DatasetItem struct {
	ID        int64           `json:"id"`
	DatasetID int64           `json:"dataset_id"`
	Data      json.RawMessage `json:"data"`
	SourceRef string          `json:"source_ref"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type ListDatasetItemsParams struct {
	DatasetID int64
	Query     string
	Limit     int
	Offset    int
}

func ListDatasetItems(ctx context.Context, db *sql.DB, p ListDatasetItemsParams) ([]DatasetItem, error) {
	q := strings.TrimSpace(p.Query)
	if q == "" {
		rows, err := db.QueryContext(ctx, `
SELECT id, dataset_id, data, source_ref, created_at, updated_at
FROM dataset_items
WHERE dataset_id = $1
ORDER BY id DESC
LIMIT $2 OFFSET $3
`, p.DatasetID, p.Limit, p.Offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanDatasetItems(rows)
	}

	pattern := "%" + q + "%"
	rows, err := db.QueryContext(ctx, `
SELECT id, dataset_id, data, source_ref, created_at, updated_at
FROM dataset_items
WHERE dataset_id = $1 AND (data::text ILIKE $2 OR source_ref ILIKE $2)
ORDER BY id DESC
LIMIT $3 OFFSET $4
`, p.DatasetID, pattern, p.Limit, p.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDatasetItems(rows)
}

func GetDatasetItem(ctx context.Context, db *sql.DB, id int64) (DatasetItem, error) {
	var it DatasetItem
	row := db.QueryRowContext(ctx, `
SELECT id, dataset_id, data, source_ref, created_at, updated_at
FROM dataset_items
WHERE id = $1
`, id)
	if err := row.Scan(&it.ID, &it.DatasetID, &it.Data, &it.SourceRef, &it.CreatedAt, &it.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return DatasetItem{}, ErrNotFound
		}
		return DatasetItem{}, err
	}
	return it, nil
}

func CreateDatasetItem(ctx context.Context, db *sql.DB, datasetID int64, data json.RawMessage, sourceRef string) (DatasetItem, error) {
	if datasetID <= 0 {
		return DatasetItem{}, ErrInvalidInput
	}
	if len(data) == 0 {
		return DatasetItem{}, ErrInvalidInput
	}
	if !json.Valid(data) {
		return DatasetItem{}, ErrInvalidInput
	}

	sourceRef = strings.TrimSpace(sourceRef)
	row := db.QueryRowContext(ctx, `
INSERT INTO dataset_items (dataset_id, data, source_ref)
VALUES ($1, $2, $3)
RETURNING id, dataset_id, data, source_ref, created_at, updated_at
`, datasetID, data, sourceRef)

	var it DatasetItem
	if err := row.Scan(&it.ID, &it.DatasetID, &it.Data, &it.SourceRef, &it.CreatedAt, &it.UpdatedAt); err != nil {
		return DatasetItem{}, err
	}
	return it, nil
}

func UpdateDatasetItem(ctx context.Context, db *sql.DB, id int64, data json.RawMessage, sourceRef string) (DatasetItem, error) {
	if id <= 0 {
		return DatasetItem{}, ErrInvalidInput
	}
	if len(data) == 0 {
		return DatasetItem{}, ErrInvalidInput
	}
	if !json.Valid(data) {
		return DatasetItem{}, ErrInvalidInput
	}

	now := time.Now().UTC()
	sourceRef = strings.TrimSpace(sourceRef)

	res, err := db.ExecContext(ctx, `
UPDATE dataset_items
SET data = $2,
    source_ref = $3,
    updated_at = $4
WHERE id = $1
`, id, data, sourceRef, now)
	if err != nil {
		return DatasetItem{}, err
	}
	a, err := res.RowsAffected()
	if err != nil {
		return DatasetItem{}, err
	}
	if a == 0 {
		return DatasetItem{}, ErrNotFound
	}
	return GetDatasetItem(ctx, db, id)
}

func DeleteDatasetItem(ctx context.Context, db *sql.DB, id int64) error {
	res, err := db.ExecContext(ctx, `DELETE FROM dataset_items WHERE id = $1`, id)
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

func DeleteDatasetItemsByDataset(ctx context.Context, db *sql.DB, datasetID int64) error {
	if datasetID <= 0 {
		return ErrInvalidInput
	}
	_, err := db.ExecContext(ctx, `DELETE FROM dataset_items WHERE dataset_id = $1`, datasetID)
	return err
}

func scanDatasetItems(rows *sql.Rows) ([]DatasetItem, error) {
	var out []DatasetItem
	for rows.Next() {
		var it DatasetItem
		if err := rows.Scan(&it.ID, &it.DatasetID, &it.Data, &it.SourceRef, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
