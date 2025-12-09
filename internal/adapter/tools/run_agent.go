package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"
)

type RunAgentTool struct {
	agentRegistry output.SimpleAgentRegistry
	logger        output.LoggerPort
}

func NewRunAgentTool(
	agentRegistry output.SimpleAgentRegistry,
	logger output.LoggerPort,
) *RunAgentTool {
	return &RunAgentTool{
		agentRegistry: agentRegistry,
		logger:        logger,
	}
}

func (t *RunAgentTool) Name() entity.ToolName {
	return entity.ToolRunAgent
}

func (t *RunAgentTool) Description() string {
	return `Run a specialized agent to handle a specific subtask. Use this to delegate work to expert agents.

Available agents:
- navigation: Navigate to URLs, explore pages, find elements and sections
- extraction: Extract structured data, lists, tables from pages
- form: Fill forms, click buttons, handle login/registration flows
- analysis: Analyze page content, verify information, take screenshots

Use this when you need to:
- Navigate to a new page or explore the current page structure
- Extract multiple data items from a page
- Interact with forms and submit data
- Verify page state or analyze content

Each agent has 5 iterations to complete the task and will return structured results.`
}

func (t *RunAgentTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agent_type": map[string]interface{}{
				"type": "string",
				"enum": []string{"navigation", "extraction", "form", "analysis"},
				"description": "Type of agent to run",
			},
			"task": map[string]interface{}{
				"type":        "string",
				"description": "Task for the agent to execute",
			},
		},
		"required": []string{"agent_type", "task"},
	}
}

func (t *RunAgentTool) Execute(ctx context.Context, arguments string) (string, error) {
	var args struct {
		AgentType string `json:"agent_type"`
		Task      string `json:"task"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	subAgentType := entity.SubAgentType(args.AgentType)

	t.logger.Info("Running agent", map[string]interface{}{
		"agent_type": subAgentType,
		"task":       args.Task,
	})

	agent, ok := t.agentRegistry.GetBySubType(subAgentType)
	if !ok {
		return "", fmt.Errorf("agent not found: %s", subAgentType)
	}

	result, err := agent.Execute(ctx, args.Task)
	if err != nil {
		t.logger.Error("Agent execution failed", err, map[string]interface{}{
			"agent_type": subAgentType,
		})
		return "", fmt.Errorf("agent execution failed: %w", err)
	}

	t.logger.Info("Agent completed", map[string]interface{}{
		"agent_type": subAgentType,
	})

	return result, nil
}
