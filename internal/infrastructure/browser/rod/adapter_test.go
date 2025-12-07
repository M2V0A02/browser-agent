package rod

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.False(t, cfg.Headless)
	assert.Equal(t, defaultSlowMotion, cfg.SlowMotion)
	assert.Equal(t, defaultTimeout, cfg.Timeout)
	assert.False(t, cfg.NoSandbox, "Should be secure by default")
	assert.False(t, cfg.DevTools)
	assert.False(t, cfg.DisableSecurityFeatures, "Should be secure by default")
}

func TestNewBrowserAdapter(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, adapter)
	defer adapter.Close()

	assert.NotNil(t, adapter.browser)
	assert.NotNil(t, adapter.launcher)
	assert.NotNil(t, adapter.page)
	assert.Equal(t, cfg.Timeout, adapter.timeout)
	assert.False(t, adapter.closed)
}

func TestNewBrowserAdapter_WithNilContext(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(nil, cfg)
	require.NoError(t, err)
	require.NotNil(t, adapter)
	defer adapter.Close()

	assert.True(t, adapter.IsReady())
}

func TestNewBrowserAdapter_WithZeroTimeout(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0
	cfg.Timeout = 0 // Should be auto-corrected

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	assert.Equal(t, defaultTimeout, adapter.timeout)
}

func TestBrowserAdapter_IsReady(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)

	assert.True(t, adapter.IsReady())

	adapter.Close()
	assert.False(t, adapter.IsReady())
}

func TestBrowserAdapter_Navigate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body><h1>Hello World</h1></body>
</html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	assert.NoError(t, err)
	assert.Equal(t, server.URL+"/", adapter.CurrentURL())
}

func TestBrowserAdapter_Navigate_InvalidURL(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	tests := []struct {
		name string
		url  string
	}{
		{"Empty URL", ""},
		{"Invalid scheme", "ftp://example.com"},
		{"JavaScript URL", "javascript:alert(1)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := adapter.Navigate(ctx, tt.url)
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidURL)
		})
	}
}

func TestBrowserAdapter_Click(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<body>
	<button id="testBtn">Click Me</button>
	<div id="result"></div>
	<script>
		document.getElementById('testBtn').addEventListener('click', function() {
			document.getElementById('result').textContent = 'Clicked!';
		});
	</script>
</body>
</html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	err = adapter.Click(ctx, "#testBtn")
	assert.NoError(t, err)
}

func TestBrowserAdapter_Click_WithXPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<body>
	<button id="testBtn">Click Me</button>
</body>
</html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	err = adapter.Click(ctx, "//button[@id='testBtn']")
	assert.NoError(t, err)
}

func TestBrowserAdapter_Click_ElementNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><body></body></html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0
	cfg.Timeout = 1 * time.Second

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	err = adapter.Click(ctx, "#nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "element not found")
}

func TestBrowserAdapter_Click_InvalidSelector(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Click(ctx, "")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSelector)
}

func TestBrowserAdapter_Fill(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<body>
	<input id="testInput" type="text" />
</body>
</html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	err = adapter.Fill(ctx, "#testInput", "Hello World")
	assert.NoError(t, err)

	el, err := adapter.page.Element("#testInput")
	require.NoError(t, err)
	value, err := el.Property("value")
	require.NoError(t, err)
	assert.Equal(t, "Hello World", value.String())
}

func TestBrowserAdapter_Fill_FieldNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><body></body></html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0
	cfg.Timeout = 1 * time.Second

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	err = adapter.Fill(ctx, "#nonexistent", "text")
	assert.Error(t, err)
	// Should contain either "field not found" or "timeout finding field"
	assert.True(t,
		strings.Contains(err.Error(), "field not found") ||
		strings.Contains(err.Error(), "timeout finding field"),
		"Error should mention field not found or timeout: %v", err)
}

func TestBrowserAdapter_PressEnter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<body>
	<input id="testInput" type="text" />
</body>
</html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	err = adapter.PressEnter(ctx)
	assert.NoError(t, err)
}

func TestBrowserAdapter_Scroll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<body style="height: 3000px;">
	<h1>Scrollable Content</h1>
	<div style="margin-top: 2000px;">Bottom content</div>
</body>
</html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	tests := []struct {
		name      string
		direction string
		wantErr   bool
		errorType error
	}{
		{"Scroll Down", "down", false, nil},
		{"Scroll Up", "up", false, nil},
		{"Scroll Top", "top", false, nil},
		{"Scroll Bottom", "bottom", false, nil},
		{"Scroll Down Mixed Case", "DOWN", false, nil},
		{"Scroll Down With Spaces", " down ", false, nil},
		{"Invalid Direction", "invalid", true, ErrInvalidScrollDirection},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := adapter.Scroll(ctx, tt.direction, 0)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBrowserAdapter_GetPageContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
	<h1>Hello World</h1>
	<button id="btn1">Click Me</button>
	<input id="input1" type="text" />
	<a href="/page2">Link</a>
</body>
</html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	content, err := adapter.GetPageContent(ctx)
	require.NoError(t, err)
	require.NotNil(t, content)

	assert.Equal(t, server.URL+"/", content.URL)
	assert.Equal(t, "Test Page", content.Title)
	assert.Contains(t, content.HTML, "Hello World")
	assert.NotNil(t, content.UIElements) // Should be at least empty slice, not nil
}

func TestBrowserAdapter_GetUIElements(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<body>
	<button id="btn1">Button 1</button>
	<button id="btn2" aria-label="Second Button">Button 2</button>
	<input id="input1" type="text" />
	<textarea id="textarea1"></textarea>
	<a href="/page" id="link1">Link Text</a>
	<div role="button" id="divBtn">Div Button</div>
</body>
</html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	elements, err := adapter.GetUIElements(ctx)
	require.NoError(t, err)
	assert.NotNil(t, elements)

	if len(elements) > 0 {
		var foundButton, foundInput, foundLink bool
		for _, el := range elements {
			assert.NotEmpty(t, el.ID)
			assert.NotEmpty(t, el.Type)
			assert.NotEmpty(t, el.Selector)
			assert.True(t, el.Visible)

			switch el.Type {
			case "button":
				foundButton = true
			case "input":
				foundInput = true
			case "link":
				foundLink = true
			}
		}

		assert.True(t, foundButton, "Should find button elements")
		assert.True(t, foundInput, "Should find input elements")
		assert.True(t, foundLink, "Should find link elements")
	}
}

func TestBrowserAdapter_Screenshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<body style="background-color: red; width: 800px; height: 600px;">
	<h1>Screenshot Test</h1>
</body>
</html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	screenshot, err := adapter.Screenshot(ctx)
	require.NoError(t, err)
	require.NotNil(t, screenshot)

	assert.NotEmpty(t, screenshot.Data)
	assert.Equal(t, "jpeg", screenshot.Format)
	assert.Greater(t, screenshot.Width, 0)
	assert.Greater(t, screenshot.Height, 0)
}

func TestBrowserAdapter_Screenshot_Resize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<body style="width: 2000px; height: 1500px;">
	<h1>Large Page</h1>
</body>
</html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	screenshot, err := adapter.Screenshot(ctx)
	require.NoError(t, err)

	assert.LessOrEqual(t, screenshot.Width, screenshotMaxWidth, "Width should be resized to max 1024")
}

func TestBrowserAdapter_CurrentURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><body>Test</body></html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	initialURL := adapter.CurrentURL()
	assert.Equal(t, "about:blank", initialURL)

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	currentURL := adapter.CurrentURL()
	assert.Equal(t, server.URL+"/", currentURL)
}

func TestBrowserAdapter_SetTimeout(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	assert.Equal(t, defaultTimeout, adapter.GetTimeout())

	newTimeout := 5 * time.Second
	adapter.SetTimeout(newTimeout)
	assert.Equal(t, newTimeout, adapter.GetTimeout())

	// Should ignore invalid timeout
	adapter.SetTimeout(0)
	assert.Equal(t, newTimeout, adapter.GetTimeout())

	adapter.SetTimeout(-1 * time.Second)
	assert.Equal(t, newTimeout, adapter.GetTimeout())
}

func TestBrowserAdapter_Close(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, adapter)

	assert.True(t, adapter.IsReady())

	adapter.Close()
	assert.False(t, adapter.IsReady())

	// Should not panic on second close
	assert.NotPanics(t, func() {
		adapter.Close()
	})
}

func TestBrowserAdapter_ClosedState(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)

	adapter.Close()

	// All operations should return error after close
	err = adapter.Navigate(ctx, "http://example.com")
	assert.ErrorIs(t, err, ErrBrowserNotConnected)

	err = adapter.Click(ctx, "#test")
	assert.ErrorIs(t, err, ErrBrowserNotConnected)

	err = adapter.Fill(ctx, "#test", "text")
	assert.ErrorIs(t, err, ErrBrowserNotConnected)

	err = adapter.PressEnter(ctx)
	assert.ErrorIs(t, err, ErrBrowserNotConnected)

	err = adapter.Scroll(ctx, "down", 0)
	assert.ErrorIs(t, err, ErrBrowserNotConnected)
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

func TestBrowserAdapter_IntegrationScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><title>Integration Test</title></head>
<body>
	<h1>Welcome</h1>
	<input id="searchBox" type="text" />
	<button id="searchBtn">Search</button>
	<div id="results"></div>
	<script>
		document.getElementById('searchBtn').addEventListener('click', function() {
			const query = document.getElementById('searchBox').value;
			document.getElementById('results').textContent = 'Results for: ' + query;
		});
	</script>
</body>
</html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	// Navigate
	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	// Fill search box
	err = adapter.Fill(ctx, "#searchBox", "test query")
	require.NoError(t, err)

	// Click search button
	err = adapter.Click(ctx, "#searchBtn")
	require.NoError(t, err)

	// Get page content
	content, err := adapter.GetPageContent(ctx)
	require.NoError(t, err)
	assert.Contains(t, content.HTML, "Results for: test query")

	// Take screenshot
	screenshot, err := adapter.Screenshot(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, screenshot.Data)

	// Scroll page
	err = adapter.Scroll(ctx, "bottom", 0)
	require.NoError(t, err)

	// Get UI elements
	elements, err := adapter.GetUIElements(ctx)
	require.NoError(t, err)
	assert.NotNil(t, elements)
}

func BenchmarkBrowserAdapter_Navigate(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><body>Test</body></html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(b, err)
	defer adapter.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = adapter.Navigate(ctx, server.URL)
	}
}

func BenchmarkBrowserAdapter_GetUIElements(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		html := `<!DOCTYPE html><html><body>`
		for i := 0; i < 100; i++ {
			html += fmt.Sprintf(`<button id="btn%d">Button %d</button>`, i, i)
		}
		html += `</body></html>`
		fmt.Fprint(w, html)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(b, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = adapter.GetUIElements(ctx)
	}
}

func TestBrowserAdapter_GetPageText(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
	<style>body { color: red; }</style>
</head>
<body>
	<h1>Hello World</h1>
	<p>This is a test paragraph.</p>
	<script>console.log('test');</script>
	<div style="display:none">Hidden content</div>
</body>
</html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	text, err := adapter.GetPageText(ctx)
	require.NoError(t, err)

	assert.Contains(t, text, "Hello World")
	assert.Contains(t, text, "This is a test paragraph")
	assert.NotContains(t, text, "<h1>")
	assert.NotContains(t, text, "<script>")
	assert.NotContains(t, text, "console.log")
}
