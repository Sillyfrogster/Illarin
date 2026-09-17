package body

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

const (
	placeContent  = "content"
	placeListItem = "listItem"
	placeTaskItem = "taskItem"
	placeQuote    = "quote"
	placeCallout  = "callout"
)

var placeNames = map[string]string{
	placeContent:  "a post",
	placeListItem: "a list item",
	placeTaskItem: "a task",
	placeQuote:    "a quotation",
	placeCallout:  "a callout",
	"tableCell":   "a table cell",
}

var blockNames = map[string]string{
	"paragraph":   "Prose",
	"heading":     "A heading",
	"bulletList":  "A list",
	"orderedList": "A numbered list",
	"taskList":    "A checklist",
	"quote":       "A quotation",
	"codeBlock":   "A code block",
	"table":       "A table",
	"callout":     "A callout",
	"image":       "A picture",
	"gallery":     "A gallery",
	"divider":     "A divider",
}

var mdxStatement = regexp.MustCompile(`^\s*(import|export)\s`)

func (i *importer) blocks(from ast.Node, place string) []Block {
	blocks := make([]Block, 0, 8)
	for node := from; node != nil; node = node.NextSibling() {
		if block, ok := i.block(node, place); ok {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

func (i *importer) block(node ast.Node, place string) (Block, bool) {
	built, ok := i.build(node, place)
	if !ok {
		return nil, false
	}
	if !allows(place, built.name()) {
		i.refuse(node, fmt.Sprintf("%s does not go inside %s.",
			blockNames[built.name()], placeNames[place]))
		return nil, false
	}
	return built, true
}

func (i *importer) build(node ast.Node, place string) (Block, bool) {
	switch shape := node.(type) {
	case *ast.Heading:
		return i.heading(shape)
	case *ast.Paragraph:
		return i.prose(shape, place)
	case *ast.TextBlock:
		return i.prose(shape, place)
	case *ast.List:
		return i.list(shape)
	case *ast.Blockquote:
		return i.quotation(shape)
	case *ast.FencedCodeBlock:
		return i.code(shape, shape.Info, string(shape.Language(i.source)))
	case *ast.CodeBlock:
		return i.code(shape, nil, "")
	case *ast.ThematicBreak:
		return Divider{}, true
	case *east.Table:
		return i.table(shape)
	case *ast.HTMLBlock:
		i.refuse(node, "Illarin does not carry HTML.")
	case *east.FootnoteList, *east.Footnote:
		i.refuse(node, "Illarin does not carry footnotes.")
	default:
		i.refuse(node, "Illarin does not carry this.")
	}
	return nil, false
}

func (i *importer) heading(node *ast.Heading) (Block, bool) {
	if node.Level < minHeadingLevel {
		i.refuse(node, fmt.Sprintf(
			"The post title is the page heading, so a body heading starts at level %d.",
			minHeadingLevel))
		return nil, false
	}
	if node.Level > maxHeadingLevel {
		i.refuse(node, fmt.Sprintf("Illarin goes down to level %d headings.", maxHeadingLevel))
		return nil, false
	}
	return Heading{Level: node.Level, Spans: i.spansFrom(node.FirstChild(), Span{})}, true
}

func (i *importer) prose(node ast.Node, place string) (Block, bool) {
	if place == placeContent && i.mdx(node) {
		i.refuse(node, "Illarin does not carry MDX.")
		return nil, false
	}
	if only := i.lonePicture(node); only != nil {
		return i.drawn(only)
	}
	return Paragraph{Spans: i.spansFrom(node.FirstChild(), Span{})}, true
}

func (i *importer) mdx(node ast.Node) bool {
	lines := node.Lines()
	if lines == nil || lines.Len() == 0 {
		return false
	}
	first := lines.At(0)
	return mdxStatement.Match(first.Value(i.source))
}

func (i *importer) list(node *ast.List) (Block, bool) {
	marked, plain := countChecks(node)
	if marked > 0 && plain > 0 {
		i.refuse(node, "A checklist marks every one of its items.")
		return nil, false
	}
	if node.IsOrdered() && node.Start != 1 {
		i.note(node, "A numbered list always starts at 1.")
	}
	if marked > 0 {
		return i.checklist(node)
	}
	items := make([]Item, 0, node.ChildCount())
	for entry := node.FirstChild(); entry != nil; entry = entry.NextSibling() {
		blocks, ok := i.entry(entry, placeListItem, "A list item needs something in it.")
		if !ok {
			return nil, false
		}
		items = append(items, Item{Blocks: blocks})
	}
	return List{Ordered: node.IsOrdered(), Items: items}, true
}

func (i *importer) checklist(node *ast.List) (Block, bool) {
	tasks := make([]Task, 0, node.ChildCount())
	for entry := node.FirstChild(); entry != nil; entry = entry.NextSibling() {
		blocks, ok := i.entry(entry, placeTaskItem, "A task needs something in it.")
		if !ok {
			return nil, false
		}
		tasks = append(tasks, Task{Done: isChecked(entry), Blocks: blocks})
	}
	return TaskList{Tasks: tasks}, true
}

func (i *importer) entry(node ast.Node, place, empty string) ([]Block, bool) {
	before := len(i.refusals)
	blocks := i.blocks(node.FirstChild(), place)
	if len(blocks) == 0 {
		i.emptyRefusal(node, before, empty)
		return nil, false
	}
	return blocks, true
}

func (i *importer) emptyRefusal(node ast.Node, before int, message string) {
	if len(i.refusals) == before {
		i.refuse(node, message)
	}
}

func (i *importer) quotation(node *ast.Blockquote) (Block, bool) {
	before := len(i.refusals)
	marker, from, marked := i.marker(node)
	if !marked {
		blocks := i.blocks(node.FirstChild(), placeQuote)
		if len(blocks) == 0 {
			i.emptyRefusal(node, before, "A quotation needs something in it.")
			return nil, false
		}
		return Quote{Blocks: blocks}, true
	}
	kind := strings.ToLower(marker)
	if !isCalloutKind(kind) {
		i.refuse(node, fmt.Sprintf("A callout is one of: %s.", strings.Join(CalloutKinds, ", ")))
		return nil, false
	}
	blocks := i.aside(node, from)
	if len(blocks) == 0 {
		i.emptyRefusal(node, before, "A callout needs something in it.")
		return nil, false
	}
	return Callout{Kind: kind, Blocks: blocks}, true
}

func (i *importer) marker(node *ast.Blockquote) (string, ast.Node, bool) {
	first := node.FirstChild()
	if first == nil || (first.Kind() != ast.KindParagraph && first.Kind() != ast.KindTextBlock) {
		return "", nil, false
	}
	var said strings.Builder
	for inline := first.FirstChild(); inline != nil; inline = inline.NextSibling() {
		word, ok := inline.(*ast.Text)
		if !ok {
			return "", nil, false
		}
		said.Write(word.Segment.Value(i.source))
		if word.SoftLineBreak() || word.HardLineBreak() {
			return named(said.String(), inline.NextSibling())
		}
	}
	return named(said.String(), nil)
}

func named(line string, from ast.Node) (string, ast.Node, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "[!") || !strings.HasSuffix(trimmed, "]") {
		return "", nil, false
	}
	return trimmed[len("[!") : len(trimmed)-len("]")], from, true
}

func (i *importer) aside(node *ast.Blockquote, from ast.Node) []Block {
	blocks := make([]Block, 0, 4)
	if spans := i.spansFrom(from, Span{}); spanLength(spans) > 0 {
		blocks = append(blocks, Paragraph{Spans: spans})
	}
	for next := node.FirstChild().NextSibling(); next != nil; next = next.NextSibling() {
		if block, ok := i.block(next, placeCallout); ok {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

func (i *importer) code(node ast.Node, info *ast.Text, label string) (Block, bool) {
	source := canonicalSource(i.lines(node.Lines()))
	if strings.TrimSpace(source) == "" {
		i.refuse(node, "A code block needs code in it.")
		return nil, false
	}
	language := strings.TrimSpace(label)
	if language == "" {
		language = "plain"
	}
	if !isLanguage(language) {
		about := ast.Node(info)
		if info == nil {
			about = node
		}
		i.note(about, "Illarin has no label for "+language+" code, so this block is labelled plain.")
		language = "plain"
	}
	return CodeBlock{Language: language, Source: source}, true
}

func (i *importer) table(node *east.Table) (Block, bool) {
	rows := make([]Row, 0, node.ChildCount())
	for line := node.FirstChild(); line != nil; line = line.NextSibling() {
		row := Row{Cells: i.cells(line, line.Kind() == east.KindTableHeader)}
		if len(rows) > 0 && len(row.Cells) != len(rows[0].Cells) {
			i.refuse(line, "Every row of a table holds the same number of cells.")
			return nil, false
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		i.refuse(node, "A table needs at least one row.")
		return nil, false
	}
	return Table{Rows: rows}, true
}

func (i *importer) cells(line ast.Node, heading bool) []Cell {
	cells := make([]Cell, 0, 4)
	for cell := line.FirstChild(); cell != nil; cell = cell.NextSibling() {
		blocks := make([]Block, 0, 1)
		if spans := i.spansFrom(cell.FirstChild(), Span{}); len(spans) > 0 {
			blocks = append(blocks, Paragraph{Spans: spans})
		}
		cells = append(cells, Cell{Heading: heading, Blocks: blocks})
	}
	return cells
}

func (i *importer) lonePicture(node ast.Node) *ast.Image {
	var only *ast.Image
	for inline := node.FirstChild(); inline != nil; inline = inline.NextSibling() {
		if drawn, ok := inline.(*ast.Image); ok {
			if only != nil {
				return nil
			}
			only = drawn
			continue
		}
		if word, ok := inline.(*ast.Text); ok &&
			strings.TrimSpace(string(word.Segment.Value(i.source))) == "" {
			continue
		}
		return nil
	}
	return only
}

func (i *importer) drawn(node *ast.Image) (Block, bool) {
	named, carried := strings.CutPrefix(string(node.Destination), MediaPrefix)
	held, err := uuid.Parse(named)
	if !carried || err != nil {
		i.refuse(node, "A picture names a file already uploaded to this post, as "+
			MediaPrefix+"<id>. Upload the file first and place the id it answers with.")
		return nil, false
	}
	if marked(node) {
		i.note(node, "The description of a picture is plain text.")
	}
	alt := strings.TrimSpace(i.plain(node))
	if alt == "" {
		i.refuse(node, "Describe the picture for a reader who cannot see it.")
		return nil, false
	}
	return Image{
		MediaID: held.String(),
		Alt:     alt,
		Caption: strings.TrimSpace(string(node.Title)),
	}, true
}

func countChecks(node *ast.List) (int, int) {
	marked, plain := 0, 0
	for entry := node.FirstChild(); entry != nil; entry = entry.NextSibling() {
		if isChecked(entry) || checkbox(entry) != nil {
			marked++
			continue
		}
		plain++
	}
	return marked, plain
}

func isChecked(entry ast.Node) bool {
	box := checkbox(entry)
	return box != nil && box.IsChecked
}

func checkbox(entry ast.Node) *east.TaskCheckBox {
	first := entry.FirstChild()
	if first == nil {
		return nil
	}
	box, ok := first.FirstChild().(*east.TaskCheckBox)
	if !ok {
		return nil
	}
	return box
}

func marked(node ast.Node) bool {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if _, plain := child.(*ast.Text); !plain {
			return true
		}
	}
	return false
}
