package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type Proposal struct {
	ID        int64           `json:"id"`
	Payload   json.RawMessage `json:"payload"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	DecidedAt *time.Time      `json:"decided_at"`
}

func CreateProposal(ctx context.Context, db *sql.DB, payload json.RawMessage) (Proposal, error) {
	row := db.QueryRowContext(ctx, `
INSERT INTO proposals (payload, status)
VALUES ($1, $2)
RETURNING id, payload, status, created_at, decided_at
`, payload, ProposalStatusPending)

	var out Proposal
	if err := row.Scan(&out.ID, &out.Payload, &out.Status, &out.CreatedAt, &out.DecidedAt); err != nil {
		return Proposal{}, err
	}
	return out, nil
}

func ListProposals(ctx context.Context, db *sql.DB, status string) ([]Proposal, error) {
	rows, err := db.QueryContext(ctx, `
SELECT id, payload, status, created_at, decided_at
FROM proposals
WHERE status = $1
ORDER BY id DESC
LIMIT 500
`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Proposal
	for rows.Next() {
		var p Proposal
		if err := rows.Scan(&p.ID, &p.Payload, &p.Status, &p.CreatedAt, &p.DecidedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func GetProposalForDecision(ctx context.Context, tx *sql.Tx, id int64) (Proposal, error) {
	var p Proposal
	err := tx.QueryRowContext(ctx, `
SELECT id, payload, status, created_at, decided_at
FROM proposals
WHERE id = $1 AND status = $2
`, id, ProposalStatusPending).Scan(&p.ID, &p.Payload, &p.Status, &p.CreatedAt, &p.DecidedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return Proposal{}, ErrNotFound
		}
		return Proposal{}, err
	}
	return p, nil
}

func MarkProposalApproved(ctx context.Context, tx *sql.Tx, id int64, now time.Time) error {
	res, err := tx.ExecContext(ctx, `
UPDATE proposals
SET status = $2, decided_at = $3
WHERE id = $1 AND status = $4
`, id, ProposalStatusApproved, now, ProposalStatusPending)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func MarkProposalRejected(ctx context.Context, db *sql.DB, id int64) error {
	res, err := db.ExecContext(ctx, `
UPDATE proposals
SET status = $2, decided_at = now()
WHERE id = $1 AND status = $3
`, id, ProposalStatusRejected, ProposalStatusPending)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
