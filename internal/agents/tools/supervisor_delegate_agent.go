package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"browser-agent/internal/domain/ports"
	"browser-agent/internal/infrastructure/browser/rodwrapper"
)

type SupervisorDelegateAgent struct {
	page   *rodwrapper.Page
	logger ports.Logger
}

var _ Tool = (*SupervisorDelegateAgent)(nil)

func NewSupervisorDelegateAgentTool(page *rodwrapper.Page, logger ports.Logger) Tool {
	return &SupervisorDelegateAgent{
		page:   page,
		logger: logger,
	}
}

func (t *SupervisorDelegateAgent) Name() string {
	return "delegate_agent"
}

func (t *SupervisorDelegateAgent) Type() string {
	return "supervisor"
}

func (t *SupervisorDelegateAgent) Description() string {
	return "Передаёт управление другому специализированному агенту (например, 'login_agent', 'search_agent', 'checkout_agent' и т.д.). Используется супервизором для делегирования подзадач. Вход — JSON с полем 'agent' (имя агента) и опциональным 'task' (новое описание задачи)."
}

func (t *SupervisorDelegateAgent) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agent": map[string]interface{}{
				"type":        "string",
				"description": "Имя агента, которому делегируется задача (например: login_agent, search_agent)",
			},
			"task": map[string]interface{}{
				"type":        "string",
				"description": "Новое или уточнённое описание задачи для делегируемого агента",
				"example":     "Войди в аккаунт используя логин user@example.com",
			},
		},
		"required": []string{"agent"},
	}
}

type delegateInput struct {
	Agent string `json:"agent"`
	Task  string `json:"task"`
}

func (t *SupervisorDelegateAgent) Call(ctx context.Context, input string) (string, error) {
	var args delegateInput
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return "", fmt.Errorf("invalid JSON input: %w", err)
	}

	if args.Agent == "" {
		return "", fmt.Errorf("agent name is required")
	}

	t.logger.Logf("Supervisor делегирует задачу агенту: %s", args.Agent)
	if args.Task != "" {
		t.logger.Logf("Новое задание: %s", args.Task)
	}

	// Специальная ошибка, которую ловит Supervisor и переключает агента
	return "", nil
}
