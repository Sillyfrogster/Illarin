package edit

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) SaveAssetBlock(c *gin.Context) {
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
	owner, ok := api.Verified(c, "saving an asset")
	if !ok {
		return
	}
	var request SaveAssetBlockRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send valid JSON with a title, layout, width and elements. Every element id must be a UUID; display must be rich or verbatim, and an image size small, medium or large.")
		return
	}
	update, err := blockUpdate(request)
	if err != nil {
		api.Refuse(c, http.StatusBadRequest, err.Error())
		return
	}
	candidate := &asset.Candidate{Version: version}
	saved, err := h.blocks.SaveBlock(
		c.Request.Context(), owner.ID, id, blockID, update, candidate)
	if work.CandidateResult(c, candidate, err) {
		return
	}
	var exposure private.ExposureRefusal
	if errors.As(err, &exposure) {
		c.JSON(http.StatusConflict, SealedExposureRefusal{
			Error: "Saving this makes " + joinNames(exposure.Prompts) +
				" readable by anyone, and puts ordinary downloads back on the asset.",
			Code:    SealedExposureRefusalCodeSealedExposure,
			Prompts: exposure.Prompts,
		})
		return
	}
	switch {
	case errors.Is(err, asset.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such block.")
	case errors.Is(err, block.ErrInvalid):
		api.Refuse(c, http.StatusBadRequest, err.Error())
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not save the block.")
	default:
		blocks, conversionErr := block.ToBlocks(saved.Kind, []block.Block{saved.Block})
		if conversionErr != nil {
			api.Refuse(c, http.StatusInternalServerError, "Could not read the saved block.")
			return
		}
		c.JSON(http.StatusOK, blocks[0])
	}
}

func blockUpdate(request SaveAssetBlockRequest) (BlockUpdate, error) {
	elements := make([]block.Element, len(request.Elements))
	for i, incoming := range request.Elements {
		elementType := block.Type(incoming.Type)
		role := block.Role("")
		if incoming.Role != nil {
			role = block.Role(*incoming.Role)
		}
		content, err := block.DecodeContent(elementType, incoming.Content)
		if err != nil {
			name := block.Element{Type: elementType, Role: role}.Label()
			if name == "" {
				name = fmt.Sprintf("Element %d", i+1)
			}
			return BlockUpdate{}, fmt.Errorf("%s content is malformed: %w", name, err)
		}
		display := block.Display("")
		if incoming.Display != nil {
			display = block.Display(*incoming.Display)
		}
		itemSize := block.ItemSize("")
		if incoming.ItemSize != nil {
			itemSize = block.ItemSize(*incoming.ItemSize)
		}
		elements[i] = block.Element{
			ID: incoming.Id, Type: elementType, Role: role,
			Slot:    block.Slot(incoming.Slot),
			Options: block.Options{Display: display, ItemSize: itemSize},
			Content: content,
		}
	}
	var allowedApps *[]string
	if request.AllowedApps != nil {
		apps := make([]string, len(*request.AllowedApps))
		for i, app := range *request.AllowedApps {
			apps[i] = string(app)
		}
		allowedApps = &apps
	}
	return BlockUpdate{
		Title:           request.Title,
		Layout:          block.Layout(request.Layout),
		Width:           block.Width(request.Width),
		Elements:        elements,
		AllowedApps:     allowedApps,
		ExposeProtected: request.ExposeProtected != nil && *request.ExposeProtected,
	}, nil
}

func joinNames(names []string) string {
	switch len(names) {
	case 1:
		return names[0]
	default:
		return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
	}
}
