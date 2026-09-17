package upload

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handlers) ListVaultPictures(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.SignedIn(c, "reading the vault")
	if !ok {
		return
	}
	pictures, err := h.uploads.ListVault(c.Request.Context(), owner.ID, id)
	switch {
	case errors.Is(err, asset.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such asset.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not read the vault.")
	default:
		listed := VaultPictureList{Pictures: make([]VaultPicture, 0, len(pictures))}
		for _, picture := range pictures {
			listed.Pictures = append(listed.Pictures, toAPIVaultPicture(picture))
		}
		c.JSON(http.StatusOK, listed)
	}
}

func (h *Handlers) PlaceVaultPicture(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	pictureID, ok := api.PathID(c, "pictureId")
	if !ok {
		return
	}
	version, ok := api.WorkingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "placing a picture")
	if !ok {
		return
	}
	var request PlaceVaultPictureRequest
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
	candidate := &asset.Candidate{Version: version}
	saved, err := h.uploads.PlaceVaultPicture(
		c.Request.Context(), owner.ID, id, pictureID, mediaID, candidate)
	if work.CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrNotFound), errors.Is(err, ErrVaultPictureNotFound):
		api.Refuse(c, http.StatusNotFound, "No such picture is waiting in the vault.")
	case errors.Is(err, ErrVaultPictureNeedsMedia), errors.Is(err, asset.ErrInvalidBlock), errors.Is(err, asset.ErrMediaNotFound):
		api.Refuse(c, http.StatusBadRequest, err.Error())
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not place the picture.")
	default:
		blocks, conversionErr := block.ToBlocks(saved.Kind, saved.Blocks)
		if conversionErr != nil {
			api.Refuse(c, http.StatusInternalServerError, "Could not read the page after placing the picture.")
			return
		}
		c.JSON(http.StatusOK, blocks)
	}
}

func (h *Handlers) DiscardVaultPicture(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	pictureID, ok := api.PathID(c, "pictureId")
	if !ok {
		return
	}
	version, ok := api.WorkingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "discarding a picture")
	if !ok {
		return
	}
	candidate := &asset.Candidate{Version: version}
	err := h.uploads.DiscardVaultPicture(c.Request.Context(), owner.ID, id, pictureID, candidate)
	if work.CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrNotFound), errors.Is(err, ErrVaultPictureNotFound):
		api.Refuse(c, http.StatusNotFound, "No such picture is waiting in the vault.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not discard the picture.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func toAPIVaultPicture(picture WaitingPicture) VaultPicture {
	listed := VaultPicture{
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
