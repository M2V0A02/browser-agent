package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"browser-agent/internal/domain/ports"
	"browser-agent/internal/infrastructure/browser/rodwrapper"
)

type BrowserFill struct {
	page   *rodwrapper.Page
	logger ports.Logger
}

var _ Tool = (*BrowserFill)(nil)

func NewBrowserFillTool(page *rodwrapper.Page, logger ports.Logger) Tool {
	return &BrowserFill{
		page:   page,
		logger: logger,
	}
}

func (t *BrowserFill) Name() string {
	return "fill"
}

func (t *BrowserFill) Type() string {
	return "browser"
}

type fillInput struct {
	Selector string `json:"selector"`
	Text     string `json:"text"`
}

func (t *BrowserFill) Description() string {
	return "Fills an input field, textarea, or content-editable element with the provided text. Automatically clears any existing content."
}

func (t *BrowserFill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"selector": map[string]interface{}{
				"type":        "string",
				"description": `CSS selector (e.g. "input[type='text']") or XPath (e.g. "//input[@placeholder='Search']"). Use XPath if selector starts with / or contains 'xpath='.`,
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "The text to input into the field. Special keys are not supported (use browser_press for key events).",
			},
		},
		"required": []string{"selector", "text"},
	}
}

func (t *BrowserFill) Call(ctx context.Context, input string) (string, error) {
	var params fillInput

	if err := json.Unmarshal([]byte(input), &params); err != nil {
		// Fallback to legacy format 'selector | text'
		parts := strings.SplitN(input, "|", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid input format: expected JSON or 'selector | text'")
		}
		params.Selector = strings.TrimSpace(parts[0])
		params.Text = strings.TrimSpace(parts[1])
	}

	if params.Selector == "" {
		return "", fmt.Errorf("selector is required")
	}

	el, err := t.page.Element(params.Selector)
	if err != nil {
		return "", fmt.Errorf("field not found: %s", params.Selector)
	}

	// Clear existing value first
	if err := el.SelectAllText(); err == nil {
		if err := el.Input(""); err != nil {
			return "", fmt.Errorf("failed to clear field: %w", err)
		}
	}

	if err := el.Input(params.Text); err != nil {
		return "", fmt.Errorf("input failed: %w", err)
	}

	return fmt.Sprintf("text '%s' entered into field: %s", params.Text, params.Selector), nil
}
