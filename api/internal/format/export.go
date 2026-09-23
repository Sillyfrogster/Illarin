package format

import (
	"context"
	"slices"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
)

const (
	OriginalFormatIllarin = "illarin"
	OriginalFormatV1      = "v1"
)

type ExportWork struct {
	Type      string
	Header    Header
	Elements  []block.Element
	Cover     *ExportMedia
	Images    map[uuid.UUID]ExportMedia
	Preserved []Remainder
	Upload    []byte
}

type ExportMedia struct {
	MediaType string
	Data      []byte
	URL       string
}

func (a ExportWork) Element(role block.Role) (block.Element, bool) {
	for _, element := range a.Elements {
		if element.Role == role && element.Content != nil && !element.Content.Empty() {
			return element, true
		}
	}
	return block.Element{}, false
}

func (a ExportWork) Content(role block.Role) (block.Content, bool) {
	element, ok := a.Element(role)
	if !ok {
		return nil, false
	}
	return element.Content, true
}

func (a ExportWork) Text(role block.Role) string {
	content, ok := a.Content(role)
	if !ok {
		return ""
	}
	prose, isProse := content.(block.Prose)
	if !isProse {
		return ""
	}
	return prose.Text
}

type MainFile struct {
	Body      []byte
	MediaType string
	Extension string
}

type Writer interface {
	Module
	Write(context.Context, ExportWork) (MainFile, error)
}

// TravelsWithOriginalFormat says whether data preserved from one original format goes into target's export, even when that format has no module left.
func (r *Registry) TravelsWithOriginalFormat(originalFormat string, target Declaration) bool {
	if slices.Contains(target.PreservesOriginalFormats, originalFormat) {
		return true
	}
	declared, known := r.Declaration(originalFormat)
	return known && declared.Preservation.Body == target.Preservation.Body &&
		slices.Equal(declared.Preservation.Container, target.Preservation.Container)
}

// Filename names a written file after its work, version and format
func Filename(name, update, label, extension string) string {
	parts := make([]string, 0, 3)
	for _, part := range []string{name, update, label} {
		if slug := filenameSlug(part); slug != "" {
			parts = append(parts, slug)
		}
	}
	if len(parts) == 0 {
		return "download" + extension
	}
	return strings.Join(parts, "-") + extension
}

func filenameSlug(text string) string {
	slug := make([]rune, 0, len(text))
	for _, letter := range strings.ToLower(text) {
		switch {
		case letter >= 'a' && letter <= 'z', letter >= '0' && letter <= '9':
			slug = append(slug, letter)
		case len(slug) > 0 && slug[len(slug)-1] != '-':
			slug = append(slug, '-')
		}
	}
	return strings.Trim(string(slug), "-")
}
