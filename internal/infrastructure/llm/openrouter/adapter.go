package openrouter

import (
	"context"
	"errors"
	"fmt"
	"io"

	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"

	"github.com/sashabaranov/go-openai"
)

var _ output.LLMPort = (*OpenRouterAdapter)(nil)

type OpenRouterAdapter struct {
	client *openai.Client
	model  string
}

type Config struct {
	APIKey  string
	Model   string
	BaseURL string
}

func DefaultConfig(apiKey, model string) Config {
	return Config{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: "https://openrouter.ai/api/v1",
	}
}

func NewOpenRouterAdapter(cfg Config) *OpenRouterAdapter {
	config := openai.DefaultConfig(cfg.APIKey)
	config.BaseURL = cfg.BaseURL

	return &OpenRouterAdapter{
		client: openai.NewClientWithConfig(config),
		model:  cfg.Model,
	}
}

func (a *OpenRouterAdapter) Chat(ctx context.Context, req output.ChatRequest) (*output.ChatResponse, error) {
	messages := convertMessages(req.Messages)
	tools := convertTools(req.Tools)

	resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       a.model,
		Messages:    messages,
		Tools:       tools,
		ToolChoice:  "auto",
		Temperature: req.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := resp.Choices[0]
	return &output.ChatResponse{
		Message: convertResponseMessage(choice.Message),
	}, nil
}

func (a *OpenRouterAdapter) ChatStream(ctx context.Context, req output.ChatRequest, onChunk func(output.StreamChunk)) (*output.ChatResponse, error) {
	messages := convertMessages(req.Messages)
	tools := convertTools(req.Tools)

	stream, err := a.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:       a.model,
		Messages:    messages,
		Tools:       tools,
		ToolChoice:  "auto",
		Temperature: req.Temperature,
		Stream:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("chat stream failed: %w", err)
	}
	defer stream.Close()

	var finalMessage entity.Message
	finalMessage.Role = entity.RoleAssistant
	toolCallsMap := make(map[int]*entity.ToolCall)
	var thinkingContent string
	var textContent string

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("stream recv error: %w", err)
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta

		if delta.ReasoningContent != "" {
			thinkingContent += delta.ReasoningContent
		}

		if delta.Content != "" {
			textContent += delta.Content
			if onChunk != nil {
				onChunk(output.StreamChunk{
					Content: delta.Content,
				})
			}
		}

		for _, tc := range delta.ToolCalls {
			idx := *tc.Index
			if existing, ok := toolCallsMap[idx]; ok {
				existing.Arguments += tc.Function.Arguments
				if tc.Function.Name != "" {
					existing.Name = tc.Function.Name
				}
				if tc.ID != "" {
					existing.ID = tc.ID
				}
			} else {
				toolCallsMap[idx] = &entity.ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				}
			}
		}
	}

	if thinkingContent != "" {
		finalMessage.ContentBlocks = append(finalMessage.ContentBlocks, entity.ContentBlock{
			Type:     entity.ContentTypeThinking,
			Thinking: thinkingContent,
		})
	}

	if textContent != "" {
		finalMessage.ContentBlocks = append(finalMessage.ContentBlocks, entity.ContentBlock{
			Type: entity.ContentTypeText,
			Text: textContent,
		})
		finalMessage.Content = textContent
	}

	for i := 0; i < len(toolCallsMap); i++ {
		if tc, ok := toolCallsMap[i]; ok {
			finalMessage.ToolCalls = append(finalMessage.ToolCalls, *tc)
			finalMessage.ContentBlocks = append(finalMessage.ContentBlocks, entity.ContentBlock{
				Type:    entity.ContentTypeToolUse,
				ToolUse: tc,
			})
		}
	}

	if onChunk != nil {
		onChunk(output.StreamChunk{
			ToolCalls: finalMessage.ToolCalls,
			Done:      true,
		})
	}

	return &output.ChatResponse{
		Message: finalMessage,
	}, nil
}

func convertMessages(messages []entity.Message) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, 0, len(messages))
	for _, msg := range messages {
		oaiMsg := openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}

		if msg.ToolCallID != "" {
			oaiMsg.ToolCallID = msg.ToolCallID
		}
		if msg.Name != "" {
			oaiMsg.Name = msg.Name
		}

		if len(msg.ContentBlocks) > 0 {
			var textContent string
			for _, block := range msg.ContentBlocks {
				if block.Type == entity.ContentTypeText {
					textContent += block.Text
				}
			}
			if textContent != "" {
				oaiMsg.Content = textContent
			}
		}

		for _, tc := range msg.ToolCalls {
			oaiMsg.ToolCalls = append(oaiMsg.ToolCalls, openai.ToolCall{
				ID:   tc.ID,
				Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			})
		}

		result = append(result, oaiMsg)
	}
	return result
}

func convertTools(tools []entity.ToolDefinition) []openai.Tool {
	result := make([]openai.Tool, 0, len(tools))
	for _, t := range tools {
		result = append(result, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}
	return result
}

func convertResponseMessage(msg openai.ChatCompletionMessage) entity.Message {
	result := entity.Message{
		Role:    entity.MessageRole(msg.Role),
		Content: msg.Content,
	}

	if msg.Content != "" {
		result.ContentBlocks = append(result.ContentBlocks, entity.ContentBlock{
			Type: entity.ContentTypeText,
			Text: msg.Content,
		})
	}

	for _, tc := range msg.ToolCalls {
		toolCall := entity.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		}
		result.ToolCalls = append(result.ToolCalls, toolCall)
		result.ContentBlocks = append(result.ContentBlocks, entity.ContentBlock{
			Type:    entity.ContentTypeToolUse,
			ToolUse: &toolCall,
		})
	}

	return result
}
