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
	Selector string
	Limit    int
	Extract  map[string]string
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
	Type  string
	Query string
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
	Attributes map[string]string
}
