// internal/infrastructure/browser/ui_extractor/ui_extractor.go
package rodwrapper

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-rod/rod"
)

type ExtractConfig struct {
	OnlyInViewport   bool
	MaxElements      int
	PriorityKeywords []string
}

var DefaultConfig = ExtractConfig{
	OnlyInViewport: true,
	MaxElements:    500,
}

type UIElement struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Text       string `json:"text"`
	AriaLabel  string `json:"aria_label,omitempty"`
	Role       string `json:"role,omitempty"`
	Visible    bool   `json:"visible"`
	InViewport bool   `json:"in_viewport"`
	Count      string `json:"count,omitempty"`
	Selector   string `json:"selector"`
}

func ExtractUI(page *Page, cfg *ExtractConfig) ([]UIElement, error) {
	if cfg == nil {
		cfg = &DefaultConfig
	}

	var result []UIElement
	counter := 0
	seen := make(map[string]bool)

	countRe := regexp.MustCompile(`\(([\d\.,\s]+)\)`)

	add := func(el *rod.Element, typ string) {
		if el == nil || counter >= cfg.MaxElements {
			return
		}

		visible, err := el.Visible()
		if err != nil || !visible {
			return
		}

		inViewport := true
		if cfg.OnlyInViewport {
			// rod не имеет IsIntersectingViewport → делаем через JS
			inView, _ := el.Eval(`() => {
				const rect = this.getBoundingClientRect();
				return rect.top < window.innerHeight && rect.bottom >= 0 &&
				rect.left < window.innerWidth && rect.right >= 0;
			}`)
			if b := inView.Value.Bool(); b {
				inViewport = b
			}
		}

		// Фильтр по ключевым словам
		if len(cfg.PriorityKeywords) > 0 {
			text, _ := el.Text()
			aria, _ := el.Attribute("aria-label")
			tooltip, _ := el.Attribute("data-tooltip")
			found := false
			for _, kw := range cfg.PriorityKeywords {
				lower := strings.ToLower
				if strings.Contains(lower(text), lower(kw)) ||
					(aria != nil && strings.Contains(lower(*aria), lower(kw))) ||
					(tooltip != nil && strings.Contains(lower(*tooltip), lower(kw))) {
					found = true
					break
				}
			}
			if !found {
				return
			}
		}

		selector := bestSelector(el)
		if seen[selector] {
			return
		}
		seen[selector] = true

		text, _ := el.Text()
		text = strings.TrimSpace(text)
		aria, _ := el.Attribute("aria-label")
		role, _ := el.Attribute("role")
		tooltip, _ := el.Attribute("data-tooltip")
		title, _ := el.Attribute("title")

		element := UIElement{
			ID:         fmt.Sprintf("ui-%04d", counter),
			Type:       typ,
			Text:       text,
			AriaLabel:  firstNonEmpty([]string{ptrToString(aria), ptrToString(tooltip), ptrToString(title), text}...),
			Role:       ptrToString(role),
			Visible:    true,
			InViewport: inViewport,
			Selector:   selector,
		}

		// Обработка папок со счётчиком
		if typ == "link" || typ == "folder" {
			if m := countRe.FindStringSubmatch(text); len(m) > 1 {
				element.Type = "folder"
				element.Count = strings.ReplaceAll(m[1], " ", "")
				element.Text = strings.TrimSpace(countRe.ReplaceAllString(text, ""))
			}
		}

		result = append(result, element)
		counter++
	}

	// === Исправленные вызовы Elements (всегда проверяем ошибку) ===
	elements, err := page.Elements("button, [role='button'], [data-tooltip], [aria-label]:not([aria-label=''])")
	if err == nil {
		for _, el := range elements {
			add(el, "button")
		}
	}

	elements, err = page.Elements("input[type='checkbox'], [role='checkbox']")
	if err == nil {
		for _, el := range elements {
			add(el, "checkbox")
		}
	}

	elements, err = page.Elements("a")
	if err == nil {
		for _, el := range elements {
			add(el, "link")
		}
	}

	elements, err = page.Elements("[data-test-id], [data-testid]")
	if err == nil {
		for _, el := range elements {
			add(el, "element")
		}
	}

	return result, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func bestSelector(el *rod.Element) string {
	return el.MustElementX("@").String()
}

func ptrToString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
