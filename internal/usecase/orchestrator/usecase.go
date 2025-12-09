package orchestrator

import (
	"context"
	"fmt"

	"browser-agent/internal/application/port/input"
	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"
)

const (
	maxIterations     = 30
	maxObservationLen = 20000
)

var _ input.TaskExecutor = (*UseCase)(nil)

type UseCase struct {
	llm          output.LLMPort
	agentTools   output.ToolRegistry
	logger       output.LoggerPort
	systemPrompt string
}

func New(
	llm output.LLMPort,
	agentTools output.ToolRegistry,
	logger output.LoggerPort,
	systemPrompt string,
) *UseCase {
	return &UseCase{
		llm:          llm,
		agentTools:   agentTools,
		logger:       logger,
		systemPrompt: systemPrompt,
	}
}

func (uc *UseCase) Execute(ctx context.Context, task string) (*input.ExecuteResult, error) {
	uc.logger.Info("Orchestrator executing task", "task", task)

	messages := []entity.Message{
		{Role: entity.RoleSystem, Content: uc.systemPrompt},
		{Role: entity.RoleUser, Content: task},
	}

	toolDefs := uc.agentTools.Definitions()

	for iter := 1; iter <= maxIterations; iter++ {
		uc.logger.Debug("Orchestrator iteration", "iteration", iter)

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
			uc.logger.Info("Task completed", "iterations", iter)
			return &input.ExecuteResult{
				FinalAnswer: resp.Message.Content,
				Iterations:  iter,
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
	tool, ok := uc.agentTools.Get(tc.Name)
	if !ok {
		uc.logger.Warn("Unknown agent tool called", "name", tc.Name)
		return fmt.Sprintf("Error: unknown agent tool '%s'", tc.Name)
	}

	uc.logger.Info("Executing agent tool", "name", tc.Name, "args", tc.Arguments)

	result, err := tool.Execute(ctx, tc.Arguments)
	if err != nil {
		uc.logger.Error("Agent tool execution failed", "name", tc.Name, "error", err)
		return "Error: " + err.Error()
	}

	if len(result) > maxObservationLen {
		result = result[:maxObservationLen] + "\n... (truncated)"
	}

	uc.logger.Debug("Agent tool completed", "name", tc.Name, "resultLen", len(result))
	return result
}
