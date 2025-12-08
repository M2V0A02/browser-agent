package output

import (
	"context"

	"browser-agent/internal/domain/entity"
)

type LLMPort interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}

type ChatRequest struct {
	Messages    []entity.Message
	Tools       []entity.ToolDefinition
	Temperature float32
}

type ChatResponse struct {
	Message entity.Message
}
