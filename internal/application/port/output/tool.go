package output

import (
	"context"

	"browser-agent/internal/domain/entity"
)

type ToolPort interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(ctx context.Context, arguments string) (string, error)
}

type ToolRegistry interface {
	Register(tool ToolPort)
	Get(name string) (ToolPort, bool)
	All() []ToolPort
	Definitions() []entity.ToolDefinition
}
