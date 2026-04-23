package orchestration

import "context"

// StepResult là kết quả mỗi step trả về (bất kỳ dạng gì)
type StepResult interface {
	// Summary trả về JSON string để lưu vào DB
	Summary() string
}

// Step là interface mỗi bước phải implement
type Step interface {
	Name() string
	Run(ctx context.Context) (StepResult, error)
}
