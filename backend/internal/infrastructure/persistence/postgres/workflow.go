package postgres

import (
	"Agent_Crawl/internal/domain/model"
	"context"
)

func CreateWorkflow(ctx context.Context, db DB, wf model.WorkflowExecution) error {
	_, err := db.Exec(ctx, `
        INSERT INTO workflow_executions (id, workflow_name, status, started_at, error_msg)
        VALUES ($1, $2, $3, $4, $5)
    `, wf.ID, wf.WorkflowName, wf.Status, wf.StartedAt, wf.ErrorMsg)
	return err
}

func UpdateWorkflow(ctx context.Context, db DB, wf model.WorkflowExecution) error {
	_, err := db.Exec(ctx, `
        UPDATE workflow_executions
        SET status=$2, completed_at=$3, error_msg=$4
        WHERE id=$1
    `, wf.ID, wf.Status, wf.CompletedAt, wf.ErrorMsg)
	return err
}

func CreateStep(ctx context.Context, db DB, s model.StepExecution) error {
	_, err := db.Exec(ctx, `
        INSERT INTO step_executions
          (id, workflow_id, step_name, status, attempts, result_summary_json, error_msg, started_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
    `, s.ID, s.WorkflowID, s.StepName, s.Status, s.Attempts,
		s.ResultSummaryJSON, s.ErrorMsg, s.StartedAt)
	return err
}

func UpdateStep(ctx context.Context, db DB, s model.StepExecution) error {
	_, err := db.Exec(ctx, `
        UPDATE step_executions
        SET status=$2, attempts=$3, result_summary_json=$4, error_msg=$5, completed_at=$6
        WHERE id=$1
    `, s.ID, s.Status, s.Attempts, s.ResultSummaryJSON, s.ErrorMsg, s.CompletedAt)
	return err
}

func ListWorkflows(ctx context.Context, db DB, limit int) ([]model.WorkflowExecution, error) {
	rows, err := db.Query(ctx, `
        SELECT id, workflow_name, status, started_at, completed_at, error_msg
        FROM workflow_executions
        ORDER BY started_at DESC
        LIMIT $1
    `, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.WorkflowExecution
	for rows.Next() {
		var wf model.WorkflowExecution
		if err := rows.Scan(&wf.ID, &wf.WorkflowName, &wf.Status,
			&wf.StartedAt, &wf.CompletedAt, &wf.ErrorMsg); err != nil {
			return nil, err
		}
		result = append(result, wf)
	}
	return result, rows.Err()
}

func ListSteps(ctx context.Context, db DB, workflowID string) ([]model.StepExecution, error) {
	rows, err := db.Query(ctx, `
        SELECT id, workflow_id, step_name, status, attempts,
               result_summary_json, error_msg, started_at, completed_at
        FROM step_executions
        WHERE workflow_id=$1
        ORDER BY started_at ASC
    `, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.StepExecution
	for rows.Next() {
		var s model.StepExecution
		if err := rows.Scan(&s.ID, &s.WorkflowID, &s.StepName, &s.Status, &s.Attempts,
			&s.ResultSummaryJSON, &s.ErrorMsg, &s.StartedAt, &s.CompletedAt); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}
