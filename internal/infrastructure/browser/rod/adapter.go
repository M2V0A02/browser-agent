package rod

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"net/url"
	"strings"
	"sync"
	"time"

	"browser-agent/internal/application/port/output"
	"browser-agent/internal/domain/entity"

	"github.com/disintegration/imaging"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/gson"
)

var _ output.BrowserPort = (*BrowserAdapter)(nil)

const (
	defaultTimeout     = 10 * time.Second
	defaultSlowMotion  = 500 * time.Millisecond
	navigationWaitTime = 5 * time.Second
	clickWaitTime      = 2 * time.Second
	enterWaitTime      = 1 * time.Second
	scrollWaitTime     = 800 * time.Millisecond

	maxUIElements = 100

	screenshotMaxWidth      = 1024
	screenshotQuality       = 75
	screenshotFormatQuality = 80

	schemeHTTP  = "http"
	schemeHTTPS = "https"
)

var (
	ErrBrowserNotInitialized  = errors.New("browser is not initialized")
	ErrPageNotInitialized     = errors.New("page is not initialized")
	ErrInvalidURL             = errors.New("invalid URL")
	ErrInvalidSelector        = errors.New("invalid selector")
	ErrElementNotFound        = errors.New("element not found")
	ErrBrowserNotConnected    = errors.New("browser is not connected")
	ErrContextCanceled        = errors.New("context was canceled")
	ErrInvalidScrollDirection = errors.New("invalid scroll direction")
)

type BrowserAdapter struct {
	browser  *rod.Browser
	launcher *launcher.Launcher
	page     *rod.Page
	timeout  time.Duration
	mu       sync.RWMutex
	closed   bool
}

type BrowserConfig struct {
	Headless                bool
	SlowMotion              time.Duration
	Timeout                 time.Duration
	NoSandbox               bool
	DevTools                bool
	DisableSecurityFeatures bool
}

func DefaultConfig() BrowserConfig {
	return BrowserConfig{
		Headless:                false,
		SlowMotion:              defaultSlowMotion,
		Timeout:                 defaultTimeout,
		NoSandbox:               false,
		DevTools:                false,
		DisableSecurityFeatures: false,
	}
}

func NewBrowserAdapter(ctx context.Context, config BrowserConfig) (*BrowserAdapter, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if config.Timeout <= 0 {
		config.Timeout = defaultTimeout
	}

	launcherInstance := launcher.New().
		Headless(config.Headless).
		Devtools(config.DevTools).
		NoSandbox(config.NoSandbox).
		Delete("use-mock-keychain")

	if config.DisableSecurityFeatures {
		launcherInstance = launcherInstance.
			Set("disable-web-security").
			Set("allow-running-insecure-content")
	}

	launchURL, err := launcherInstance.Context(ctx).Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().
		ControlURL(launchURL).
		Trace(true).
		SlowMotion(config.SlowMotion).
		MustConnect()

	page := browser.MustPage("about:blank")

	adapter := &BrowserAdapter{
		browser:  browser,
		launcher: launcherInstance,
		page:     page,
		timeout:  config.Timeout,
		closed:   false,
	}

	return adapter, nil
}

func (b *BrowserAdapter) IsReady() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed || b.browser == nil || b.page == nil {
		return false
	}

	_, err := b.page.Info()
	return err == nil
}

func (b *BrowserAdapter) Navigate(ctx context.Context, targetURL string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := b.validateURL(targetURL); err != nil {
		return err
	}

	if err := b.checkState(); err != nil {
		return err
	}

	if err := b.page.Context(ctx).Navigate(targetURL); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%w: %v", ErrContextCanceled, ctx.Err())
		}
		return fmt.Errorf("navigation failed: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, navigationWaitTime)
	defer cancel()

	if err := b.page.Context(waitCtx).WaitLoad(); err != nil {
		if waitCtx.Err() != nil {
			return fmt.Errorf("wait load timeout: %w", err)
		}
		return fmt.Errorf("wait load failed: %w", err)
	}

	_ = b.page.Context(ctx).WaitIdle(navigationWaitTime)

	return nil
}

func (b *BrowserAdapter) Click(ctx context.Context, selector string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := b.validateSelector(selector); err != nil {
		return err
	}

	if err := b.checkState(); err != nil {
		return err
	}

	element, err := b.findElement(ctx, selector)
	if err != nil {
		return fmt.Errorf("element not found for selector %q: %w", selector, err)
	}

	if err := element.Context(ctx).Click(proto.InputMouseButtonLeft, 1); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%w: %v", ErrContextCanceled, ctx.Err())
		}
		return fmt.Errorf("click failed: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, clickWaitTime)
	defer cancel()
	_ = b.page.Context(waitCtx).WaitIdle(clickWaitTime)

	return nil
}

func (b *BrowserAdapter) ClickWithChanges(ctx context.Context, selector string) (*entity.ClickResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := b.validateSelector(selector); err != nil {
		return &entity.ClickResult{Success: false, Error: err.Error()}, err
	}

	if err := b.checkState(); err != nil {
		return &entity.ClickResult{Success: false, Error: err.Error()}, err
	}

	beforeURL := b.CurrentURL()
	beforeElements, _ := b.GetUIElements(ctx)
	beforeCount := len(beforeElements)

	element, err := b.findElement(ctx, selector)
	if err != nil {
		return &entity.ClickResult{
			Success: false,
			Error:   fmt.Sprintf("element not found for selector %q: %v", selector, err),
		}, err
	}

	if err := element.Context(ctx).Click(proto.InputMouseButtonLeft, 1); err != nil {
		if ctx.Err() != nil {
			return &entity.ClickResult{Success: false, Error: "context canceled"}, ErrContextCanceled
		}
		return &entity.ClickResult{Success: false, Error: fmt.Sprintf("click failed: %v", err)}, err
	}

	waitCtx, cancel := context.WithTimeout(ctx, clickWaitTime)
	defer cancel()
	_ = b.page.Context(waitCtx).WaitIdle(clickWaitTime)

	afterURL := b.CurrentURL()
	afterElements, _ := b.GetUIElements(ctx)
	afterCount := len(afterElements)

	changes := &entity.PageChanges{
		URLChanged: beforeURL != afterURL,
		NewURL:     afterURL,
	}

	if afterCount > beforeCount {
		newElements := []entity.UIElement{}
		beforeMap := make(map[string]bool)
		for _, el := range beforeElements {
			beforeMap[el.Selector] = true
		}
		for _, el := range afterElements {
			if !beforeMap[el.Selector] {
				newElements = append(newElements, el)
				if el.Role == "dialog" || strings.Contains(strings.ToLower(el.Text), "modal") {
					changes.ModalOpened = true
				}
			}
		}
		changes.NewElements = newElements
	} else if afterCount < beforeCount {
		changes.ElementsRemoved = beforeCount - afterCount
		changes.ModalClosed = true
	}

	return &entity.ClickResult{
		Success: true,
		Changes: changes,
	}, nil
}

func (b *BrowserAdapter) BatchClick(ctx context.Context, selectors []string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := b.checkState(); err != nil {
		return err
	}

	for i, selector := range selectors {
		if err := b.validateSelector(selector); err != nil {
			return fmt.Errorf("invalid selector at index %d (%q): %w", i, selector, err)
		}

		element, err := b.findElement(ctx, selector)
		if err != nil {
			return fmt.Errorf("element not found at index %d for selector %q: %w", i, selector, err)
		}

		if err := element.Context(ctx).Click(proto.InputMouseButtonLeft, 1); err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("click %d/%d failed: %w", i+1, len(selectors), ErrContextCanceled)
			}
			return fmt.Errorf("click %d/%d failed on selector %q: %w", i+1, len(selectors), selector, err)
		}

		time.Sleep(300 * time.Millisecond)
	}

	waitCtx, cancel := context.WithTimeout(ctx, clickWaitTime)
	defer cancel()
	_ = b.page.Context(waitCtx).WaitIdle(clickWaitTime)

	return nil
}

func (b *BrowserAdapter) Fill(ctx context.Context, selector, text string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := b.validateSelector(selector); err != nil {
		return err
	}

	if err := b.checkState(); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	element, err := b.page.Context(timeoutCtx).Element(selector)
	if err != nil {
		if timeoutCtx.Err() != nil {
			return fmt.Errorf("timeout finding field %q: %w", selector, err)
		}
		return fmt.Errorf("field not found %q: %w", selector, err)
	}

	if err := element.Context(ctx).SelectAllText(); err == nil {
		if err := element.Context(ctx).Input(""); err != nil {
			return fmt.Errorf("failed to clear field: %w", err)
		}
	}

	if err := element.Context(ctx).Input(text); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%w: %v", ErrContextCanceled, ctx.Err())
		}
		return fmt.Errorf("input failed: %w", err)
	}

	return nil
}

func (b *BrowserAdapter) BatchFill(ctx context.Context, fields map[string]string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := b.checkState(); err != nil {
		return err
	}

	for selector, text := range fields {
		if err := b.validateSelector(selector); err != nil {
			return fmt.Errorf("invalid selector %q: %w", selector, err)
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, b.timeout)
		element, err := b.page.Context(timeoutCtx).Element(selector)
		cancel()

		if err != nil {
			return fmt.Errorf("field not found %q: %w", selector, err)
		}

		if err := element.Context(ctx).SelectAllText(); err == nil {
			if err := element.Context(ctx).Input(""); err != nil {
				return fmt.Errorf("failed to clear field %q: %w", selector, err)
			}
		}

		if err := element.Context(ctx).Input(text); err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("input failed for %q: %w", selector, ErrContextCanceled)
			}
			return fmt.Errorf("input failed for %q: %w", selector, err)
		}
	}

	return nil
}

func (b *BrowserAdapter) PressEnter(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := b.checkState(); err != nil {
		return err
	}

	bodyElement, err := b.page.Context(ctx).Element("body")
	if err != nil {
		return fmt.Errorf("failed to find body element: %w", err)
	}

	if err := bodyElement.Context(ctx).Input("\n"); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%w: %v", ErrContextCanceled, ctx.Err())
		}
		return fmt.Errorf("failed to press Enter: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, enterWaitTime)
	defer cancel()
	_ = b.page.Context(waitCtx).WaitIdle(enterWaitTime)

	return nil
}

type ScrollDirection string

const (
	ScrollDown   ScrollDirection = "down"
	ScrollUp     ScrollDirection = "up"
	ScrollTop    ScrollDirection = "top"
	ScrollBottom ScrollDirection = "bottom"
)

func (b *BrowserAdapter) Scroll(ctx context.Context, direction string, amount int) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := b.checkState(); err != nil {
		return err
	}

	normalizedDirection := ScrollDirection(strings.ToLower(strings.TrimSpace(direction)))

	scrollScripts := map[ScrollDirection]string{
		ScrollDown:   "() => window.scrollBy(0, window.innerHeight * 2)",
		ScrollUp:     "() => window.scrollBy(0, -window.innerHeight * 2)",
		ScrollTop:    "() => window.scrollTo(0, 0)",
		ScrollBottom: "() => window.scrollTo(0, document.body.scrollHeight)",
	}

	script, exists := scrollScripts[normalizedDirection]
	if !exists {
		return fmt.Errorf("%w: %s (valid: down, up, top, bottom)", ErrInvalidScrollDirection, direction)
	}

	_, err := b.page.Context(ctx).Eval(script)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%w: %v", ErrContextCanceled, ctx.Err())
		}
		return fmt.Errorf("scroll failed: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, scrollWaitTime)
	defer cancel()
	_ = b.page.Context(waitCtx).WaitIdle(scrollWaitTime)

	return nil
}

func (b *BrowserAdapter) GetPageContent(ctx context.Context) (*entity.PageContent, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := b.checkState(); err != nil {
		return nil, err
	}

	info, err := b.page.Context(ctx).Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get page info: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	bodyElement, err := b.page.Context(timeoutCtx).Element("body")
	if err != nil {
		return nil, fmt.Errorf("body not found: %w", err)
	}

	html, err := bodyElement.HTML()
	if err != nil {
		return nil, fmt.Errorf("failed to get HTML: %w", err)
	}

	elements, err := b.GetUIElements(ctx)
	if err != nil {
		elements = []entity.UIElement{}
	}

	return &entity.PageContent{
		URL:        info.URL,
		Title:      info.Title,
		HTML:       html,
		UIElements: elements,
	}, nil
}

func (b *BrowserAdapter) GetPageText(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := b.checkState(); err != nil {
		return "", err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	result, err := b.page.Context(timeoutCtx).Eval(`() => document.body.innerText`)
	if err != nil {
		return "", fmt.Errorf("failed to get page text: %w", err)
	}

	var text string
	if err := result.Value.Unmarshal(&text); err != nil {
		return "", fmt.Errorf("failed to unmarshal text: %w", err)
	}

	return text, nil
}

func (b *BrowserAdapter) GetUIElements(ctx context.Context) ([]entity.UIElement, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := b.checkState(); err != nil {
		return nil, err
	}

	collector := newElementCollector(maxUIElements)

	_ = b.collectElementsByType(ctx, collector, "button",
		"button, [role='button'], [data-tooltip], [aria-label]:not([aria-label=''])")

	_ = b.collectElementsByType(ctx, collector, "input", "input, textarea")

	_ = b.collectElementsByType(ctx, collector, "link", "a")

	return collector.getElements(), nil
}

func (b *BrowserAdapter) GetPageContext(ctx context.Context) (*entity.PageContext, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := b.checkState(); err != nil {
		return nil, err
	}

	info, err := b.page.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get page info: %w", err)
	}

	visibleElements, err := b.GetUIElements(ctx)
	if err != nil {
		visibleElements = []entity.UIElement{}
	}

	pageText, err := b.GetPageText(ctx)
	if err != nil {
		pageText = ""
	}

	const maxTextLength = 1000
	if len(pageText) > maxTextLength {
		pageText = pageText[:maxTextLength] + "..."
	}

	return &entity.PageContext{
		URL:             info.URL,
		Title:           info.Title,
		VisibleElements: visibleElements,
		TextContent:     pageText,
		ElementCount:    len(visibleElements),
	}, nil
}

func (b *BrowserAdapter) QueryElements(ctx context.Context, req entity.QueryElementsRequest) (*entity.QueryElementsResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := b.checkState(); err != nil {
		return nil, err
	}

	if req.Limit <= 0 {
		req.Limit = 20
	}

	if req.Limit > 100 {
		req.Limit = 100
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	jsCode := `(selector, limit, extractConfig) => {
		const found = document.querySelectorAll(selector);
		const elements = Array.from(found).slice(0, limit);

		return elements.map((element, index) => {
			const data = {};

			for (const [subSelector, extractType] of Object.entries(extractConfig)) {
				try {
					const targetEl = (subSelector === '_self') ? element : element.querySelector(subSelector);

					if (!targetEl && subSelector !== '_self') {
						data[subSelector] = '';
						continue;
					}

					if (extractType === 'text') {
						data[subSelector] = targetEl.innerText?.trim() || '';
					} else if (extractType === 'html') {
						data[subSelector] = targetEl.innerHTML || '';
					} else if (extractType === 'selector') {
						// Возвращаем селектор элемента для последующего клика
						if (targetEl.id) {
							data[subSelector] = '#' + targetEl.id;
						} else if (targetEl.className) {
							const classes = targetEl.className.split(' ').filter(c => c).join('.');
							if (classes) {
								data[subSelector] = targetEl.tagName.toLowerCase() + '.' + classes;
							} else {
								data[subSelector] = targetEl.tagName.toLowerCase();
							}
						} else {
							data[subSelector] = targetEl.tagName.toLowerCase();
						}
					} else if (extractType.startsWith('attr:')) {
						const attrName = extractType.substring(5);
						data[subSelector] = targetEl.getAttribute(attrName) || '';
					}
				} catch (e) {
					data[subSelector] = '';
				}
			}

			let elementSelector = '';
			if (element.id) {
				elementSelector = '#' + element.id;
			} else if (element.className) {
				const classes = element.className.split(' ').filter(c => c).join('.');
				if (classes) {
					elementSelector = element.tagName.toLowerCase() + '.' + classes;
				}
			}

			return {
				index: index,
				selector: elementSelector,
				data: data
			};
		});
	}`

	result, err := b.page.Context(timeoutCtx).Eval(jsCode, req.Selector, req.Limit, req.Extract)
	if err != nil {
		return nil, fmt.Errorf("failed to query elements: %w", err)
	}

	var rawResults []map[string]interface{}
	if err := result.Value.Unmarshal(&rawResults); err != nil {
		return nil, fmt.Errorf("failed to unmarshal query results: %w", err)
	}

	elements := make([]entity.ElementData, 0, len(rawResults))
	for _, raw := range rawResults {
		index := 0
		if idx, ok := raw["index"].(float64); ok {
			index = int(idx)
		}

		selector := ""
		if sel, ok := raw["selector"].(string); ok {
			selector = sel
		}

		data := make(map[string]string)
		if dataMap, ok := raw["data"].(map[string]interface{}); ok {
			for k, v := range dataMap {
				if strVal, ok := v.(string); ok {
					data[k] = strVal
				}
			}
		}

		elements = append(elements, entity.ElementData{
			Index:    index,
			Selector: selector,
			Data:     data,
		})
	}

	return &entity.QueryElementsResult{
		Elements: elements,
		Count:    len(elements),
	}, nil
}

func (b *BrowserAdapter) Search(ctx context.Context, req entity.SearchRequest) (*entity.SearchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := b.checkState(); err != nil {
		return nil, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	switch req.Type {
	case "text":
		return b.searchByText(timeoutCtx, req.Query)
	case "id":
		return b.searchByID(timeoutCtx, req.Query)
	case "attribute":
		return b.searchByAttribute(timeoutCtx, req.Query)
	default:
		return nil, fmt.Errorf("invalid search type: %s, must be 'text', 'id', or 'attribute'", req.Type)
	}
}

func (b *BrowserAdapter) searchByText(ctx context.Context, query string) (*entity.SearchResult, error) {
	jsCode := `(searchText) => {
		const walker = document.createTreeWalker(
			document.body,
			NodeFilter.SHOW_TEXT,
			{
				acceptNode: function(node) {
					if (node.nodeValue.trim().length === 0) return NodeFilter.FILTER_REJECT;
					const parent = node.parentElement;
					if (!parent) return NodeFilter.FILTER_REJECT;
					const style = window.getComputedStyle(parent);
					if (style.display === 'none' || style.visibility === 'hidden') return NodeFilter.FILTER_REJECT;
					return NodeFilter.FILTER_ACCEPT;
				}
			}
		);

		const searchLower = searchText.toLowerCase();
		const matches = [];
		let node;

		while (node = walker.nextNode()) {
			const text = node.nodeValue;
			const textLower = text.toLowerCase();
			const index = textLower.indexOf(searchLower);

			if (index !== -1) {
				const start = Math.max(0, index - 100);
				const end = Math.min(text.length, index + searchText.length + 100);
				const context = text.substring(start, end);
				matches.push((start > 0 ? '...' : '') + context + (end < text.length ? '...' : ''));
			}
		}

		return matches;
	}`

	result, err := b.page.Context(ctx).Eval(jsCode, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search text: %w", err)
	}

	var matches []string
	if err := result.Value.Unmarshal(&matches); err != nil {
		return nil, fmt.Errorf("failed to unmarshal text search results: %w", err)
	}

	if len(matches) == 0 {
		return &entity.SearchResult{
			Type:    "text",
			Found:   false,
			Content: "",
		}, nil
	}

	totalLength := 0
	content := ""
	for i, match := range matches {
		if totalLength+len(match) > 1000 {
			break
		}
		if i > 0 {
			content += "\n---\n"
			totalLength += 5
		}
		content += match
		totalLength += len(match)
	}

	return &entity.SearchResult{
		Type:    "text",
		Found:   true,
		Content: content,
	}, nil
}

func (b *BrowserAdapter) searchByID(ctx context.Context, id string) (*entity.SearchResult, error) {
	jsCode := `(id) => {
		const elements = document.querySelectorAll('[id*="' + id + '"]');
		return Array.from(elements).map(el => {
			const attrs = {};
			for (const attr of el.attributes) {
				attrs[attr.name] = attr.value;
			}

			let selector = '';
			if (el.id) {
				selector = '#' + el.id;
			} else if (el.className) {
				const classes = el.className.split(' ').filter(c => c).join('.');
				if (classes) {
					selector = el.tagName.toLowerCase() + '.' + classes;
				} else {
					selector = el.tagName.toLowerCase();
				}
			} else {
				selector = el.tagName.toLowerCase();
			}

			return {
				id: el.id,
				selector: selector,
				attributes: attrs
			};
		});
	}`

	result, err := b.page.Context(ctx).Eval(jsCode, id)
	if err != nil {
		return nil, fmt.Errorf("failed to search by id: %w", err)
	}

	var rawElements []map[string]interface{}
	if err := result.Value.Unmarshal(&rawElements); err != nil {
		return nil, fmt.Errorf("failed to unmarshal id search results: %w", err)
	}

	if len(rawElements) == 0 {
		return &entity.SearchResult{
			Type:     "id",
			Found:    false,
			Elements: []entity.SearchElement{},
		}, nil
	}

	elements := make([]entity.SearchElement, 0, len(rawElements))
	for _, raw := range rawElements {
		elem := entity.SearchElement{
			Attributes: make(map[string]string),
		}

		if id, ok := raw["id"].(string); ok {
			elem.ID = id
		}

		if selector, ok := raw["selector"].(string); ok {
			elem.Selector = selector
		}

		if attrs, ok := raw["attributes"].(map[string]interface{}); ok {
			for k, v := range attrs {
				if strVal, ok := v.(string); ok {
					elem.Attributes[k] = strVal
				}
			}
		}

		elements = append(elements, elem)
	}

	return &entity.SearchResult{
		Type:     "id",
		Found:    true,
		Elements: elements,
	}, nil
}

func (b *BrowserAdapter) searchByAttribute(ctx context.Context, query string) (*entity.SearchResult, error) {
	jsCode := `(attrQuery) => {
		const parts = attrQuery.split('=');
		const attrName = parts[0].trim();
		const attrValue = parts.length > 1 ? parts[1].trim() : '';

		let elements;
		if (attrValue) {
			elements = document.querySelectorAll('[' + attrName + '*="' + attrValue + '"]');
		} else {
			elements = document.querySelectorAll('[' + attrName + ']');
		}

		return Array.from(elements).map(el => {
			const attrs = {};
			for (const attr of el.attributes) {
				attrs[attr.name] = attr.value;
			}

			let selector = '';
			if (el.id) {
				selector = '#' + el.id;
			} else if (el.className) {
				const classes = el.className.split(' ').filter(c => c).join('.');
				if (classes) {
					selector = el.tagName.toLowerCase() + '.' + classes;
				} else {
					selector = el.tagName.toLowerCase();
				}
			} else {
				selector = el.tagName.toLowerCase();
			}

			return {
				id: el.id,
				selector: selector,
				attributes: attrs
			};
		});
	}`

	result, err := b.page.Context(ctx).Eval(jsCode, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search by attribute: %w", err)
	}

	var rawElements []map[string]interface{}
	if err := result.Value.Unmarshal(&rawElements); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attribute search results: %w", err)
	}

	if len(rawElements) == 0 {
		return &entity.SearchResult{
			Type:     "attribute",
			Found:    false,
			Elements: []entity.SearchElement{},
		}, nil
	}

	elements := make([]entity.SearchElement, 0, len(rawElements))
	for _, raw := range rawElements {
		elem := entity.SearchElement{
			Attributes: make(map[string]string),
		}

		if id, ok := raw["id"].(string); ok {
			elem.ID = id
		}

		if selector, ok := raw["selector"].(string); ok {
			elem.Selector = selector
		}

		if attrs, ok := raw["attributes"].(map[string]interface{}); ok {
			for k, v := range attrs {
				if strVal, ok := v.(string); ok {
					elem.Attributes[k] = strVal
				}
			}
		}

		elements = append(elements, elem)
	}

	return &entity.SearchResult{
		Type:     "attribute",
		Found:    true,
		Elements: elements,
	}, nil
}

func (b *BrowserAdapter) Screenshot(ctx context.Context) (*entity.Screenshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := b.checkState(); err != nil {
		return nil, err
	}

	imageBytes, err := b.page.Context(ctx).Screenshot(true, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: gson.Int(screenshotFormatQuality),
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %v", ErrContextCanceled, ctx.Err())
		}
		return nil, fmt.Errorf("screenshot failed: %w", err)
	}

	return b.processScreenshot(imageBytes)
}

func (b *BrowserAdapter) CurrentURL() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.page == nil {
		return ""
	}

	info, err := b.page.Info()
	if err != nil {
		return ""
	}

	return info.URL
}

func (b *BrowserAdapter) SetTimeout(timeout time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if timeout > 0 {
		b.timeout = timeout
	}
}

func (b *BrowserAdapter) GetTimeout() time.Duration {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.timeout
}

func (b *BrowserAdapter) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.closed = true

	if b.browser != nil {
		_ = b.browser.Close()
		b.browser = nil
	}

	if b.launcher != nil {
		b.launcher.Kill()
		b.launcher.Cleanup()
		b.launcher = nil
	}

	b.page = nil
}

func (b *BrowserAdapter) checkState() error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return ErrBrowserNotConnected
	}

	if b.browser == nil {
		return ErrBrowserNotInitialized
	}

	if b.page == nil {
		return ErrPageNotInitialized
	}

	return nil
}

func (b *BrowserAdapter) validateURL(targetURL string) error {
	if strings.TrimSpace(targetURL) == "" {
		return fmt.Errorf("%w: URL cannot be empty", ErrInvalidURL)
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != schemeHTTP && scheme != schemeHTTPS && scheme != "about" {
		return fmt.Errorf("%w: unsupported scheme %q (only http, https, about allowed)",
			ErrInvalidURL, scheme)
	}

	return nil
}

func (b *BrowserAdapter) validateSelector(selector string) error {
	if strings.TrimSpace(selector) == "" {
		return fmt.Errorf("%w: selector cannot be empty", ErrInvalidSelector)
	}
	return nil
}

func (b *BrowserAdapter) findElement(ctx context.Context, selector string) (*rod.Element, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	var element *rod.Element
	var err error

	if isXPathSelector(selector) {
		element, err = b.page.Context(timeoutCtx).ElementX(selector)
	} else {
		element, err = b.page.Context(timeoutCtx).Element(selector)
	}

	if err != nil {
		if timeoutCtx.Err() != nil {
			return nil, fmt.Errorf("timeout: %w", err)
		}
		return nil, fmt.Errorf("%w: %v", ErrElementNotFound, err)
	}

	return element, nil
}

func isXPathSelector(selector string) bool {
	selector = strings.TrimSpace(selector)
	return strings.HasPrefix(selector, "/") ||
		strings.HasPrefix(selector, "(") ||
		strings.Contains(selector, "xpath=")
}

func (b *BrowserAdapter) collectElementsByType(
	ctx context.Context,
	collector *elementCollector,
	elementType string,
	cssSelector string,
) error {
	elements, err := b.page.Context(ctx).Elements(cssSelector)
	if err != nil {
		return fmt.Errorf("failed to find %s elements: %w", elementType, err)
	}

	for _, element := range elements {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		collector.tryAddElement(element, elementType)
	}

	return nil
}

func (b *BrowserAdapter) processScreenshot(imageBytes []byte) (*entity.Screenshot, error) {
	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return nil, fmt.Errorf("image decode failed: %w", err)
	}

	if img.Bounds().Dx() > screenshotMaxWidth {
		img = imaging.Resize(img, screenshotMaxWidth, 0, imaging.Lanczos)
	}

	buffer := new(bytes.Buffer)
	if err := jpeg.Encode(buffer, img, &jpeg.Options{Quality: screenshotQuality}); err != nil {
		return nil, fmt.Errorf("jpeg encode failed: %w", err)
	}

	return &entity.Screenshot{
		Data:   buffer.Bytes(),
		Format: "jpeg",
		Width:  img.Bounds().Dx(),
		Height: img.Bounds().Dy(),
	}, nil
}

type elementCollector struct {
	elements    []entity.UIElement
	seen        map[string]bool
	counter     int
	maxElements int
}

func newElementCollector(maxElements int) *elementCollector {
	return &elementCollector{
		elements:    make([]entity.UIElement, 0, maxElements),
		seen:        make(map[string]bool),
		counter:     0,
		maxElements: maxElements,
	}
}

func (c *elementCollector) tryAddElement(element *rod.Element, elementType string) {
	if element == nil || c.counter >= c.maxElements {
		return
	}

	visible, err := element.Visible()
	if err != nil || !visible {
		return
	}

	inViewport, err := c.isInViewport(element)
	if err != nil || !inViewport {
		return
	}

	selector := element.String()
	if selector == "" || c.seen[selector] {
		return
	}
	c.seen[selector] = true

	text, _ := element.Text()
	text = strings.TrimSpace(text)

	ariaLabel, _ := element.Attribute("aria-label")
	role, _ := element.Attribute("role")

	uiElement := entity.UIElement{
		ID:        fmt.Sprintf("ui-%04d", c.counter),
		Type:      elementType,
		Text:      text,
		AriaLabel: pointerToString(ariaLabel),
		Role:      pointerToString(role),
		Selector:  selector,
	}

	c.elements = append(c.elements, uiElement)
	c.counter++
}

func (c *elementCollector) getElements() []entity.UIElement {
	return c.elements
}

func (c *elementCollector) isInViewport(element *rod.Element) (bool, error) {
	var inViewport bool
	result, err := element.Eval(`() => {
		const rect = this.getBoundingClientRect();
		return rect.top < window.innerHeight &&
		       rect.bottom > 0 &&
		       rect.left < window.innerWidth &&
		       rect.right > 0;
	}`)
	if err != nil {
		return false, err
	}
	err = result.Value.Unmarshal(&inViewport)
	return inViewport, err
}

func pointerToString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
