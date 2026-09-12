package http

import (
	"errors"
	"math"
	"net/http"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

func (h *Handlers) DownloadSource(c *gin.Context, id types.UUID) {
	viewerID, ok := h.viewerID(c)
	if !ok {
		return
	}
	download, err := h.assets.DownloadSource(c.Request.Context(), id, viewerID)
	if err != nil {
		h.downloadError(c, err)
		return
	}
	h.handOffDownload(c, download)
}

func (h *Handlers) DownloadExport(
	c *gin.Context,
	id types.UUID,
	target string,
	params DownloadExportParams,
) {
	viewerID, ok := h.viewerID(c)
	if !ok {
		return
	}
	gallery, ok := chosenGallery(params.Images)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "no such download"})
		return
	}
	var err error
	var download asset.Export
	if params.Version == nil {
		download, err = h.assets.DownloadExport(c.Request.Context(), id, viewerID, target, gallery)
	} else {
		download, err = h.assets.DownloadRecordedExport(
			c.Request.Context(), id, viewerID, *params.Version, target, gallery)
	}
	if err != nil {
		h.downloadError(c, err)
		return
	}
	h.handOffExport(c, download)
}

func chosenGallery(images *string) (*asset.GallerySelection, bool) {
	if images == nil {
		return nil, true
	}
	chosen := asset.GallerySelection{Images: []uuid.UUID{}}
	for _, written := range strings.Split(*images, ",") {
		written = strings.TrimSpace(written)
		if written == "" {
			continue
		}
		mediaID, err := uuid.Parse(written)
		if err != nil {
			return nil, false
		}
		chosen.Images = append(chosen.Images, mediaID)
	}
	return &chosen, true
}

func (h *Handlers) downloadError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, asset.ErrNotFound), errors.Is(err, asset.ErrTargetNotOffered),
		errors.Is(err, asset.ErrLinkedInstallOnly):
		c.JSON(http.StatusNotFound, gin.H{"error": "no such download"})
	case errors.Is(err, asset.ErrExportTooLarge):
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": oversizedDownload})
	case errors.Is(err, asset.ErrExportImageUnreadable):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": unreadableDownloadImage})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read the file"})
	}
}

const (
	oversizedDownload = "Those images make a file larger than Illarin will produce. " +
		"Leave some of them out and try again."
	unreadableDownloadImage = "One of this asset's images could not be read, " +
		"so Illarin made no file rather than one missing a picture. Try again in a moment."
)

func (h *Handlers) handOffExport(c *gin.Context, download asset.Export) {
	if download.Event != nil {
		if err := h.assets.RecordDownload(c.Request.Context(), *download.Event); err != nil {
			h.downloadError(c, err)
			return
		}
	}
	c.Header("Content-Disposition", `attachment; filename="`+download.Filename+`"`)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Illarin-Export-Target", download.Target)
	c.Data(http.StatusOK, download.MediaType, download.Body)
}

func (h *Handlers) handOffDownload(c *gin.Context, download asset.SourceDownload) {
	if err := h.assets.RecordDownload(c.Request.Context(), download.Event); err != nil {
		h.downloadError(c, err)
		return
	}
	disposition := "attachment"
	mediaType := "application/octet-stream"
	if download.Inline {
		disposition = "inline"
		mediaType = download.MediaType
	}
	c.Header("Content-Disposition", disposition)
	c.Header("Content-Type", mediaType)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Accel-Redirect", download.InternalRedirect)
	c.Status(http.StatusOK)
}

func (h *Handlers) GetMediaVariant(
	c *gin.Context,
	mediaID types.UUID,
	variant GetMediaVariantParamsVariant,
	derivativeVersion int,
	params GetMediaVariantParams,
) {
	viewerID, ok := h.viewerID(c)
	if !ok {
		return
	}
	if derivativeVersion < 1 || uint64(derivativeVersion) > math.MaxUint32 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no such media variant"})
		return
	}
	download, err := h.assets.MediaVariant(c.Request.Context(), asset.MediaRequest{
		MediaID:   uuid.UUID(mediaID),
		Variant:   string(variant),
		Version:   uint32(derivativeVersion),
		ViewerID:  viewerID,
		Expires:   valueOrEmpty(params.Expires),
		Signature: valueOrEmpty(params.Signature),
	})
	if errors.Is(err, asset.ErrMediaNotFound) {
		h.sharedImageVariant(c, uuid.UUID(mediaID), string(variant), uint32(derivativeVersion), params)
		return
	}
	if errors.Is(err, storage.ErrInsufficientSpace) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "The image is temporarily unavailable."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read the image"})
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
			redirect, mediaType, err := h.publications.MarkVariant(ctx, mediaID, variant, version)
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
			errors.Is(err, publication.ErrMarkNotFound),
			errors.Is(err, publication.ErrPostMediaNotFound):
			continue
		case errors.Is(err, storage.ErrInsufficientSpace):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "The image is temporarily unavailable."})
			return
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read the image"})
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
	c.JSON(http.StatusNotFound, gin.H{"error": "no such media variant"})
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func toAPIMedia(found asset.Media) Media {
	return Media{
		Id:                types.UUID(found.ID),
		AssetId:           types.UUID(found.AssetID),
		Role:              MediaRole(found.Role),
		Width:             found.Width,
		Height:            found.Height,
		DerivativeVersion: int(found.DerivativeVersion),
	}
}
