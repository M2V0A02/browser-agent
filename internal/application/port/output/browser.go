package output

import (
	"context"

	"browser-agent/internal/domain/entity"
)

type BrowserPort interface {
	Navigate(ctx context.Context, url string) error
	Click(ctx context.Context, selector string) error
	Fill(ctx context.Context, selector, text string) error
	PressEnter(ctx context.Context) error
	Scroll(ctx context.Context, direction string, amount int) error

	GetPageContent(ctx context.Context) (*entity.PageContent, error)
	GetPageText(ctx context.Context) (string, error)
	GetUIElements(ctx context.Context) ([]entity.UIElement, error)
	Screenshot(ctx context.Context) (*entity.Screenshot, error)

	CurrentURL() string
	Close()
}
