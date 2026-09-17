package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) AddAssetBlock(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	version, ok := workingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := h.verifiedAccount(c, "adding a block")
	if !ok {
		return
	}
	var request AddAssetBlockRequest
	if err := decodeOneJSON(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name the block to add and the element it starts with."})
		return
	}
	candidate := &asset.Candidate{Version: version}
	saved, err := h.assets.AddBlock(
		c.Request.Context(), owner.ID, id,
		block.DefinitionID(request.Definition), block.Type(request.ElementType), candidate)
	if candidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such asset."})
	case errors.Is(err, asset.ErrInvalidBlock):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not add the block."})
	default:
		blocks, conversionErr := toAPIBlocks(saved.Kind, []block.Block{saved.Block})
		if conversionErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the new block."})
			return
		}
		c.JSON(http.StatusCreated, blocks[0])
	}
}

func (h *Handlers) ArrangeAssetBlocks(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	version, ok := workingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := h.verifiedAccount(c, "arranging an asset")
	if !ok {
		return
	}
	var request ArrangeAssetBlocksRequest
	if err := decodeOneJSON(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send every block once with its id, hidden state and width."})
		return
	}
	arrangement := make([]asset.BlockArrangement, len(request.Blocks))
	for i, choice := range request.Blocks {
		arrangement[i] = asset.BlockArrangement{
			ID: choice.Id, Hidden: choice.Hidden, Width: block.Width(choice.Width),
		}
	}
	candidate := &asset.Candidate{Version: version}
	saved, err := h.assets.ArrangeBlocks(c.Request.Context(), owner.ID, id, arrangement, candidate)
	if candidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such asset."})
	case errors.Is(err, asset.ErrInvalidBlock):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not arrange the blocks."})
	default:
		blocks, conversionErr := toAPIBlocks(saved.Kind, saved.Blocks)
		if conversionErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the arranged blocks."})
			return
		}
		c.JSON(http.StatusOK, blocks)
	}
}

func (h *Handlers) RemoveAssetBlock(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	blockID, ok := pathID(c, "blockId")
	if !ok {
		return
	}
	version, ok := workingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := h.verifiedAccount(c, "removing a block")
	if !ok {
		return
	}
	candidate := &asset.Candidate{Version: version}
	err := h.assets.RemoveBlock(c.Request.Context(), owner.ID, id, blockID, candidate)
	if candidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such block."})
	case errors.Is(err, asset.ErrInvalidBlock):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not remove the block."})
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) MoveAssetBlockContent(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	blockID, ok := pathID(c, "blockId")
	if !ok {
		return
	}
	version, ok := workingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := h.verifiedAccount(c, "moving block content")
	if !ok {
		return
	}
	var request MoveAssetBlockContentRequest
	if err := decodeOneJSON(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Choose the block that should keep this content."})
		return
	}
	candidate := &asset.Candidate{Version: version}
	saved, err := h.assets.MoveBlockContent(
		c.Request.Context(), owner.ID, id, blockID, request.DestinationBlockId, candidate)
	if candidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such block."})
	case errors.Is(err, asset.ErrInvalidBlock):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not move the block content."})
	default:
		blocks, conversionErr := toAPIBlocks(saved.Kind, saved.Blocks)
		if conversionErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the arranged blocks."})
			return
		}
		c.JSON(http.StatusOK, blocks)
	}
}
