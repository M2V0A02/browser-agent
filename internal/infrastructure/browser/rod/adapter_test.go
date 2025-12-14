package rod

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Pure unit tests (fast, no browser required)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.False(t, cfg.Headless)
	assert.Equal(t, defaultSlowMotion, cfg.SlowMotion)
	assert.Equal(t, defaultTimeout, cfg.Timeout)
	assert.False(t, cfg.NoSandbox, "Should be secure by default")
	assert.False(t, cfg.DevTools)
	assert.False(t, cfg.DisableSecurityFeatures, "Should be secure by default")
}

func TestPointerToString(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{
			name:     "Nil pointer",
			input:    nil,
			expected: "",
		},
		{
			name:     "Non-nil pointer",
			input:    stringPtr("test"),
			expected: "test",
		},
		{
			name:     "Empty string",
			input:    stringPtr(""),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pointerToString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsXPathSelector(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		expected bool
	}{
		{"XPath with slash", "//div", true},
		{"XPath with parenthesis", "(//div)", true},
		{"XPath with prefix", "xpath=//div", true},
		{"CSS selector", "#test", false},
		{"CSS class", ".test", false},
		{"CSS element", "div", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isXPathSelector(tt.selector)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
