package input

import "context"

type ExecuteResult struct {
	FinalAnswer string
	Iterations  int
}

type TaskExecutor interface {
	Execute(ctx context.Context, task string) (*ExecuteResult, error)
}
