package rodwrapper

import (
	"strings"
	"testing"
)

// Utility: компактно проверяем включение/исключение
func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

func TestCleanHTML_RemovesScriptStyle(t *testing.T) {
	html := `
<body>
    <div id="main">Hello</div>
    <script>alert("hi")</script>
    <style>.x {}</style>
</body>`

	out := CleanHTMLForAgent(html, &DefaultCleanConfig)

	if contains(out, "<script") || contains(out, "<style") {
		t.Errorf("script/style tags must be removed, output: %s", out)
	}
	if !contains(out, `id="main"`) {
		t.Errorf("expected to keep normal elements")
	}
}

func TestCleanHTML_RemovesComments(t *testing.T) {
	html := `
<body>
    <!-- comment -->
    <div>Text</div>
</body>`

	out := CleanHTMLForAgent(html, &DefaultCleanConfig)

	if contains(out, "comment") {
		t.Errorf("HTML comments must be removed")
	}
}

func TestCleanHTML_KeepsUsefulAttributes(t *testing.T) {
	html := `
<body>
    <a href="https://example.com" class="link" id="x" data-x="1" aria-hidden="true">Go</a>
</body>`

	out := CleanHTMLForAgent(html, &DefaultCleanConfig)

	if !contains(out, `href="https://example.com"`) {
		t.Errorf("href must be kept")
	}
	if !contains(out, `class="link"`) {
		t.Errorf("class must be kept")
	}
	if !contains(out, `id="x"`) {
		t.Errorf("id must be kept")
	}

	if contains(out, `data-x`) {
		t.Errorf("data-* attribute must be removed")
	}
	if contains(out, `aria-hidden`) {
		t.Errorf("aria-* attribute must be removed")
	}
}

func TestCleanHTML_RemovesInlineStyles(t *testing.T) {
	html := `
<body>
    <div style="color:red" class="ok">Hi</div>
</body>`

	out := CleanHTMLForAgent(html, &DefaultCleanConfig)

	if contains(out, "style=") {
		t.Errorf("style attribute must be removed")
	}
	if !contains(out, `class="ok"`) {
		t.Errorf("class must remain")
	}
}

func TestCleanHTML_RemovesMediaGarbageAttributes(t *testing.T) {
	html := `
<body>
    <img src="x.jpg" srcset="a,b,c" sizes="100w" loading="lazy">
</body>`

	out := CleanHTMLForAgent(html, &DefaultCleanConfig)

	if contains(out, `srcset=`) || contains(out, `sizes=`) ||
		contains(out, `loading=`) || contains(out, `decoding=`) {
		t.Errorf("garbage media attributes must be removed")
	}
	if !contains(out, `src="x.jpg"`) {
		t.Errorf("src must remain")
	}
}

func TestCleanHTML_RemovesHeadMetaLink(t *testing.T) {
	html := `
<html>
<head>
    <meta charset="utf-8">
    <link rel="stylesheet" href="x.css">
</head>
<body>
    <p>Hi</p>
</body>
</html>`

	out := CleanHTMLForAgent(html, &DefaultCleanConfig)

	if contains(out, "<head") || contains(out, "<meta") || contains(out, "<link") {
		t.Errorf("head/meta/link must be removed")
	}
	if !contains(out, "<p") {
		t.Errorf("body content must remain")
	}
}

func TestCleanHTML_Truncation(t *testing.T) {
	var big strings.Builder
	big.WriteString("<body>")
	for i := 0; i < 20000; i++ {
		big.WriteString("<div>test</div>")
	}
	big.WriteString("</body>")

	out := CleanHTMLForAgent(big.String(), &DefaultCleanConfig)

	if len(out) > 130500 {
		t.Errorf("output must be truncated near 130 KB")
	}
	if !contains(out, "HTML truncated") {
		t.Errorf("truncation notice must appear")
	}
}
