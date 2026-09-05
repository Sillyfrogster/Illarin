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
	markStrike = "strike"
	markCode   = "code"
	markLink   = "link"
)

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
	fields, kind, err := r.node(path, raw)
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
		case markBold, markItalic, markStrike, markCode:
			if err := onlyKeys(here, fields, "type"); err != nil {
				return err
			}
			span.wear(kind)
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

func (s *Span) wear(mark string) {
	switch mark {
	case markBold:
		s.Bold = true
	case markItalic:
		s.Italic = true
	case markStrike:
		s.Strike = true
	case markCode:
		s.Code = true
	}
}

func checkAddress(path, address string) error {
	if said := addressProblem(address); said != "" {
		return Problem{Path: path, Message: said}
	}
	return nil
}

// addressProblem says why a link address cannot be followed, or nothing.
func addressProblem(address string) string {
	if len(address) > maxAddress {
		return "This link address is too long."
	}
	if strings.ContainsFunc(address, isControl) || strings.ContainsRune(address, ' ') {
		return "A link address carries no spaces or control characters."
	}
	if strings.HasPrefix(address, "https://") && len(address) > len("https://") {
		return ""
	}
	if strings.HasPrefix(address, "mailto:") && len(address) > len("mailto:") {
		return ""
	}
	return "A link goes to an https address or a mailto address."
}
