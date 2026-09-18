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

func (h *Handlers) GetMediaVariant(c *gin.Context) {
	mediaID, ok := api.PathID(c, "media_id")
	if !ok {
		return
	}
	variant := c.Param("variant")
	derivativeVersion, ok := api.PathNumber(c, "derivative_version")
	if !ok {
		return
	}
	q := api.ReadQuery(c)
	params := GetMediaVariantParams{
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
	if derivativeVersion < 1 || uint64(derivativeVersion) > math.MaxUint32 {
		api.Refuse(c, http.StatusNotFound, "no such media variant")
		return
	}
	download, err := h.assets.MediaVariant(c.Request.Context(), work.MediaRequest{
		MediaID:   mediaID,
		Variant:   variant,
		Version:   uint32(derivativeVersion),
		ViewerID:  viewerID,
		Expires:   valueOrEmpty(params.Expires),
		Signature: valueOrEmpty(params.Signature),
	})
	if errors.Is(err, work.ErrMediaNotFound) {
		h.sharedImageVariant(c, mediaID, variant, uint32(derivativeVersion), params)
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

func (h *Handlers) sharedImageVariant(
	c *gin.Context,
	mediaID uuid.UUID,
	variant string,
	version uint32,
	params GetMediaVariantParams,
) {
	ctx := c.Request.Context()
	owners := []func() (string, string, bool, error){
		func() (string, string, bool, error) {
			redirect, mediaType, err := h.accounts.AvatarVariant(ctx, mediaID, variant, version)
			return redirect, mediaType, false, err
		},
		func() (string, string, bool, error) {
			return h.publications.PostMediaVariant(ctx, mediaID, variant, version,
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
	api.Refuse(c, http.StatusNotFound, "no such media variant")
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func toAPIMedia(found work.Media) Media {
	return Media{
		Id:                found.ID,
		AssetId:           found.AssetID,
		Role:              MediaRole(found.Role),
		Width:             found.Width,
		Height:            found.Height,
		DerivativeVersion: int(found.DerivativeVersion),
	}
}
