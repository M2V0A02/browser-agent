package agents

import (
	"browser-agent/internal/domain/ports"
	"browser-agent/internal/infrastructure/browser/rodwrapper"
)

type BrowserCore struct {
	page   *rodwrapper.Page
	logger ports.Logger
}

func NewBrowserCore(page *rodwrapper.Page, logger ports.Logger) *BrowserCore {
	return &BrowserCore{page: page, logger: logger}
}

func (c *BrowserCore) Page() *rodwrapper.Page {
	return c.page
}
