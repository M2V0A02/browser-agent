package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"browser-agent/internal/domain/ports"
	"browser-agent/internal/infrastructure/browser/rodwrapper"
)

// BrowserExtract реализует инструмент извлечения очищенного HTML.
type BrowserExtract struct {
	page   *rodwrapper.Page
	logger ports.Logger
}

var _ Tool = (*BrowserExtract)(nil)

// NewBrowserExtractTool создаёт инструмент извлечения HTML.
func NewBrowserExtractTool(page *rodwrapper.Page, logger ports.Logger) Tool {
	return &BrowserExtract{
		page:   page,
		logger: logger,
	}
}

func (t *BrowserExtract) Name() string {
	return "extract"
}

func (t *BrowserExtract) Type() string {
	return "browser"
}

// ExtractInput определяет входные параметры.
type ExtractInput struct {
	Selector string `json:"selector,omitempty"`
}

func (t *BrowserExtract) Description() string {
	return "Extracts Text from HTML."
}

func (t *BrowserExtract) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"selector": map[string]interface{}{
				"type":        "string",
				"description": `CSS selector (e.g. "#content", ".article") to extract a specific element. If not provided, extracts the entire <body> element.`,
			},
		},
		"required": []string{},
	}
}

func (t *BrowserExtract) Call(ctx context.Context, input string) (string, error) {
	var args ExtractInput

	if input != "" {
		if err := json.Unmarshal([]byte(input), &args); err != nil {
			return "", fmt.Errorf("invalid JSON input: %w", err)
		}
	}

	var rawHTML string

	if args.Selector != "" {
		t.logger.Logf("Extracting HTML by selector: %s", args.Selector)
		el, err := t.page.Element(args.Selector)
		if err != nil {
			return fmt.Sprintf("element not found by selector '%s': %s", args.Selector, err), nil
		}
		rawHTML, err = el.Text()
		if err != nil {
			return "", fmt.Errorf("failed to get element HTML: %w", err)
		}
	} else {
		t.logger.Logf("Extracting entire body of the page")
		body, err := t.page.Element("body")
		if err != nil {
			return "", fmt.Errorf("body not found on page: %w", err)
		}
		rawHTML, err = body.Text()
		if err != nil {
			return "", fmt.Errorf("failed to get body HTML: %w", err)
		}
	}

	logPreview := rawHTML
	if len(logPreview) > 200 {
		logPreview = logPreview[:200] + "..."
	}

	t.logger.Logf("TOOL %s completed: extracted %d characters: %s",
		t.Name(), len(rawHTML), logPreview)

	return rawHTML, nil
}
