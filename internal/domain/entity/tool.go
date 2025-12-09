package entity

type ToolName string

const (
	ToolBrowserNavigate     ToolName = "browser_navigate"
	ToolBrowserClick        ToolName = "browser_click"
	ToolBrowserFill         ToolName = "browser_fill"
	ToolBrowserScroll       ToolName = "browser_scroll"
	ToolBrowserScreenshot   ToolName = "browser_screenshot"
	ToolBrowserPressEnter   ToolName = "browser_press_enter"
	ToolBrowserObserve      ToolName = "browser_observe"
	ToolBrowserQueryElements ToolName = "browser_query_elements"
	ToolBrowserSearch       ToolName = "browser_search"

	ToolRunAgent ToolName = "run_agent"

	ToolUserAskQuestion   ToolName = "user_ask_question"
	ToolUserWaitAction    ToolName = "user_wait_action"
)

func (t ToolName) String() string {
	return string(t)
}

type SubAgentType string

const (
	SubAgentNavigation SubAgentType = "navigation"
	SubAgentExtraction SubAgentType = "extraction"
	SubAgentForm       SubAgentType = "form"
	SubAgentAnalysis   SubAgentType = "analysis"
)

func (t SubAgentType) String() string {
	return string(t)
}
