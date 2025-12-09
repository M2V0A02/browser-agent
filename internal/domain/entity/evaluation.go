package entity

type EvaluationResult struct {
	Success     bool     `json:"success"`
	Confidence  float64  `json:"confidence"`
	Issues      []string `json:"issues"`
	Feedback    string   `json:"feedback"`
	ShouldRetry bool     `json:"should_retry"`
}

type EvaluationCriteria struct {
	TaskDescription string
	ActualResult    string
	AgentType       AgentType
}
