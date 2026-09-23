package edit

import (
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
)

type AddWorkBlockRequest struct {
	Definition  string            `json:"definition"`
	ElementType block.ElementType `json:"elementType"`
}

type ArrangeWorkBlocksRequest struct {
	Blocks []struct {
		Hidden bool                                `json:"hidden"`
		Id     uuid.UUID                           `json:"id"`
		Width  ArrangeWorkBlocksRequestBlocksWidth `json:"width"`
	} `json:"blocks"`
}

type ArrangeWorkBlocksRequestBlocksWidth string

const (
	ArrangeWorkBlocksRequestBlocksWidthFull      ArrangeWorkBlocksRequestBlocksWidth = "full"
	ArrangeWorkBlocksRequestBlocksWidthHalf      ArrangeWorkBlocksRequestBlocksWidth = "half"
	ArrangeWorkBlocksRequestBlocksWidthThird     ArrangeWorkBlocksRequestBlocksWidth = "third"
	ArrangeWorkBlocksRequestBlocksWidthTwoThirds ArrangeWorkBlocksRequestBlocksWidth = "two_thirds"
)

type MoveWorkBlockContentRequest struct {
	DestinationBlockId uuid.UUID `json:"destinationBlockId"`
}
