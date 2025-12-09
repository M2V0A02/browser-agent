package prompts

import (
	_ "embed"
)

//go:embed orchestrator.txt
var OrchestratorPrompt string

//go:embed navigation.txt
var NavigationPrompt string

//go:embed extraction.txt
var ExtractionPrompt string

//go:embed form.txt
var FormPrompt string
