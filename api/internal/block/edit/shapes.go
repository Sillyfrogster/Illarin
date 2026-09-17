package edit

import (
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
)

type AddAssetBlockRequest struct {
	Definition  string            `json:"definition"`
	ElementType block.ElementType `json:"elementType"`
}

type ArrangeAssetBlocksRequest struct {
	Blocks []struct {
		Hidden bool                                 `json:"hidden"`
		Id     uuid.UUID                            `json:"id"`
		Width  ArrangeAssetBlocksRequestBlocksWidth `json:"width"`
	} `json:"blocks"`
}

type ArrangeAssetBlocksRequestBlocksWidth string

const (
	ArrangeAssetBlocksRequestBlocksWidthFull      ArrangeAssetBlocksRequestBlocksWidth = "full"
	ArrangeAssetBlocksRequestBlocksWidthHalf      ArrangeAssetBlocksRequestBlocksWidth = "half"
	ArrangeAssetBlocksRequestBlocksWidthThird     ArrangeAssetBlocksRequestBlocksWidth = "third"
	ArrangeAssetBlocksRequestBlocksWidthTwoThirds ArrangeAssetBlocksRequestBlocksWidth = "two_thirds"
)

type MoveAssetBlockContentRequest struct {
	DestinationBlockId uuid.UUID `json:"destinationBlockId"`
}
