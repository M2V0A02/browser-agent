package agents

import (
	"context"
	"fmt"
	"strings"

	"browser-agent/internal/agents/tools"
	t "browser-agent/internal/agents/tools"
	"browser-agent/internal/domain/adapter"
	"browser-agent/internal/domain/ports"
	"browser-agent/internal/infrastructure/env"

	"github.com/sashabaranov/go-openai"
)

const maxIterations = 50

type ReactAgent struct {
	client   *openai.Client
	model    string
	tools    []openai.Tool
	toolMap  map[string]t.Tool
	messages []openai.ChatCompletionMessage
}

func NewReactAgent(browser ports.BrowserCore, envService *env.EnvService, logger ports.Logger) (*ReactAgent, error) {
	config := openai.DefaultConfig(envService.MustGet("OPENROUTER_API_KEY"))
	config.BaseURL = "https://openrouter.ai/api/v1"

	client := openai.NewClientWithConfig(config)

	toolList := tools.NewGeneralTools(browser, logger)
	var openaiTools []openai.Tool
	toolMap := make(map[string]t.Tool)
	for _, tool := range toolList {
		openaiAdapter := adapter.NewOpenAiAdapter(tool)
		openaiTools = append(openaiTools, openaiAdapter.ToOpenAITool())
		toolMap[tool.Name()] = tool
	}

	return &ReactAgent{
		client:   client,
		model:    envService.MustGet("OPENROUTER_MODEL_NAME"),
		tools:    openaiTools,
		toolMap:  toolMap,
		messages: []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleSystem, Content: "Ты — автономный браузерный агент. Думай шаг за шагом и используй инструменты."}},
	}, nil
}

func (a *ReactAgent) Run(ctx context.Context, task string) (*AgentResult, error) {
	a.messages = append(a.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: task,
	})

	iteration := 0

	for iteration < maxIterations {
		iteration++

		stream, err := a.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
			Model:       a.model,
			Messages:    a.messages,
			Tools:       a.tools,
			ToolChoice:  "auto",
			Temperature: 0.0,
			Stream:      true,
		})
		if err != nil {
			return nil, err
		}

		var thought strings.Builder
		var assistantMsg openai.ChatCompletionMessage
		// Map для накопления tool calls по индексу (streaming приходит частями)
		toolCallsMap := make(map[int]*openai.ToolCall)

		for {
			chunk, err := stream.Recv()
			if err != nil {
				break
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			delta := chunk.Choices[0].Delta

			if delta.Content != "" {
				thought.WriteString(delta.Content)
				assistantMsg.Content += delta.Content
			}

			// Мержим tool calls по индексу
			for _, tc := range delta.ToolCalls {
				idx := *tc.Index
				if existing, ok := toolCallsMap[idx]; ok {
					// Дополняем существующий tool call
					existing.Function.Arguments += tc.Function.Arguments
					if tc.Function.Name != "" {
						existing.Function.Name = tc.Function.Name
					}
					if tc.ID != "" {
						existing.ID = tc.ID
					}
				} else {
					// Создаём новый tool call
					toolCallsMap[idx] = &openai.ToolCall{
						ID:   tc.ID,
						Type: tc.Type,
						Function: openai.FunctionCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				}
			}
		}
		stream.Close()

		// Конвертируем map в slice
		for i := 0; i < len(toolCallsMap); i++ {
			if tc, ok := toolCallsMap[i]; ok {
				assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, *tc)
			}
		}

		a.messages = append(a.messages, assistantMsg)

		// Если есть tool calls — выполняем их
		if len(assistantMsg.ToolCalls) > 0 {
			for _, tc := range assistantMsg.ToolCalls {
				name := tc.Function.Name
				args := tc.Function.Arguments

				var obs string
				var err error

				tool, ok := a.toolMap[name]
				if !ok {
					obs = fmt.Sprintf("Error: unknown tool '%s'", name)
				} else {
					obs, err = tool.Call(ctx, args)
					if err != nil {
						obs = "Error: " + err.Error()
					}
				}

				// По-прежнему обрезаем огромные ответы, чтобы не убить контекст
				if len(obs) > 20000 {
					obs = obs[:20000] + "\n... (обрезано)"
				}

				a.messages = append(a.messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					ToolCallID: tc.ID,
					Name:       name,
					Content:    obs,
				})
			}
			continue
		}

		// Финальный ответ
		return &AgentResult{
			FinalAnswer: assistantMsg.Content,
			FullLog:     nil, // логи полностью убраны
		}, nil
	}

	return nil, fmt.Errorf("max iterations exceeded")
}

// ExecuteWithStreaming implements Agent interface
func (a *ReactAgent) ExecuteWithStreaming(ctx context.Context, task string, onChunk func(LogEntry)) (*AgentResult, error) {
	return a.Run(ctx, task)
}

// Execute is a simple wrapper for Run
func (a *ReactAgent) Execute(ctx context.Context, task string) (string, error) {
	result, err := a.Run(ctx, task)
	if err != nil {
		return "", err
	}
	return result.FinalAnswer, nil
}
