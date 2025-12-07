package openrouter

import (
	"testing"

	"browser-agent/internal/domain/entity"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

func TestConvertResponseMessage_WithContent(t *testing.T) {
	msg := openai.ChatCompletionMessage{
		Role:    "assistant",
		Content: "Hello, world!",
	}

	result := convertResponseMessage(msg)

	assert.Equal(t, entity.RoleAssistant, result.Role)
	assert.Equal(t, "Hello, world!", result.Content)
	assert.Len(t, result.ContentBlocks, 1)
	assert.Equal(t, entity.ContentTypeText, result.ContentBlocks[0].Type)
	assert.Equal(t, "Hello, world!", result.ContentBlocks[0].Text)
}

func TestConvertResponseMessage_WithToolCalls(t *testing.T) {
	msg := openai.ChatCompletionMessage{
		Role:    "assistant",
		Content: "",
		ToolCalls: []openai.ToolCall{
			{
				ID:   "call_123",
				Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{
					Name:      "navigate",
					Arguments: `{"url":"https://example.com"}`,
				},
			},
		},
	}

	result := convertResponseMessage(msg)

	assert.Equal(t, entity.RoleAssistant, result.Role)
	assert.Len(t, result.ToolCalls, 1)
	assert.Equal(t, "call_123", result.ToolCalls[0].ID)
	assert.Equal(t, "navigate", result.ToolCalls[0].Name)
	assert.Len(t, result.ContentBlocks, 1)
	assert.Equal(t, entity.ContentTypeToolUse, result.ContentBlocks[0].Type)
	assert.NotNil(t, result.ContentBlocks[0].ToolUse)
}

func TestConvertMessages_WithContentBlocks(t *testing.T) {
	messages := []entity.Message{
		{
			Role:    entity.RoleUser,
			Content: "Hello",
		},
		{
			Role:    entity.RoleAssistant,
			Content: "Hi there",
			ContentBlocks: []entity.ContentBlock{
				{
					Type:     entity.ContentTypeThinking,
					Thinking: "Let me think about this...",
				},
				{
					Type: entity.ContentTypeText,
					Text: "Hi there",
				},
			},
		},
	}

	result := convertMessages(messages)

	assert.Len(t, result, 2)
	assert.Equal(t, "user", result[0].Role)
	assert.Equal(t, "Hello", result[0].Content)
	assert.Equal(t, "assistant", result[1].Role)
	assert.Equal(t, "<thinking>\nLet me think about this...\n</thinking>\nHi there", result[1].Content)
}

func TestConvertMessages_EmptyContentWithBlocks(t *testing.T) {
	messages := []entity.Message{
		{
			Role:    entity.RoleAssistant,
			Content: "",
			ContentBlocks: []entity.ContentBlock{
				{
					Type: entity.ContentTypeText,
					Text: "Response text",
				},
			},
		},
	}

	result := convertMessages(messages)

	assert.Len(t, result, 1)
	assert.Equal(t, "Response text", result[0].Content)
}
