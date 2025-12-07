package adapter

import (
	t "browser-agent/internal/agents/tools"

	"github.com/sashabaranov/go-openai"
)

// Адаптер для использования t.Tool в стиле openai.Tool
type OpenAiAdapter struct {
	Tool t.Tool
}

// Конструктор адаптера
func NewOpenAiAdapter(tool t.Tool) *OpenAiAdapter {
	return &OpenAiAdapter{
		Tool: tool,
	}
}

// Преобразуем в openai.Tool с вызовом метода Parameters()
func (a *OpenAiAdapter) ToOpenAITool() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction, // OpenAI API требует "function"
		Function: &openai.FunctionDefinition{
			Name:        a.Tool.Name(),
			Description: a.Tool.Description(),
			Parameters:  a.Tool.Parameters(),
		},
	}
}
