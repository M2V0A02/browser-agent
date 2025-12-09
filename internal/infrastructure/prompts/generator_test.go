package prompts

import (
	"context"
	"strings"
	"testing"

	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"
)

type mockAgent struct {
	subType     entity.SubAgentType
	description string
}

func (m *mockAgent) GetType() entity.AgentType {
	return entity.AgentType(m.subType)
}

func (m *mockAgent) GetSubAgentType() entity.SubAgentType {
	return m.subType
}

func (m *mockAgent) GetDescription() string {
	return m.description
}

func (m *mockAgent) Execute(ctx context.Context, task string) (string, error) {
	return "", nil
}

type mockAgentRegistry struct {
	agents []output.SimpleAgent
}

func (r *mockAgentRegistry) Register(agent output.SimpleAgent) {
	r.agents = append(r.agents, agent)
}

func (r *mockAgentRegistry) GetBySubType(subType entity.SubAgentType) (output.SimpleAgent, bool) {
	for _, agent := range r.agents {
		if agent.GetSubAgentType() == subType {
			return agent, true
		}
	}
	return nil, false
}

func (r *mockAgentRegistry) List() []output.SimpleAgent {
	return r.agents
}

func TestGenerateOrchestratorPrompt(t *testing.T) {
	registry := &mockAgentRegistry{}

	registry.Register(&mockAgent{
		subType:     entity.SubAgentNavigation,
		description: "Navigate to URLs and explore pages",
	})
	registry.Register(&mockAgent{
		subType:     entity.SubAgentExtraction,
		description: "Extract data from pages",
	})
	registry.Register(&mockAgent{
		subType:     entity.SubAgentForm,
		description: "Fill forms and click buttons",
	})

	template := `Test template

## AVAILABLE AGENTS

{{range .Agents -}}
- {{.Name}}: {{.Description}}
{{end}}`

	result, err := GenerateOrchestratorPrompt(template, registry)
	if err != nil {
		t.Fatalf("GenerateOrchestratorPrompt failed: %v", err)
	}

	if !strings.Contains(result, "Test template") {
		t.Error("Result should contain base template text")
	}

	if !strings.Contains(result, "navigation: Navigate to URLs and explore pages") {
		t.Error("Result should contain navigation agent description")
	}

	if !strings.Contains(result, "extraction: Extract data from pages") {
		t.Error("Result should contain extraction agent description")
	}

	if !strings.Contains(result, "form: Fill forms and click buttons") {
		t.Error("Result should contain form agent description")
	}

	t.Logf("Generated prompt:\n%s", result)
}

func TestGenerateOrchestratorPromptEmptyRegistry(t *testing.T) {
	registry := &mockAgentRegistry{}

	template := `Test template

{{range .Agents -}}
- {{.Name}}: {{.Description}}
{{end}}`

	result, err := GenerateOrchestratorPrompt(template, registry)
	if err != nil {
		t.Fatalf("GenerateOrchestratorPrompt failed: %v", err)
	}

	if !strings.Contains(result, "Test template") {
		t.Error("Result should contain base template text")
	}
}

func TestGenerateOrchestratorPromptInvalidTemplate(t *testing.T) {
	registry := &mockAgentRegistry{}
	registry.Register(&mockAgent{
		subType:     entity.SubAgentNavigation,
		description: "Test agent",
	})

	invalidTemplate := `Test {{.InvalidField}}`

	_, err := GenerateOrchestratorPrompt(invalidTemplate, registry)
	if err == nil {
		t.Error("Expected error for invalid template, got nil")
	}
}
