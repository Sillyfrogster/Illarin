package edit

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) AddWorkBlock(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	version, ok := api.WorkingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "adding a block")
	if !ok {
		return
	}
	var request AddWorkBlockRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Name the block to add and the element it starts with.")
		return
	}
	candidate := &work.Candidate{Version: version}
	saved, err := h.blocks.AddBlock(
		c.Request.Context(), owner.ID, id,
		block.DefinitionID(request.Definition), block.Type(request.ElementType), candidate)
	if page.CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such work.")
	case errors.Is(err, block.ErrInvalid):
		api.Refuse(c, http.StatusBadRequest, err.Error())
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not add the block.")
	default:
		blocks, conversionErr := block.ToBlocks(saved.Type, []block.Block{saved.Block})
		if conversionErr != nil {
			api.Refuse(c, http.StatusInternalServerError, "Could not read the new block.")
			return
		}
		c.JSON(http.StatusCreated, blocks[0])
	}
}

func (h *Handlers) ArrangeWorkBlocks(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	version, ok := api.WorkingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "arranging a work")
	if !ok {
		return
	}
	var request ArrangeWorkBlocksRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send every block once with its id, hidden state and width.")
		return
	}
	arrangement := make([]BlockArrangement, len(request.Blocks))
	for i, choice := range request.Blocks {
		arrangement[i] = BlockArrangement{
			ID: choice.Id, Hidden: choice.Hidden, Width: block.Width(choice.Width),
		}
	}
	candidate := &work.Candidate{Version: version}
	saved, err := h.blocks.ArrangeBlocks(c.Request.Context(), owner.ID, id, arrangement, candidate)
	if page.CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such work.")
	case errors.Is(err, block.ErrInvalid):
		api.Refuse(c, http.StatusBadRequest, err.Error())
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not arrange the blocks.")
	default:
		blocks, conversionErr := block.ToBlocks(saved.Type, saved.Blocks)
		if conversionErr != nil {
			api.Refuse(c, http.StatusInternalServerError, "Could not read the arranged blocks.")
			return
		}
		c.JSON(http.StatusOK, blocks)
	}
}

func (h *Handlers) RemoveWorkBlock(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	blockID, ok := api.PathID(c, "blockId")
	if !ok {
		return
	}
	version, ok := api.WorkingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "removing a block")
	if !ok {
		return
	}
	candidate := &work.Candidate{Version: version}
	err := h.blocks.RemoveBlock(c.Request.Context(), owner.ID, id, blockID, candidate)
	if page.CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such block.")
	case errors.Is(err, block.ErrInvalid):
		api.Refuse(c, http.StatusBadRequest, err.Error())
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not remove the block.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) MoveWorkBlockContent(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	blockID, ok := api.PathID(c, "blockId")
	if !ok {
		return
	}
	version, ok := api.WorkingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "moving block content")
	if !ok {
		return
	}
	var request MoveWorkBlockContentRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Choose the block that should keep this content.")
		return
	}
	candidate := &work.Candidate{Version: version}
	saved, err := h.blocks.MoveBlockContent(
		c.Request.Context(), owner.ID, id, blockID, request.DestinationBlockId, candidate)
	if page.CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such block.")
	case errors.Is(err, block.ErrInvalid):
		api.Refuse(c, http.StatusBadRequest, err.Error())
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not move the block content.")
	default:
		blocks, conversionErr := block.ToBlocks(saved.Type, saved.Blocks)
		if conversionErr != nil {
			api.Refuse(c, http.StatusInternalServerError, "Could not read the arranged blocks.")
			return
		}
		c.JSON(http.StatusOK, blocks)
	}
}
