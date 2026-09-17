package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) ListPreservedNamespaces(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	owner, ok := h.verifiedAccount(c, "reading preserved data")
	if !ok {
		return
	}
	found, err := h.assets.PreservedNamespaces(c.Request.Context(), owner.ID, id)
	switch {
	case errors.Is(err, asset.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such asset."})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load extra file data. Try again."})
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
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	namespace := c.Param("namespace")
	version, ok := workingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := h.verifiedAccount(c, "deleting preserved data")
	if !ok {
		return
	}
	candidate := &asset.Candidate{Version: version}
	err := h.assets.DeletePreservedNamespace(
		c.Request.Context(), owner.ID, id, namespace, candidate)
	if candidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "This asset preserves no such data."})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete the preserved data."})
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) ExportSealedContent(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	owner, ok := h.verifiedAccount(c, "reading sealed content")
	if !ok {
		return
	}
	sealed, err := h.assets.OpenSealedContent(c.Request.Context(), owner.ID, id)
	switch {
	case errors.Is(err, asset.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "This asset holds no sealed content."})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the sealed content."})
	default:
		c.Header("Content-Disposition", `attachment; filename="`+sealed.Filename+`"`)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Cache-Control", "private, no-store")
		c.Data(http.StatusOK, sealed.MediaType, sealed.Body)
	}
}
