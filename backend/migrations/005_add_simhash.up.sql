ALTER TABLE documents ADD COLUMN IF NOT EXISTS content_simhash BIGINT NULL;
CREATE INDEX IF NOT EXISTS idx_documents_simhash ON documents (content_simhash);
