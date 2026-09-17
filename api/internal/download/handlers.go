package download

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	downloads *Service
	accounts  *account.Service
}

func NewHandlers(downloads *Service, accounts *account.Service) *Handlers {
	return &Handlers{downloads: downloads, accounts: accounts}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/assets/:id/updates/:number/downloads", d.JSON, h.GetRecordedVersionDownloads)
	routes.Handle(http.MethodGet, "/download/:id", d.Download, h.DownloadSource)
	routes.Handle(http.MethodGet, "/download/:id/:target", d.Download, h.DownloadExport)
}

func (h *Handlers) DownloadSource(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	viewerID, ok := api.ViewerID(c)
	if !ok {
		return
	}
	download, err := h.downloads.Source(c.Request.Context(), id, viewerID)
	if err != nil {
		Refuse(c, err)
		return
	}
	h.HandOffSource(c, download)
}

func (h *Handlers) DownloadExport(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	target := c.Param("target")
	q := api.ReadQuery(c)
	params := DownloadExportParams{
		Images:  api.QueryText[string](q, "images"),
		Version: api.QueryNumber(q, "version"),
	}
	if q.Refused(c) {
		return
	}
	viewerID, ok := api.ViewerID(c)
	if !ok {
		return
	}
	gallery, ok := chosenGallery(params.Images)
	if !ok {
		api.Refuse(c, http.StatusNotFound, "no such download")
		return
	}
	var err error
	var download Export
	if params.Version == nil {
		download, err = h.downloads.OpenExport(c.Request.Context(), id, viewerID, target, gallery)
	} else {
		download, err = h.downloads.OpenRecordedExport(
			c.Request.Context(), id, viewerID, *params.Version, target, gallery)
	}
	if err != nil {
		Refuse(c, err)
		return
	}
	h.HandOffExport(c, download)
}

func chosenGallery(images *string) (*GallerySelection, bool) {
	if images == nil {
		return nil, true
	}
	chosen := GallerySelection{Images: []uuid.UUID{}}
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

// Refuse answers a download that could not be made
func Refuse(c *gin.Context, err error) {
	switch {
	case errors.Is(err, asset.ErrNotFound), errors.Is(err, ErrTargetNotOffered),
		errors.Is(err, ErrLinkedInstallOnly):
		api.Refuse(c, http.StatusNotFound, "no such download")
	case errors.Is(err, ErrExportTooLarge):
		api.Refuse(c, http.StatusRequestEntityTooLarge, oversizedDownload)
	case errors.Is(err, ErrExportImageUnreadable):
		api.Refuse(c, http.StatusServiceUnavailable, unreadableDownloadImage)
	default:
		api.Refuse(c, http.StatusInternalServerError, "could not read the file")
	}
}

const (
	oversizedDownload = "Those images make a file larger than Illarin will produce. " +
		"Leave some of them out and try again."
	unreadableDownloadImage = "One of this asset's images could not be read, " +
		"so Illarin made no file rather than one missing a picture. Try again in a moment."
)

func (h *Handlers) HandOffExport(c *gin.Context, download Export) {
	if download.Event != nil {
		if err := h.downloads.Record(c.Request.Context(), *download.Event); err != nil {
			Refuse(c, err)
			return
		}
	}
	c.Header("Content-Disposition", `attachment; filename="`+download.Filename+`"`)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Illarin-Export-Target", download.Target)
	c.Data(http.StatusOK, download.MediaType, download.Body)
}

func (h *Handlers) HandOffSource(c *gin.Context, download Source) {
	if err := h.downloads.Record(c.Request.Context(), download.Event); err != nil {
		Refuse(c, err)
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

func (h *Handlers) GetRecordedVersionDownloads(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	number, ok := api.PathNumber(c, "number")
	if !ok {
		return
	}
	q := api.ReadQuery(c)
	params := GetRecordedVersionDownloadsParams{
		Nsfw: api.QueryText[GetRecordedVersionDownloadsParamsNsfw](q, "nsfw"),
	}
	if q.Refused(c) {
		return
	}
	viewerID, ok := api.ViewerID(c)
	if !ok {
		return
	}
	var requested *string
	if params.Nsfw != nil {
		value := string(*params.Nsfw)
		requested = &value
	}
	visibility, ok := work.ReaderVisibility(c, h.accounts, requested)
	if !ok {
		return
	}
	offered, err := h.downloads.RecordedDownloads(c.Request.Context(), id, viewerID, number, visibility)
	if errors.Is(err, asset.ErrNotFound) {
		api.Refuse(c, http.StatusNotFound, "No such version.")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the version's downloads.")
		return
	}
	blocks, err := block.ToBlocks(offered.Kind, offered.Blocks)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the version's downloads.")
		return
	}
	c.JSON(http.StatusOK, RecordedVersionDownloads{
		Version:           work.ToRecordedVersion(offered.Version),
		Kind:              RecordedVersionDownloadsKind(offered.Kind),
		LinkedInstallOnly: offered.LinkedInstallOnly,
		Downloads:         work.ToDownloads(offered.Downloads),
		AppTargets:        work.ToAppTargets(offered.AppTargets),
		Blocks:            blocks,
		Media:             work.ToImages(offered.Media),
	})
}

// LinkedInstanceFile hands a connected app the file it was sent
func (h *Handlers) LinkedInstanceFile(c *gin.Context, assetID uuid.UUID, target string) {
	if target == asset.RawDownloadTarget {
		download, err := h.downloads.SourceForLinkedInstance(c.Request.Context(), assetID)
		if err != nil {
			Refuse(c, err)
			return
		}
		h.HandOffSource(c, download)
		return
	}
	download, err := h.downloads.OpenExportForLinkedInstance(c.Request.Context(), assetID, target)
	if err != nil {
		Refuse(c, err)
		return
	}
	h.HandOffExport(c, download)
}
