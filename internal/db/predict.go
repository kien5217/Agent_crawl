package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

func UpdateDocumentML(ctx context.Context, conn *pgx.Conn,
	docID int64,
	modelName string,
	modelVersion int,
	mlTopicID string,
	mlConfidence float32,
	mlScoresJSON string,
	predictedAt time.Time,
) error {
	_, err := conn.Exec(ctx, `
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
