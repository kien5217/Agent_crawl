package orchestration

import (
	"context"

	"fmt"
	"time"

	"Agent_Crawl/internal/domain/model"
	"Agent_Crawl/internal/domain/repository"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Orchestrator struct {
	repo       repository.WorkflowRepository
	maxRetries int
}

func NewOrchestrator(repo repository.WorkflowRepository, maxRetries int) *Orchestrator {
	return &Orchestrator{repo: repo, maxRetries: maxRetries}
}

// Run chạy một WorkflowDef từ đầu đến cuối, lưu mọi trạng thái vào DB.
func (o *Orchestrator) Run(ctx context.Context, def WorkflowDef) error {
	wfExec := model.WorkflowExecution{
		ID:           uuid.New().String(),
		WorkflowName: def.Name,
		Status:       model.WorkflowRunning,
		StartedAt:    time.Now(),
	}
	if err := o.repo.CreateWorkflow(ctx, wfExec); err != nil {
		return fmt.Errorf("orchestrator: cannot persist workflow: %w", err)
	}

	log.Info().Str("workflow_id", wfExec.ID).Str("name", def.Name).Msg("workflow started")

	for _, step := range def.Steps {
		stepExec := model.StepExecution{
			ID:         uuid.New().String(),
			WorkflowID: wfExec.ID,
			StepName:   step.Name(),
			Status:     model.StepRunning,
			StartedAt:  time.Now(),
		}
		if err := o.repo.CreateStep(ctx, stepExec); err != nil {
			return fmt.Errorf("orchestrator: cannot persist step: %w", err)
		}

		// --- Chạy step với retry ---
		var result StepResult
		var lastErr error
		for attempt := 1; attempt <= o.maxRetries; attempt++ {
			stepExec.Attempts = attempt
			result, lastErr = step.Run(ctx)
			if lastErr == nil {
				break
			}
			log.Warn().Err(lastErr).Str("step", step.Name()).
				Int("attempt", attempt).Msg("step failed, retrying")
		}

		now := time.Now()
		stepExec.CompletedAt = &now

		if lastErr != nil {
			// Step thất bại sau hết số lần retry
			stepExec.Status = model.StepFailed
			stepExec.ErrorMsg = lastErr.Error()
			_ = o.repo.UpdateStep(ctx, stepExec)

			wfExec.Status = model.WorkflowFailed
			wfExec.CompletedAt = &now
			wfExec.ErrorMsg = fmt.Sprintf("step %q failed: %s", step.Name(), lastErr)
			_ = o.repo.UpdateWorkflow(ctx, wfExec)

			log.Error().Str("workflow_id", wfExec.ID).Str("step", step.Name()).
				Err(lastErr).Msg("workflow failed")
			return lastErr
		}

		// --- Lưu kết quả step ---
		if result != nil {
			stepExec.ResultSummaryJSON = result.Summary()
		}
		stepExec.Status = model.StepSuccess
		_ = o.repo.UpdateStep(ctx, stepExec)

		log.Info().Str("step", step.Name()).Str("result", stepExec.ResultSummaryJSON).
			Msg("step succeeded")

		// --- Kiểm tra quality gate ---
		if gate, ok := def.Gates[step.Name()]; ok {
			pass, reason := gate(result)
			if !pass {
				wfExec.Status = model.WorkflowHalted
				wfExec.CompletedAt = &now
				wfExec.ErrorMsg = fmt.Sprintf("gate failed after %q: %s", step.Name(), reason)
				_ = o.repo.UpdateWorkflow(ctx, wfExec)

				log.Warn().Str("workflow_id", wfExec.ID).Str("step", step.Name()).
					Str("reason", reason).Msg("workflow halted by quality gate")
				return fmt.Errorf("halted: %s", reason)
			}
		}
	}

	now := time.Now()
	wfExec.Status = model.WorkflowCompleted
	wfExec.CompletedAt = &now
	_ = o.repo.UpdateWorkflow(ctx, wfExec)

	log.Info().Str("workflow_id", wfExec.ID).Msg("workflow completed")
	return nil
}
