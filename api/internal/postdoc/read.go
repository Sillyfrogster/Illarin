package postdoc

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	markBold   = "bold"
	markItalic = "italic"
	markCode   = "code"
	markLink   = "link"
)

const (
	maxNodes        = 2000
	maxDepth        = 6
	maxSpanText     = 5000
	maxDocumentText = 200000
	maxAddress      = 2000
	minHeadingLevel = 2
	maxHeadingLevel = 4
)

// Problem says what is wrong with a document and where it is.
type Problem struct {
	Path    string
	Message string
}

func (p Problem) Error() string {
	return p.Path + ": " + p.Message
}

// blockPlaces is the set of blocks each container accepts.
var blockPlaces = map[string][]string{
	"content":  {"paragraph", "heading", "bulletList", "orderedList", "quote", "divider"},
	"listItem": {"paragraph", "bulletList", "orderedList"},
	"quote":    {"paragraph", "bulletList", "orderedList"},
}

// Read turns submitted bytes into a document, refusing anything outside the
// vocabulary rather than keeping it invisibly.
func Read(raw []byte) (Document, error) {
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		return Document{}, Problem{Path: "document", Message: "Send the post body as a JSON object."}
	}
	if err := onlyKeys("document", body, "version", "content"); err != nil {
		return Document{}, err
	}
	version, err := readInt("document.version", body, "version")
	if err != nil {
		return Document{}, err
	}
	if version != Version {
		return Document{}, Problem{
			Path:    "document.version",
			Message: fmt.Sprintf("This build writes and reads post document version %d.", Version),
		}
	}
	state := &reader{}
	blocks, err := state.blocks("document.content", body["content"], "content", 1)
	if err != nil {
		return Document{}, err
	}
	return Document{Blocks: blocks}, nil
}

// Empty answers whether a document would publish as a blank page.
func (d Document) Empty() bool {
	return d.textLength() == 0
}

func (d Document) textLength() int {
	total := 0
	var walk func([]Block)
	walk = func(blocks []Block) {
		for _, block := range blocks {
			switch shape := block.(type) {
			case Paragraph:
				total += spanLength(shape.Spans)
			case Heading:
				total += spanLength(shape.Spans)
			case List:
				for _, item := range shape.Items {
					walk(item.Blocks)
				}
			case Quote:
				walk(shape.Blocks)
			}
		}
	}
	walk(d.Blocks)
	return total
}

func spanLength(spans []Span) int {
	total := 0
	for _, span := range spans {
		total += len(strings.TrimSpace(span.Text))
	}
	return total
}

// reader carries the counts that bound one document as it is read.
type reader struct {
	nodes int
	text  int
}

func (r *reader) blocks(path string, raw json.RawMessage, place string, depth int) ([]Block, error) {
	if depth > maxDepth {
		return nil, Problem{Path: path, Message: "This post nests its structures too deeply."}
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, Problem{Path: path, Message: "This has to be a list of blocks."}
	}
	blocks := make([]Block, 0, len(entries))
	for index, entry := range entries {
		here := fmt.Sprintf("%s.%d", path, index)
		block, err := r.block(here, entry, place, depth)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

func (r *reader) block(path string, raw json.RawMessage, place string, depth int) (Block, error) {
	r.nodes++
	if r.nodes > maxNodes {
		return nil, Problem{Path: path, Message: "This post has too many blocks."}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, Problem{Path: path, Message: "A block has to be a JSON object."}
	}
	kind, err := readString(path+".type", fields, "type")
	if err != nil {
		return nil, err
	}
	if !allows(place, kind) {
		return nil, Problem{
			Path:    path + ".type",
			Message: fmt.Sprintf("%q cannot go here. Allowed: %s.", kind, strings.Join(blockPlaces[place], ", ")),
		}
	}
	switch kind {
	case "paragraph":
		return r.paragraph(path, fields)
	case "heading":
		return r.heading(path, fields)
	case "bulletList", "orderedList":
		return r.list(path, fields, kind == "orderedList", depth)
	case "quote":
		return r.quote(path, fields, depth)
	case "divider":
		if err := onlyKeys(path, fields, "type"); err != nil {
			return nil, err
		}
		return Divider{}, nil
	}
	return nil, Problem{Path: path + ".type", Message: fmt.Sprintf("%q is not a post block.", kind)}
}

func (r *reader) paragraph(path string, fields map[string]json.RawMessage) (Block, error) {
	if err := onlyKeys(path, fields, "type", "content"); err != nil {
		return nil, err
	}
	spans, err := r.spans(path+".content", fields["content"])
	if err != nil {
		return nil, err
	}
	return Paragraph{Spans: spans}, nil
}

func (r *reader) heading(path string, fields map[string]json.RawMessage) (Block, error) {
	if err := onlyKeys(path, fields, "type", "level", "content"); err != nil {
		return nil, err
	}
	level, err := readInt(path+".level", fields, "level")
	if err != nil {
		return nil, err
	}
	if level < minHeadingLevel || level > maxHeadingLevel {
		return nil, Problem{
			Path: path + ".level",
			Message: fmt.Sprintf(
				"The post title is the page heading, so a body heading is level %d to %d.",
				minHeadingLevel, maxHeadingLevel,
			),
		}
	}
	spans, err := r.spans(path+".content", fields["content"])
	if err != nil {
		return nil, err
	}
	if spanLength(spans) == 0 {
		return nil, Problem{Path: path + ".content", Message: "A heading needs words."}
	}
	return Heading{Level: level, Spans: spans}, nil
}

func (r *reader) list(
	path string,
	fields map[string]json.RawMessage,
	ordered bool,
	depth int,
) (Block, error) {
	if err := onlyKeys(path, fields, "type", "content"); err != nil {
		return nil, err
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(fields["content"], &entries); err != nil {
		return nil, Problem{Path: path + ".content", Message: "A list holds a list of items."}
	}
	if len(entries) == 0 {
		return nil, Problem{Path: path + ".content", Message: "A list needs at least one item."}
	}
	items := make([]Item, 0, len(entries))
	for index, entry := range entries {
		here := fmt.Sprintf("%s.content.%d", path, index)
		item, err := r.item(here, entry, depth+1)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return List{Ordered: ordered, Items: items}, nil
}

func (r *reader) item(path string, raw json.RawMessage, depth int) (Item, error) {
	r.nodes++
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return Item{}, Problem{Path: path, Message: "A list item has to be a JSON object."}
	}
	kind, err := readString(path+".type", fields, "type")
	if err != nil {
		return Item{}, err
	}
	if kind != "listItem" {
		return Item{}, Problem{Path: path + ".type", Message: "A list holds list items."}
	}
	if err := onlyKeys(path, fields, "type", "content"); err != nil {
		return Item{}, err
	}
	blocks, err := r.blocks(path+".content", fields["content"], "listItem", depth+1)
	if err != nil {
		return Item{}, err
	}
	if len(blocks) == 0 {
		return Item{}, Problem{Path: path + ".content", Message: "A list item needs content."}
	}
	return Item{Blocks: blocks}, nil
}

func (r *reader) quote(path string, fields map[string]json.RawMessage, depth int) (Block, error) {
	if err := onlyKeys(path, fields, "type", "content"); err != nil {
		return nil, err
	}
	blocks, err := r.blocks(path+".content", fields["content"], "quote", depth+1)
	if err != nil {
		return nil, err
	}
	if len(blocks) == 0 {
		return nil, Problem{Path: path + ".content", Message: "A quotation needs content."}
	}
	return Quote{Blocks: blocks}, nil
}

func (r *reader) spans(path string, raw json.RawMessage) ([]Span, error) {
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, Problem{Path: path, Message: "This has to be a list of text runs."}
	}
	spans := make([]Span, 0, len(entries))
	for index, entry := range entries {
		here := fmt.Sprintf("%s.%d", path, index)
		span, err := r.span(here, entry)
		if err != nil {
			return nil, err
		}
		spans = append(spans, span)
	}
	return spans, nil
}

func (r *reader) span(path string, raw json.RawMessage) (Span, error) {
	r.nodes++
	if r.nodes > maxNodes {
		return Span{}, Problem{Path: path, Message: "This post has too many blocks."}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return Span{}, Problem{Path: path, Message: "A text run has to be a JSON object."}
	}
	kind, err := readString(path+".type", fields, "type")
	if err != nil {
		return Span{}, err
	}
	if kind != "text" {
		return Span{}, Problem{Path: path + ".type", Message: "Only text runs go inside a paragraph."}
	}
	if err := onlyKeys(path, fields, "type", "text", "marks"); err != nil {
		return Span{}, err
	}
	text, err := readString(path+".text", fields, "text")
	if err != nil {
		return Span{}, err
	}
	if !utf8.ValidString(text) {
		return Span{}, Problem{Path: path + ".text", Message: "This text is not valid UTF-8."}
	}
	if len(text) > maxSpanText {
		return Span{}, Problem{Path: path + ".text", Message: "This run of text is too long."}
	}
	r.text += len(text)
	if r.text > maxDocumentText {
		return Span{}, Problem{Path: path + ".text", Message: "This post is too long."}
	}
	span := Span{Text: text}
	if marks, carried := fields["marks"]; carried {
		if err := readMarks(path+".marks", marks, &span); err != nil {
			return Span{}, err
		}
	}
	return span, nil
}

func readMarks(path string, raw json.RawMessage, span *Span) error {
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return Problem{Path: path, Message: "Marks are a list."}
	}
	for index, entry := range entries {
		here := fmt.Sprintf("%s.%d", path, index)
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(entry, &fields); err != nil {
			return Problem{Path: here, Message: "A mark has to be a JSON object."}
		}
		kind, err := readString(here+".type", fields, "type")
		if err != nil {
			return err
		}
		switch kind {
		case markBold:
			if err := onlyKeys(here, fields, "type"); err != nil {
				return err
			}
			span.Bold = true
		case markItalic:
			if err := onlyKeys(here, fields, "type"); err != nil {
				return err
			}
			span.Italic = true
		case markCode:
			if err := onlyKeys(here, fields, "type"); err != nil {
				return err
			}
			span.Code = true
		case markLink:
			if err := onlyKeys(here, fields, "type", "href"); err != nil {
				return err
			}
			address, err := readString(here+".href", fields, "href")
			if err != nil {
				return err
			}
			if err := checkAddress(here+".href", address); err != nil {
				return err
			}
			span.Link = address
		default:
			return Problem{Path: here + ".type", Message: fmt.Sprintf("%q is not a post mark.", kind)}
		}
	}
	return nil
}

func checkAddress(path, address string) error {
	if len(address) > maxAddress {
		return Problem{Path: path, Message: "This link address is too long."}
	}
	if strings.HasPrefix(address, "https://") && len(address) > len("https://") {
		return nil
	}
	if strings.HasPrefix(address, "mailto:") && len(address) > len("mailto:") {
		return nil
	}
	return Problem{Path: path, Message: "A link goes to an https address or a mailto address."}
}

func allows(place, kind string) bool {
	for _, allowed := range blockPlaces[place] {
		if allowed == kind {
			return true
		}
	}
	return false
}

func onlyKeys(path string, fields map[string]json.RawMessage, allowed ...string) error {
	for key := range fields {
		known := false
		for _, name := range allowed {
			if key == name {
				known = true
				break
			}
		}
		if !known {
			return Problem{
				Path:    path + "." + key,
				Message: fmt.Sprintf("%q is not part of a post document.", key),
			}
		}
	}
	return nil
}

func readString(path string, fields map[string]json.RawMessage, key string) (string, error) {
	raw, carried := fields[key]
	if !carried {
		return "", Problem{Path: path, Message: fmt.Sprintf("%q is missing.", key)}
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", Problem{Path: path, Message: fmt.Sprintf("%q has to be text.", key)}
	}
	return value, nil
}

func readInt(path string, fields map[string]json.RawMessage, key string) (int, error) {
	raw, carried := fields[key]
	if !carried {
		return 0, Problem{Path: path, Message: fmt.Sprintf("%q is missing.", key)}
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, Problem{Path: path, Message: fmt.Sprintf("%q has to be a whole number.", key)}
	}
	return value, nil
}
