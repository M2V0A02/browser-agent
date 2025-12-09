package entity

type AgentType string

const (
	AgentTypeOrchestrator AgentType = "orchestrator"
	AgentTypeNavigation   AgentType = "navigation"
	AgentTypeExtraction   AgentType = "extraction"
	AgentTypeForm         AgentType = "form"
	AgentTypeAnalysis     AgentType = "analysis"
)

type AgentRequest struct {
	Type          AgentType
	Task          string
	Context       map[string]interface{}
	MaxIterations int
}

type AgentResponse struct {
	Success    bool
	Result     string
	Iterations int
	Error      string
}
