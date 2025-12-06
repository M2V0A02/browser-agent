package agents

import (
	"browser-agent/internal/domain/ports"
	"browser-agent/internal/infrastructure/env"
	"context"
	"fmt"
)

type Agent interface {
	ExecuteWithStreaming(ctx context.Context, task string, onChunk func(LogEntry)) (*AgentResult, error)
}

func NewAgent(browser ports.BrowserCore, envService *env.EnvService, logger ports.Logger) (Agent, error) {
	agent, err := NewReactAgent(browser, envService, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create ReactAgent: %w", err)
	}
	return agent, nil
}

// NewModernAgent is an alias for NewReactAgent for backwards compatibility
func NewModernAgent(browser ports.BrowserCore, envService *env.EnvService, logger ports.Logger) (*ReactAgent, error) {
	return NewReactAgent(browser, envService, logger)
}
