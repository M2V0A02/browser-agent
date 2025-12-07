package entity

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

type Task struct {
	ID          string
	Description string
	Status      TaskStatus
}

type TaskResult struct {
	TaskID      string
	FinalAnswer string
	Success     bool
	Error       string
}
