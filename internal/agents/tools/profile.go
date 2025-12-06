package tools

import (
	"browser-agent/internal/domain/ports"
)

func NewGeneralTools(core ports.BrowserCore, logger ports.Logger) []Tool {
	page, _ := core.Page()
	return []Tool{
		NewBrowserNavigateTool(page, logger),
		NewBrowserClickTool(page, logger),
		NewBrowserFillTool(page, logger),
		NewBrowserPressEnterTool(page, logger),
		NewBrowserScrollTool(page, logger),
		NewBrowserExtractTool(page, logger),

		NewUserAskTool(page, logger),
		NewUserWaitActionTool(page, logger),
	}
}

func NewSupervisorTools(core ports.BrowserCore, logger ports.Logger) []Tool {
	page, _ := core.Page()
	return []Tool{
		NewSupervisorDelegateAgentTool(page, logger),
	}
}
