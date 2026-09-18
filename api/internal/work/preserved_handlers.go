package work

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) ListPreservedNamespaces(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "reading preserved data")
	if !ok {
		return
	}
	found, err := h.works.PreservedNamespaces(c.Request.Context(), owner.ID, id)
	switch {
	case errors.Is(err, asset.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such asset.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not load extra file data. Try again.")
	default:
		served := make([]PreservedNamespace, 0, len(found))
		for _, namespace := range found {
			served = append(served, PreservedNamespace{
				Name: namespace.Name, Bytes: namespace.Bytes,
			})
		}
		c.JSON(http.StatusOK, served)
	}
}

func (h *Handlers) DeletePreservedNamespace(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	namespace := c.Param("namespace")
	version, ok := api.WorkingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "deleting preserved data")
	if !ok {
		return
	}
	candidate := &asset.Candidate{Version: version}
	err := h.works.DeletePreservedNamespace(
		c.Request.Context(), owner.ID, id, namespace, candidate)
	if CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "This asset preserves no such data.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not delete the preserved data.")
	default:
		c.Status(http.StatusNoContent)
	}
}
