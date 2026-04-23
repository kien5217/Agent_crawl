package postgres

import (
	"context"
	"time"

	"Agent_Crawl/internal/domain/model"
)

func ListDocsForWeakLabel(ctx context.Context, db DB, limit int) ([]model.LearningDocument, error) {
	if limit <= 0 || limit > 50000 {
		limit = 5000
	}
	rows, err := db.Query(ctx, `
		SELECT d.id, d.title, d.content_text
		FROM documents d
		LEFT JOIN labels_weak lw ON lw.document_id = d.id
		WHERE lw.document_id IS NULL
		  AND length(d.content_text) >= 200
		ORDER BY d.id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.LearningDocument
	for rows.Next() {
		var r model.LearningDocument
		if err := rows.Scan(&r.ID, &r.Title, &r.Content); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func UpsertWeakLabel(ctx context.Context, db DB, docID int64, topicID string, confidence float32, ruleID string) error {
	_, err := db.Exec(ctx, `
		INSERT INTO labels_weak (document_id, topic_id, confidence, rule_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (document_id, topic_id, rule_id)
		DO UPDATE SET confidence = GREATEST(labels_weak.confidence, EXCLUDED.confidence)
	`, docID, topicID, confidence, ruleID)
	return err
}

func ListTrainingSet(ctx context.Context, db DB, minWeakConf float32, limit int) ([]model.LearningDocument, error) {
	if limit <= 0 || limit > 200000 {
		limit = 50000
	}
	rows, err := db.Query(ctx, `
		WITH gold AS (
		  SELECT d.id, d.title, d.content_text, lg.topic_id
		  FROM labels_gold lg
		  JOIN documents d ON d.id = lg.document_id
		),
		weak AS (
		  SELECT d.id, d.title, d.content_text, lw.topic_id
		  FROM labels_weak lw
		  JOIN documents d ON d.id = lw.document_id
		  LEFT JOIN labels_gold lg ON lg.document_id = d.id
		  WHERE lg.document_id IS NULL
		    AND lw.confidence >= $1
		)
		SELECT * FROM gold
		UNION ALL
		SELECT * FROM weak
		ORDER BY id DESC
		LIMIT $2
	`, minWeakConf, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.LearningDocument
	for rows.Next() {
		var r model.LearningDocument
		if err := rows.Scan(&r.ID, &r.Title, &r.Content, &r.TopicID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func SaveModel(ctx context.Context, db DB, name string, version int, classesJSON []byte, blob []byte) error {
	_, err := db.Exec(ctx, `
		INSERT INTO models (name, version, classes, blob)
		VALUES ($1, $2, $3::jsonb, $4)
		ON CONFLICT (name, version) DO UPDATE SET
		  classes = EXCLUDED.classes,
		  blob = EXCLUDED.blob,
		  created_at = now()
	`, name, version, string(classesJSON), blob)
	return err
}

func LoadLatestModel(ctx context.Context, db DB, name string) (version int, blob []byte, err error) {
	err = db.QueryRow(ctx, `
		SELECT version, blob
		FROM models
		WHERE name = $1
		ORDER BY version DESC
		LIMIT 1
	`, name).Scan(&version, &blob)
	return
}

func ListUnlabeledDocs(ctx context.Context, db DB, limit int) (ids []int64, titles []string, contents []string, err error) {
	if limit <= 0 || limit > 200000 {
		limit = 50000
	}
	rows, err := db.Query(ctx, `
		SELECT d.id, d.title, d.content_text
		FROM documents d
		LEFT JOIN labels_gold lg ON lg.document_id = d.id
		WHERE lg.document_id IS NULL
		  AND length(d.content_text) >= 200
		ORDER BY d.id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var t, c string
		if err := rows.Scan(&id, &t, &c); err != nil {
			return nil, nil, nil, err
		}
		ids = append(ids, id)
		titles = append(titles, t)
		contents = append(contents, c)
	}
	return ids, titles, contents, rows.Err()
}

func EnqueueLabelQueue(ctx context.Context, db DB, docID int64, reason string, margin float32) error {
	_, err := db.Exec(ctx, `
		INSERT INTO label_queue (document_id, reason, margin, status)
		VALUES ($1, $2, $3, 'pending')
		ON CONFLICT (document_id) DO UPDATE SET
		  reason = EXCLUDED.reason,
		  margin = EXCLUDED.margin
	`, docID, reason, margin)
	return err
}

func UpdateDocumentML(ctx context.Context, db DB,
	docID int64,
	modelName string,
	modelVersion int,
	mlTopicID string,
	mlConfidence float32,
	mlScoresJSON string,
	predictedAt time.Time,
) error {
	_, err := db.Exec(ctx, `
		UPDATE documents
		SET ml_topic_id = $2,
		    ml_confidence = $3,
		    ml_scores = $4::jsonb,
		    ml_model_name = $5,
		    ml_model_version = $6,
		    ml_predicted_at = $7,
		    updated_at = now()
		WHERE id = $1
	`, docID, mlTopicID, mlConfidence, mlScoresJSON, modelName, modelVersion, predictedAt)
	return err
}
