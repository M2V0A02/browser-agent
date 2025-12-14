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
	Limit int    `json:"limit"` // optional limit for results
}

type SearchResult struct {
	Type     string
	Found    bool
	Query    string
	Count    int
	Results  []SearchResultItem
	// Deprecated: use Results instead
	Content  string
	Elements []SearchElement
}

type SearchResultItem struct {
	Element    string            `json:"element"`     // tag name (div, section, h2, etc.)
	Text       string            `json:"text"`        // text content
	Selector   string            `json:"selector"`    // CSS selector to access element
	ID         string            `json:"id,omitempty"` // element ID if present
	Classes    []string          `json:"classes,omitempty"` // element classes if present
	Attributes map[string]string `json:"attributes,omitempty"` // key attributes
	Parent     *ParentInfo       `json:"parent,omitempty"` // parent element info
	Match      string            `json:"match,omitempty"` // what exactly matched (for contains search)
}

type ParentInfo struct {
	Element  string   `json:"element"`
	Selector string   `json:"selector"`
	ID       string   `json:"id,omitempty"`
	Classes  []string `json:"classes,omitempty"`
}

type SearchElement struct {
	ID         string
	Selector   string
	TagName    string
	Text       string
	Attributes map[string]string
}

// PageStructure represents semantic page structure
type PageStructure struct {
	URL             string
	Title           string
	Elements        []StructureElement
	RepeatedClasses map[string]int // class name -> count (only classes appearing >= 2 times)
}

type StructureElement struct {
	TagName    string
	Selector   string
	ID         string
	Classes    []string
	Text       string // first 100 chars
	Level      int    // nesting level for tree display
	Children   int    // number of children
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
