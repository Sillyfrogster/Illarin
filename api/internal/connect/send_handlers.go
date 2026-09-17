package connect

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxLibraryBodyBytes = 256 << 10

func (h *Handlers) CollectDeliveries(c *gin.Context) {
	noStoreLink(c)
	instance, ok := h.instance(c, ScopeAssetReceive)
	if !ok {
		return
	}
	var request CollectDeliveries
	if !readLinkJSON(c, &request) {
		return
	}
	collected, err := h.sends.Collect(
		c.Request.Context(), instance, uuidsFrom(request.Acknowledge),
	)
	if err != nil {
		h.deliveryError(c, err)
		return
	}
	if len(collected.Work) == 0 && len(collected.Withheld) == 0 {
		c.Status(http.StatusNoContent)
		return
	}
	items := make([]DeliveryWork, 0, len(collected.Work))
	for _, released := range collected.Work {
		items = append(items, toAPIDeliveryWork(released))
	}
	c.JSON(http.StatusOK, DeliveryWorkList{
		Deliveries: items, Withheld: toAPIWithheldNotices(collected.Withheld),
	})
}

func (h *Handlers) SyncLibrary(c *gin.Context) {
	noStoreLink(c)
	instance, ok := h.instance(c, ScopeLibrarySync)
	if !ok {
		return
	}
	var request LibraryReport
	if !api.ReadBoundedJSON(c, &request, maxLibraryBodyBytes, "The library report is too large.") {
		return
	}
	result, err := h.sends.Sync(c.Request.Context(), instance, toLibraryReport(request))
	if err != nil {
		h.deliveryError(c, err)
		return
	}
	c.JSON(http.StatusOK, LibraryReportResult{
		Accepted: result.Accepted, Removed: result.Removed, Ignored: result.Ignored,
		Withheld: toAPIWithheldNotices(result.Withheld),
	})
}

func (h *Handlers) GetAssetInstances(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	noStoreLink(c)
	creator, ok := api.SignedIn(c, "sending an asset to an application")
	if !ok {
		return
	}
	found, err := h.sends.AssetInstances(c.Request.Context(), creator.ID, id)
	if err != nil {
		h.deliveryError(c, err)
		return
	}
	items := make([]AssetInstance, 0, len(found.Items))
	for _, state := range found.Items {
		items = append(items, toAPIAssetInstance(state))
	}
	c.JSON(http.StatusOK, AssetInstanceList{
		ContentGeneration: found.ContentGeneration, Items: items,
	})
}

func (h *Handlers) SendAssetToInstance(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	if !api.FromIllarin(c) {
		return
	}
	noStoreLink(c)
	creator, ok := api.SignedIn(c, "sending an asset to an application")
	if !ok || !api.RequireBrowser(c, h.apps.BrowserOrigin()) {
		return
	}
	var request SendAssetRequest
	if !readLinkJSON(c, &request) {
		return
	}
	queued, err := h.sends.Queue(
		c.Request.Context(), creator.ID, request.InstanceId, id,
	)
	if err != nil {
		h.deliveryError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, toAPIQueuedDelivery(queued))
}

func (h *Handlers) DiscardDelivery(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	if !api.FromIllarin(c) {
		return
	}
	noStoreLink(c)
	creator, ok := api.SignedIn(c, "managing deliveries")
	if !ok || !api.RequireBrowser(c, h.apps.BrowserOrigin()) {
		return
	}
	if err := h.sends.Discard(c.Request.Context(), creator.ID, id); err != nil {
		h.deliveryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) DownloadDeliveryExport(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	q := api.ReadQuery(c)
	params := DownloadDeliveryExportParams{
		Expires:   api.QueryRequired(q, "expires"),
		Signature: api.QueryRequired(q, "signature"),
	}
	if q.Refused(c) {
		return
	}
	c.Header("Cache-Control", "private, no-store")
	assetID, target, err := h.sends.Artifact(
		c.Request.Context(), id, params.Expires, params.Signature,
	)
	if err != nil {
		h.deliveryArtifactError(c, err)
		return
	}
	h.files.LinkedInstanceFile(c, assetID, target)
}

func (h *Handlers) deliveryArtifactError(c *gin.Context, err error) {
	if errors.Is(err, ErrArtifactNotFound) {
		api.Refuse(c, http.StatusNotFound, "no such download")
		return
	}
	api.Refuse(c, http.StatusInternalServerError, "could not read the file")
}

func (h *Handlers) deliveryError(c *gin.Context, err error) {
	var limited *RateLimitError
	switch {
	case errors.As(err, &limited):
		seconds := int((limited.After + time.Second - 1) / time.Second)
		c.Header("Retry-After", strconv.Itoa(seconds))
		api.Refuse(c, http.StatusTooManyRequests, "Too many requests. Try again later.")
	case errors.Is(err, ErrTooManyCollectors):
		c.Header("Retry-After", strconv.Itoa(collectorsBusySeconds))
		api.Refuse(c, http.StatusServiceUnavailable, "Too many applications are waiting for work. Try again shortly.")
	case errors.Is(err, ErrNoInstanceOfYours):
		api.Refuse(c, http.StatusNotFound, "No live application of yours has that id.")
	case errors.Is(err, ErrMissingScope):
		api.Refuse(c, http.StatusForbidden, "That application cannot receive assets.")
	case errors.Is(err, ErrAssetNotFound), errors.Is(err, ErrAssetNotSendable):
		api.Refuse(c, http.StatusNotFound, "No asset that can be sent has that id.")
	case errors.Is(err, ErrNoTarget):
		api.Refuse(c, http.StatusConflict, "That application accepts no format this asset can be written in.")
	case errors.Is(err, ErrCannotInstall):
		api.Refuse(c, http.StatusConflict, "That application does not install extensions from Illarin.")
	case errors.Is(err, ErrQueueFull):
		api.Refuse(c, http.StatusConflict, "That application already has as many deliveries waiting as it may hold.")
	case errors.Is(err, ErrDeliveryNotFound):
		api.Refuse(c, http.StatusNotFound, "No delivery of yours has that id.")
	case errors.Is(err, ErrLibraryTooLarge):
		api.Refuse(c, http.StatusRequestEntityTooLarge, "Report fewer installed assets in one request.")
	case errors.Is(err, ErrLibraryReport):
		api.Refuse(c, http.StatusBadRequest, "That report is not valid.")
	case errors.Is(err, ErrLibraryVersion):
		api.Refuse(c, http.StatusBadRequest, "The application version must be printable text of at most 64 characters.")
	case errors.Is(err, ErrAcknowledgement):
		api.Refuse(c, http.StatusBadRequest, "Acknowledge at most 32 deliveries in one request.")
	default:
		api.Refuse(c, http.StatusInternalServerError, "Could not complete the request.")
	}
}

const collectorsBusySeconds = 30

func toLibraryReport(request LibraryReport) ReportedLibrary {
	entries := make([]ReportedEntry, 0, len(request.Entries))
	for _, entry := range request.Entries {
		entries = append(entries, ReportedEntry{
			AssetID: entry.AssetId, ContentGeneration: entry.ContentGeneration,
		})
	}
	var removed []uuid.UUID
	if request.Removed != nil {
		removed = uuidsFrom(*request.Removed)
	}
	report := ReportedLibrary{
		Snapshot: request.Snapshot, Entries: entries, Removed: removed,
	}
	if request.ApplicationVersion != nil {
		report.ApplicationVersion = *request.ApplicationVersion
	}
	return report
}

func toAPIDeliveryWork(released Work) DeliveryWork {
	artifacts := make([]DeliveryArtifact, 0, len(released.Artifacts))
	for _, artifact := range released.Artifacts {
		item := DeliveryArtifact{
			Kind: DeliveryArtifactKind(artifact.Kind), Url: artifact.URL,
		}
		if artifact.MediaID != nil {
			mediaID := *artifact.MediaID
			role, isCover := artifact.Role, artifact.IsCover
			item.MediaId, item.Role, item.IsCover = &mediaID, &role, &isCover
		}
		artifacts = append(artifacts, item)
	}
	return DeliveryWork{
		Id: released.ID, AssetId: released.AssetID,
		ContentGeneration: released.ContentGeneration, Kind: released.Kind,
		Name: released.Name, Format: released.Format, Label: released.Label,
		QueuedAt: released.QueuedAt, LeaseExpiresAt: released.LeaseExpiresAt,
		Artifacts: artifacts,
	}
}

func toAPIQueuedDelivery(queued Delivery) QueuedDelivery {
	item := QueuedDelivery{
		Id: queued.ID, InstanceId: queued.InstanceID,
		AssetId: queued.AssetID, State: QueuedDeliveryState(queued.State),
		QueuedAt: queued.QueuedAt, SettledAt: queued.SettledAt, ExpiresAt: queued.ExpiresAt,
		UpdatesInstall: queued.UpdatesInstall,
	}
	if queued.Reason != "" {
		reason := QueuedDeliveryReason(queued.Reason)
		item.Reason = &reason
	}
	return item
}

func toAPIAssetInstance(state InstanceState) AssetInstance {
	item := AssetInstance{
		InstanceId: state.InstanceID, ApplicationName: state.ApplicationName,
		InstanceName: state.InstanceName, LastSeenAt: state.LastSeenAt,
		CanReceive: state.CanReceive, ReportsLibrary: state.ReportsLibrary,
		InstalledGeneration: state.InstalledGeneration,
		UpdateAvailable:     state.UpdateAvailable,
	}
	if state.Delivery != nil {
		queued := toAPIQueuedDelivery(*state.Delivery)
		item.Delivery = &queued
	}
	return item
}

func toAPIWithheldNotices(notices []WithheldWork) []WithheldNotice {
	items := make([]WithheldNotice, 0, len(notices))
	for _, notice := range notices {
		items = append(items, WithheldNotice{
			AssetId: notice.AssetID, Name: notice.Name, WithheldAt: notice.WithheldAt,
		})
	}
	return items
}

func uuidsFrom(values []uuid.UUID) []uuid.UUID {
	converted := make([]uuid.UUID, len(values))
	for index, value := range values {
		converted[index] = value
	}
	return converted
}
