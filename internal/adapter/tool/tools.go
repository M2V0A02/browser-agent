package tool

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"browser-agent/internal/application/port/output"
)

type NavigateTool struct {
	browser output.BrowserPort
	logger  output.LoggerPort
}

func NewNavigateTool(browser output.BrowserPort, logger output.LoggerPort) *NavigateTool {
	return &NavigateTool{browser: browser, logger: logger}
}

func (t *NavigateTool) Name() string        { return "navigate" }
func (t *NavigateTool) Description() string { return "Navigates browser to URL" }
func (t *NavigateTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to navigate to",
			},
		},
		"required": []string{"url"},
	}
}

func (t *NavigateTool) Execute(ctx context.Context, args string) (string, error) {
	var input struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", err
	}
	if err := t.browser.Navigate(ctx, input.URL); err != nil {
		return "", err
	}
	return fmt.Sprintf("Navigated to %s", t.browser.CurrentURL()), nil
}

type ClickTool struct {
	browser output.BrowserPort
	logger  output.LoggerPort
}

func NewClickTool(browser output.BrowserPort, logger output.LoggerPort) *ClickTool {
	return &ClickTool{browser: browser, logger: logger}
}

func (t *ClickTool) Name() string        { return "click" }
func (t *ClickTool) Description() string { return "Clicks element by selector" }
func (t *ClickTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"selector": map[string]interface{}{
				"type":        "string",
				"description": "CSS or XPath selector",
			},
		},
		"required": []string{"selector"},
	}
}

func (t *ClickTool) Execute(ctx context.Context, args string) (string, error) {
	var input struct {
		Selector string `json:"selector"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", err
	}
	if err := t.browser.Click(ctx, input.Selector); err != nil {
		return "", err
	}
	return "Click successful", nil
}

type FillTool struct {
	browser output.BrowserPort
	logger  output.LoggerPort
}

func NewFillTool(browser output.BrowserPort, logger output.LoggerPort) *FillTool {
	return &FillTool{browser: browser, logger: logger}
}

func (t *FillTool) Name() string        { return "fill" }
func (t *FillTool) Description() string { return "Fills input field with text" }
func (t *FillTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"selector": map[string]interface{}{
				"type":        "string",
				"description": "CSS selector for input",
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "Text to input",
			},
		},
		"required": []string{"selector", "text"},
	}
}

func (t *FillTool) Execute(ctx context.Context, args string) (string, error) {
	var input struct {
		Selector string `json:"selector"`
		Text     string `json:"text"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", err
	}
	if err := t.browser.Fill(ctx, input.Selector, input.Text); err != nil {
		return "", err
	}
	return fmt.Sprintf("Filled '%s' with text", input.Selector), nil
}

type ScrollTool struct {
	browser output.BrowserPort
	logger  output.LoggerPort
}

func NewScrollTool(browser output.BrowserPort, logger output.LoggerPort) *ScrollTool {
	return &ScrollTool{browser: browser, logger: logger}
}

func (t *ScrollTool) Name() string        { return "scroll" }
func (t *ScrollTool) Description() string { return "Scrolls page in direction" }
func (t *ScrollTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"direction": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"up", "down", "top", "bottom"},
				"description": "Scroll direction",
			},
		},
		"required": []string{"direction"},
	}
}

func (t *ScrollTool) Execute(ctx context.Context, args string) (string, error) {
	var input struct {
		Direction string `json:"direction"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", err
	}
	if err := t.browser.Scroll(ctx, input.Direction, 0); err != nil {
		return "", err
	}
	return fmt.Sprintf("Scrolled %s", input.Direction), nil
}

type ScreenshotTool struct {
	browser output.BrowserPort
	logger  output.LoggerPort
}

func NewScreenshotTool(browser output.BrowserPort, logger output.LoggerPort) *ScreenshotTool {
	return &ScreenshotTool{browser: browser, logger: logger}
}

func (t *ScreenshotTool) Name() string        { return "screenshot" }
func (t *ScreenshotTool) Description() string { return "Takes screenshot of page" }
func (t *ScreenshotTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

func (t *ScreenshotTool) Execute(ctx context.Context, args string) (string, error) {
	screenshot, err := t.browser.Screenshot(ctx)
	if err != nil {
		return "", err
	}
	b64 := base64.StdEncoding.EncodeToString(screenshot.Data)
	return fmt.Sprintf("data:image/%s;base64,%s", screenshot.Format, b64), nil
}

type ExtractTool struct {
	browser output.BrowserPort
	logger  output.LoggerPort
}

func NewExtractTool(browser output.BrowserPort, logger output.LoggerPort) *ExtractTool {
	return &ExtractTool{browser: browser, logger: logger}
}

func (t *ExtractTool) Name() string        { return "extract" }
func (t *ExtractTool) Description() string { return "Extracts text from page" }
func (t *ExtractTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

func (t *ExtractTool) Execute(ctx context.Context, args string) (string, error) {
	content, err := t.browser.GetPageContent(ctx)
	if err != nil {
		return "", err
	}
	return content.HTML, nil
}

type UISummaryTool struct {
	browser output.BrowserPort
	logger  output.LoggerPort
}

func NewUISummaryTool(browser output.BrowserPort, logger output.LoggerPort) *UISummaryTool {
	return &UISummaryTool{browser: browser, logger: logger}
}

func (t *UISummaryTool) Name() string        { return "ui_summary" }
func (t *UISummaryTool) Description() string { return "Returns list of UI elements" }
func (t *UISummaryTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

func (t *UISummaryTool) Execute(ctx context.Context, args string) (string, error) {
	elements, err := t.browser.GetUIElements(ctx)
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(elements, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type PressEnterTool struct {
	browser output.BrowserPort
	logger  output.LoggerPort
}

func NewPressEnterTool(browser output.BrowserPort, logger output.LoggerPort) *PressEnterTool {
	return &PressEnterTool{browser: browser, logger: logger}
}

func (t *PressEnterTool) Name() string        { return "press_enter" }
func (t *PressEnterTool) Description() string { return "Presses Enter key" }
func (t *PressEnterTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

func (t *PressEnterTool) Execute(ctx context.Context, args string) (string, error) {
	if err := t.browser.PressEnter(ctx); err != nil {
		return "", err
	}
	return "Enter pressed", nil
}
