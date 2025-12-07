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

	maxUIElements = 500

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
		ID:         fmt.Sprintf("ui-%04d", c.counter),
		Type:       elementType,
		Text:       text,
		AriaLabel:  pointerToString(ariaLabel),
		Role:       pointerToString(role),
		Visible:    true,
		InViewport: true,
		Selector:   selector,
	}

	c.elements = append(c.elements, uiElement)
	c.counter++
}

func (c *elementCollector) getElements() []entity.UIElement {
	return c.elements
}

func pointerToString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
