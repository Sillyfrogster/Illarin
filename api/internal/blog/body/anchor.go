package body

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	maxAnchor      = 80
	mintedAnchor   = 60
	fallbackAnchor = "section"
)

func (r *reader) anchor(
	path string,
	fields map[string]json.RawMessage,
	spans []Span,
) (string, error) {
	given, carried := fields["anchor"]
	if !carried {
		return r.mint(spans), nil
	}
	var chosen string
	if err := json.Unmarshal(given, &chosen); err != nil {
		return "", Problem{Path: path + ".anchor", Message: `"anchor" has to be text.`}
	}
	if err := checkAnchor(path+".anchor", chosen); err != nil {
		return "", err
	}
	if r.anchors[chosen] {
		return "", Problem{
			Path:    path + ".anchor",
			Message: fmt.Sprintf("Another heading already answers to %q.", chosen),
		}
	}
	r.anchors[chosen] = true
	return chosen, nil
}

func checkAnchor(path, chosen string) error {
	if chosen == "" {
		return Problem{
			Path:    path,
			Message: "Give the heading an address, or leave the anchor out and Illarin writes one.",
		}
	}
	if len(chosen) > maxAnchor {
		return Problem{Path: path, Message: "This heading address is too long."}
	}
	if chosen != slug(chosen) {
		return Problem{
			Path: path,
			Message: "A heading address is lower-case letters, digits and single hyphens, " +
				"such as \"release-notes\".",
		}
	}
	return nil
}

func (r *reader) mint(spans []Span) string {
	stem := slug(cut(words(spans), mintedAnchor))
	if stem == "" {
		stem = fallbackAnchor
	}
	if !r.anchors[stem] {
		r.anchors[stem] = true
		return stem
	}
	for next := 2; ; next++ {
		candidate := stem + "-" + strconv.Itoa(next)
		if !r.anchors[candidate] {
			r.anchors[candidate] = true
			return candidate
		}
	}
}

func words(spans []Span) string {
	var said strings.Builder
	for _, span := range spans {
		said.WriteString(span.Text)
	}
	return said.String()
}

func slug(said string) string {
	var out strings.Builder
	hyphen := false
	for _, letter := range strings.ToLower(said) {
		switch {
		case letter >= 'a' && letter <= 'z', letter >= '0' && letter <= '9':
			if hyphen && out.Len() > 0 {
				out.WriteByte('-')
			}
			hyphen = false
			out.WriteRune(letter)
		default:
			hyphen = true
		}
	}
	return out.String()
}

func cut(said string, limit int) string {
	if len(said) <= limit {
		return said
	}
	shortened := said[:limit]
	if space := strings.LastIndexAny(shortened, " \t-_"); space > limit/2 {
		return shortened[:space]
	}
	return shortened
}
