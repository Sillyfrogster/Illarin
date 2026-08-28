// Package postdoc owns the structured body Illarin keeps for a post. Whatever
// editor writes a post, this vocabulary is what gets stored and rendered.
package postdoc

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// Version is the document version every writer emits.
const Version = 1

// Document is one post body, an ordered run of blocks.
type Document struct {
	Blocks []Block
}

// Block is one structure in a document. The set of them is closed.
type Block interface {
	name() string
	writeJSON(*bytes.Buffer) error
}

// Paragraph is a run of prose.
type Paragraph struct {
	Spans []Span
}

// Heading is a section title inside the body. The post title is the page
// heading, so a body heading starts at level two.
type Heading struct {
	Level int
	Spans []Span
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

// Quote is quoted material.
type Quote struct {
	Blocks []Block
}

// Divider separates two parts of a post.
type Divider struct{}

// Span is a run of text carrying the marks that apply to all of it.
type Span struct {
	Text   string
	Bold   bool
	Italic bool
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
func (Item) name() string    { return "listItem" }
func (Quote) name() string   { return "quote" }
func (Divider) name() string { return "divider" }

// MarshalJSON writes the one canonical form of a document, so a round trip
// through any client leaves the stored bytes comparable.
func (d Document) MarshalJSON() ([]byte, error) {
	var out bytes.Buffer
	out.WriteString(`{"version":`)
	out.WriteString(strconv.Itoa(Version))
	out.WriteString(`,"content":`)
	if err := writeBlocks(&out, d.Blocks); err != nil {
		return nil, err
	}
	out.WriteString(`}`)
	return out.Bytes(), nil
}

func (p Paragraph) writeJSON(out *bytes.Buffer) error {
	out.WriteString(`{"type":"paragraph","content":`)
	if err := writeSpans(out, p.Spans); err != nil {
		return err
	}
	out.WriteString(`}`)
	return nil
}

func (h Heading) writeJSON(out *bytes.Buffer) error {
	out.WriteString(`{"type":"heading","level":`)
	out.WriteString(strconv.Itoa(h.Level))
	out.WriteString(`,"content":`)
	if err := writeSpans(out, h.Spans); err != nil {
		return err
	}
	out.WriteString(`}`)
	return nil
}

func (l List) writeJSON(out *bytes.Buffer) error {
	out.WriteString(`{"type":"`)
	out.WriteString(l.name())
	out.WriteString(`","content":[`)
	for index, item := range l.Items {
		if index > 0 {
			out.WriteString(`,`)
		}
		if err := item.writeJSON(out); err != nil {
			return err
		}
	}
	out.WriteString(`]}`)
	return nil
}

func (i Item) writeJSON(out *bytes.Buffer) error {
	out.WriteString(`{"type":"listItem","content":`)
	if err := writeBlocks(out, i.Blocks); err != nil {
		return err
	}
	out.WriteString(`}`)
	return nil
}

func (q Quote) writeJSON(out *bytes.Buffer) error {
	out.WriteString(`{"type":"quote","content":`)
	if err := writeBlocks(out, q.Blocks); err != nil {
		return err
	}
	out.WriteString(`}`)
	return nil
}

func (Divider) writeJSON(out *bytes.Buffer) error {
	out.WriteString(`{"type":"divider"}`)
	return nil
}

func writeBlocks(out *bytes.Buffer, blocks []Block) error {
	out.WriteString(`[`)
	for index, block := range blocks {
		if index > 0 {
			out.WriteString(`,`)
		}
		if err := block.writeJSON(out); err != nil {
			return err
		}
	}
	out.WriteString(`]`)
	return nil
}

func writeSpans(out *bytes.Buffer, spans []Span) error {
	out.WriteString(`[`)
	for index, span := range spans {
		if index > 0 {
			out.WriteString(`,`)
		}
		if err := span.writeJSON(out); err != nil {
			return err
		}
	}
	out.WriteString(`]`)
	return nil
}

func (s Span) writeJSON(out *bytes.Buffer) error {
	out.WriteString(`{"type":"text","text":`)
	text, err := json.Marshal(s.Text)
	if err != nil {
		return err
	}
	out.Write(text)
	marks := s.marks()
	if len(marks) == 0 {
		out.WriteString(`}`)
		return nil
	}
	out.WriteString(`,"marks":[`)
	for index, mark := range marks {
		if index > 0 {
			out.WriteString(`,`)
		}
		out.WriteString(`{"type":"`)
		out.WriteString(mark)
		out.WriteString(`"`)
		if mark == markLink {
			href, err := json.Marshal(s.Link)
			if err != nil {
				return err
			}
			out.WriteString(`,"href":`)
			out.Write(href)
		}
		out.WriteString(`}`)
	}
	out.WriteString(`]}`)
	return nil
}

// marks answers the marks on a span in the one order the canonical form uses.
func (s Span) marks() []string {
	ordered := make([]string, 0, 4)
	if s.Bold {
		ordered = append(ordered, markBold)
	}
	if s.Italic {
		ordered = append(ordered, markItalic)
	}
	if s.Code {
		ordered = append(ordered, markCode)
	}
	if s.Link != "" {
		ordered = append(ordered, markLink)
	}
	return ordered
}
