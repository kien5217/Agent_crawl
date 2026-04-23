-- Topics loaded from config; stored for reference (optional)
CREATE TABLE IF NOT EXISTS topics (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  keywords JSONB NOT NULL DEFAULT '{}'::jsonb,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sources (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  domain TEXT NOT NULL,
  rss_url TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  trust_score INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- CREATE TYPE queue_status AS ENUM ('pending', 'processing', 'done', 'failed');
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'queue_status') THEN
    CREATE TYPE queue_status AS ENUM ('pending', 'processing', 'done', 'failed');
  END IF;
END
$$;

CREATE TABLE IF NOT EXISTS crawl_queue (
  id BIGSERIAL PRIMARY KEY,
  topic_id TEXT NOT NULL,
  source_id TEXT NOT NULL,
  url TEXT NOT NULL,
  domain TEXT NOT NULL,
  status queue_status NOT NULL DEFAULT 'pending',
  priority INT NOT NULL DEFAULT 0,
  attempts INT NOT NULL DEFAULT 0,
  next_run_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_error TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- avoid enqueue duplicates
CREATE UNIQUE INDEX IF NOT EXISTS uq_crawl_queue_url ON crawl_queue (url);

CREATE INDEX IF NOT EXISTS idx_crawl_queue_pick
  ON crawl_queue (status, next_run_at, priority, id);

CREATE TABLE IF NOT EXISTS documents (
  id BIGSERIAL PRIMARY KEY,
  url TEXT NOT NULL,
  canonical_url TEXT NOT NULL,
  domain TEXT NOT NULL,
  source_id TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  published_at TIMESTAMPTZ NULL,
  author TEXT NOT NULL DEFAULT '',
  content_text TEXT NOT NULL DEFAULT '',
  content_hash TEXT NOT NULL DEFAULT '',
  topic_id TEXT NOT NULL,
  topic_scores JSONB NOT NULL DEFAULT '{}'::jsonb,
  lang TEXT NOT NULL DEFAULT 'vi',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_documents_canonical ON documents (canonical_url);
CREATE INDEX IF NOT EXISTS idx_documents_topic_time ON documents (topic_id, published_at DESC);
CREATE INDEX IF NOT EXISTS idx_documents_domain_time ON documents (domain, published_at DESC);
CREATE INDEX IF NOT EXISTS idx_documents_hash ON documents (content_hash);