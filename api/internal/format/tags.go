package format

import (
	"errors"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	MaxTags     = 32
	MaxTagRunes = 64
)

var ErrInvalidTags = errors.New("tags must be unique, nonempty, at most 32, and at most 64 characters each")

func CheckTags(tags []string) ([]string, error) {
	if len(tags) > MaxTags {
		return nil, ErrInvalidTags
	}
	checked := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || utf8.RuneCountInString(tag) > MaxTagRunes || slices.Contains(checked, tag) {
			return nil, ErrInvalidTags
		}
		checked = append(checked, tag)
	}
	return checked, nil
}
