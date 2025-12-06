package tools

import (
	"browser-agent/internal/domain/ports"
	"browser-agent/internal/infrastructure/browser/rodwrapper"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type BrowserScroll struct {
	page   *rodwrapper.Page
	logger ports.Logger
}

var _ Tool = (*BrowserScroll)(nil)

func NewBrowserScrollTool(page *rodwrapper.Page, logger ports.Logger) Tool {
	return &BrowserScroll{
		page:   page,
		logger: logger,
	}
}

func (t *BrowserScroll) Name() string {
	return "scroll"
}

func (t *BrowserScroll) Type() string {
	return "browser"
}

type scrollInput struct {
	Direction string `json:"direction"`
}

func (t *BrowserScroll) Description() string {
	return "Scrolls the page in different directions or to specific positions."
}

func (t *BrowserScroll) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"direction": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"down", "up", "top", "bottom"},
				"description": "Direction to scroll: 'down' (scroll down), 'up' (scroll up), 'top' (scroll to top), 'bottom' (scroll to bottom)",
			},
		},
		"required": []string{"direction"},
	}
}

func (t *BrowserScroll) Call(ctx context.Context, input string) (string, error) {
	var params scrollInput

	if input != "" {
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			params.Direction = strings.ToLower(strings.TrimSpace(input))
		}
	}

	direction := strings.ToLower(strings.TrimSpace(params.Direction))

	switch direction {
	case "down":
		t.logger.Logf("Scrolling down by 2 viewport heights")
		t.page.Eval(`() => window.scrollBy(0, window.innerHeight * 2)`)
	case "up":
		t.logger.Logf("Scrolling up by 2 viewport heights")
		t.page.Eval(`() => window.scrollBy(0, -window.innerHeight * 2)`)
	case "top":
		t.logger.Logf("Scrolling to top of page")
		t.page.Eval(`() => window.scrollTo(0, 0)`)
	case "bottom":
		t.logger.Logf("Scrolling to bottom of page")
		t.page.Eval(`() => window.scrollTo(0, document.body.scrollHeight)`)
	default:
		return "", fmt.Errorf("unknown scroll direction: %s (available: down, up, top, bottom)", direction)
	}

	t.page.WaitIdle(800 * time.Millisecond)

	action := fmt.Sprintf("Scrolled %s", direction)
	t.logger.Logf("TOOL %s completed: %s", t.Name(), action)
	return action, nil
}
