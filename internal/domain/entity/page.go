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

type PageContext struct {
	URL             string
	Title           string
	VisibleElements []UIElement
	TextContent     string
	ElementCount    int
}

type QueryElementsRequest struct {
	Selector string            `json:"selector"`
	Limit    int               `json:"limit"`
	Extract  map[string]string `json:"extract"`
}

type QueryElementsResult struct {
	Elements []ElementData
	Count    int
}

type ElementData struct {
	Index    int
	Selector string
	Data     map[string]string
}

type SearchRequest struct {
	Type  string `json:"type"`
	Query string `json:"query"`
}

type SearchResult struct {
	Type     string
	Found    bool
	Content  string
	Elements []SearchElement
}

type SearchElement struct {
	ID         string
	Selector   string
	TagName    string
	Text       string
	Attributes map[string]string
}

type PageChanges struct {
	NewElements     []UIElement
	URLChanged      bool
	NewURL          string
	ModalOpened     bool
	ModalClosed     bool
	ElementsRemoved int
}

type ClickResult struct {
	Success bool
	Changes *PageChanges
	Error   string
}
