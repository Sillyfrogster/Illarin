package format

import (
	"context"
	"slices"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
)

const (
	OriginIllarin = "illarin"
	OriginV1      = "v1"
)

type ExportAsset struct {
	Kind      string
	Header    Header
	Elements  []block.Element
	Cover     *ExportMedia
	Images    map[uuid.UUID]ExportMedia
	Preserved []Remainder
}

type ExportMedia struct {
	MediaType string
	Data      []byte
	URL       string
}

func (a ExportAsset) Element(role block.Role) (block.Element, bool) {
	for _, element := range a.Elements {
		if element.Role == role && element.Content != nil && !element.Content.Empty() {
			return element, true
		}
	}
	return block.Element{}, false
}

func (a ExportAsset) Content(role block.Role) (block.Content, bool) {
	element, ok := a.Element(role)
	if !ok {
		return nil, false
	}
	return element.Content, true
}

func (a ExportAsset) Text(role block.Role) string {
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

type Artifact struct {
	Body      []byte
	MediaType string
	Extension string
}

type Writer interface {
	Module
	Write(context.Context, ExportAsset) (Artifact, error)
}

func TravelsWithOrigin(origin, target Declaration) bool {
	if slices.Contains(target.PreservesOrigins, origin.ID) {
		return true
	}
	return origin.Preservation.Body == target.Preservation.Body &&
		slices.Equal(origin.Preservation.Container, target.Preservation.Container)
}
