-- 1) Nhãn yếu (bootstrapping từ keyword rules)
CREATE TABLE IF NOT EXISTS labels_weak (
  document_id BIGINT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  topic_id TEXT NOT NULL,
  confidence REAL NOT NULL DEFAULT 0.0,
  rule_id TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (document_id, topic_id, rule_id)
);

CREATE INDEX IF NOT EXISTS idx_labels_weak_topic ON labels_weak (topic_id, confidence DESC);

-- 2) Nhãn thật (do người gán)
CREATE TABLE IF NOT EXISTS labels_gold (
  document_id BIGINT PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
  topic_id TEXT NOT NULL,
  labeled_by TEXT NOT NULL DEFAULT '',
  labeled_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_labels_gold_topic ON labels_gold (topic_id, labeled_at DESC);

-- 3) Lưu model (vectorizer + weights) để worker/API/CLI dùng lại
CREATE TABLE IF NOT EXISTS models (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  version INT NOT NULL DEFAULT 1,
  classes JSONB NOT NULL,
  blob BYTEA NOT NULL,        -- model JSON bytes (vectorizer + logreg)
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_models_name_version ON models (name, version);

-- 4) Hàng đợi để người gán nhãn (active learning picks)
--CREATE TYPE labelq_status AS ENUM ('pending', 'labeled', 'skipped');
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'labelq_status') THEN
    CREATE TYPE labelq_status AS ENUM ('pending', 'labeled', 'skipped');
  END IF;
END
$$;

CREATE TABLE IF NOT EXISTS label_queue (
  id BIGSERIAL PRIMARY KEY,
  document_id BIGINT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  status labelq_status NOT NULL DEFAULT 'pending',
  reason TEXT NOT NULL DEFAULT '',
  margin REAL NOT NULL DEFAULT 1.0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  labeled_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_label_queue_doc ON label_queue (document_id);
CREATE INDEX IF NOT EXISTS idx_label_queue_pending ON label_queue (status, created_at DESC);