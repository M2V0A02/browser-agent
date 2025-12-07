package output

import (
	"context"

	"browser-agent/internal/domain/entity"
)

type LLMPort interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error)
}

type ChatRequest struct {
	Messages    []entity.Message
	Tools       []entity.ToolDefinition
	Temperature float32
}

type ChatResponse struct {
	Message entity.Message
}

type StreamChunk struct {
	Content   string
	ToolCalls []entity.ToolCall
	Done      bool
}
