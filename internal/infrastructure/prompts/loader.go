package prompts

import (
	_ "embed"
)

//go:embed system.txt
var DefaultSystemPrompt string
