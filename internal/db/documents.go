package db

import (
	"context"

	"Agent_Crawl/internal/config"

	"github.com/jackc/pgx/v5"
)

// UpsertTopics inserts or updates topics in the database. It takes a TopicsFile struct which contains a list of topics, and for each topic, it performs an upsert operation based on the topic ID. If a topic with the same ID already exists, it updates the name, keywords, and updated_at timestamp. If it doesn't exist, it inserts a new record with enabled set to true.
func UpsertTopics(ctx context.Context, conn *pgx.Conn, tf config.TopicsFile) error {
	for _, t := range tf.Topics {
		_, err := conn.Exec(ctx, `
			INSERT INTO topics (id, name, keywords, enabled)
			VALUES ($1, $2, $3::jsonb, TRUE)
			ON CONFLICT (id) DO UPDATE SET
			  name = EXCLUDED.name,
			  keywords = EXCLUDED.keywords,
			  updated_at = now()
		`, t.ID, t.Name, "{}")
		if err != nil {
			return err
		}
	}
	return nil
}

// insert and update nguồn tin vào bảng sources. Nếu nguồn tin đã tồn tại (dựa trên id), nó sẽ cập nhật tên, domain, rss_url, enabled và updated_at. Nếu không tồn tại, nó sẽ chèn một bản ghi mới với enabled được đặt thành giá trị trong cấu hình.
func UpsertSources(ctx context.Context, conn *pgx.Conn, sf config.SourcesFile) error {
	for _, s := range sf.Sources {
		_, err := conn.Exec(ctx, `
			INSERT INTO sources (id, name, domain, rss_url, enabled)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id) DO UPDATE SET
			  name = EXCLUDED.name,
			  domain = EXCLUDED.domain,
			  rss_url = EXCLUDED.rss_url,
			  enabled = EXCLUDED.enabled,
			  updated_at = now()
		`, s.ID, s.Name, s.Domain, s.RSSURL, s.Enabled)
		if err != nil {
			return err
		}
	}
	return nil
}
