package form

import (
	"context"
	"fmt"

	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"
	"browser-agent/internal/usecase/evaluator"
)

const (
	maxIterations     = 5
	maxRetries        = 2
	maxObservationLen = 20000
)

var _ output.SimpleAgent = (*Agent)(nil)

type Agent struct {
	llm             output.LLMPort
	tools           output.ToolRegistry
	logger          output.LoggerPort
	userInteraction output.UserInteractionPort
	systemPrompt    string
	evaluator       *evaluator.Evaluator
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
		evaluator:       evaluator.New(llm, logger),
	}
}

func (a *Agent) GetType() entity.AgentType {
	return entity.AgentTypeForm
}

func (a *Agent) GetSubAgentType() entity.SubAgentType {
	return entity.SubAgentForm
}

func (a *Agent) GetDescription() string {
	return "Fill forms, click buttons, and modify page content. Use ONLY for interactions that change page state. Does NOT navigate to new URLs."
}

func (a *Agent) Execute(ctx context.Context, task string) (string, error) {
	a.logger.Info("Form agent executing", "task", task)

	var lastResult string
	originalTask := task

	for retry := 0; retry <= maxRetries; retry++ {
		if retry > 0 {
			a.logger.Info("Retrying form interaction with feedback", "retry", retry)
		}

		result, err := a.executeWithIterations(ctx, task)
		if err != nil {
			return "", err
		}

		lastResult = result

		eval, err := a.evaluator.Evaluate(ctx, entity.EvaluationCriteria{
			TaskDescription: originalTask,
			ActualResult:    result,
			AgentType:       entity.AgentTypeForm,
		})
		if err != nil {
			a.logger.Warn("Evaluation failed, returning result anyway", "error", err)
			return result, nil
		}

		if eval.Success {
			a.logger.Info("Form interaction successful",
				"confidence", eval.Confidence,
				"retry_number", retry,
			)
			return result, nil
		}

		if !eval.ShouldRetry || retry == maxRetries {
			a.logger.Warn("Form interaction suboptimal but not retrying",
				"confidence", eval.Confidence,
				"issues", eval.Issues,
			)
			return result, nil
		}

		a.logger.Info("Form interaction needs improvement, retrying",
			"confidence", eval.Confidence,
			"issues", eval.Issues,
		)

		task = fmt.Sprintf("%s\n\nPREVIOUS ATTEMPT FEEDBACK:\n%s\n\nIssues to fix:\n", originalTask, eval.Feedback)
		for i, issue := range eval.Issues {
			task += fmt.Sprintf("%d. %s\n", i+1, issue)
		}
	}

	return lastResult, nil
}

func (a *Agent) executeWithIterations(ctx context.Context, task string) (string, error) {
	messages := []entity.Message{
		{Role: entity.RoleSystem, Content: a.systemPrompt},
		{Role: entity.RoleUser, Content: task},
	}

	toolDefs := a.filterTools()

	for iter := 1; iter <= maxIterations; iter++ {
		a.userInteraction.ShowIteration(ctx, iter, maxIterations)
		a.logger.Debug("Form agent iteration", "iteration", iter)

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
