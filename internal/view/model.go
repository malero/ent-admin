package view

// PageKind identifies the generic page model passed from the runtime to the
// renderer. The renderer can change without changing route dispatch.
type PageKind string

const (
	PageIndex  PageKind = "index"
	PageList   PageKind = "list"
	PageDetail PageKind = "detail"
	PageError  PageKind = "error"
)

type Field struct {
	Name      string
	Label     string
	Kind      string
	Sensitive bool
}

type Entity struct {
	Name        string
	Label       string
	PluralLabel string
	Route       string
	Fields      []Field
}

type Record struct {
	ID     string
	Values map[string]any
}

type Page struct {
	Kind        PageKind
	StatusCode  int
	BasePath    string
	CurrentPath string
	Title       string
	Entities    []Entity
	Entity      *Entity
	Records     []Record
	Record      *Record
	Error       string
}
