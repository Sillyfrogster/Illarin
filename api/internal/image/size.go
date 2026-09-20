package image

import (
	"errors"
	"math"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/blog"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handlers) GetImageSize(c *gin.Context) {
	mediaID, ok := api.PathID(c, "media_id")
	if !ok {
		return
	}
	size := c.Param("size")
	imageSizeVersion, ok := api.PathNumber(c, "image_size_version")
	if !ok {
		return
	}
	q := api.ReadQuery(c)
	params := GetImageSizeParams{
		Expires:   api.QueryText[string](q, "expires"),
		Signature: api.QueryText[string](q, "signature"),
	}
	if q.Refused(c) {
		return
	}
	viewerID, ok := api.ViewerID(c)
	if !ok {
		return
	}
	if imageSizeVersion < 1 || uint64(imageSizeVersion) > math.MaxUint32 {
		api.Refuse(c, http.StatusNotFound, "no such media size")
		return
	}
	download, err := h.works.ImageSize(c.Request.Context(), work.MediaRequest{
		MediaID:   mediaID,
		Size:      size,
		Version:   uint32(imageSizeVersion),
		ViewerID:  viewerID,
		Expires:   valueOrEmpty(params.Expires),
		Signature: valueOrEmpty(params.Signature),
	})
	if errors.Is(err, work.ErrMediaNotFound) {
		h.sharedImageSize(c, mediaID, size, uint32(imageSizeVersion), params)
		return
	}
	if errors.Is(err, storage.ErrInsufficientSpace) {
		api.Refuse(c, http.StatusServiceUnavailable, "The image is temporarily unavailable.")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "could not read the image")
		return
	}
	cache := "public, max-age=31536000, immutable"
	if download.Private {
		cache = "private, no-store"
	}
	c.Header("Cache-Control", cache)
	c.Header("Content-Disposition", "inline")
	c.Header("Content-Type", download.MediaType)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Accel-Redirect", download.InternalRedirect)
	c.Status(http.StatusOK)
}

func (h *Handlers) sharedImageSize(
	c *gin.Context,
	mediaID uuid.UUID,
	size string,
	version uint32,
	params GetImageSizeParams,
) {
	ctx := c.Request.Context()
	owners := []func() (string, string, bool, error){
		func() (string, string, bool, error) {
			redirect, mediaType, err := h.accounts.AvatarImageSize(ctx, mediaID, size, version)
			return redirect, mediaType, false, err
		},
		func() (string, string, bool, error) {
			return h.posts.PostImageSize(ctx, mediaID, size, version,
				valueOrEmpty(params.Expires), valueOrEmpty(params.Signature))
		},
	}
	for _, owner := range owners {
		redirect, mediaType, private, err := owner()
		switch {
		case errors.Is(err, account.ErrProfileMediaNotFound),
			errors.Is(err, blog.ErrPostMediaNotFound):
			continue
		case errors.Is(err, storage.ErrInsufficientSpace):
			api.Refuse(c, http.StatusServiceUnavailable, "The image is temporarily unavailable.")
			return
		case err != nil:
			api.Refuse(c, http.StatusInternalServerError, "could not read the image")
			return
		}
		cache := "public, max-age=31536000, immutable"
		if private {
			cache = "private, no-store"
		}
		c.Header("Cache-Control", cache)
		c.Header("Content-Disposition", "inline")
		c.Header("Content-Type", mediaType)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Accel-Redirect", redirect)
		c.Status(http.StatusOK)
		return
	}
	api.Refuse(c, http.StatusNotFound, "no such media size")
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func toAPIMedia(found work.Media) Media {
	return Media{
		Id:               found.ID,
		WorkId:           found.WorkID,
		Role:             MediaRole(found.Role),
		Width:            found.Width,
		Height:           found.Height,
		ImageSizeVersion: int(found.ImageSizeVersion),
	}
}
