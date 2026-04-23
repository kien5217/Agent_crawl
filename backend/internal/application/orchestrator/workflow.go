package orchestration

// GateFn là hàm kiểm tra kết quả của một step trước khi chạy step tiếp theo.
// Trả về false → workflow bị halt.
type GateFn func(result StepResult) (pass bool, reason string)

// WorkflowDef định nghĩa một workflow (không có state, chỉ là bản thiết kế)
type WorkflowDef struct {
	Name  string
	Steps []Step
	Gates map[string]GateFn // key = Step.Name()
}
