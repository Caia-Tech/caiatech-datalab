-- Generic dataset items (JSONB) so DataLab can manage any dataset type.

ALTER TABLE datasets
  ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'items';

CREATE TABLE IF NOT EXISTS dataset_items (
  id BIGSERIAL PRIMARY KEY,
  dataset_id BIGINT NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
  data JSONB NOT NULL,
  source_ref TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS dataset_items_dataset_idx ON dataset_items(dataset_id);
CREATE INDEX IF NOT EXISTS dataset_items_data_gin_idx ON dataset_items USING gin (data);

-- Fast substring search across the JSON text representation.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS dataset_items_data_text_trgm_idx
  ON dataset_items USING gin ((data::text) gin_trgm_ops);

-- Backfill dataset kinds: mark datasets with existing conversations as 'conversations' unless explicitly set.
UPDATE datasets d
SET kind = 'conversations'
WHERE d.kind = 'items'
  AND EXISTS (SELECT 1 FROM conversations c WHERE c.dataset_id = d.id);
