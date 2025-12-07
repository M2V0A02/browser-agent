package rod

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"strings"
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

type BrowserAdapter struct {
	browser  *rod.Browser
	launcher *launcher.Launcher
	page     *rod.Page
	timeout  time.Duration
}

type BrowserConfig struct {
	Headless    bool
	SlowMotion  time.Duration
	Timeout     time.Duration
	NoSandbox   bool
	DevTools    bool
}

func DefaultConfig() BrowserConfig {
	return BrowserConfig{
		Headless:   false,
		SlowMotion: 500 * time.Millisecond,
		Timeout:    10 * time.Second,
		NoSandbox:  true,
		DevTools:   false,
	}
}

func NewBrowserAdapter(ctx context.Context, cfg BrowserConfig) (*BrowserAdapter, error) {
	l := launcher.New().
		Headless(cfg.Headless).
		Devtools(cfg.DevTools).
		NoSandbox(cfg.NoSandbox).
		Delete("use-mock-keychain").
		Set("disable-web-security").
		Set("allow-running-insecure-content").
		Set("disable-setuid-sandbox")

	url, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().
		ControlURL(url).
		Trace(true).
		SlowMotion(cfg.SlowMotion).
		MustConnect()

	page := browser.MustPage("about:blank")

	return &BrowserAdapter{
		browser:  browser,
		launcher: l,
		page:     page,
		timeout:  cfg.Timeout,
	}, nil
}

func (b *BrowserAdapter) Navigate(ctx context.Context, url string) error {
	if err := b.page.Navigate(url); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}
	b.page.MustWaitLoad()
	b.page.WaitIdle(5 * time.Second)
	return nil
}

func (b *BrowserAdapter) Click(ctx context.Context, selector string) error {
	var el *rod.Element
	var err error

	if strings.HasPrefix(selector, "/") || strings.Contains(selector, "xpath") {
		el, err = b.page.Timeout(b.timeout).ElementX(selector)
	} else {
		el, err = b.page.Timeout(b.timeout).Element(selector)
	}
	if err != nil {
		return fmt.Errorf("element not found: %s: %w", selector, err)
	}

	if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("click failed: %w", err)
	}

	b.page.WaitIdle(2 * time.Second)
	return nil
}

func (b *BrowserAdapter) Fill(ctx context.Context, selector, text string) error {
	el, err := b.page.Timeout(b.timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("field not found: %s: %w", selector, err)
	}

	if err := el.SelectAllText(); err == nil {
		_ = el.Input("")
	}

	if err := el.Input(text); err != nil {
		return fmt.Errorf("input failed: %w", err)
	}

	return nil
}

func (b *BrowserAdapter) PressEnter(ctx context.Context) error {
	el := b.page.MustElement("body")
	if err := el.Input("\r"); err != nil {
		return fmt.Errorf("failed to press Enter: %w", err)
	}
	b.page.WaitIdle(1 * time.Second)
	return nil
}

func (b *BrowserAdapter) Scroll(ctx context.Context, direction string, amount int) error {
	direction = strings.ToLower(strings.TrimSpace(direction))

	switch direction {
	case "down":
		b.page.Eval(`() => window.scrollBy(0, window.innerHeight * 2)`)
	case "up":
		b.page.Eval(`() => window.scrollBy(0, -window.innerHeight * 2)`)
	case "top":
		b.page.Eval(`() => window.scrollTo(0, 0)`)
	case "bottom":
		b.page.Eval(`() => window.scrollTo(0, document.body.scrollHeight)`)
	default:
		return fmt.Errorf("unknown scroll direction: %s", direction)
	}

	b.page.WaitIdle(800 * time.Millisecond)
	return nil
}

func (b *BrowserAdapter) GetPageContent(ctx context.Context) (*entity.PageContent, error) {
	info := b.page.MustInfo()

	body, err := b.page.Timeout(b.timeout).Element("body")
	if err != nil {
		return nil, fmt.Errorf("body not found: %w", err)
	}

	html, err := body.HTML()
	if err != nil {
		return nil, fmt.Errorf("failed to get HTML: %w", err)
	}

	elements, err := b.GetUIElements(ctx)
	if err != nil {
		elements = nil
	}

	return &entity.PageContent{
		URL:        info.URL,
		Title:      info.Title,
		HTML:       html,
		UIElements: elements,
	}, nil
}

func (b *BrowserAdapter) GetUIElements(ctx context.Context) ([]entity.UIElement, error) {
	var result []entity.UIElement
	seen := make(map[string]bool)
	counter := 0
	maxElements := 500

	add := func(el *rod.Element, typ string) {
		if el == nil || counter >= maxElements {
			return
		}

		visible, err := el.Visible()
		if err != nil || !visible {
			return
		}

		selector := el.MustElementX("@").String()
		if seen[selector] {
			return
		}
		seen[selector] = true

		text, _ := el.Text()
		text = strings.TrimSpace(text)
		aria, _ := el.Attribute("aria-label")
		role, _ := el.Attribute("role")

		element := entity.UIElement{
			ID:         fmt.Sprintf("ui-%04d", counter),
			Type:       typ,
			Text:       text,
			AriaLabel:  ptrToString(aria),
			Role:       ptrToString(role),
			Visible:    true,
			InViewport: true,
			Selector:   selector,
		}

		result = append(result, element)
		counter++
	}

	elements, err := b.page.Elements("button, [role='button'], [data-tooltip], [aria-label]:not([aria-label=''])")
	if err == nil {
		for _, el := range elements {
			add(el, "button")
		}
	}

	elements, err = b.page.Elements("input, textarea")
	if err == nil {
		for _, el := range elements {
			add(el, "input")
		}
	}

	elements, err = b.page.Elements("a")
	if err == nil {
		for _, el := range elements {
			add(el, "link")
		}
	}

	return result, nil
}

func (b *BrowserAdapter) Screenshot(ctx context.Context) (*entity.Screenshot, error) {
	imgBytes, err := b.page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: gson.Int(80),
	})
	if err != nil {
		return nil, fmt.Errorf("screenshot failed: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("image decode failed: %w", err)
	}

	if img.Bounds().Dx() > 1024 {
		img = imaging.Resize(img, 1024, 0, imaging.Lanczos)
	}

	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 75}); err != nil {
		return nil, fmt.Errorf("jpeg encode failed: %w", err)
	}

	return &entity.Screenshot{
		Data:   buf.Bytes(),
		Format: "jpeg",
		Width:  img.Bounds().Dx(),
		Height: img.Bounds().Dy(),
	}, nil
}

func (b *BrowserAdapter) CurrentURL() string {
	return b.page.MustInfo().URL
}

func (b *BrowserAdapter) Close() {
	if b.browser != nil {
		_ = b.browser.Close()
	}
	if b.launcher != nil {
		b.launcher.Kill()
		b.launcher.Cleanup()
	}
}

func ptrToString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
