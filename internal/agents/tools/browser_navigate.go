package tools

import (
	"browser-agent/internal/infrastructure/browser/rodwrapper"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

var _ Tool = (*BrowserNavigate)(nil)

type BrowserNavigate struct {
	page   *rodwrapper.Page
	logger Logger
}

type Logger interface {
	Logf(format string, args ...any)
}

func NewBrowserNavigateTool(page *rodwrapper.Page, logger Logger) Tool {
	return &BrowserNavigate{
		page:   page,
		logger: logger,
	}
}

func (t *BrowserNavigate) Name() string {
	return "navigate"
}

func (t *BrowserNavigate) Type() string {
	return "browser"
}

type navigateInput struct {
	URL string `json:"url"`
}

func (t *BrowserNavigate) Description() string {
	return "Navigates the browser to the specified URL and waits until the page is fully loaded and idle."
}

func (t *BrowserNavigate) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "Full URL to navigate to. Must include protocol (https:// or http://).",
			},
		},
		"required": []string{"url"},
	}
}

func (t *BrowserNavigate) Call(ctx context.Context, input string) (string, error) {
	var args navigateInput
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}
	if args.URL == "" {
		return "", fmt.Errorf("url parameter is required")
	}

	t.logger.Logf("Navigating to: %s", args.URL)

	if err := t.page.Navigate(args.URL); err != nil {
		return "", fmt.Errorf("navigation failed: %w", err)
	}

	t.page.MustWaitLoad()
	t.page.WaitIdle(5 * time.Second)

	currentURL := t.page.MustInfo().URL
	t.logger.Logf("Successfully loaded: %s", currentURL)

	return fmt.Sprintf("Successfully navigated to %s", currentURL), nil
}
