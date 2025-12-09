package input

import (
	"context"

	"browser-agent/internal/domain/entity"
)

type AgentExecutor interface {
	Execute(ctx context.Context, req entity.AgentRequest) (*entity.AgentResponse, error)
	GetType() entity.AgentType
}
