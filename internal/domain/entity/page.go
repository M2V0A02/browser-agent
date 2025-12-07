package entity

type PageContent struct {
	URL        string
	Title      string
	HTML       string
	UIElements []UIElement
}

type UIElement struct {
	ID        string
	Type      string
	Text      string
	AriaLabel string
	Role      string
	Selector  string
}

type Screenshot struct {
	Data   []byte
	Format string
	Width  int
	Height int
}
