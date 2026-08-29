// Package postdoc owns the structured body Illarin keeps for a post. Whatever
// editor writes a post, this vocabulary is what gets stored and rendered.
package postdoc

// Version is the document version every writer emits.
const Version = 2

// firstVersion is the oldest stored version a reader still upgrades.
const firstVersion = 1

// Document is one post body, an ordered run of blocks.
type Document struct {
	Blocks []Block
}

// Block is one structure in a document. The set of them is closed.
type Block interface {
	name() string
	writeJSON(*writer)
}

// Paragraph is a run of prose.
type Paragraph struct {
	Spans []Span
}

// Heading is a section title inside the body. The post title is the page
// heading, so a body heading starts at level two.
type Heading struct {
	Level  int
	Anchor string
	Spans  []Span
}

// List is a bulleted or numbered run of items.
type List struct {
	Ordered bool
	Items   []Item
}

// Item is one entry in a list.
type Item struct {
	Blocks []Block
}

// TaskList is a run of items a reader can see the state of.
type TaskList struct {
	Tasks []Task
}

// Task is one entry in a task list.
type Task struct {
	Done   bool
	Blocks []Block
}

// Quote is quoted material.
type Quote struct {
	Blocks []Block
}

// CodeBlock is source in one language, kept exactly as it was written.
type CodeBlock struct {
	Language string
	Source   string
}

// Table is a rectangle of cells. Its heading cells fill the first row, the
// first column, or both.
type Table struct {
	Rows []Row
}

// Row is one row of a table.
type Row struct {
	Cells []Cell
}

// Cell is one cell of a table.
type Cell struct {
	Heading bool
	Blocks  []Block
}

// Callout is an aside that says what kind of aside it is.
type Callout struct {
	Kind   string
	Blocks []Block
}

// Divider separates two parts of a post.
type Divider struct{}

// Span is a run of text carrying the marks that apply to all of it.
type Span struct {
	Text   string
	Bold   bool
	Italic bool
	Strike bool
	Code   bool
	Link   string
}

func (Paragraph) name() string { return "paragraph" }
func (Heading) name() string   { return "heading" }
func (l List) name() string {
	if l.Ordered {
		return "orderedList"
	}
	return "bulletList"
}
func (Item) name() string      { return "listItem" }
func (TaskList) name() string  { return "taskList" }
func (Task) name() string      { return "taskItem" }
func (Quote) name() string     { return "quote" }
func (CodeBlock) name() string { return "codeBlock" }
func (Table) name() string     { return "table" }
func (Row) name() string       { return "tableRow" }
func (Cell) name() string      { return "tableCell" }
func (Callout) name() string   { return "callout" }
func (Divider) name() string   { return "divider" }
