package body

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	maxNodes        = 4000
	maxDepth        = 8
	maxSpanText     = 5000
	maxDocumentText = 200000
	maxCodeSource   = 20000
	maxAddress      = 2000
	maxTableRows    = 60
	maxTableColumns = 10
	minHeadingLevel = 2
	maxHeadingLevel = 4
)

type Problem struct {
	Path    string
	Message string
}

func (p Problem) Error() string {
	return p.Path + ": " + p.Message
}

var blockPlaces = map[string][]string{
	"content": {
		"paragraph", "heading", "bulletList", "orderedList", "taskList",
		"quote", "codeBlock", "table", "callout", "image", "gallery", "divider",
	},
	"listItem":  {"paragraph", "bulletList", "orderedList", "taskList", "codeBlock"},
	"taskItem":  {"paragraph", "bulletList", "orderedList", "taskList", "codeBlock"},
	"quote":     {"paragraph", "bulletList", "orderedList", "codeBlock"},
	"callout":   {"paragraph", "bulletList", "orderedList", "taskList", "codeBlock"},
	"tableCell": {"paragraph"},
}

func Read(raw []byte) (Document, error) {
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		return Document{}, Problem{Path: "body", Message: "Send the post body as a JSON object."}
	}
	if err := onlyKeys("body", body, "version", "content"); err != nil {
		return Document{}, err
	}
	version, err := readInt("body.version", body, "version")
	if err != nil {
		return Document{}, err
	}
	if version < firstVersion || version > Version {
		return Document{}, Problem{
			Path: "body.version",
			Message: fmt.Sprintf(
				"This build reads post body versions %d to %d and writes %d.",
				firstVersion, Version, Version,
			),
		}
	}
	state := &reader{anchors: map[string]bool{}}
	blocks, err := state.blocks("body.content", body["content"], "content", 1)
	if err != nil {
		return Document{}, err
	}
	return Document{Blocks: blocks}, nil
}

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
			case TaskList:
				for _, task := range shape.Tasks {
					walk(task.Blocks)
				}
			case Quote:
				walk(shape.Blocks)
			case CodeBlock:
				total += len(strings.TrimSpace(shape.Source))
			case Table:
				for _, row := range shape.Rows {
					for _, cell := range row.Cells {
						walk(cell.Blocks)
					}
				}
			case Callout:
				walk(shape.Blocks)
			case Image:
				total += len(shape.Alt)
			case Gallery:
				for _, picture := range shape.Images {
					total += len(picture.Alt)
				}
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

type reader struct {
	nodes   int
	text    int
	anchors map[string]bool
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
	fields, calloutType, err := r.node(path, raw)
	if err != nil {
		return nil, err
	}
	if !allows(place, calloutType) {
		if !isBlock(calloutType) {
			return nil, Problem{
				Path:    path + ".type",
				Message: fmt.Sprintf("%q is not a post block.", calloutType),
			}
		}
		return nil, Problem{
			Path:    path + ".type",
			Message: fmt.Sprintf("%q cannot go here. Allowed: %s.", calloutType, strings.Join(blockPlaces[place], ", ")),
		}
	}
	switch calloutType {
	case "paragraph":
		return r.paragraph(path, fields)
	case "heading":
		return r.heading(path, fields)
	case "bulletList", "orderedList":
		return r.list(path, fields, calloutType == "orderedList", depth)
	case "taskList":
		return r.taskList(path, fields, depth)
	case "quote":
		return r.quote(path, fields, depth)
	case "codeBlock":
		return r.codeBlock(path, fields)
	case "table":
		return r.table(path, fields, depth)
	case "callout":
		return r.callout(path, fields, depth)
	case "image":
		return r.image(path, fields)
	case "gallery":
		return r.gallery(path, fields)
	case "divider":
		if err := onlyKeys(path, fields, "type"); err != nil {
			return nil, err
		}
		return Divider{}, nil
	}
	return nil, Problem{Path: path + ".type", Message: fmt.Sprintf("%q is not a post block.", calloutType)}
}

func (r *reader) node(path string, raw json.RawMessage) (map[string]json.RawMessage, string, error) {
	r.nodes++
	if r.nodes > maxNodes {
		return nil, "", Problem{Path: path, Message: "This post has too many blocks."}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, "", Problem{Path: path, Message: "A block has to be a JSON object."}
	}
	calloutType, err := readString(path+".type", fields, "type")
	if err != nil {
		return nil, "", err
	}
	return fields, calloutType, nil
}

func isBlock(calloutType string) bool {
	for _, allowed := range blockPlaces {
		if contains(allowed, calloutType) {
			return true
		}
	}
	return false
}

func allows(place, calloutType string) bool {
	return contains(blockPlaces[place], calloutType)
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
				Message: fmt.Sprintf("%q is not part of a post body.", key),
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

func readBool(path string, fields map[string]json.RawMessage, key string) (bool, error) {
	raw, carried := fields[key]
	if !carried {
		return false, Problem{Path: path, Message: fmt.Sprintf("%q is missing.", key)}
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, Problem{Path: path, Message: fmt.Sprintf("%q has to be true or false.", key)}
	}
	return value, nil
}
