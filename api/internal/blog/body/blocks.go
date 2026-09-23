package body

import (
	"encoding/json"
	"fmt"
	"strings"
)

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
	if err := onlyKeys(path, fields, "type", "level", "anchor", "content"); err != nil {
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
	anchor, err := r.anchor(path, fields, spans)
	if err != nil {
		return nil, err
	}
	return Heading{Level: level, Anchor: anchor, Spans: spans}, nil
}

func (r *reader) list(
	path string,
	fields map[string]json.RawMessage,
	ordered bool,
	depth int,
) (Block, error) {
	entries, err := entryList(path, fields, "A list holds a list of items.")
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, Problem{Path: path + ".content", Message: "A list needs at least one item."}
	}
	items := make([]Item, 0, len(entries))
	for index, entry := range entries {
		here := fmt.Sprintf("%s.content.%d", path, index)
		_, blocks, err := r.entry(here, entry, "listItem", depth)
		if err != nil {
			return nil, err
		}
		items = append(items, Item{Blocks: blocks})
	}
	return List{Ordered: ordered, Items: items}, nil
}

func (r *reader) taskList(path string, fields map[string]json.RawMessage, depth int) (Block, error) {
	entries, err := entryList(path, fields, "A task list holds a list of tasks.")
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, Problem{Path: path + ".content", Message: "A task list needs at least one task."}
	}
	tasks := make([]Task, 0, len(entries))
	for index, entry := range entries {
		here := fmt.Sprintf("%s.content.%d", path, index)
		fields, blocks, err := r.entry(here, entry, "taskItem", depth)
		if err != nil {
			return nil, err
		}
		done, err := readBool(here+".done", fields, "done")
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, Task{Done: done, Blocks: blocks})
	}
	return TaskList{Tasks: tasks}, nil
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

func (r *reader) callout(path string, fields map[string]json.RawMessage, depth int) (Block, error) {
	if err := onlyKeys(path, fields, "type", "kind", "content"); err != nil {
		return nil, err
	}
	calloutType, err := readString(path+".kind", fields, "kind")
	if err != nil {
		return nil, err
	}
	if !isCalloutType(calloutType) {
		return nil, Problem{
			Path:    path + ".kind",
			Message: fmt.Sprintf("A callout is one of: %s.", strings.Join(CalloutTypes, ", ")),
		}
	}
	blocks, err := r.blocks(path+".content", fields["content"], "callout", depth+1)
	if err != nil {
		return nil, err
	}
	if len(blocks) == 0 {
		return nil, Problem{Path: path + ".content", Message: "A callout needs content."}
	}
	return Callout{Type: calloutType, Blocks: blocks}, nil
}

func (r *reader) codeBlock(path string, fields map[string]json.RawMessage) (Block, error) {
	if err := onlyKeys(path, fields, "type", "language", "source"); err != nil {
		return nil, err
	}
	language, err := readString(path+".language", fields, "language")
	if err != nil {
		return nil, err
	}
	if !isLanguage(language) {
		return nil, Problem{
			Path:    path + ".language",
			Message: fmt.Sprintf("Illarin labels code with one of: %s.", strings.Join(Languages, ", ")),
		}
	}
	source, err := readString(path+".source", fields, "source")
	if err != nil {
		return nil, err
	}
	source = canonicalSource(source)
	if strings.TrimSpace(source) == "" {
		return nil, Problem{Path: path + ".source", Message: "A code block needs code in it."}
	}
	if len(source) > maxCodeSource {
		return nil, Problem{Path: path + ".source", Message: "This code block is too long."}
	}
	if column := strings.IndexFunc(source, isControl); column >= 0 {
		return nil, Problem{
			Path:    path + ".source",
			Message: fmt.Sprintf("This code carries a control character at byte %d.", column),
		}
	}
	r.text += len(source)
	if r.text > maxDocumentText {
		return nil, Problem{Path: path + ".source", Message: "This post is too long."}
	}
	return CodeBlock{Language: language, Source: source}, nil
}

func (r *reader) table(path string, fields map[string]json.RawMessage, depth int) (Block, error) {
	entries, err := entryList(path, fields, "A table holds a list of rows.")
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, Problem{Path: path + ".content", Message: "A table needs at least one row."}
	}
	if len(entries) > maxTableRows {
		return nil, Problem{Path: path + ".content", Message: "This table has too many rows."}
	}
	rows := make([]Row, 0, len(entries))
	for index, entry := range entries {
		here := fmt.Sprintf("%s.content.%d", path, index)
		row, err := r.tableRow(here, entry, depth+1)
		if err != nil {
			return nil, err
		}
		if index > 0 && len(row.Cells) != len(rows[0].Cells) {
			return nil, Problem{
				Path: here,
				Message: fmt.Sprintf(
					"Every row of a table holds the same number of cells. This one holds %d and the first holds %d.",
					len(row.Cells), len(rows[0].Cells),
				),
			}
		}
		rows = append(rows, row)
	}
	if err := checkTableHeadings(path, rows); err != nil {
		return nil, err
	}
	return Table{Rows: rows}, nil
}

func (r *reader) tableRow(path string, raw json.RawMessage, depth int) (Row, error) {
	fields, calloutType, err := r.node(path, raw)
	if err != nil {
		return Row{}, err
	}
	if calloutType != "tableRow" {
		return Row{}, Problem{Path: path + ".type", Message: "A table holds table rows."}
	}
	entries, err := entryList(path, fields, "A table row holds a list of cells.")
	if err != nil {
		return Row{}, err
	}
	if len(entries) == 0 {
		return Row{}, Problem{Path: path + ".content", Message: "A table row needs at least one cell."}
	}
	if len(entries) > maxTableColumns {
		return Row{}, Problem{Path: path + ".content", Message: "This table has too many columns."}
	}
	cells := make([]Cell, 0, len(entries))
	for index, entry := range entries {
		here := fmt.Sprintf("%s.content.%d", path, index)
		cell, err := r.tableCell(here, entry, depth+1)
		if err != nil {
			return Row{}, err
		}
		cells = append(cells, cell)
	}
	return Row{Cells: cells}, nil
}

func (r *reader) tableCell(path string, raw json.RawMessage, depth int) (Cell, error) {
	fields, calloutType, err := r.node(path, raw)
	if err != nil {
		return Cell{}, err
	}
	if calloutType != "tableCell" {
		return Cell{}, Problem{Path: path + ".type", Message: "A table row holds table cells."}
	}
	if err := onlyKeys(path, fields, "type", "heading", "content"); err != nil {
		return Cell{}, err
	}
	heading := false
	if _, carried := fields["heading"]; carried {
		if heading, err = readBool(path+".heading", fields, "heading"); err != nil {
			return Cell{}, err
		}
	}
	blocks, err := r.blocks(path+".content", fields["content"], "tableCell", depth+1)
	if err != nil {
		return Cell{}, err
	}
	return Cell{Heading: heading, Blocks: blocks}, nil
}

func checkTableHeadings(path string, rows []Row) error {
	headingRow := wholeHeading(rows[0].Cells)
	first := make([]Cell, 0, len(rows))
	for _, row := range rows {
		first = append(first, row.Cells[0])
	}
	headingColumn := wholeHeading(first)
	for down, row := range rows {
		for across, cell := range row.Cells {
			want := (down == 0 && headingRow) || (across == 0 && headingColumn)
			if cell.Heading == want {
				continue
			}
			return Problem{
				Path: fmt.Sprintf("%s.content.%d.content.%d.heading", path, down, across),
				Message: "A table's heading cells fill its first row, its first column, or both. " +
					"This one does not follow from the cells around it.",
			}
		}
	}
	return nil
}

func (r *reader) entry(
	path string,
	raw json.RawMessage,
	place string,
	depth int,
) (map[string]json.RawMessage, []Block, error) {
	fields, calloutType, err := r.node(path, raw)
	if err != nil {
		return nil, nil, err
	}
	if calloutType != place {
		return nil, nil, Problem{
			Path:    path + ".type",
			Message: fmt.Sprintf("This list holds %s entries.", place),
		}
	}
	allowed := []string{"type", "content"}
	if place == "taskItem" {
		allowed = append(allowed, "done")
	}
	if err := onlyKeys(path, fields, allowed...); err != nil {
		return nil, nil, err
	}
	blocks, err := r.blocks(path+".content", fields["content"], place, depth+1)
	if err != nil {
		return nil, nil, err
	}
	if len(blocks) == 0 {
		return nil, nil, Problem{Path: path + ".content", Message: "This entry needs content."}
	}
	return fields, blocks, nil
}

func entryList(
	path string,
	fields map[string]json.RawMessage,
	complaint string,
) ([]json.RawMessage, error) {
	if err := onlyKeys(path, fields, "type", "content"); err != nil {
		return nil, err
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(fields["content"], &entries); err != nil {
		return nil, Problem{Path: path + ".content", Message: complaint}
	}
	return entries, nil
}

func canonicalSource(source string) string {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	source = strings.ReplaceAll(source, "\r", "\n")
	return strings.TrimRight(source, "\n")
}

func wholeHeading(cells []Cell) bool {
	for _, cell := range cells {
		if !cell.Heading {
			return false
		}
	}
	return true
}

func isControl(letter rune) bool {
	return letter < 0x20 && letter != '\n' && letter != '\t'
}
