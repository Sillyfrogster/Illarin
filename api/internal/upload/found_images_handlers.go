package upload

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handlers) ListFoundImages(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.SignedIn(c, "reading found images")
	if !ok {
		return
	}
	pictures, err := h.uploads.ListFoundImages(c.Request.Context(), owner.ID, id)
	switch {
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such work.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not read found images.")
	default:
		listed := FoundImageList{Pictures: make([]FoundImage, 0, len(pictures))}
		for _, picture := range pictures {
			listed.Pictures = append(listed.Pictures, toAPIFoundImage(picture))
		}
		c.JSON(http.StatusOK, listed)
	}
}

func (h *Handlers) PlaceFoundImage(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	pictureID, ok := api.PathID(c, "pictureId")
	if !ok {
		return
	}
	version, ok := api.DraftedChangesVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "placing a picture")
	if !ok {
		return
	}
	var request PlaceFoundImageRequest
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || (len(body) > 0 && api.DecodeOneJSON(bytes.NewReader(body), &request) != nil) {
		api.Refuse(c, http.StatusBadRequest, "Send the id of the uploaded picture, or nothing.")
		return
	}
	var mediaID *uuid.UUID
	if request.MediaId != nil {
		media := uuid.UUID(*request.MediaId)
		mediaID = &media
	}
	candidate := &work.Candidate{Version: version}
	saved, err := h.uploads.PlaceFoundImage(
		c.Request.Context(), owner.ID, id, pictureID, mediaID, candidate)
	if page.CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, work.ErrNotFound), errors.Is(err, ErrFoundImageNotFound):
		api.Refuse(c, http.StatusNotFound, "No such picture is waiting in found images.")
	case errors.Is(err, ErrFoundImageNeedsCopy), errors.Is(err, work.ErrInvalidBlock), errors.Is(err, work.ErrMediaNotFound):
		api.Refuse(c, http.StatusBadRequest, err.Error())
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not place the picture.")
	default:
		blocks, conversionErr := block.ToBlocks(saved.Type, saved.Blocks)
		if conversionErr != nil {
			api.Refuse(c, http.StatusInternalServerError, "Could not read the page after placing the picture.")
			return
		}
		c.JSON(http.StatusOK, blocks)
	}
}

func (h *Handlers) DiscardFoundImage(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	pictureID, ok := api.PathID(c, "pictureId")
	if !ok {
		return
	}
	version, ok := api.DraftedChangesVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "discarding a picture")
	if !ok {
		return
	}
	candidate := &work.Candidate{Version: version}
	err := h.uploads.DiscardFoundImage(c.Request.Context(), owner.ID, id, pictureID, candidate)
	if page.CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, work.ErrNotFound), errors.Is(err, ErrFoundImageNotFound):
		api.Refuse(c, http.StatusNotFound, "No such picture is waiting in found images.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not discard the picture.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func toAPIFoundImage(picture WaitingPicture) FoundImage {
	listed := FoundImage{
		Id: picture.ID, Address: picture.Address, Name: picture.Name, Section: picture.Section,
	}
	if picture.BlockID != nil {
		blockID := *picture.BlockID
		listed.BlockId = &blockID
	}
	if picture.MediaID != nil {
		listed.Media = &struct {
			Height   int       `json:"height"`
			Id       uuid.UUID `json:"id"`
			ThumbUrl string    `json:"thumbUrl"`
			Width    int       `json:"width"`
		}{Height: picture.Height, Id: *picture.MediaID, ThumbUrl: picture.ThumbURL, Width: picture.Width}
	}
	return listed
}
