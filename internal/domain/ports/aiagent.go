package ports

import "context"

type AIAgent interface {
	Execute(ctx context.Context, task string) (finalAnswer string, err error)
}
