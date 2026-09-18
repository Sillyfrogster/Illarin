package image

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/upload"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) AddMedia(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	version, ok := api.WorkingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}
	parts, err := c.Request.MultipartReader()
	if err != nil {
		upload.RefuseFile(c, api.FormRefusal{
			Reason: "send the image as form data, with a metadata part and a file part",
			Cause:  err,
		}, h.maxUploadBytes)
		return
	}
	metadata, err := readMediaMetadata(parts)
	if err != nil {
		upload.RefuseFile(c, err, h.maxUploadBytes)
		return
	}
	file, err := api.NextPart(parts, api.FilePart)
	if err != nil {
		upload.RefuseFile(c, err, h.maxUploadBytes)
		return
	}
	limitedFile := http.MaxBytesReader(c.Writer, file, h.maxUploadBytes)
	defer limitedFile.Close()
	candidate := &work.Candidate{Version: version}
	added, err := h.works.AddMedia(c.Request.Context(), work.AddMediaInput{
		OwnerID: owner.ID,
		WorkID:  id,
		Role:    work.MediaRole(metadata.Role),
		File:    limitedFile,
	}, candidate)
	if page.CandidateResult(c, candidate, err) {
		return
	}
	if errors.Is(err, work.ErrMediaNotFound) || errors.Is(err, work.ErrNotFound) {
		api.Refuse(c, http.StatusNotFound, "no such work")
		return
	}
	if err != nil {
		upload.RefuseFile(c, err, h.maxUploadBytes)
		return
	}
	c.JSON(http.StatusCreated, toAPIMedia(added))
}

func (h *Handlers) ListMedia(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	viewerID, ok := api.ViewerID(c)
	if !ok {
		return
	}
	found, err := h.works.ListMedia(c.Request.Context(), id, viewerID)
	if errors.Is(err, work.ErrNotFound) {
		api.Refuse(c, http.StatusNotFound, "no such work")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "could not list the images")
		return
	}
	items := make([]Media, 0, len(found))
	for _, item := range found {
		items = append(items, toAPIMedia(item))
	}
	c.JSON(http.StatusOK, MediaList{Items: items})
}
