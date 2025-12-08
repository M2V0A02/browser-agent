package executor

import (
	"context"
	"fmt"

	"browser-agent/internal/application/port/input"
	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"
)

var _ input.TaskExecutor = (*UseCase)(nil)

const (
	maxIterations     = 50
	maxObservationLen = 20000
)

type UseCase struct {
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
) *UseCase {
	return &UseCase{
		llm:          llm,
		tools:        tools,
		logger:       logger,
		systemPrompt: systemPrompt,
	}
}

func (uc *UseCase) Execute(ctx context.Context, task string) (*input.ExecuteResult, error) {
	messages := []entity.Message{
		{Role: entity.RoleSystem, Content: uc.systemPrompt},
		{Role: entity.RoleUser, Content: task},
	}

	toolDefs := uc.tools.Definitions()

	for iteration := 1; iteration <= maxIterations; iteration++ {
		uc.logger.Debug("Starting iteration", "iteration", iteration)

		resp, err := uc.llm.Chat(ctx, output.ChatRequest{
			Messages:    messages,
			Tools:       toolDefs,
			Temperature: 0.0,
		})
		if err != nil {
			return nil, fmt.Errorf("llm request failed: %w", err)
		}

		messages = append(messages, resp.Message)

		if len(resp.Message.ToolCalls) == 0 {
			return &input.ExecuteResult{
				FinalAnswer: resp.Message.Content,
				Iterations:  iteration,
			}, nil
		}

		for _, tc := range resp.Message.ToolCalls {
			observation := uc.executeTool(ctx, tc)

			messages = append(messages, entity.Message{
				Role:       entity.RoleTool,
				ToolCallID: tc.ID,
				Name:       tc.Name,
				Content:    observation,
			})
		}
	}

	return nil, fmt.Errorf("max iterations (%d) exceeded", maxIterations)
}

func (uc *UseCase) executeTool(ctx context.Context, tc entity.ToolCall) string {
	tool, ok := uc.tools.Get(tc.Name)
	if !ok {
		uc.logger.Warn("Unknown tool called", "name", tc.Name)
		return fmt.Sprintf("Error: unknown tool '%s'", tc.Name)
	}

	uc.logger.Info("Executing tool", "name", tc.Name, "args", tc.Arguments)

	result, err := tool.Execute(ctx, tc.Arguments)
	if err != nil {
		uc.logger.Error("Tool execution failed", "name", tc.Name, "error", err)
		return "Error: " + err.Error()
	}

	if len(result) > maxObservationLen {
		result = result[:maxObservationLen] + "\n... (truncated)"
	}

	uc.logger.Debug("Tool completed", "name", tc.Name, "resultLen", len(result))
	return result
}
