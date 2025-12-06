package tools

import (
	"browser-agent/internal/infrastructure/browser/rodwrapper"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

var _ Tool = (*BrowserClick)(nil)

type BrowserClick struct {
	page   *rodwrapper.Page
	logger Logger
}

func NewBrowserClickTool(page *rodwrapper.Page, logger Logger) Tool {
	return &BrowserClick{
		page:   page,
		logger: logger,
	}
}

func (t *BrowserClick) Name() string {
	return "click"
}

func (t *BrowserClick) Type() string {
	return "browser"
}

type clickInput struct {
	Selector string `json:"selector"`
}

func (t *BrowserClick) Description() string {
	return "Clicks the first visible element matched by the provided CSS or XPath selector. Automatically scrolls the element into view."
}

func (t *BrowserClick) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"selector": map[string]interface{}{
				"type":        "string",
				"description": `CSS selector (e.g. "button#submit") or XPath (e.g. "//button[text()='Submit']"). Use XPath if the selector starts with / or contains 'xpath='.`,
			},
		},
		"required": []string{"selector"},
	}
}

func (t *BrowserClick) Call(ctx context.Context, input string) (string, error) {
	t.logger.Logf("browser_click: %s", input)

	var el *rod.Element
	var err error

	var args clickInput
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	selector := strings.TrimSpace(args.Selector)
	if strings.HasPrefix(selector, "/") || strings.Contains(selector, "xpath") {
		el, err = t.page.ElementX(selector)
	} else {
		el, err = t.page.Element(selector)
	}
	if err != nil {
		return "", fmt.Errorf("элемент не найден: %s", selector)
	}

	if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return "", fmt.Errorf("клик не удался: %w", err)
	}

	t.page.WaitIdle(2 * time.Second)
	return "клик выполнен успешно", nil
}
