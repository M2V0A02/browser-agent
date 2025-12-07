package service

import (
	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"
)

var _ output.ToolRegistry = (*ToolRegistryImpl)(nil)

type ToolRegistryImpl struct {
	tools map[string]output.ToolPort
}

func NewToolRegistry() *ToolRegistryImpl {
	return &ToolRegistryImpl{
		tools: make(map[string]output.ToolPort),
	}
}

func (r *ToolRegistryImpl) Register(tool output.ToolPort) {
	r.tools[tool.Name()] = tool
}

func (r *ToolRegistryImpl) Get(name string) (output.ToolPort, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *ToolRegistryImpl) All() []output.ToolPort {
	result := make([]output.ToolPort, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}

func (r *ToolRegistryImpl) Definitions() []entity.ToolDefinition {
	result := make([]entity.ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, entity.ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}
	return result
}
