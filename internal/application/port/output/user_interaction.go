package output

import "context"

type UserInteractionPort interface {
	AskQuestion(ctx context.Context, question string) (string, error)
	WaitForUserAction(ctx context.Context, message string) error

	ShowIteration(ctx context.Context, iteration, maxIterations int)
	ShowToolStart(ctx context.Context, toolName, arguments string)
	ShowToolResult(ctx context.Context, toolName, result string, isError bool)
	ShowThinking(ctx context.Context, content string)
}
