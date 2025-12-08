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
	GetPageContext(ctx context.Context) (*entity.PageContext, error)
	Screenshot(ctx context.Context) (*entity.Screenshot, error)
	QueryElements(ctx context.Context, req entity.QueryElementsRequest) (*entity.QueryElementsResult, error)
	Search(ctx context.Context, req entity.SearchRequest) (*entity.SearchResult, error)

	CurrentURL() string
	Close()
}
