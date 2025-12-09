package service

import (
	"browser-agent/internal/application/port/input"
	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"
)

var _ output.ToolRegistry = (*ToolRegistryImpl)(nil)

type ToolRegistryImpl struct {
	tools map[entity.ToolName]output.ToolPort
}

func NewToolRegistry() *ToolRegistryImpl {
	return &ToolRegistryImpl{
		tools: make(map[entity.ToolName]output.ToolPort),
	}
}

func (r *ToolRegistryImpl) Register(tool output.ToolPort) {
	r.tools[tool.Name()] = tool
}

func (r *ToolRegistryImpl) Get(name entity.ToolName) (output.ToolPort, bool) {
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

var _ output.AgentRegistry = (*AgentRegistryImpl)(nil)

type AgentRegistryImpl struct {
	agents map[entity.AgentType]input.AgentExecutor
}

func NewAgentRegistry() *AgentRegistryImpl {
	return &AgentRegistryImpl{
		agents: make(map[entity.AgentType]input.AgentExecutor),
	}
}

func (r *AgentRegistryImpl) Register(agent input.AgentExecutor) {
	r.agents[agent.GetType()] = agent
}

func (r *AgentRegistryImpl) Get(agentType entity.AgentType) (input.AgentExecutor, bool) {
	agent, ok := r.agents[agentType]
	return agent, ok
}

func (r *AgentRegistryImpl) List() []entity.AgentType {
	result := make([]entity.AgentType, 0, len(r.agents))
	for agentType := range r.agents {
		result = append(result, agentType)
	}
	return result
}

var _ output.SimpleAgentRegistry = (*SimpleAgentRegistryImpl)(nil)

type SimpleAgentRegistryImpl struct {
	agents map[entity.AgentType]output.SimpleAgent
}

func NewSimpleAgentRegistry() *SimpleAgentRegistryImpl {
	return &SimpleAgentRegistryImpl{
		agents: make(map[entity.AgentType]output.SimpleAgent),
	}
}

func (r *SimpleAgentRegistryImpl) Register(agent output.SimpleAgent) {
	r.agents[agent.GetType()] = agent
}

func (r *SimpleAgentRegistryImpl) Get(agentType entity.AgentType) (output.SimpleAgent, bool) {
	agent, ok := r.agents[agentType]
	return agent, ok
}

func (r *SimpleAgentRegistryImpl) List() []output.SimpleAgent {
	result := make([]output.SimpleAgent, 0, len(r.agents))
	for _, agent := range r.agents {
		result = append(result, agent)
	}
	return result
}
