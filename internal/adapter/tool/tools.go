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

func (t *NavigateTool) Name() string { return "navigate" }
func (t *NavigateTool) Description() string {
	return "Navigate browser to a URL. Use this to open web pages or follow links. Accepts full URLs (https://example.com) or partial URLs. Returns the final URL after navigation (may differ due to redirects). Use this as the first step when starting a new task or when you need to go to a different website."
}
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

func (t *ClickTool) Name() string { return "click" }
func (t *ClickTool) Description() string {
	return "Click on a page element using CSS selector or XPath. Use this to interact with buttons, links, checkboxes, or any clickable element. First use ui_summary to get available selectors. Supports both CSS selectors (e.g., '#submit-btn', '.menu-item') and XPath (e.g., '//button[text()=\"Submit\"]'). Returns success message or error if element not found."
}
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

func (t *FillTool) Name() string { return "fill" }
func (t *FillTool) Description() string {
	return "Fill text into an input field or textarea using CSS selector. Use this to enter data into forms, search boxes, or text areas. First use ui_summary to find the correct selector. Clears existing content before filling. For submitting forms, use press_enter after filling or click on submit button."
}
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

func (t *ScrollTool) Name() string { return "scroll" }
func (t *ScrollTool) Description() string {
	return "Scroll the page in specified direction. Directions: 'up' (scroll up one viewport), 'down' (scroll down one viewport), 'top' (scroll to page top), 'bottom' (scroll to page bottom). Use this to reveal content below the fold, navigate long pages, or before taking screenshots of different page sections. After scrolling, use ui_summary to see newly visible elements."
}
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

func (t *ScreenshotTool) Name() string { return "screenshot" }
func (t *ScreenshotTool) Description() string {
	return "Capture a screenshot of the current visible viewport. Returns base64-encoded image data URL. Use this when you need visual confirmation of page state, to verify UI appearance, or to show results to user. The screenshot only captures the visible portion - use scroll to capture different sections. Useful after navigation or interactions to confirm success."
}
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

func (t *ExtractTool) Name() string { return "extract" }
func (t *ExtractTool) Description() string {
	return "Extract the text content of the current page. Returns clean text without HTML tags, scripts, or styles. Use this when you need to read page content, extract specific data, or analyze text information. For interactive elements info, use ui_summary instead. For visual verification, use screenshot. This returns only visible text content."
}
func (t *ExtractTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

func (t *ExtractTool) Execute(ctx context.Context, args string) (string, error) {
	text, err := t.browser.GetPageText(ctx)
	if err != nil {
		return "", err
	}
	return text, nil
}

type UISummaryTool struct {
	browser output.BrowserPort
	logger  output.LoggerPort
}

func NewUISummaryTool(browser output.BrowserPort, logger output.LoggerPort) *UISummaryTool {
	return &UISummaryTool{browser: browser, logger: logger}
}

func (t *UISummaryTool) Name() string { return "ui_summary" }
func (t *UISummaryTool) Description() string {
	return "Get structured list of interactive UI elements on current page. Returns JSON array with buttons, links, inputs, and their selectors. Use this BEFORE click or fill operations to find correct selectors. Shows element type, text content, and CSS selector. This is your primary tool for understanding what elements are available to interact with. Call after navigation or scroll to see available options."
}
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
	data, err := json.Marshal(elements)
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

func (t *PressEnterTool) Name() string { return "press_enter" }
func (t *PressEnterTool) Description() string {
	return "Press the Enter key on the keyboard. Use this to submit forms after filling input fields, trigger search actions, or confirm inputs. Common workflow: fill input field with search query, then press_enter to submit. Alternative to clicking submit buttons when Enter key submission is supported."
}
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

type AskQuestionTool struct {
	userInteraction output.UserInteractionPort
	logger          output.LoggerPort
}

func NewAskQuestionTool(userInteraction output.UserInteractionPort, logger output.LoggerPort) *AskQuestionTool {
	return &AskQuestionTool{userInteraction: userInteraction, logger: logger}
}

func (t *AskQuestionTool) Name() string { return "ask_question" }
func (t *AskQuestionTool) Description() string {
	return "Ask the user a question and wait for their response. Use this when you need information from the user to proceed with the task, such as credentials, preferences, or clarifications. The tool will pause execution until the user provides an answer. Returns the user's text response."
}
func (t *AskQuestionTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"question": map[string]interface{}{
				"type":        "string",
				"description": "The question to ask the user",
			},
		},
		"required": []string{"question"},
	}
}

func (t *AskQuestionTool) Execute(ctx context.Context, args string) (string, error) {
	var input struct {
		Question string `json:"question"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", err
	}
	answer, err := t.userInteraction.AskQuestion(ctx, input.Question)
	if err != nil {
		return "", err
	}
	return answer, nil
}

type WaitUserActionTool struct {
	userInteraction output.UserInteractionPort
	logger          output.LoggerPort
}

func NewWaitUserActionTool(userInteraction output.UserInteractionPort, logger output.LoggerPort) *WaitUserActionTool {
	return &WaitUserActionTool{userInteraction: userInteraction, logger: logger}
}

func (t *WaitUserActionTool) Name() string { return "wait_user_action" }
func (t *WaitUserActionTool) Description() string {
	return "Pause execution and wait for the user to complete a manual action in the browser. Use this when you need the user to handle tasks that cannot be automated, such as solving CAPTCHA challenges, completing 2FA authentication, manual login steps, or any other human verification. Provide a clear message explaining what action the user needs to perform. The tool will wait until the user presses Enter to confirm completion."
}
func (t *WaitUserActionTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Instructions for the user explaining what action they need to perform",
			},
		},
		"required": []string{"message"},
	}
}

func (t *WaitUserActionTool) Execute(ctx context.Context, args string) (string, error) {
	var input struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", err
	}
	if err := t.userInteraction.WaitForUserAction(ctx, input.Message); err != nil {
		return "", err
	}
	return "User confirmed action completion", nil
}
