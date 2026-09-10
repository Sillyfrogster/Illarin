package postdoc

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

const maxMarkdownSource = 400000

const maxNotes = 40

const MediaPrefix = "media:"

type Note struct {
	Line    int
	Message string
}

type Refused struct {
	Notes []Note
}

func (r Refused) Error() string {
	said := make([]string, 0, len(r.Notes))
	for _, note := range r.Notes {
		said = append(said, fmt.Sprintf("line %d: %s", note.Line, note.Message))
	}
	return strings.Join(said, "; ")
}

var constrained = goldmark.New(goldmark.WithExtensions(
	extension.Table, extension.Strikethrough, extension.TaskList, extension.Footnote,
))

func FromMarkdown(source string) ([]byte, []Note, error) {
	if len(source) > maxMarkdownSource {
		return nil, nil, Refused{Notes: []Note{{
			Line: 1, Message: "This Markdown is longer than a post can be.",
		}}}
	}
	raw := []byte(source)
	state := &importer{source: raw, starts: lineStarts(raw)}
	root := constrained.Parser().Parse(text.NewReader(raw))
	blocks := state.blocks(root.FirstChild(), placeContent)
	if len(state.refusals) > 0 {
		return nil, nil, Refused{Notes: settle(state.refusals)}
	}
	written, err := json.Marshal(Document{Blocks: blocks})
	if err != nil {
		return nil, nil, fmt.Errorf("write the imported body: %w", err)
	}
	read, err := Read(written)
	if err != nil {
		return nil, nil, err
	}
	canonical, err := json.Marshal(read)
	if err != nil {
		return nil, nil, fmt.Errorf("write the imported body: %w", err)
	}
	return canonical, settle(state.notes), nil
}

type importer struct {
	source   []byte
	starts   []int
	notes    []Note
	refusals []Note
}

func (i *importer) note(node ast.Node, message string) {
	i.notes = append(i.notes, Note{Line: i.lineOf(node), Message: message})
}

func (i *importer) refuse(node ast.Node, message string) {
	i.refusals = append(i.refusals, Note{Line: i.lineOf(node), Message: message})
}

func settle(notes []Note) []Note {
	sort.SliceStable(notes, func(one, two int) bool {
		return notes[one].Line < notes[two].Line
	})
	notes = once(notes)
	if len(notes) <= maxNotes {
		return notes
	}
	kept := notes[:maxNotes:maxNotes]
	return append(kept, Note{
		Line:    notes[maxNotes].Line,
		Message: "There is more past this line. Settle these and import again.",
	})
}

func once(notes []Note) []Note {
	kept := make([]Note, 0, len(notes))
	said := make(map[Note]bool, len(notes))
	for _, note := range notes {
		if said[note] {
			continue
		}
		said[note] = true
		kept = append(kept, note)
	}
	return kept
}

func lineStarts(source []byte) []int {
	starts := []int{0}
	for at, letter := range source {
		if letter == '\n' {
			starts = append(starts, at+1)
		}
	}
	return starts
}

func (i *importer) lineOf(node ast.Node) int {
	at, found := offsetOf(node)
	if !found {
		return 1
	}
	return sort.SearchInts(i.starts, at+1)
}

func offsetOf(node ast.Node) (int, bool) {
	for at := node; at != nil; at = at.Parent() {
		if found, ok := selfOffset(at); ok {
			return found, true
		}
	}
	return 0, false
}

func selfOffset(node ast.Node) (int, bool) {
	if word, ok := node.(*ast.Text); ok {
		return word.Segment.Start, true
	}
	if node.Type() != ast.TypeInline {
		if lines := node.Lines(); lines != nil && lines.Len() > 0 {
			return lines.At(0).Start, true
		}
	}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if at, ok := selfOffset(child); ok {
			return at, true
		}
	}
	return 0, false
}

func (i *importer) lines(segments *text.Segments) string {
	var out strings.Builder
	for at := range segments.Len() {
		segment := segments.At(at)
		out.Write(segment.Value(i.source))
	}
	return out.String()
}
