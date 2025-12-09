package prompts

import (
	"bytes"
	"sort"
	"text/template"

	"browser-agent/internal/application/port/output"
)

type AgentInfo struct {
	Name        string
	Description string
}

type OrchestratorPromptData struct {
	Agents []AgentInfo
}

func GenerateOrchestratorPrompt(baseTemplate string, agentRegistry output.SimpleAgentRegistry) (string, error) {
	agents := agentRegistry.List()
	agentInfos := make([]AgentInfo, 0, len(agents))

	for _, agent := range agents {
		agentInfos = append(agentInfos, AgentInfo{
			Name:        string(agent.GetSubAgentType()),
			Description: agent.GetDescription(),
		})
	}

	sort.Slice(agentInfos, func(i, j int) bool {
		return agentInfos[i].Name < agentInfos[j].Name
	})

	data := OrchestratorPromptData{
		Agents: agentInfos,
	}

	tmpl, err := template.New("orchestrator").Parse(baseTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
