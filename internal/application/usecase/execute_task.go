package usecase

import (
	"context"
	"fmt"

	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"
)

const (
	maxIterations     = 50
	maxObservationLen = 20000
)

type ExecuteTaskUseCase struct {
	llm      output.LLMPort
	tools    output.ToolRegistry
	logger   output.LoggerPort
	systemPrompt string
}

type ExecuteTaskConfig struct {
	SystemPrompt string
}

func DefaultExecuteTaskConfig() ExecuteTaskConfig {
	return ExecuteTaskConfig{
		SystemPrompt: "Ты — автономный браузерный агент. Думай шаг за шагом и используй инструменты.",
	}
}

func NewExecuteTaskUseCase(
	llm output.LLMPort,
	tools output.ToolRegistry,
	logger output.LoggerPort,
	cfg ExecuteTaskConfig,
) *ExecuteTaskUseCase {
	return &ExecuteTaskUseCase{
		llm:          llm,
		tools:        tools,
		logger:       logger,
		systemPrompt: cfg.SystemPrompt,
	}
}

type ExecuteResult struct {
	FinalAnswer string
	Iterations  int
}

func (uc *ExecuteTaskUseCase) Execute(ctx context.Context, task string) (*ExecuteResult, error) {
	messages := []entity.Message{
		{Role: entity.RoleSystem, Content: uc.systemPrompt},
		{Role: entity.RoleUser, Content: task},
	}

	toolDefs := uc.tools.Definitions()

	for iteration := 1; iteration <= maxIterations; iteration++ {
		uc.logger.Debug("Starting iteration", "iteration", iteration)

		resp, err := uc.llm.ChatStream(ctx, output.ChatRequest{
			Messages:    messages,
			Tools:       toolDefs,
			Temperature: 0.0,
		}, nil)
		if err != nil {
			return nil, fmt.Errorf("llm request failed: %w", err)
		}

		messages = append(messages, resp.Message)

		if len(resp.Message.ToolCalls) == 0 {
			return &ExecuteResult{
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

func (uc *ExecuteTaskUseCase) executeTool(ctx context.Context, tc entity.ToolCall) string {
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
