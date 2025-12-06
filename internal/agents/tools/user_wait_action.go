package tools

import (
	"context"
	"encoding/json"

	"browser-agent/internal/domain/ports"
	"browser-agent/internal/infrastructure/browser/rodwrapper"
)

type UserWaitAction struct {
	page   *rodwrapper.Page
	logger ports.Logger
}

var _ Tool = (*UserWaitAction)(nil)

func NewUserWaitActionTool(page *rodwrapper.Page, logger ports.Logger) Tool {
	return &UserWaitAction{
		page:   page,
		logger: logger,
	}
}

func (t *UserWaitAction) Name() string {
	return "wait_action"
}

func (t *UserWaitAction) Type() string {
	return "user"
}

func (t *UserWaitAction) Description() string {
	return "Pauses automated execution and transfers control to the user. Use when human intervention is required (e.g., to solve CAPTCHA, complete authentication, or perform manual actions). The browser session remains open for user interaction."
}

func (t *UserWaitAction) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

func (t *UserWaitAction) Call(ctx context.Context, input string) (string, error) {
	t.logger.Logf("UserWaitAction: transferring control to user")

	type response struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	}

	resp := response{
		Message: "Browser control transferred to user. Please perform required actions manually.",
		Status:  "waiting_for_user",
	}

	jsonResp, _ := json.Marshal(resp)
	return string(jsonResp), nil
}
