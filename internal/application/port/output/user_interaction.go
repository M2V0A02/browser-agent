package output

import "context"

type UserInteractionPort interface {
	AskQuestion(ctx context.Context, question string) (string, error)
	WaitForUserAction(ctx context.Context, message string) error
}
