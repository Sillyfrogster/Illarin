package postdoc

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

func (i *importer) spansFrom(start ast.Node, wearing Span) []Span {
	spans := make([]Span, 0, 4)
	for node := start; node != nil; node = node.NextSibling() {
		spans = append(spans, i.inline(node, wearing)...)
	}
	return join(spans)
}

func (i *importer) inline(node ast.Node, wearing Span) []Span {
	switch shape := node.(type) {
	case *ast.Text:
		return i.words(shape, wearing)
	case *ast.String:
		return []Span{wearing.saying(string(shape.Value))}
	case *ast.CodeSpan:
		return []Span{wearing.wearing(markCode).saying(i.plain(shape))}
	case *ast.Emphasis:
		return i.spansFrom(shape.FirstChild(), wearing.wearing(emphasis(shape.Level)))
	case *east.Strikethrough:
		return i.spansFrom(shape.FirstChild(), wearing.wearing(markStrike))
	case *ast.Link:
		return i.link(shape, wearing)
	case *ast.AutoLink:
		return i.autolink(shape, wearing)
	case *east.TaskCheckBox:
		return nil
	case *ast.Image:
		i.refuse(node, "A picture stands on a line of its own.")
	case *ast.RawHTML:
		i.refuse(node, "Illarin does not carry HTML.")
	case *east.FootnoteLink, *east.FootnoteBacklink:
		i.refuse(node, "Illarin does not carry footnotes.")
	default:
		i.refuse(node, "Illarin does not carry this.")
	}
	return nil
}

func (i *importer) words(node *ast.Text, wearing Span) []Span {
	said := string(node.Segment.Value(i.source))
	if node.HardLineBreak() {
		i.note(node, "A line break inside a paragraph becomes a space.")
	}
	if node.SoftLineBreak() || node.HardLineBreak() {
		said += " "
	}
	return []Span{wearing.saying(said)}
}

func (i *importer) link(node *ast.Link, wearing Span) []Span {
	address := string(node.Destination)
	if said := addressProblem(address); said != "" {
		i.refuse(node, said)
		return nil
	}
	if len(node.Title) > 0 {
		i.note(node, "A link carries no hover title.")
	}
	worn := wearing
	worn.Link = address
	return i.spansFrom(node.FirstChild(), worn)
}

func (i *importer) autolink(node *ast.AutoLink, wearing Span) []Span {
	address := string(node.URL(i.source))
	if node.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(address, "mailto:") {
		address = "mailto:" + address
	}
	if said := addressProblem(address); said != "" {
		i.refuse(node, said)
		return nil
	}
	worn := wearing
	worn.Link = address
	return []Span{worn.saying(string(node.Label(i.source)))}
}

func (i *importer) plain(node ast.Node) string {
	var said strings.Builder
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch shape := child.(type) {
		case *ast.Text:
			said.Write(shape.Segment.Value(i.source))
		case *ast.String:
			said.Write(shape.Value)
		default:
			said.WriteString(i.plain(child))
		}
	}
	return said.String()
}

func join(spans []Span) []Span {
	joined := make([]Span, 0, len(spans))
	for _, span := range spans {
		if span.Text == "" {
			continue
		}
		last := len(joined) - 1
		if last >= 0 && joined[last].sameMarks(span) {
			joined[last].Text += span.Text
			continue
		}
		joined = append(joined, span)
	}
	return joined
}

func emphasis(level int) string {
	if level >= 2 {
		return markBold
	}
	return markItalic
}

func (s Span) wearing(mark string) Span {
	s.wear(mark)
	return s
}

func (s Span) saying(text string) Span {
	s.Text = text
	return s
}

func (s Span) sameMarks(other Span) bool {
	return s.Bold == other.Bold && s.Italic == other.Italic &&
		s.Strike == other.Strike && s.Code == other.Code && s.Link == other.Link
}
