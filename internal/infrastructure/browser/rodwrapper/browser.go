package rodwrapper

import (
	"context"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// Browser — обёртка над *rod.Browser с корректным закрытием
type Browser struct {
	browser  *rod.Browser
	launcher *launcher.Launcher // важно! чтобы корректно убить процесс Chrome
}

// NewBrowser — запускает браузер с правильной очисткой ресурсов
func NewBrowser(ctx context.Context) (*Browser, error) {
	// Настраиваем лаунчер (только флаги запуска, без SlowMotion!)
	l := launcher.New().
		Headless(false).
		Devtools(false).
		NoSandbox(true).
		Delete("use-mock-keychain").
		Set("disable-web-security").
		Set("allow-running-insecure-content").
		Set("disable-setuid-sandbox")

	// Запускаем и получаем URL для подключения
	url, err := l.Launch()
	if err != nil {
		return nil, err
	}

	// Подключаемся к браузеру
	browser := rod.New().
		ControlURL(url).
		Trace(true).                        // логи действий (опционально)
		SlowMotion(500 * time.Millisecond). // ← ЗДЕСЬ! Замедление действий браузера
		MustConnect()

	return &Browser{
		browser:  browser,
		launcher: l,
	}, nil
}

// Page — создаёт новую страницу и сразу открывает about:blank
func (b *Browser) Page() (*Page, error) {
	rodPage := b.browser.MustPage("about:blank")
	page := NewPage(rodPage)
	return page, nil
}

// Close — корректно закрывает и браузер, и процесс Chrome
func (b *Browser) Close() {
	if b.browser != nil {
		_ = b.browser.Close()
	}
	if b.launcher != nil {
		b.launcher.Kill() // убиваем процесс Chrome
		b.launcher.Cleanup()
	}
}

type Page struct {
	*rod.Page
	defaultTimeout time.Duration
}

func NewPage(rodPage *rod.Page) *Page {
	return &Page{
		Page:           rodPage,
		defaultTimeout: 10 * time.Second,
	}
}

func (p *Page) Element(selector string) (*rod.Element, error) {
	return p.Page.Timeout(p.defaultTimeout).Element(selector)
}

func (p *Page) ElementR(selector, regex string) (*rod.Element, error) {
	return p.Page.Timeout(p.defaultTimeout).ElementR(selector, regex)
}

func (p *Page) WaitElement(selector string) (*rod.Element, error) {
	return p.Page.Timeout(p.defaultTimeout).Element(selector)
}
