package navigation

import (
	"context"
	"fmt"

	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"
)

const (
	maxIterations     = 10
	maxObservationLen = 20000
)

var _ output.SimpleAgent = (*Agent)(nil)

type Agent struct {
	llm             output.LLMPort
	tools           output.ToolRegistry
	logger          output.LoggerPort
	userInteraction output.UserInteractionPort
	systemPrompt    string
}

func New(
	llm output.LLMPort,
	tools output.ToolRegistry,
	logger output.LoggerPort,
	userInteraction output.UserInteractionPort,
	systemPrompt string,
) *Agent {
	return &Agent{
		llm:             llm,
		tools:           tools,
		logger:          logger,
		userInteraction: userInteraction,
		systemPrompt:    systemPrompt,
	}
}

func (a *Agent) GetType() entity.AgentType {
	return entity.AgentTypeNavigation
}

func (a *Agent) GetSubAgentType() entity.SubAgentType {
	return entity.SubAgentNavigation
}

func (a *Agent) GetDescription() string {
	return "Navigate to URLs and verify pages loaded. Does NOT analyze structure, find selectors, fill forms, or extract data."
}

func (a *Agent) Execute(ctx context.Context, task string) (string, error) {
	a.logger.Info("Navigation agent executing", "task", task)
	return a.executeWithIterations(ctx, task)
}

func (a *Agent) executeWithIterations(ctx context.Context, task string) (string, error) {
	messages := []entity.Message{
		{Role: entity.RoleSystem, Content: a.systemPrompt},
		{Role: entity.RoleUser, Content: task},
	}

	toolDefs := a.filterTools()

	for iter := 1; iter <= maxIterations; iter++ {
		a.userInteraction.ShowIteration(ctx, iter, maxIterations)
		a.logger.Debug("Navigation agent iteration", "iteration", iter)

		resp, err := a.llm.Chat(ctx, output.ChatRequest{
			Messages:    messages,
			Tools:       toolDefs,
			Temperature: 0.0,
		})
		if err != nil {
			return "", fmt.Errorf("llm request failed: %w", err)
		}

		if resp.Message.Content != "" {
			a.userInteraction.ShowThinking(ctx, resp.Message.Content)
		}

		messages = append(messages, resp.Message)

		if len(resp.Message.ToolCalls) == 0 {
			return resp.Message.Content, nil
		}

		for _, tc := range resp.Message.ToolCalls {
			a.userInteraction.ShowToolStart(ctx, tc.Name, tc.Arguments)
			observation := a.executeTool(ctx, tc)

			isError := false
			if len(observation) > 7 && observation[:7] == "Error: " {
				isError = true
			}

			a.userInteraction.ShowToolResult(ctx, tc.Name, observation, isError)

			messages = append(messages, entity.Message{
				Role:       entity.RoleTool,
				ToolCallID: tc.ID,
				Name:       tc.Name,
				Content:    observation,
			})
		}
	}

	// Summary iteration: force agent to provide final report
	a.logger.Info("Max iterations reached, requesting final summary")
	messages = append(messages, entity.Message{
		Role: entity.RoleUser,
		Content: fmt.Sprintf(`CRITICAL: Maximum iterations reached. You MUST provide your FINAL REPORT now.

Format your response as:
- If task completed successfully: Provide your success report as instructed
- If task failed: Start with "FAILED:" and provide detailed failure report as instructed in your prompt
- If task partially completed: Start with "PARTIAL SUCCESS:" and explain what was done

This is your LAST response. Do NOT call any tools. Provide text response ONLY.`),
	})

	summaryResp, err := a.llm.Chat(ctx, output.ChatRequest{
		Messages:    messages,
		Tools:       nil, // No tools allowed in summary iteration
		Temperature: 0.0,
	})
	if err != nil {
		return "", fmt.Errorf("summary iteration failed: %w", err)
	}

	a.logger.Info("Summary report received", "contentLen", len(summaryResp.Message.Content))
	return summaryResp.Message.Content, nil
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
		entity.ToolBrowserNavigate,
		entity.ToolBrowserObserve,
		entity.ToolBrowserScroll,
		entity.ToolBrowserSearch,
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
