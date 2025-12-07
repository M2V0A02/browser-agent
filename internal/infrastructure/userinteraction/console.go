package userinteraction

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"browser-agent/internal/application/port/output"
)

var _ output.UserInteractionPort = (*ConsoleUserInteraction)(nil)

type ConsoleUserInteraction struct {
	reader *bufio.Reader
}

func NewConsoleUserInteraction() *ConsoleUserInteraction {
	return &ConsoleUserInteraction{
		reader: bufio.NewReader(os.Stdin),
	}
}

func (u *ConsoleUserInteraction) AskQuestion(ctx context.Context, question string) (string, error) {
	fmt.Printf("\n[USER INPUT REQUIRED] %s\n> ", question)

	answer, err := u.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read user input: %w", err)
	}

	return strings.TrimSpace(answer), nil
}

func (u *ConsoleUserInteraction) WaitForUserAction(ctx context.Context, message string) error {
	fmt.Printf("\n[USER ACTION REQUIRED] %s\n", message)
	fmt.Print("Press Enter when done...")

	_, err := u.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to wait for user: %w", err)
	}

	return nil
}
