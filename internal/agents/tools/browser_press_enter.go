package tools

import (
	"browser-agent/internal/domain/ports"
	"browser-agent/internal/infrastructure/browser/rodwrapper"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

type BrowserPressEnter struct {
	page   *rodwrapper.Page
	logger ports.Logger
}

var _ Tool = (*BrowserPressEnter)(nil)

func NewBrowserPressEnterTool(page *rodwrapper.Page, logger ports.Logger) Tool {
	return &BrowserPressEnter{
		page:   page,
		logger: logger,
	}
}

func (t *BrowserPressEnter) Name() string {
	return "press_enter"
}

func (t *BrowserPressEnter) Type() string {
	return "browser"
}

type pressEnterInput struct {
	Selector string `json:"selector,omitempty"`
}

func (t *BrowserPressEnter) Description() string {
	return "Simulates pressing the Enter key. Can target a specific focused element or press Enter globally. Useful for submitting forms, confirming dialogs, or triggering search."
}

func (t *BrowserPressEnter) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"selector": map[string]interface{}{
				"type":        "string",
				"description": `CSS selector for the element to press Enter on (e.g., "input[type='text']"). If not provided, presses Enter on the currently focused element or body.`,
			},
		},
		"required": []string{},
	}
}

func (t *BrowserPressEnter) Call(ctx context.Context, input string) (string, error) {
	var params pressEnterInput

	if input != "" {
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			selector := strings.TrimSpace(input)
			if selector != "" {
				params.Selector = selector
			}
		}
	}

	var el *rod.Element
	var err error

	if params.Selector != "" {
		t.logger.Logf("Pressing Enter on element: %s", params.Selector)
		el, err = t.page.Element(params.Selector)
		if err != nil {
			return "", fmt.Errorf("element not found: %s: %w", params.Selector, err)
		}
	} else {
		t.logger.Logf("Pressing Enter on body (global)")
		el = t.page.MustElement("body")
	}

	if err := el.Input("E007"); err != nil {
		if err := el.Input("\r"); err != nil {
			return "", fmt.Errorf("failed to press Enter: %w", err)
		}
	}

	t.page.WaitIdle(1 * time.Second)

	action := "Enter key pressed"
	if params.Selector != "" {
		action = fmt.Sprintf("Enter key pressed on element: %s", params.Selector)
	}

	t.logger.Logf("TOOL %s completed: %s", t.Name(), action)
	return action, nil
}
