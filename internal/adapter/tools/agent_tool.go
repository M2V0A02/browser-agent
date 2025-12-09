package tool

import (
	"context"
	"encoding/json"

	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"
)

var _ output.ToolPort = (*AgentTool)(nil)

type AgentTool struct {
	agent  output.SimpleAgent
	logger output.LoggerPort
}

func NewAgentTool(agent output.SimpleAgent, logger output.LoggerPort) *AgentTool {
	return &AgentTool{
		agent:  agent,
		logger: logger,
	}
}

func (t *AgentTool) Name() entity.ToolName {
	return t.agent.GetName()
}

func (t *AgentTool) Description() string {
	return t.agent.GetDescription()
}

func (t *AgentTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task": map[string]interface{}{
				"type":        "string",
				"description": "The specific task for this agent to perform",
			},
		},
		"required": []string{"task"},
	}
}

func (t *AgentTool) Execute(ctx context.Context, arguments string) (string, error) {
	var args struct {
		Task string `json:"task"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", "tool", t.Name(), "error", err)
		return "", err
	}

	t.logger.Info("Agent tool delegating", "agent", t.Name(), "task", args.Task)

	result, err := t.agent.Execute(ctx, args.Task)
	if err != nil {
		t.logger.Error("Agent execution failed", "agent", t.Name(), "error", err)
		return "", err
	}

	return result, nil
}
