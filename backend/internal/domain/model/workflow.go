package model

import "time"

// WorkflowStatus là trạng thái của một lần chạy workflow
type WorkflowStatus string

const (
	WorkflowPending   WorkflowStatus = "pending"
	WorkflowRunning   WorkflowStatus = "running"
	WorkflowCompleted WorkflowStatus = "completed"
	WorkflowFailed    WorkflowStatus = "failed"
	WorkflowHalted    WorkflowStatus = "halted"
)

// StepStatus là trạng thái của một bước trong workflow
type StepStatus string

const (
	StepPending StepStatus = "pending"
	StepRunning StepStatus = "running"
	StepSuccess StepStatus = "success"
	StepSkipped StepStatus = "skipped"
	StepFailed  StepStatus = "failed"
)

// WorkflowExecution là bản ghi một lần chạy workflow (lưu vào DB)
type WorkflowExecution struct {
	ID           string
	WorkflowName string
	Status       WorkflowStatus
	StartedAt    time.Time
	CompletedAt  *time.Time // nil nếu chưa xong
	ErrorMsg     string     // lý do fail/halt nếu có
}

// StepExecution là bản ghi một bước trong một lần chạy workflow
type StepExecution struct {
	ID                string
	WorkflowID        string // FK → WorkflowExecution.ID
	StepName          string
	Status            StepStatus
	Attempts          int
	ResultSummaryJSON string // JSON kết quả mỗi step (số URL, số doc, score...)
	ErrorMsg          string
	StartedAt         time.Time
	CompletedAt       *time.Time
}
