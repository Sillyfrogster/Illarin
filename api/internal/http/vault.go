package http

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

func (h *Handlers) ListVaultPictures(c *gin.Context, id types.UUID) {
	owner, ok := h.signedInAccount(c, "reading the vault")
	if !ok {
		return
	}
	pictures, err := h.assets.ListVault(c.Request.Context(), owner.ID, uuid.UUID(id))
	switch {
	case errors.Is(err, asset.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such asset."})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the vault."})
	default:
		listed := VaultPictureList{Pictures: make([]VaultPicture, 0, len(pictures))}
		for _, picture := range pictures {
			listed.Pictures = append(listed.Pictures, toAPIVaultPicture(picture))
		}
		c.JSON(http.StatusOK, listed)
	}
}

func (h *Handlers) PlaceVaultPicture(c *gin.Context, id, pictureID types.UUID, params PlaceVaultPictureParams) {
	owner, ok := h.verifiedAccount(c, "placing a picture")
	if !ok {
		return
	}
	var request PlaceVaultPictureRequest
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || (len(body) > 0 && decodeOneJSON(bytes.NewReader(body), &request) != nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the id of the uploaded picture, or nothing."})
		return
	}
	var mediaID *uuid.UUID
	if request.MediaId != nil {
		media := uuid.UUID(*request.MediaId)
		mediaID = &media
	}
	candidate := &asset.Candidate{Version: params.XWorkingCopyVersion}
	saved, err := h.assets.PlaceVaultPicture(
		c.Request.Context(), owner.ID, uuid.UUID(id), uuid.UUID(pictureID), mediaID, candidate)
	if candidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrNotFound), errors.Is(err, asset.ErrVaultPictureNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such picture is waiting in the vault."})
	case errors.Is(err, asset.ErrVaultPictureNeedsMedia), errors.Is(err, asset.ErrInvalidBlock), errors.Is(err, asset.ErrMediaNotFound):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not place the picture."})
	default:
		blocks, conversionErr := toAPIBlocks(saved.Kind, saved.Blocks)
		if conversionErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the page after placing the picture."})
			return
		}
		c.JSON(http.StatusOK, blocks)
	}
}

func (h *Handlers) DiscardVaultPicture(c *gin.Context, id, pictureID types.UUID, params DiscardVaultPictureParams) {
	owner, ok := h.verifiedAccount(c, "discarding a picture")
	if !ok {
		return
	}
	candidate := &asset.Candidate{Version: params.XWorkingCopyVersion}
	err := h.assets.DiscardVaultPicture(c.Request.Context(), owner.ID, uuid.UUID(id), uuid.UUID(pictureID), candidate)
	if candidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrNotFound), errors.Is(err, asset.ErrVaultPictureNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such picture is waiting in the vault."})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not discard the picture."})
	default:
		c.Status(http.StatusNoContent)
	}
}

func toAPIVaultPicture(picture asset.VaultPicture) VaultPicture {
	listed := VaultPicture{
		Id: types.UUID(picture.ID), Address: picture.Address, Name: picture.Name, Section: picture.Section,
	}
	if picture.BlockID != nil {
		blockID := types.UUID(*picture.BlockID)
		listed.BlockId = &blockID
	}
	if picture.MediaID != nil {
		listed.Media = &struct {
			Height   int        `json:"height"`
			Id       types.UUID `json:"id"`
			ThumbUrl string     `json:"thumbUrl"`
			Width    int        `json:"width"`
		}{Height: picture.Height, Id: types.UUID(*picture.MediaID), ThumbUrl: picture.ThumbURL, Width: picture.Width}
	}
	return listed
}
