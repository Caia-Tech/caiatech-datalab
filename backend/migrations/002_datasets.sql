CREATE TABLE IF NOT EXISTS datasets (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE conversations
  ADD COLUMN IF NOT EXISTS dataset_id BIGINT;

-- Ensure a 'default' dataset exists.
INSERT INTO datasets (name)
VALUES ('default')
ON CONFLICT (name) DO NOTHING;

-- Create datasets for any existing sources.
INSERT INTO datasets (name)
SELECT DISTINCT source
FROM conversations
WHERE source IS NOT NULL AND source <> ''
ON CONFLICT (name) DO NOTHING;

-- Assign conversations to dataset matching their source.
UPDATE conversations c
SET dataset_id = d.id
FROM datasets d
WHERE c.dataset_id IS NULL
  AND c.source IS NOT NULL
  AND c.source <> ''
  AND d.name = c.source;

-- Assign any remaining conversations to default.
UPDATE conversations c
SET dataset_id = (SELECT id FROM datasets WHERE name = 'default')
WHERE c.dataset_id IS NULL;

ALTER TABLE conversations
  ALTER COLUMN dataset_id SET NOT NULL;

ALTER TABLE conversations
  ADD CONSTRAINT conversations_dataset_id_fk
  FOREIGN KEY (dataset_id) REFERENCES datasets(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS conversations_dataset_idx ON conversations(dataset_id);
CREATE INDEX IF NOT EXISTS conversations_dataset_split_status_idx ON conversations(dataset_id, split, status);
