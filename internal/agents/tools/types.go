package tools

import "context"

type Tool interface {
	Name() string
	Type() string
	Description() string
	Call(ctx context.Context, input string) (string, error)
	Parameters() map[string]interface{}
}
