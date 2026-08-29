package postdoc

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// writer builds the one canonical form of a document.
type writer struct {
	out bytes.Buffer
}

// MarshalJSON writes the one canonical form of a document, so a round trip
// through any client leaves the stored bytes comparable.
func (d Document) MarshalJSON() ([]byte, error) {
	var w writer
	w.out.WriteString(`{"version":`)
	w.out.WriteString(strconv.Itoa(Version))
	w.out.WriteString(`,"content":`)
	w.blocks(d.Blocks)
	w.out.WriteString(`}`)
	return w.out.Bytes(), nil
}

func (w *writer) blocks(blocks []Block) {
	w.out.WriteString(`[`)
	for index, block := range blocks {
		if index > 0 {
			w.out.WriteString(`,`)
		}
		block.writeJSON(w)
	}
	w.out.WriteString(`]`)
}

func (w *writer) open(kind string) {
	w.out.WriteString(`{"type":"`)
	w.out.WriteString(kind)
	w.out.WriteString(`"`)
}

func (w *writer) key(name string) {
	w.out.WriteString(`,"`)
	w.out.WriteString(name)
	w.out.WriteString(`":`)
}

func (w *writer) text(name, value string) {
	w.key(name)
	encoded, _ := json.Marshal(value)
	w.out.Write(encoded)
}

func (w *writer) content(blocks []Block) {
	w.key("content")
	w.blocks(blocks)
	w.out.WriteString(`}`)
}

func (p Paragraph) writeJSON(w *writer) {
	w.open("paragraph")
	w.key("content")
	w.spans(p.Spans)
	w.out.WriteString(`}`)
}

func (h Heading) writeJSON(w *writer) {
	w.open("heading")
	w.key("level")
	w.out.WriteString(strconv.Itoa(h.Level))
	w.text("anchor", h.Anchor)
	w.key("content")
	w.spans(h.Spans)
	w.out.WriteString(`}`)
}

func (l List) writeJSON(w *writer) {
	w.open(l.name())
	w.key("content")
	w.out.WriteString(`[`)
	for index, item := range l.Items {
		if index > 0 {
			w.out.WriteString(`,`)
		}
		item.writeJSON(w)
	}
	w.out.WriteString(`]}`)
}

func (i Item) writeJSON(w *writer) {
	w.open("listItem")
	w.content(i.Blocks)
}

func (t TaskList) writeJSON(w *writer) {
	w.open("taskList")
	w.key("content")
	w.out.WriteString(`[`)
	for index, task := range t.Tasks {
		if index > 0 {
			w.out.WriteString(`,`)
		}
		task.writeJSON(w)
	}
	w.out.WriteString(`]}`)
}

func (t Task) writeJSON(w *writer) {
	w.open("taskItem")
	w.key("done")
	w.out.WriteString(strconv.FormatBool(t.Done))
	w.content(t.Blocks)
}

func (q Quote) writeJSON(w *writer) {
	w.open("quote")
	w.content(q.Blocks)
}

func (c CodeBlock) writeJSON(w *writer) {
	w.open("codeBlock")
	w.text("language", c.Language)
	w.text("source", c.Source)
	w.out.WriteString(`}`)
}

func (t Table) writeJSON(w *writer) {
	w.open("table")
	w.key("content")
	w.out.WriteString(`[`)
	for index, row := range t.Rows {
		if index > 0 {
			w.out.WriteString(`,`)
		}
		row.writeJSON(w)
	}
	w.out.WriteString(`]}`)
}

func (r Row) writeJSON(w *writer) {
	w.open("tableRow")
	w.key("content")
	w.out.WriteString(`[`)
	for index, cell := range r.Cells {
		if index > 0 {
			w.out.WriteString(`,`)
		}
		cell.writeJSON(w)
	}
	w.out.WriteString(`]}`)
}

func (c Cell) writeJSON(w *writer) {
	w.open("tableCell")
	if c.Heading {
		w.key("heading")
		w.out.WriteString(`true`)
	}
	w.content(c.Blocks)
}

func (c Callout) writeJSON(w *writer) {
	w.open("callout")
	w.text("kind", c.Kind)
	w.content(c.Blocks)
}

func (i Image) writeJSON(w *writer) {
	w.open("image")
	w.picture(i)
}

func (g Gallery) writeJSON(w *writer) {
	w.open("gallery")
	w.key("content")
	w.out.WriteString(`[`)
	for index, picture := range g.Images {
		if index > 0 {
			w.out.WriteString(`,`)
		}
		w.open("galleryImage")
		w.picture(picture)
	}
	w.out.WriteString(`]}`)
}

// picture writes the fields an image carries and closes the node.
func (w *writer) picture(image Image) {
	w.text("mediaId", image.MediaID)
	w.text("alt", image.Alt)
	if image.Caption != "" {
		w.text("caption", image.Caption)
	}
	w.out.WriteString(`}`)
}

func (Divider) writeJSON(w *writer) {
	w.open("divider")
	w.out.WriteString(`}`)
}

func (w *writer) spans(spans []Span) {
	w.out.WriteString(`[`)
	for index, span := range spans {
		if index > 0 {
			w.out.WriteString(`,`)
		}
		span.writeJSON(w)
	}
	w.out.WriteString(`]`)
}

func (s Span) writeJSON(w *writer) {
	w.open("text")
	w.text("text", s.Text)
	marks := s.marks()
	if len(marks) == 0 {
		w.out.WriteString(`}`)
		return
	}
	w.key("marks")
	w.out.WriteString(`[`)
	for index, mark := range marks {
		if index > 0 {
			w.out.WriteString(`,`)
		}
		w.open(mark)
		if mark == markLink {
			w.text("href", s.Link)
		}
		w.out.WriteString(`}`)
	}
	w.out.WriteString(`]}`)
}

// marks answers the marks on a span in the one order the canonical form uses.
func (s Span) marks() []string {
	ordered := make([]string, 0, 5)
	if s.Bold {
		ordered = append(ordered, markBold)
	}
	if s.Italic {
		ordered = append(ordered, markItalic)
	}
	if s.Strike {
		ordered = append(ordered, markStrike)
	}
	if s.Code {
		ordered = append(ordered, markCode)
	}
	if s.Link != "" {
		ordered = append(ordered, markLink)
	}
	return ordered
}
