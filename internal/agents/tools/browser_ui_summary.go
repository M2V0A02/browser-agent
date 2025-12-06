package tools

import (
	"browser-agent/internal/infrastructure/browser/rodwrapper"
	"context"
	"encoding/json"
	"fmt"
)

var _ Tool = (*BrowserUISummary)(nil)

type BrowserUISummary struct {
	page   *rodwrapper.Page
	logger Logger
}

type UIElement struct {
	ID        string `json:"id"`
	Type      string `json:"type"` // button, checkbox, link, menu, input
	Text      string `json:"text"`
	AriaLabel string `json:"aria_label,omitempty"`
	Role      string `json:"role,omitempty"`
	Visible   bool   `json:"visible"`
	Count     string `json:"count,omitempty"`
	Selector  string `json:"selector"`
}

func NewBrowserUISummaryTool(page *rodwrapper.Page, logger Logger) Tool {
	return &BrowserUISummary{page: page, logger: logger}
}

func (t *BrowserUISummary) Name() string {
	return "ui_summary"
}

func (t *BrowserUISummary) Type() string {
	return "browser"
}

func (t *BrowserUISummary) Description() string {
	return "Returns a structured list of all visible interactive UI elements on the page (buttons, checkboxes, links, inputs, etc.) as JSON array. Includes reliable CSS selectors for each element. Use this instead of browser_extract for quick interface understanding and getting ready-to-use selectors."
}

func (t *BrowserUISummary) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

func (t *BrowserUISummary) Call(ctx context.Context, input string) (string, error) {
	t.logger.Logf("Starting browser_ui_summary")

	cfg := &rodwrapper.ExtractConfig{
		OnlyInViewport: true,
		MaxElements:    400,
	}

	elements, err := rodwrapper.ExtractUI(t.page, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to extract UI elements: %w", err)
	}

	jsonData, err := json.MarshalIndent(elements, "", "  ")
	if err != nil {
		return "", fmt.Errorf("json marshal error: %w", err)
	}

	t.logger.Logf("ui_summary: found %d elements, %d bytes", len(elements), len(jsonData))
	return string(jsonData), nil
}
