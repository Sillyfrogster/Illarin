package page

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) DeleteAsset(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}
	err := h.works.Delete(c.Request.Context(), owner.ID, id)
	switch {
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "no such asset")
	case errors.Is(err, work.ErrAssetFrozen):
		api.Refuse(c, http.StatusConflict, "A withheld asset cannot be deleted.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not delete the asset.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) RestoreAsset(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}
	err := h.works.Restore(c.Request.Context(), owner.ID, id)
	switch {
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "no such recoverable asset")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not restore the asset.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) ListDeletedAssets(c *gin.Context) {
	handle := c.Param("handle")
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}
	if !strings.EqualFold(owner.Handle, handle) {
		api.Refuse(c, http.StatusNotFound, "no such deleted listing")
		return
	}
	found, err := h.works.Deleted(c.Request.Context(), owner.ID, strings.ToLower(handle))
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not list deleted assets.")
		return
	}
	items := make([]DeletedAsset, len(found))
	for i, item := range found {
		items[i] = DeletedAsset{
			Id: item.ID, Name: item.Name, Kind: DeletedAssetKind(item.Kind),
			DeletedAt: item.DeletedAt, RecoverableUntil: item.RecoverableUntil,
		}
	}
	c.JSON(http.StatusOK, DeletedAssetList{Items: items})
}
