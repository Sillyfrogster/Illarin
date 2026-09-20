package page

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) DeleteWork(c *gin.Context) {
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
		api.Refuse(c, http.StatusNotFound, "no such work")
	case errors.Is(err, work.ErrWorkFrozen):
		api.Refuse(c, http.StatusConflict, "A taken-down work cannot be deleted.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not delete the work.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) RestoreWork(c *gin.Context) {
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
		api.Refuse(c, http.StatusNotFound, "no such recoverable work")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not restore the work.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) ListDeletedWorks(c *gin.Context) {
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
		api.Refuse(c, http.StatusInternalServerError, "Could not list deleted works.")
		return
	}
	c.JSON(http.StatusOK, DeletedWorkList{Items: found})
}
