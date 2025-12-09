package evaluator

import (
	"strings"
	"testing"

	"browser-agent/internal/domain/entity"
)

func TestParseEvaluationResponse_ValidJSON(t *testing.T) {
	e := &Evaluator{}

	jsonResponse := `{
  "success": true,
  "confidence": 0.9,
  "issues": ["minor issue"],
  "feedback": "good job",
  "should_retry": false
}`

	result, err := e.parseEvaluationResponse(jsonResponse)
	if err != nil {
		t.Fatalf("parseEvaluationResponse failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success=true")
	}

	if result.Confidence != 0.9 {
		t.Errorf("Expected confidence=0.9, got %f", result.Confidence)
	}

	if len(result.Issues) != 1 || result.Issues[0] != "minor issue" {
		t.Errorf("Expected issues=[\"minor issue\"], got %v", result.Issues)
	}

	if result.Feedback != "good job" {
		t.Errorf("Expected feedback=\"good job\", got %s", result.Feedback)
	}

	if result.ShouldRetry {
		t.Error("Expected should_retry=false")
	}
}

func TestParseEvaluationResponse_WithTextAround(t *testing.T) {
	e := &Evaluator{}

	response := `Here's my evaluation:

{
  "success": false,
  "confidence": 0.3,
  "issues": ["selectors missing", "no data extracted"],
  "feedback": "Need to use observe first",
  "should_retry": true
}

Hope this helps!`

	result, err := e.parseEvaluationResponse(response)
	if err != nil {
		t.Fatalf("parseEvaluationResponse failed: %v", err)
	}

	if result.Success {
		t.Error("Expected success=false")
	}

	if result.Confidence != 0.3 {
		t.Errorf("Expected confidence=0.3, got %f", result.Confidence)
	}

	if len(result.Issues) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(result.Issues))
	}

	if !result.ShouldRetry {
		t.Error("Expected should_retry=true")
	}
}

func TestParseEvaluationResponse_InvalidJSON(t *testing.T) {
	e := &Evaluator{}

	invalidResponse := "This is not JSON at all"

	_, err := e.parseEvaluationResponse(invalidResponse)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestBuildEvaluationPrompt_Extraction(t *testing.T) {
	e := &Evaluator{}

	prompt := e.buildEvaluationPrompt(entity.EvaluationCriteria{
		TaskDescription: "Extract 10 emails",
		ActualResult:    "Extracted 10 emails successfully",
		AgentType:       entity.AgentTypeExtraction,
	})

	if !strings.Contains(prompt, "Evaluator Agent") {
		t.Error("Prompt should contain 'Evaluator Agent'")
	}

	if !strings.Contains(prompt, "data actually extracted") {
		t.Error("Prompt should contain extraction-specific criteria")
	}

	if !strings.Contains(prompt, "JSON") {
		t.Error("Prompt should request JSON format")
	}
}

func TestBuildEvaluationPrompt_Navigation(t *testing.T) {
	e := &Evaluator{}

	prompt := e.buildEvaluationPrompt(entity.EvaluationCriteria{
		TaskDescription: "Navigate to example.com",
		ActualResult:    "Navigated successfully",
		AgentType:       entity.AgentTypeNavigation,
	})

	if !strings.Contains(prompt, "navigate") {
		t.Error("Prompt should contain navigation-specific criteria")
	}

	if !strings.Contains(prompt, "parent and child selectors") {
		t.Error("Prompt should mention selector requirements")
	}
}

func TestBuildEvaluationPrompt_Form(t *testing.T) {
	e := &Evaluator{}

	prompt := e.buildEvaluationPrompt(entity.EvaluationCriteria{
		TaskDescription: "Fill login form",
		ActualResult:    "Form filled successfully",
		AgentType:       entity.AgentTypeForm,
	})

	if !strings.Contains(prompt, "form fields filled") {
		t.Error("Prompt should contain form-specific criteria")
	}

	if !strings.Contains(prompt, "confirmation") {
		t.Error("Prompt should ask for confirmation of success")
	}
}
