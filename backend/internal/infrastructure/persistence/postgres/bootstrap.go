package postgres

import (
	"context"

	"Agent_Crawl/internal/domain/config"
)

func UpsertTopics(ctx context.Context, db DB, tf config.TopicsFile) error {
	for _, t := range tf.Topics {
		_, err := db.Exec(ctx, `
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

func UpsertSources(ctx context.Context, db DB, sf config.SourcesFile) error {
	for _, s := range sf.Sources {
		_, err := db.Exec(ctx, `
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
