package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"browser-agent/internal/domain/ports"
	"browser-agent/internal/infrastructure/browser/rodwrapper"
)

type UserAsk struct {
	page   *rodwrapper.Page
	logger ports.Logger
}

var _ Tool = (*UserAsk)(nil)

func NewUserAskTool(page *rodwrapper.Page, logger ports.Logger) Tool {
	return &UserAsk{
		page:   page,
		logger: logger,
	}
}

func (t *UserAsk) Name() string {
	return "ask"
}

func (t *UserAsk) Type() string {
	return "user"
}

type userAskInput struct {
	Question string `json:"question"`
}

func (t *UserAsk) Description() string {
	return "Pauses execution and asks the user for additional information or confirmation. Use when you need human input (e.g., solving CAPTCHA, choosing from options, entering text, etc.)."
}

func (t *UserAsk) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"question": map[string]interface{}{
				"type":        "string",
				"description": "The question to present to the user. Be clear about what information or action is needed.",
			},
		},
		"required": []string{"question"},
	}
}

func (t *UserAsk) Call(ctx context.Context, input string) (string, error) {
	var args userAskInput
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return "", fmt.Errorf("invalid JSON input: %w", err)
	}
	if args.Question == "" {
		return "", fmt.Errorf("question is required")
	}

	fmt.Printf("\n%s\n", args.Question)
	fmt.Print("Your answer: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("input read error: %w", err)
	}

	response = strings.TrimSpace(response)
	t.logger.Logf("UserAsk: Question='%s', Answer='%s'", args.Question, response)

	return response, nil
}
