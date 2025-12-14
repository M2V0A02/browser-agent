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

func (t *NavigateTool) Name() entity.ToolName { return entity.ToolBrowserNavigate }
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

func (t *ClickTool) Name() entity.ToolName { return entity.ToolBrowserClick }
func (t *ClickTool) Description() string {
	return "Click on page elements. Supports single click, batch clicking multiple elements, and observing changes after click. Use 'selectors' array with one element for single click, or multiple elements for batch operations (up to 50 elements). Set 'observe' to true to see what changed after clicking (new modals, buttons, URL changes) - only works with single element. Batch clicks are executed sequentially without returning to LLM between clicks."
}
func (t *ClickTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"selectors": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Array of CSS selectors to click (max 50). For single click use array with one element. Example: [\"#button1\"] or [\"#checkbox1\", \"#checkbox2\", \"#checkbox3\"]",
				"maxItems":    50,
			},
			"observe": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, observe and return page changes after clicking (new elements, modals, URL changes). Only works with single element in selectors array.",
				"default":     false,
			},
		},
		"required": []string{"selectors"},
	}
}

func (t *ClickTool) Execute(ctx context.Context, args string) (string, error) {
	var input struct {
		Selectors []string `json:"selectors"`
		Observe   bool     `json:"observe"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", err
	}

	if len(input.Selectors) == 0 {
		return "", fmt.Errorf("selectors array is required and must not be empty")
	}

	if len(input.Selectors) > 50 {
		return "", fmt.Errorf("too many selectors (max 50, got %d)", len(input.Selectors))
	}

	if input.Observe && len(input.Selectors) > 1 {
		return "", fmt.Errorf("observe mode only works with single element, not batch")
	}

	if input.Observe {
		result, err := t.browser.ClickWithChanges(ctx, input.Selectors[0])
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

	if len(input.Selectors) == 1 {
		if err := t.browser.Click(ctx, input.Selectors[0]); err != nil {
			return "", err
		}
		return "Click successful", nil
	}

	if err := t.browser.BatchClick(ctx, input.Selectors); err != nil {
		return "", err
	}
	return fmt.Sprintf("Successfully clicked %d elements", len(input.Selectors)), nil
}

type FillTool struct {
	browser output.BrowserPort
	logger  output.LoggerPort
}

func NewFillTool(browser output.BrowserPort, logger output.LoggerPort) *FillTool {
	return &FillTool{browser: browser, logger: logger}
}

func (t *FillTool) Name() entity.ToolName { return entity.ToolBrowserFill }
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
				"type":          "object",
				"description":   "Map of CSS selectors to values for batch filling. Example: {\"#name\": \"John\", \"#email\": \"john@example.com\", \"#phone\": \"123-456-7890\"}",
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

func (t *ScrollTool) Name() entity.ToolName { return entity.ToolBrowserScroll }
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

func (t *ScreenshotTool) Name() entity.ToolName { return entity.ToolBrowserScreenshot }
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

type PressEnterTool struct {
	browser output.BrowserPort
	logger  output.LoggerPort
}

func NewPressEnterTool(browser output.BrowserPort, logger output.LoggerPort) *PressEnterTool {
	return &PressEnterTool{browser: browser, logger: logger}
}

func (t *PressEnterTool) Name() entity.ToolName { return entity.ToolBrowserPressEnter }
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

func (t *AskQuestionTool) Name() entity.ToolName { return entity.ToolUserAskQuestion }
func (t *AskQuestionTool) Description() string {
	return "Always use this tool if you don't have enough context to work effectively."
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

func (t *WaitUserActionTool) Name() entity.ToolName { return entity.ToolUserWaitAction }
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

func (t *ObserveTool) Name() entity.ToolName { return entity.ToolBrowserObserve }
func (t *ObserveTool) Description() string {
	return "Observe the current state of the page. Three modes: 1) 'interactive' (default) - shows interactive elements (buttons, links, inputs); 2) 'structure' - shows semantic page structure (sections, headers, key divs with IDs) - USE THIS to understand page layout and find element selectors; 3) 'full' - combines both. Use 'structure' mode when you need to find selectors for content blocks, articles, or specific page sections."
}
func (t *ObserveTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"mode": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"interactive", "structure", "full"},
				"description": "Observation mode: 'interactive' for buttons/links/inputs, 'structure' for page layout and content sections, 'full' for both",
				"default":     "structure",
			},
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "Maximum elements to return (default: 50)",
				"default":     50,
			},
		},
		"required": []string{},
	}
}

func (t *ObserveTool) Execute(ctx context.Context, args string) (string, error) {
	var input struct {
		Mode  string  `json:"mode"`
		Limit float64 `json:"limit"`
	}

	// Set defaults
	input.Mode = "structure"
	input.Limit = 50

	if args != "" && args != "{}" {
		if err := json.Unmarshal([]byte(args), &input); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}

	if input.Mode == "" {
		input.Mode = "structure"
	}

	switch input.Mode {
	case "interactive":
		return t.observeInteractive(ctx)
	case "structure":
		return t.observeStructure(ctx, int(input.Limit))
	case "full":
		interactive, _ := t.observeInteractive(ctx)
		structure, _ := t.observeStructure(ctx, int(input.Limit))
		return interactive + "\n\n" + structure, nil
	default:
		return "", fmt.Errorf("invalid mode: %s (must be 'interactive', 'structure', or 'full')", input.Mode)
	}
}

func (t *ObserveTool) observeInteractive(ctx context.Context) (string, error) {
	pageCtx, err := t.browser.GetPageContext(ctx)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf(`PAGE OBSERVATION (Interactive Mode):

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

func (t *ObserveTool) observeStructure(ctx context.Context, limit int) (string, error) {
	structure, err := t.browser.GetPageStructure(ctx)
	if err != nil {
		return "", err
	}

	if limit <= 0 {
		limit = 50
	}

	result := fmt.Sprintf(`PAGE STRUCTURE:

URL: %s
Title: %s

SEMANTIC STRUCTURE:
`, structure.URL, structure.Title)

	count := 0
	for _, el := range structure.Elements {
		if count >= limit {
			result += fmt.Sprintf("\n... and %d more elements (use limit parameter to see more)\n", len(structure.Elements)-limit)
			break
		}

		// Create indentation based on level
		indent := ""
		for i := 0; i < el.Level; i++ {
			indent += "  "
		}

		// Format element
		elementStr := fmt.Sprintf("%s<%s>", indent, el.TagName)

		// Add ID if present
		if el.ID != "" {
			elementStr += fmt.Sprintf(" #%s", el.ID)
		}

		// Add key classes (max 2)
		if len(el.Classes) > 0 {
			classStr := ""
			maxClasses := 2
			if len(el.Classes) < maxClasses {
				maxClasses = len(el.Classes)
			}
			for i := 0; i < maxClasses; i++ {
				classStr += "." + el.Classes[i]
			}
			elementStr += classStr
		}

		// Add text preview if present
		if el.Text != "" {
			textPreview := el.Text
			if len(textPreview) > 50 {
				textPreview = textPreview[:50] + "..."
			}
			elementStr += fmt.Sprintf(": \"%s\"", textPreview)
		}

		// Add selector
		elementStr += fmt.Sprintf(" [%s]", el.Selector)

		result += elementStr + "\n"
		count++
	}

	result += "\nKEY SELECTORS (use these in search/query/click/fill tools):\n"
	selectorCount := 0
	for _, el := range structure.Elements {
		if selectorCount >= 10 {
			break
		}
		if el.ID != "" || (len(el.Classes) > 0 && (el.TagName == "section" || el.TagName == "div" || el.TagName == "main" || el.TagName == "article")) {
			desc := el.TagName
			if el.Text != "" {
				textPreview := el.Text
				if len(textPreview) > 30 {
					textPreview = textPreview[:30] + "..."
				}
				desc += fmt.Sprintf(": \"%s\"", textPreview)
			}
			result += fmt.Sprintf("- %s → %s\n", desc, el.Selector)
			selectorCount++
		}
	}

	return result, nil
}

type QueryElementsTool struct {
	browser output.BrowserPort
	logger  output.LoggerPort
}

func NewQueryElementsTool(browser output.BrowserPort, logger output.LoggerPort) *QueryElementsTool {
	return &QueryElementsTool{browser: browser, logger: logger}
}

func (t *QueryElementsTool) Name() entity.ToolName { return entity.ToolBrowserQueryElements }
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
	var input struct {
		Selector string            `json:"selector"`
		Limit    float64           `json:"limit"`
		Extract  map[string]string `json:"extract"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if input.Selector == "" {
		return "", fmt.Errorf("selector is required")
	}

	if len(input.Extract) == 0 {
		return "", fmt.Errorf("extract map is required and must not be empty")
	}

	req := entity.QueryElementsRequest{
		Selector: input.Selector,
		Limit:    int(input.Limit),
		Extract:  input.Extract,
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

func (t *SearchTool) Name() entity.ToolName { return entity.ToolBrowserSearch }
func (t *SearchTool) Description() string {
	return "Search for elements on the page. ALWAYS returns selectors for found elements. Four search types: 1) 'text' - exact text match, returns elements with selector and parent info; 2) 'contains' - partial text match (e.g., 'Избранная' finds 'Избранная статья'); 3) 'selector' - CSS selector with wildcard support (e.g., '[class*=\"featured\"]'); 4) 'id' - search by element ID. All types return JSON with element info, selector for interaction, and parent context. Use 'contains' when you're not sure of exact text. Use 'selector' to find elements by class/attribute patterns."
}
func (t *SearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"text", "contains", "selector", "id"},
				"description": "Search type: 'text' (exact text), 'contains' (partial text), 'selector' (CSS selector with wildcards), 'id' (element ID)",
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query. For 'text'/'contains' - text to find. For 'selector' - CSS selector (e.g., '[class*=\"mp-\"]', '#main > div'). For 'id' - element ID",
			},
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "Maximum results to return (default: 10, max: 50)",
				"default":     10,
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

	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 50 {
		req.Limit = 50
	}

	result, err := t.browser.Search(ctx, req)
	if err != nil {
		return "", err
	}

	return formatSearchResult(result), nil
}

func formatSearchResult(result *entity.SearchResult) string {
	if !result.Found {
		return fmt.Sprintf("No results found for %s search: \"%s\"", result.Type, result.Query)
	}

	// Use new Results format if available
	if len(result.Results) > 0 {
		jsonBytes, err := json.MarshalIndent(map[string]interface{}{
			"type":  result.Type,
			"query": result.Query,
			"found": result.Count,
			"results": result.Results,
		}, "", "  ")
		if err != nil {
			return fmt.Sprintf("Error formatting results: %v", err)
		}
		return string(jsonBytes)
	}

	// Fallback to old format for backward compatibility
	switch result.Type {
	case "text":
		return fmt.Sprintf("Found text:\n\n%s", result.Content)

	case "id", "attribute":
		if len(result.Elements) == 0 {
			return fmt.Sprintf("No elements found for %s search", result.Type)
		}

		output := fmt.Sprintf("Found %d element(s):\n\n", len(result.Elements))
		for i, elem := range result.Elements {
			output += fmt.Sprintf("#%d <%s> [%s]\n", i+1, elem.TagName, elem.Selector)

			if elem.Text != "" {
				output += fmt.Sprintf("  Text: %q\n", elem.Text)
			}

			if elem.ID != "" {
				output += fmt.Sprintf("  ID: %q\n", elem.ID)
			}

			importantAttrs := []string{"type", "name", "value", "placeholder", "href", "aria-label", "title", "role"}
			for _, attr := range importantAttrs {
				if v, ok := elem.Attributes[attr]; ok && v != "" {
					output += fmt.Sprintf("  %s: %q\n", attr, v)
				}
			}

			output += "\n"
		}
		return output

	default:
		return "Unknown search type"
	}
}
