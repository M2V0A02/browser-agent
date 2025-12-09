package entity

type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

type ContentBlockType string

const (
	ContentTypeText     ContentBlockType = "text"
	ContentTypeThinking ContentBlockType = "thinking"
	ContentTypeToolUse  ContentBlockType = "tool_use"
)

type ContentBlock struct {
	Type      ContentBlockType
	Text      string
	Thinking  string
	ToolUse   *ToolCall
}

type Message struct {
	Role       MessageRole
	Content    string
	ContentBlocks []ContentBlock
	ToolCalls  []ToolCall
	ToolCallID string
	Name       string
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type ToolDefinition struct {
	Name        ToolName
	Description string
	Parameters  map[string]interface{}
}
