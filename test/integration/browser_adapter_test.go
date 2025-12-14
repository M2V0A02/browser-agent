package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"browser-agent/internal/domain/entity"
	"browser-agent/internal/infrastructure/browser/rod"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function for string pointer
func stringPtr(s string) *string {
	return &s
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
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	assert.NoError(t, err)
	assert.Equal(t, server.URL+"/", adapter.CurrentURL())
}

func TestBrowserAdapter_Navigate_InvalidURL(t *testing.T) {
	ctx := context.Background()
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
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
			assert.ErrorIs(t, err, rod.ErrInvalidURL)
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
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
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
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
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
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0
	cfg.Timeout = 1 * time.Second

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	err = adapter.Click(ctx, "#nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "element not found")
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
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	err = adapter.Fill(ctx, "#testInput", "Hello World")
	assert.NoError(t, err)
}

func TestBrowserAdapter_Fill_FieldNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><body></body></html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0
	cfg.Timeout = 1 * time.Second

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	err = adapter.Fill(ctx, "#nonexistent", "text")
	assert.Error(t, err)
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
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
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
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
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
		{"Invalid Direction", "invalid", true, rod.ErrInvalidScrollDirection},
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
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
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
	assert.NotNil(t, content.UIElements)
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
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	elements, err := adapter.GetUIElements(ctx)
	require.NoError(t, err)
	assert.NotNil(t, elements)

	if len(elements) > 0 {
		var foundButton bool
		for _, el := range elements {
			assert.NotEmpty(t, el.ID)
			assert.NotEmpty(t, el.Type)
			assert.NotEmpty(t, el.Selector)

			if el.Type == "button" {
				foundButton = true
			}
		}

		assert.True(t, foundButton, "Should find button elements")
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
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
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

func TestBrowserAdapter_CurrentURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><body>Test</body></html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	initialURL := adapter.CurrentURL()
	assert.Equal(t, "about:blank", initialURL)

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	currentURL := adapter.CurrentURL()
	assert.Equal(t, server.URL+"/", currentURL)
}

func TestBrowserAdapter_GetPageText(t *testing.T) {
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
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
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

func TestBrowserAdapter_Search_ID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<body>
	<div id="main-container" class="container" data-test="value1">Main Content</div>
	<button id="submit-button" type="submit">Submit</button>
	<input id="email-input" type="email" name="email">
</body>
</html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	result, err := adapter.Search(ctx, entity.SearchRequest{
		Type:  "id",
		Query: "submit",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Found)
	assert.Equal(t, "id", result.Type)
	assert.Greater(t, len(result.Elements), 0)

	found := false
	for _, elem := range result.Elements {
		if elem.ID == "submit-button" {
			found = true
			assert.Equal(t, "#submit-button", elem.Selector)
			assert.Equal(t, "submit", elem.Attributes["type"])
			break
		}
	}
	assert.True(t, found, "Expected to find submit-button element")
}

func TestBrowserAdapter_Search_Attribute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<body>
	<div data-testid="component-1" class="test">Component 1</div>
	<div data-testid="component-2" class="test">Component 2</div>
	<button data-action="submit" type="submit">Submit</button>
</body>
</html>`)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Navigate(ctx, server.URL)
	require.NoError(t, err)

	result, err := adapter.Search(ctx, entity.SearchRequest{
		Type:  "attribute",
		Query: "data-testid=component",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Found)
	assert.Equal(t, "attribute", result.Type)
	assert.Equal(t, 2, len(result.Elements))

	for _, elem := range result.Elements {
		assert.Contains(t, elem.Attributes["data-testid"], "component")
	}
}

func TestBrowserAdapter_IntegrationScenario(t *testing.T) {
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
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
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

// Browser lifecycle tests (moved from unit tests - require real browser)

func TestNewBrowserAdapter_WithNilContext(t *testing.T) {
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(nil, cfg)
	require.NoError(t, err)
	require.NotNil(t, adapter)
	defer adapter.Close()

	assert.True(t, adapter.IsReady())
}

func TestNewBrowserAdapter_WithZeroTimeout(t *testing.T) {
	ctx := context.Background()
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0
	cfg.Timeout = 0 // Should be auto-corrected

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	assert.Equal(t, rod.DefaultConfig().Timeout, adapter.GetTimeout())
}

func TestBrowserAdapter_IsReady(t *testing.T) {
	ctx := context.Background()
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)

	assert.True(t, adapter.IsReady())

	adapter.Close()
	assert.False(t, adapter.IsReady())
}

func TestBrowserAdapter_SetTimeout(t *testing.T) {
	ctx := context.Background()
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	defaultTimeout := adapter.GetTimeout()
	assert.NotZero(t, defaultTimeout)

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
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
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
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)

	adapter.Close()

	// All operations should return error after close
	err = adapter.Navigate(ctx, "http://example.com")
	assert.ErrorIs(t, err, rod.ErrBrowserNotConnected)

	err = adapter.Click(ctx, "#test")
	assert.ErrorIs(t, err, rod.ErrBrowserNotConnected)

	err = adapter.Fill(ctx, "#test", "text")
	assert.ErrorIs(t, err, rod.ErrBrowserNotConnected)

	err = adapter.PressEnter(ctx)
	assert.ErrorIs(t, err, rod.ErrBrowserNotConnected)

	err = adapter.Scroll(ctx, "down", 0)
	assert.ErrorIs(t, err, rod.ErrBrowserNotConnected)
}

func TestBrowserAdapter_Click_InvalidSelector(t *testing.T) {
	ctx := context.Background()
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	err = adapter.Click(ctx, "")
	assert.Error(t, err)
	assert.ErrorIs(t, err, rod.ErrInvalidSelector)
}

func TestBrowserAdapter_ValidateURL(t *testing.T) {
	ctx := context.Background()
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	tests := []struct {
		name      string
		url       string
		shouldErr bool
	}{
		{"Valid HTTP", "http://example.com", false},
		{"Valid HTTPS", "https://example.com", false},
		{"Valid about:blank", "about:blank", false},
		{"Valid file URL", "file:///path/to/file.html", false},
		{"Empty URL", "", true},
		{"Invalid scheme FTP", "ftp://example.com", true},
		{"Invalid scheme javascript", "javascript:alert(1)", true},
		{"Invalid scheme data", "data:text/html,<html>", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call Navigate instead of validateURL (private method)
			err := adapter.Navigate(ctx, tt.url)
			if tt.shouldErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, rod.ErrInvalidURL)
			} else {
				// For valid URLs, navigation might fail for other reasons
				// but should NOT be ErrInvalidURL
				if err != nil {
					assert.NotErrorIs(t, err, rod.ErrInvalidURL)
				}
			}
		})
	}
}

func TestBrowserAdapter_ValidateSelector(t *testing.T) {
	ctx := context.Background()
	cfg := rod.DefaultConfig()
	cfg.Headless = true
	cfg.SlowMotion = 0

	adapter, err := rod.NewBrowserAdapter(ctx, cfg)
	require.NoError(t, err)
	defer adapter.Close()

	tests := []struct {
		name      string
		selector  string
		shouldErr bool
	}{
		{"Valid ID selector", "#test", false},
		{"Valid class selector", ".test", false},
		{"Valid element selector", "div", false},
		{"Valid complex selector", "div.class#id", false},
		{"Valid XPath", "//div[@id='test']", false},
		{"Empty selector", "", true},
		{"Whitespace only", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call Click with invalid selector to test validation
			err := adapter.Click(ctx, tt.selector)
			if tt.shouldErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, rod.ErrInvalidSelector)
			} else {
				// For valid selectors, click might fail for other reasons
				// but should NOT be ErrInvalidSelector
				if err != nil {
					assert.NotErrorIs(t, err, rod.ErrInvalidSelector)
				}
			}
		})
	}
}
