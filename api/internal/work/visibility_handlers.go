package work

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) SetAssetDiscovery(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}
	var request AssetDiscoveryRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil || !request.Discovery.Valid() {
		api.Refuse(c, http.StatusBadRequest, "Choose listed or unlisted.")
		return
	}
	err := h.works.SetDiscovery(
		c.Request.Context(), owner.ID, id, asset.Discovery(request.Discovery),
	)
	switch {
	case errors.Is(err, asset.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "no such asset")
	case errors.Is(err, asset.ErrAssetFrozen):
		api.Refuse(c, http.StatusConflict, "A withheld asset cannot be changed.")
	case errors.Is(err, asset.ErrAssetIsDraft):
		api.Refuse(c, http.StatusConflict, "Discovery applies once the asset is published.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not save the catalog listing. Try again.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) PublishAsset(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	version, ok := api.WorkingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "publishing an asset")
	if !ok {
		return
	}
	candidate := &asset.Candidate{Version: version}
	items, err := h.works.Publish(c.Request.Context(), owner.ID, id, candidate)
	if CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrPublishFloor):
		notReady := PublishRefusalCodeNotReady
		c.JSON(http.StatusConflict, PublishRefusal{
			Error:     "This draft is not ready to publish yet.",
			Code:      &notReady,
			Readiness: ToReadiness(items),
		})
		return
	case errors.Is(err, ErrAlreadyPublished):
		published := PublishRefusalCodeAlreadyPublished
		c.JSON(http.StatusConflict, PublishRefusal{
			Error: "This asset is already published.", Code: &published,
		})
		return
	case errors.Is(err, asset.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such draft.")
		return
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not publish the asset.")
		return
	}

	visibility, ok := ReaderVisibility(c, h.accounts, nil)
	if !ok {
		return
	}
	found, err := h.works.Detail(c.Request.Context(), id, &owner.ID, visibility)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the published asset.")
		return
	}
	page, err := ToPage(found, visibility)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the published asset.")
		return
	}
	c.JSON(http.StatusOK, page)
}

func ToReadiness(items []asset.ReadinessItem) *[]ReadinessItem {
	if items == nil {
		return nil
	}
	out := make([]ReadinessItem, 0, len(items))
	for _, item := range items {
		served := ReadinessItem{
			Id: item.ID, Label: item.Label, Detail: item.Detail, Met: item.Met,
		}
		if item.BlockID != nil {
			blockID := *item.BlockID
			served.BlockId = &blockID
		}
		out = append(out, served)
	}
	return &out
}
