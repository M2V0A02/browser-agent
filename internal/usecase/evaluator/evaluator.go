package evaluator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"
)

type Evaluator struct {
	llm    output.LLMPort
	logger output.LoggerPort
}

func New(llm output.LLMPort, logger output.LoggerPort) *Evaluator {
	return &Evaluator{
		llm:    llm,
		logger: logger,
	}
}

func (e *Evaluator) Evaluate(ctx context.Context, criteria entity.EvaluationCriteria) (*entity.EvaluationResult, error) {
	prompt := e.buildEvaluationPrompt(criteria)

	messages := []entity.Message{
		{Role: entity.RoleSystem, Content: prompt},
		{Role: entity.RoleUser, Content: fmt.Sprintf("Task: %s\n\nActual Result:\n%s", criteria.TaskDescription, criteria.ActualResult)},
	}

	resp, err := e.llm.Chat(ctx, output.ChatRequest{
		Messages:    messages,
		Temperature: 0.0,
	})
	if err != nil {
		return nil, fmt.Errorf("evaluation llm request failed: %w", err)
	}

	result, err := e.parseEvaluationResponse(resp.Message.Content)
	if err != nil {
		e.logger.Warn("Failed to parse evaluation response, assuming success", "error", err)
		return &entity.EvaluationResult{
			Success:    true,
			Confidence: 0.5,
			Issues:     []string{},
			Feedback:   "",
			ShouldRetry: false,
		}, nil
	}

	e.logger.Info("Evaluation completed",
		"success", result.Success,
		"confidence", result.Confidence,
		"should_retry", result.ShouldRetry,
		"issues_count", len(result.Issues),
	)

	return result, nil
}

func (e *Evaluator) buildEvaluationPrompt(criteria entity.EvaluationCriteria) string {
	basePrompt := `You are an Evaluator Agent. Your job is to assess if an agent successfully completed its task.

Analyze the task description and the actual result, then provide evaluation in JSON format.

Response format (MUST be valid JSON):
{
  "success": true/false,
  "confidence": 0.0-1.0,
  "issues": ["issue1", "issue2"],
  "feedback": "specific feedback for improvement",
  "should_retry": true/false
}

Evaluation criteria:`

	switch criteria.AgentType {
	case entity.AgentTypeNavigation:
		basePrompt += `
- Did the agent navigate to the requested URL?
- Are the required elements found and their selectors provided?
- Is the page structure information clear and actionable?
- If asked to find elements, are BOTH parent and child selectors provided?

SUCCESS if:
✓ Navigation completed or elements found
✓ Selectors are specific (e.g., ".class-name", not "button")
✓ Information is actionable for next agent

SHOULD_RETRY if:
✗ Selectors are too generic or missing
✗ No elements found when they should exist
✗ Page didn't load properly`

	case entity.AgentTypeExtraction:
		basePrompt += `
- Was data actually extracted (not just "found" or "identified")?
- Are all requested fields present in the result?
- Is data structured and parseable?
- Are selectors included for interactive elements (checkboxes, buttons)?

SUCCESS if:
✓ Data is extracted with actual values
✓ All requested fields are present
✓ Format is structured (numbered list, table)
✓ Selectors provided for follow-up actions

SHOULD_RETRY if:
✗ No data extracted, only descriptions
✗ Missing requested fields
✗ Result says "couldn't find" or similar
✗ Used all 5 iterations without success`

	case entity.AgentTypeForm:
		basePrompt += `
- Were the requested form fields filled?
- Were buttons clicked as requested?
- Is there confirmation of success (page redirect, success message)?
- Are specific errors reported if something failed?

SUCCESS if:
✓ Form filled with provided data
✓ Actions completed (click, submit)
✓ Confirmation of result (redirect, message)

SHOULD_RETRY if:
✗ Form fields not found
✗ Click failed
✗ No confirmation of action`

	default:
		basePrompt += `
- Was the requested task completed?
- Is the result clear and actionable?
- Are there any obvious errors or failures?`
	}

	basePrompt += `

IMPORTANT:
- Be strict but fair
- Confidence should reflect certainty (1.0 = definitely successful, 0.0 = definitely failed)
- Only suggest retry if improvement is likely with feedback
- Provide specific, actionable feedback`

	return basePrompt
}

func (e *Evaluator) parseEvaluationResponse(response string) (*entity.EvaluationResult, error) {
	response = strings.TrimSpace(response)

	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")

	if start == -1 || end == -1 {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonStr := response[start : end+1]

	var result entity.EvaluationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &result, nil
}
