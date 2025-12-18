CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS conversation_messages_content_trgm_idx
  ON conversation_messages USING gin (content gin_trgm_ops);
