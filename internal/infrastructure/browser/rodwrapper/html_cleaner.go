package rodwrapper

import (
	"log"
	"strings"

	"golang.org/x/net/html"
)

type CleanConfig struct {
	TagsToRemove     []string
	AttrsToRemove    []string
	MaxOutputSize    int
	CustomAttrFilter func(attr html.Attribute) bool
}

// DefaultCleanConfig — дефолтная конфигурация
var DefaultCleanConfig = CleanConfig{
	TagsToRemove: []string{
		"script", "style", "noscript", "svg", "iframe",
		"link", "meta", "head", "title",
	},
	AttrsToRemove: []string{
		"style", "srcset", "sizes", "loading", "decoding", "fetchpriority", "tabindex",
	},
	MaxOutputSize: 130_000,
}

// CleanHTMLForAgent очищает HTML для агента, оставляя только нужное
func CleanHTMLForAgent(rawHTML string, cfg *CleanConfig) string {
	if cfg == nil {
		cfg = &DefaultCleanConfig
	}

	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		log.Printf("HTML parse error: %v", err)
		return rawHTML // fallback
	}

	body := findBodyNode(doc)
	if body == nil {
		log.Printf("No <body> found in HTML")
		return rawHTML
	}

	cleanNode(body, cfg)

	result := renderNode(body)
	result = truncateHTML(result, cfg.MaxOutputSize)

	return result
}

// findBodyNode ищет <body> в дереве HTML
func findBodyNode(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "body" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if b := findBodyNode(c); b != nil {
			return b
		}
	}
	return nil
}

// cleanNode рекурсивно удаляет комментарии, мусорные теги и фильтрует атрибуты
func cleanNode(n *html.Node, cfg *CleanConfig) {
	if n.Type == html.CommentNode {
		if n.Parent != nil {
			n.Parent.RemoveChild(n)
		}
		return
	}
	if n.Type != html.ElementNode {
		return
	}

	if isOneOf(n.Data, cfg.TagsToRemove...) {
		if n.Parent != nil {
			n.Parent.RemoveChild(n)
		}
		return
	}

	n.Attr = filterAttributes(n.Attr, cfg)

	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		cleanNode(c, cfg)
		c = next
	}
}

// filterAttributes фильтрует атрибуты узла на основе конфигурации
func filterAttributes(attrs []html.Attribute, cfg *CleanConfig) []html.Attribute {
	var kept []html.Attribute
	for _, attr := range attrs {
		if shouldRemoveAttr(attr, cfg) {
			continue
		}
		kept = append(kept, attr)
	}
	return kept
}

// shouldRemoveAttr проверяет, нужно ли удалить атрибут
func shouldRemoveAttr(attr html.Attribute, cfg *CleanConfig) bool {
	key := attr.Key
	for _, r := range cfg.AttrsToRemove {
		if key == r {
			return true
		}
	}
	if strings.HasPrefix(key, "data-") || strings.HasPrefix(key, "aria-") || strings.HasPrefix(key, "on") {
		return true
	}
	if cfg.CustomAttrFilter != nil && cfg.CustomAttrFilter(attr) {
		return true
	}
	return false
}

// renderNode преобразует узел обратно в HTML
func renderNode(n *html.Node) string {
	var sb strings.Builder
	_ = html.Render(&sb, n)
	return sb.String()
}

// truncateHTML обрезает HTML до maxSize, если нужно
func truncateHTML(htmlStr string, maxSize int) string {
	if len(htmlStr) > maxSize {
		return htmlStr[:maxSize] + "\n<!-- HTML truncated to fit token limit -->"
	}
	return htmlStr
}

// isOneOf проверяет, что s совпадает с одним из candidates
func isOneOf(s string, candidates ...string) bool {
	for _, c := range candidates {
		if s == c {
			return true
		}
	}
	return false
}
