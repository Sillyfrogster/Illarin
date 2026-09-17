package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
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
	candidate := &asset.Candidate{Version: version}
	added, err := h.assets.AddMedia(c.Request.Context(), asset.AddMediaInput{
		OwnerID: owner.ID,
		AssetID: id,
		Role:    asset.MediaRole(metadata.Role),
		File:    limitedFile,
	}, candidate)
	if work.CandidateResult(c, candidate, err) {
		return
	}
	if errors.Is(err, asset.ErrMediaNotFound) || errors.Is(err, asset.ErrNotFound) {
		api.Refuse(c, http.StatusNotFound, "no such asset")
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
	found, err := h.assets.ListMedia(c.Request.Context(), id, viewerID)
	if errors.Is(err, asset.ErrNotFound) {
		api.Refuse(c, http.StatusNotFound, "no such asset")
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
