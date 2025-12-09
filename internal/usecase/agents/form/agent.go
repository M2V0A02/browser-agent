package form

import (
	"context"
	"fmt"

	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"
)

const (
	maxIterations     = 5
	maxObservationLen = 20000
)

var _ output.SimpleAgent = (*Agent)(nil)

type Agent struct {
	llm          output.LLMPort
	tools        output.ToolRegistry
	logger       output.LoggerPort
	systemPrompt string
}

func New(
	llm output.LLMPort,
	tools output.ToolRegistry,
	logger output.LoggerPort,
	systemPrompt string,
) *Agent {
	return &Agent{
		llm:          llm,
		tools:        tools,
		logger:       logger,
		systemPrompt: systemPrompt,
	}
}

func (a *Agent) GetType() entity.AgentType {
	return entity.AgentTypeForm
}

func (a *Agent) GetName() entity.ToolName {
	return entity.ToolAgentForm
}

func (a *Agent) GetDescription() string {
	return "Fill out forms, click buttons, submit data, and handle authentication. Use this when you need to interact with input fields, buttons, or complete forms on the current page."
}

func (a *Agent) Execute(ctx context.Context, task string) (string, error) {
	a.logger.Info("Form agent executing", "task", task)

	messages := []entity.Message{
		{Role: entity.RoleSystem, Content: a.systemPrompt},
		{Role: entity.RoleUser, Content: task},
	}

	toolDefs := a.filterTools()

	for iter := 1; iter <= maxIterations; iter++ {
		a.logger.Debug("Form agent iteration", "iteration", iter)

		resp, err := a.llm.Chat(ctx, output.ChatRequest{
			Messages:    messages,
			Tools:       toolDefs,
			Temperature: 0.0,
		})
		if err != nil {
			return "", fmt.Errorf("llm request failed: %w", err)
		}

		messages = append(messages, resp.Message)

		if len(resp.Message.ToolCalls) == 0 {
			return resp.Message.Content, nil
		}

		for _, tc := range resp.Message.ToolCalls {
			observation := a.executeTool(ctx, tc)

			messages = append(messages, entity.Message{
				Role:       entity.RoleTool,
				ToolCallID: tc.ID,
				Name:       tc.Name,
				Content:    observation,
			})
		}
	}

	return "", fmt.Errorf("max iterations (%d) exceeded", maxIterations)
}

func (a *Agent) executeTool(ctx context.Context, tc entity.ToolCall) string {
	tool, ok := a.tools.Get(entity.ToolName(tc.Name))
	if !ok {
		a.logger.Warn("Unknown tool called", "name", tc.Name)
		return fmt.Sprintf("Error: unknown tool '%s'", tc.Name)
	}

	a.logger.Info("Executing tool", "name", tc.Name, "args", tc.Arguments)

	result, err := tool.Execute(ctx, tc.Arguments)
	if err != nil {
		a.logger.Error("Tool execution failed", "name", tc.Name, "error", err)
		return "Error: " + err.Error()
	}

	if len(result) > maxObservationLen {
		result = result[:maxObservationLen] + "\n... (truncated)"
	}

	a.logger.Debug("Tool completed", "name", tc.Name, "resultLen", len(result))
	return result
}

func (a *Agent) filterTools() []entity.ToolDefinition {
	allowedTools := []entity.ToolName{
		entity.ToolBrowserFill,
		entity.ToolBrowserClick,
		entity.ToolBrowserPressEnter,
		entity.ToolBrowserObserve,
		entity.ToolBrowserSearch,
		entity.ToolUserWaitAction,
		entity.ToolUserAskQuestion,
	}

	allTools := a.tools.Definitions()
	filtered := make([]entity.ToolDefinition, 0)

	for _, tool := range allTools {
		for _, allowed := range allowedTools {
			if tool.Name == allowed {
				filtered = append(filtered, tool)
				break
			}
		}
	}

	return filtered
}
