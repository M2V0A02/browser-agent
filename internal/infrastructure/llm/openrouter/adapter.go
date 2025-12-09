package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"

	"github.com/sashabaranov/go-openai"
)

var _ output.LLMPort = (*OpenRouterAdapter)(nil)

type OpenRouterAdapter struct {
	client        *openai.Client
	model         string
	logger        output.LoggerPort
	thinkingMode  bool
	thinkingBudget int
}

type Config struct {
	APIKey         string
	Model          string
	BaseURL        string
	Logger         output.LoggerPort
	ThinkingMode   bool
	ThinkingBudget int
}

func DefaultConfig(apiKey, model string) Config {
	return Config{
		APIKey:         apiKey,
		Model:          model,
		BaseURL:        "https://openrouter.ai/api/v1",
		ThinkingMode:   true,
		ThinkingBudget: 10000,
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

	if resp != nil && resp.Body != nil {
		resp.Body = &reasoningFixerReader{
			reader: resp.Body,
			logger: t.logger,
		}
	}

	return resp, err
}

type reasoningFixerReader struct {
	reader io.ReadCloser
	buffer bytes.Buffer
	logger output.LoggerPort
}

func (r *reasoningFixerReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		chunk := string(p[:n])
		fixed := strings.ReplaceAll(chunk, `"reasoning":`, `"reasoning_content":`)
		copy(p, fixed)
		return len(fixed), err
	}
	return n, err
}

func (r *reasoningFixerReader) Close() error {
	return r.reader.Close()
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
		client:         openai.NewClientWithConfig(config),
		model:          cfg.Model,
		logger:         cfg.Logger,
		thinkingMode:   cfg.ThinkingMode,
		thinkingBudget: cfg.ThinkingBudget,
	}
}

func (a *OpenRouterAdapter) Chat(ctx context.Context, req output.ChatRequest) (*output.ChatResponse, error) {
	messages := convertMessages(req.Messages)
	tools := convertTools(req.Tools)

	if a.logger != nil {
		a.logger.Debug("Creating chat completion",
			"model", a.model,
			"messagesCount", len(messages),
			"toolsCount", len(tools),
			"temperature", req.Temperature,
			"thinkingMode", a.thinkingMode,
			"thinkingBudget", a.thinkingBudget)
	}

	chatReq := openai.ChatCompletionRequest{
		Model:       a.model,
		Messages:    messages,
		Tools:       tools,
		ToolChoice:  "auto",
		Temperature: req.Temperature,
	}

	if a.thinkingMode && a.thinkingBudget > 0 {
		chatReq.MaxCompletionTokens = a.thinkingBudget
	}

	resp, err := a.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		if a.logger != nil {
			a.logger.Error("Chat completion failed", "error", err)
		}
		return nil, fmt.Errorf("chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := resp.Choices[0]
	message := convertResponseMessage(choice.Message)

	if a.logger != nil {
		toolCallsInfo := make([]map[string]string, 0, len(message.ToolCalls))
		for _, tc := range message.ToolCalls {
			toolCallsInfo = append(toolCallsInfo, map[string]string{
				"id":   tc.ID,
				"name": tc.Name,
				"args": tc.Arguments,
			})
		}

		thinkingLen := 0
		for _, block := range message.ContentBlocks {
			if block.Type == entity.ContentTypeThinking {
				thinkingLen += len(block.Thinking)
			}
		}

		a.logger.Info("LLM Response received",
			"role", message.Role,
			"content", message.Content,
			"toolCalls", toolCallsInfo,
			"contentBlocksCount", len(message.ContentBlocks),
			"thinkingLen", thinkingLen,
		)
	}

	return &output.ChatResponse{
		Message: message,
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
				Name:        t.Name.String(),
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

	if msg.ReasoningContent != "" {
		result.ContentBlocks = append(result.ContentBlocks, entity.ContentBlock{
			Type:     entity.ContentTypeThinking,
			Thinking: msg.ReasoningContent,
		})
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
