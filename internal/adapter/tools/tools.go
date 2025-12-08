package tool

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"
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
	return "Click on page elements. Supports single click, batch clicking multiple elements, and observing changes after click. Use 'selector' for single click, 'selectors' array for batch operations (up to 50 elements). Set 'observe' to true to see what changed after clicking (new modals, buttons, URL changes). Batch clicks are executed sequentially without returning to LLM between clicks."
}
func (t *ClickTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"selector": map[string]interface{}{
				"type":        "string",
				"description": "CSS or XPath selector for single element click",
			},
			"selectors": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Array of CSS selectors for batch clicking (max 50). Example: [\"#checkbox1\", \"#checkbox2\", \"#checkbox3\"]",
				"maxItems":    50,
			},
			"observe": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, observe and return page changes after clicking (new elements, modals, URL changes). Only works with single selector.",
				"default":     false,
			},
		},
		"oneOf": []map[string]interface{}{
			{"required": []string{"selector"}},
			{"required": []string{"selectors"}},
		},
	}
}

func (t *ClickTool) Execute(ctx context.Context, args string) (string, error) {
	var input struct {
		Selector  string   `json:"selector"`
		Selectors []string `json:"selectors"`
		Observe   bool     `json:"observe"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", err
	}

	if len(input.Selectors) > 0 {
		if input.Observe {
			return "", fmt.Errorf("observe mode only works with single selector, not batch")
		}
		if len(input.Selectors) > 50 {
			return "", fmt.Errorf("too many selectors (max 50, got %d)", len(input.Selectors))
		}
		if err := t.browser.BatchClick(ctx, input.Selectors); err != nil {
			return "", err
		}
		return fmt.Sprintf("Successfully clicked %d elements", len(input.Selectors)), nil
	}

	if input.Selector == "" {
		return "", fmt.Errorf("either 'selector' or 'selectors' is required")
	}

	if input.Observe {
		result, err := t.browser.ClickWithChanges(ctx, input.Selector)
		if err != nil {
			return "", err
		}
		if !result.Success {
			return "", fmt.Errorf("click failed: %s", result.Error)
		}

		output := "Click successful"
		if result.Changes != nil {
			changes := result.Changes
			if changes.URLChanged {
				output += fmt.Sprintf("\n✓ URL changed to: %s", changes.NewURL)
			}
			if changes.ModalOpened {
				output += "\n✓ Modal/dialog opened"
			}
			if changes.ModalClosed {
				output += "\n✓ Modal/dialog closed"
			}
			if len(changes.NewElements) > 0 {
				output += fmt.Sprintf("\n✓ %d new elements appeared:", len(changes.NewElements))
				for i, el := range changes.NewElements {
					if i >= 10 {
						output += fmt.Sprintf("\n  ... and %d more", len(changes.NewElements)-10)
						break
					}
					label := el.Text
					if label == "" {
						label = el.AriaLabel
					}
					if label == "" {
						label = el.Type
					}
					output += fmt.Sprintf("\n  - [%s] %s (selector: %s)", el.Type, label, el.Selector)
				}
			}
			if changes.ElementsRemoved > 0 {
				output += fmt.Sprintf("\n✓ %d elements removed", changes.ElementsRemoved)
			}
		}
		return output, nil
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
	return "Fill text into form fields. Supports single field or batch filling multiple fields. Use 'selector' and 'text' for single field, or 'fields' object for batch operations (up to 20 fields). Clears existing content before filling. All batch fills are executed without returning to LLM between fields. For submitting forms, use press_enter after filling or click on submit button."
}
func (t *FillTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"selector": map[string]interface{}{
				"type":        "string",
				"description": "CSS selector for single input field",
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "Text to input into single field",
			},
			"fields": map[string]interface{}{
				"type":        "object",
				"description": "Map of CSS selectors to values for batch filling. Example: {\"#name\": \"John\", \"#email\": \"john@example.com\", \"#phone\": \"123-456-7890\"}",
				"maxProperties": 20,
			},
		},
		"oneOf": []map[string]interface{}{
			{"required": []string{"selector", "text"}},
			{"required": []string{"fields"}},
		},
	}
}

func (t *FillTool) Execute(ctx context.Context, args string) (string, error) {
	var input struct {
		Selector string            `json:"selector"`
		Text     string            `json:"text"`
		Fields   map[string]string `json:"fields"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", err
	}

	if len(input.Fields) > 0 {
		if len(input.Fields) > 20 {
			return "", fmt.Errorf("too many fields (max 20, got %d)", len(input.Fields))
		}
		if err := t.browser.BatchFill(ctx, input.Fields); err != nil {
			return "", err
		}
		return fmt.Sprintf("Successfully filled %d fields", len(input.Fields)), nil
	}

	if input.Selector == "" || input.Text == "" {
		return "", fmt.Errorf("either ('selector' and 'text') or 'fields' is required")
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
	return "Ask the user a question and wait for their response. Use this for gathering preferences, clarifications, or non-sensitive information. Authentication and credentials are handled automatically by the system - do not request them through this tool. Returns the user's text response."
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

type ObserveTool struct {
	browser output.BrowserPort
	logger  output.LoggerPort
}

func NewObserveTool(browser output.BrowserPort, logger output.LoggerPort) *ObserveTool {
	return &ObserveTool{browser: browser, logger: logger}
}

func (t *ObserveTool) Name() string { return "observe" }
func (t *ObserveTool) Description() string {
	return "Observe the current state of the page to understand what you are looking at. Returns comprehensive information: current URL, page title, all visible interactive elements (buttons, links, inputs) with their text and selectors, and a preview of the page's text content. Use this tool when you need to understand the page context, after navigation, after scrolling, or when you're unsure what elements are available. This is your primary tool for situational awareness - it tells you what you can see and interact with on the current page."
}
func (t *ObserveTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

func (t *ObserveTool) Execute(ctx context.Context, args string) (string, error) {
	pageCtx, err := t.browser.GetPageContext(ctx)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf(`PAGE OBSERVATION:

URL: %s
Title: %s
Visible Elements: %d elements found

INTERACTIVE ELEMENTS:
`, pageCtx.URL, pageCtx.Title, pageCtx.ElementCount)

	for _, el := range pageCtx.VisibleElements {
		label := el.Text
		if label == "" && el.AriaLabel != "" {
			label = el.AriaLabel
		}
		if label == "" {
			label = "(no text)"
		}
		result += fmt.Sprintf("- [%s] %s: \"%s\" (selector: %s)\n", el.ID, el.Type, label, el.Selector)
	}

	result += fmt.Sprintf("\nPAGE CONTENT PREVIEW:\n%s\n", pageCtx.TextContent)

	return result, nil
}

type QueryElementsTool struct {
	browser output.BrowserPort
	logger  output.LoggerPort
}

func NewQueryElementsTool(browser output.BrowserPort, logger output.LoggerPort) *QueryElementsTool {
	return &QueryElementsTool{browser: browser, logger: logger}
}

func (t *QueryElementsTool) Name() string { return "query_elements" }
func (t *QueryElementsTool) Description() string {
	return "Extract structured data from repeated elements using exact CSS selector. Extracts ALL needed data from multiple elements including nested element selectors for later clicks. Perfect for emails, products, news items when you know the exact selector. Use 'search' tool first if you need to find elements by pattern. Returns compact text format. For multiple selectors, call this tool multiple times in parallel."
}
func (t *QueryElementsTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"selector": map[string]interface{}{
				"type":        "string",
				"description": "Exact CSS selector for target elements. Example: '.mail-item', 'tr.email', 'div[data-message]'",
			},
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "Maximum elements to return (default: 20, max: 100)",
			},
			"extract": map[string]interface{}{
				"type":        "object",
				"description": "Map of sub-selectors to extraction types. Types: 'text' (innerText), 'html' (innerHTML), 'selector' (returns selector for later click!), 'attr:name' (attribute). Use '_self' for main element. Example: {'.sender': 'text', '.subject': 'text', 'button.delete': 'selector', '_self': 'attr:data-id'}",
			},
		},
		"required": []string{"selector", "extract"},
	}
}

func (t *QueryElementsTool) Execute(ctx context.Context, args string) (string, error) {
	var req entity.QueryElementsRequest
	if err := json.Unmarshal([]byte(args), &req); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if req.Selector == "" {
		return "", fmt.Errorf("selector is required")
	}

	if len(req.Extract) == 0 {
		return "", fmt.Errorf("extract map is required and must not be empty")
	}

	result, err := t.browser.QueryElements(ctx, req)
	if err != nil {
		return "", err
	}

	return formatQueryResult(result), nil
}

func formatQueryResult(result *entity.QueryElementsResult) string {
	if result.Count == 0 {
		return "No elements found"
	}

	output := fmt.Sprintf("Found %d elements:\n\n", result.Count)

	for i, elem := range result.Elements {
		output += fmt.Sprintf("#%d [%s]\n", i+1, elem.Selector)

		for key, value := range elem.Data {
			if value != "" {
				output += fmt.Sprintf("  %s: %q\n", key, value)
			}
		}

		output += "\n"
	}

	return output
}

type SearchTool struct {
	browser output.BrowserPort
	logger  output.LoggerPort
}

func NewSearchTool(browser output.BrowserPort, logger output.LoggerPort) *SearchTool {
	return &SearchTool{browser: browser, logger: logger}
}

func (t *SearchTool) Name() string { return "search" }
func (t *SearchTool) Description() string {
	return "Search for information on the page with minimal output. Three search types: 1) 'text' - finds text content and returns up to 1000 characters with context; 2) 'id' - finds elements by id (or partial id match) and returns their attributes without children; 3) 'attribute' - finds elements by attribute name/value and returns element with selector. Use this for simple searches when you need concise results."
}
func (t *SearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"text", "id", "attribute"},
				"description": "Search type: 'text' (search for text content), 'id' (search by element id), 'attribute' (search by attribute)",
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query. For 'text' - text to find. For 'id' - id or partial id. For 'attribute' - attribute name or 'name=value' format",
			},
		},
		"required": []string{"type", "query"},
	}
}

func (t *SearchTool) Execute(ctx context.Context, args string) (string, error) {
	var req entity.SearchRequest
	if err := json.Unmarshal([]byte(args), &req); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if req.Type == "" {
		return "", fmt.Errorf("type is required")
	}

	if req.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	result, err := t.browser.Search(ctx, req)
	if err != nil {
		return "", err
	}

	return formatSearchResult(result), nil
}

func formatSearchResult(result *entity.SearchResult) string {
	if !result.Found {
		return fmt.Sprintf("No results found for %s search", result.Type)
	}

	switch result.Type {
	case "text":
		return fmt.Sprintf("Found text:\n\n%s", result.Content)

	case "id", "attribute":
		if len(result.Elements) == 0 {
			return fmt.Sprintf("No elements found for %s search", result.Type)
		}

		output := fmt.Sprintf("Found %d element(s):\n\n", len(result.Elements))
		for i, elem := range result.Elements {
			output += fmt.Sprintf("#%d [%s]\n", i+1, elem.Selector)
			if elem.ID != "" {
				output += fmt.Sprintf("  id: %q\n", elem.ID)
			}
			if len(elem.Attributes) > 0 {
				output += "  attributes:\n"
				for k, v := range elem.Attributes {
					if v != "" {
						output += fmt.Sprintf("    %s: %q\n", k, v)
					}
				}
			}
			output += "\n"
		}
		return output

	default:
		return "Unknown search type"
	}
}

