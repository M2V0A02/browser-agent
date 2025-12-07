package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"

	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"

	"github.com/sashabaranov/go-openai"
)

var _ output.LLMPort = (*OpenRouterAdapter)(nil)

type OpenRouterAdapter struct {
	client *openai.Client
	model  string
	logger output.LoggerPort
}

type Config struct {
	APIKey  string
	Model   string
	BaseURL string
	Logger  output.LoggerPort
}

func DefaultConfig(apiKey, model string) Config {
	return Config{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: "https://openrouter.ai/api/v1",
	}
}

type loggingTransport struct {
	base   http.RoundTripper
	logger output.LoggerPort
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.logger != nil {
		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, _ = io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		var requestData map[string]interface{}
		if len(bodyBytes) > 0 {
			json.Unmarshal(bodyBytes, &requestData)
		}

		t.logger.Info("HTTP Request",
			"method", req.Method,
			"url", req.URL.String(),
			"body", requestData,
		)
	}

	resp, err := t.base.RoundTrip(req)

	if t.logger != nil && resp != nil {
		t.logger.Info("HTTP Response",
			"status", resp.Status,
			"statusCode", resp.StatusCode,
		)
	}

	return resp, err
}

func NewOpenRouterAdapter(cfg Config) *OpenRouterAdapter {
	config := openai.DefaultConfig(cfg.APIKey)
	config.BaseURL = cfg.BaseURL

	if cfg.Logger != nil {
		transport := &loggingTransport{
			base:   http.DefaultTransport,
			logger: cfg.Logger,
		}
		config.HTTPClient = &http.Client{
			Transport: transport,
		}
	}

	return &OpenRouterAdapter{
		client: openai.NewClientWithConfig(config),
		model:  cfg.Model,
		logger: cfg.Logger,
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

	if a.logger != nil {
		totalChars := 0
		for _, msg := range messages {
			totalChars += len(msg.Content)
		}
		a.logger.Debug("Creating chat completion stream",
			"model", a.model,
			"messagesCount", len(messages),
			"toolsCount", len(tools),
			"temperature", req.Temperature,
			"totalChars", totalChars)
	}

	stream, err := a.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:       a.model,
		Messages:    messages,
		Tools:       tools,
		ToolChoice:  "auto",
		Temperature: req.Temperature,
		Stream:      true,
	})
	if err != nil {
		if a.logger != nil {
			a.logger.Error("Failed to create stream", "error", err)
		}
		return nil, fmt.Errorf("chat stream failed: %w", err)
	}
	defer stream.Close()

	var finalMessage entity.Message
	finalMessage.Role = entity.RoleAssistant
	toolCallsMap := make(map[int]*entity.ToolCall)
	var thinkingContent string
	var textContent string
	chunkCount := 0

	if a.logger != nil {
		a.logger.Debug("Starting stream reception")
	}

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled: %w", ctx.Err())
		default:
		}

		chunk, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if a.logger != nil {
					a.logger.Debug("Stream completed", "chunks", chunkCount, "thinkingLen", len(thinkingContent), "textLen", len(textContent))
				}
				break
			}
			if a.logger != nil {
				a.logger.Error("Stream recv error", "error", err, "chunks", chunkCount, "errorType", fmt.Sprintf("%T", err))
			}
			return nil, fmt.Errorf("stream recv error: %w", err)
		}

		chunkCount++

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
			if tc.Index == nil {
				continue
			}
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
		if a.logger != nil {
			a.logger.Debug("Received thinking content", "length", len(thinkingContent))
		}
		finalMessage.ContentBlocks = append(finalMessage.ContentBlocks, entity.ContentBlock{
			Type:     entity.ContentTypeThinking,
			Thinking: thinkingContent,
		})
	}

	if textContent != "" {
		if a.logger != nil {
			a.logger.Debug("Received text content", "length", len(textContent))
		}
		finalMessage.ContentBlocks = append(finalMessage.ContentBlocks, entity.ContentBlock{
			Type: entity.ContentTypeText,
			Text: textContent,
		})
		finalMessage.Content = textContent
	}

	if a.logger != nil {
		a.logger.Debug("Final message assembled",
			"contentBlocksCount", len(finalMessage.ContentBlocks),
			"toolCallsCount", len(finalMessage.ToolCalls),
			"contentLength", len(finalMessage.Content))
	}

	indices := make([]int, 0, len(toolCallsMap))
	for idx := range toolCallsMap {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	for _, idx := range indices {
		tc := toolCallsMap[idx]
		finalMessage.ToolCalls = append(finalMessage.ToolCalls, *tc)
		finalMessage.ContentBlocks = append(finalMessage.ContentBlocks, entity.ContentBlock{
			Type:    entity.ContentTypeToolUse,
			ToolUse: tc,
		})
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
			var fullContent string
			for _, block := range msg.ContentBlocks {
				if block.Type == entity.ContentTypeThinking && block.Thinking != "" {
					fullContent += "<thinking>\n" + block.Thinking + "\n</thinking>\n"
				} else if block.Type == entity.ContentTypeText && block.Text != "" {
					fullContent += block.Text
				}
			}
			if fullContent != "" {
				oaiMsg.Content = fullContent
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
