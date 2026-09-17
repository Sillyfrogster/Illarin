package body

const Version = 2

const firstVersion = 1

type Document struct {
	Blocks []Block
}

type Block interface {
	name() string
	writeJSON(*writer)
}

type Paragraph struct {
	Spans []Span
}

type Heading struct {
	Level  int
	Anchor string
	Spans  []Span
}

type List struct {
	Ordered bool
	Items   []Item
}

type Item struct {
	Blocks []Block
}

type TaskList struct {
	Tasks []Task
}

type Task struct {
	Done   bool
	Blocks []Block
}

type Quote struct {
	Blocks []Block
}

type CodeBlock struct {
	Language string
	Source   string
}

type Table struct {
	Rows []Row
}

type Row struct {
	Cells []Cell
}

type Cell struct {
	Heading bool
	Blocks  []Block
}

type Callout struct {
	Kind   string
	Blocks []Block
}

type Image struct {
	MediaID string
	Alt     string
	Caption string
}

type Gallery struct {
	Images []Image
}

type Divider struct{}

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
func (Image) name() string     { return "image" }
func (Gallery) name() string   { return "gallery" }
func (Divider) name() string   { return "divider" }
