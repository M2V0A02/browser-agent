package prompts_test

import (
	"fmt"
	"testing"

	"browser-agent/internal/application/service"
	"browser-agent/internal/infrastructure/logger"
	"browser-agent/internal/infrastructure/prompts"
	"browser-agent/internal/usecase/agents/extraction"
	"browser-agent/internal/usecase/agents/form"
	"browser-agent/internal/usecase/agents/navigation"
)

func TestRealOrchestratorPromptGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real prompt generation in short mode")
	}

	log, _ := logger.NewLoggerAdapter()
	defer log.Close()

	registry := service.NewSimpleAgentRegistry()
	tools := service.NewToolRegistry()

	registry.Register(navigation.New(nil, tools, log, ""))
	registry.Register(extraction.New(nil, tools, log, ""))
	registry.Register(form.New(nil, tools, log, ""))

	prompt, err := prompts.GenerateOrchestratorPrompt(prompts.OrchestratorPrompt, registry)
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	fmt.Println("=== GENERATED ORCHESTRATOR PROMPT ===")
	fmt.Println(prompt)
	fmt.Println("=== END OF PROMPT ===")

	if len(prompt) < 100 {
		t.Error("Generated prompt seems too short")
	}
}
