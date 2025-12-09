package output

import (
	"context"

	"browser-agent/internal/application/port/input"
	"browser-agent/internal/domain/entity"
)

type SimpleAgent interface {
	GetType() entity.AgentType
	GetName() entity.ToolName
	GetDescription() string
	Execute(ctx context.Context, task string) (string, error)
}

type AgentRegistry interface {
	Register(agent input.AgentExecutor)
	Get(agentType entity.AgentType) (input.AgentExecutor, bool)
	List() []entity.AgentType
}

type SimpleAgentRegistry interface {
	Register(agent SimpleAgent)
	Get(agentType entity.AgentType) (SimpleAgent, bool)
	List() []SimpleAgent
}
