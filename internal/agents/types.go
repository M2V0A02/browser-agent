package agents

type LogEntry struct {
	Type      string `json:"type"` // thought | action | observation | final
	Content   string `json:"content"`
	ToolName  string `json:"tool_name,omitempty"`
	Timestamp string `json:"timestamp"`
}

type AgentResult struct {
	FinalAnswer string     `json:"final_answer"`
	FullLog     []LogEntry `json:"full_log"`
}
