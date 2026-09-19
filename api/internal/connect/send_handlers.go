package connect

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxLibraryBodyBytes = 256 << 10

func (h *Handlers) CollectSends(c *gin.Context) {
	noStore(c)
	app, ok := h.connectedApp(c, PermissionReceiveWorks)
	if !ok {
		return
	}
	var request CollectSends
	if !readConnectJSON(c, &request) {
		return
	}
	collected, err := h.sends.Collect(c.Request.Context(), app, request.Acknowledge)
	if err != nil {
		h.sendError(c, err)
		return
	}
	if len(collected.Work) == 0 && len(collected.Withheld) == 0 {
		c.Status(http.StatusNoContent)
		return
	}
	items := make([]CollectedSend, 0, len(collected.Work))
	for _, released := range collected.Work {
		items = append(items, toAPICollectedSend(released))
	}
	c.JSON(http.StatusOK, CollectedSends{
		Sends: items, Withheld: toAPIWithheldNotices(collected.Withheld),
	})
}

func (h *Handlers) SyncLibrary(c *gin.Context) {
	noStore(c)
	app, ok := h.connectedApp(c, PermissionSyncLibrary)
	if !ok {
		return
	}
	var request LibraryReport
	if !api.ReadBoundedJSON(c, &request, maxLibraryBodyBytes, "The library report is too large.") {
		return
	}
	result, err := h.sends.Sync(c.Request.Context(), app, toLibraryReport(request))
	if err != nil {
		h.sendError(c, err)
		return
	}
	c.JSON(http.StatusOK, LibraryReportResult{
		Accepted: result.Accepted, Removed: result.Removed, Ignored: result.Ignored,
		Withheld: toAPIWithheldNotices(result.Withheld),
	})
}

func (h *Handlers) GetWorkConnectedApps(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	noStore(c)
	creator, ok := api.SignedIn(c, "sending a work to an app")
	if !ok {
		return
	}
	found, err := h.sends.WorkApps(c.Request.Context(), creator.ID, id)
	if err != nil {
		h.sendError(c, err)
		return
	}
	items := make([]WorkConnectedApp, 0, len(found.Items))
	for _, state := range found.Items {
		items = append(items, toAPIWorkConnectedApp(state))
	}
	c.JSON(http.StatusOK, WorkConnectedAppList{
		VersionNumber: found.VersionNumber, Items: items,
	})
}

func (h *Handlers) SendWork(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	if !api.FromIllarin(c) {
		return
	}
	noStore(c)
	creator, ok := api.SignedIn(c, "sending a work to an app")
	if !ok || !api.RequireBrowser(c, h.apps.BrowserOrigin()) {
		return
	}
	var request SendWorkRequest
	if !readConnectJSON(c, &request) {
		return
	}
	queued, err := h.sends.Queue(c.Request.Context(), creator.ID, request.ConnectedAppId, id)
	if err != nil {
		h.sendError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, toAPIQueuedSend(queued))
}

func (h *Handlers) DiscardSend(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	if !api.FromIllarin(c) {
		return
	}
	noStore(c)
	creator, ok := api.SignedIn(c, "managing sends")
	if !ok || !api.RequireBrowser(c, h.apps.BrowserOrigin()) {
		return
	}
	if err := h.sends.Discard(c.Request.Context(), creator.ID, id); err != nil {
		h.sendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// DownloadSendFile checks the signature against the address the request came in on
func (h *Handlers) DownloadSendFile(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	q := api.ReadQuery(c)
	params := SendFileParams{
		Expires:   api.QueryRequired(q, "expires"),
		Signature: api.QueryRequired(q, "signature"),
	}
	if q.Refused(c) {
		return
	}
	c.Header("Cache-Control", "private, no-store")
	pathStart := strings.TrimSuffix(c.FullPath(), ":id/export")
	workID, chosenFormat, err := h.sends.MainFile(
		c.Request.Context(), pathStart, id, params.Expires, params.Signature,
	)
	if errors.Is(err, ErrMainFileNotFound) {
		api.Refuse(c, http.StatusNotFound, "no such download")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "could not read the file")
		return
	}
	h.files.ServeSendFile(c, workID, chosenFormat)
}

func (h *Handlers) sendError(c *gin.Context, err error) {
	var limited *RateLimitError
	switch {
	case errors.As(err, &limited):
		seconds := int((limited.After + time.Second - 1) / time.Second)
		c.Header("Retry-After", strconv.Itoa(seconds))
		api.Refuse(c, http.StatusTooManyRequests, "Too many requests. Try again later.")
	case errors.Is(err, ErrTooManyCollectors):
		c.Header("Retry-After", strconv.Itoa(collectorsBusySeconds))
		api.Refuse(c, http.StatusServiceUnavailable, "Too many apps are waiting for work. Try again shortly.")
	case errors.Is(err, ErrNoAppOfYours):
		api.Refuse(c, http.StatusNotFound, "No live connected app of yours has that id.")
	case errors.Is(err, ErrMissingPermission):
		api.Refuse(c, http.StatusForbidden, "That connected app cannot receive works.")
	case errors.Is(err, ErrWorkNotFound), errors.Is(err, ErrWorkNotSendable):
		api.Refuse(c, http.StatusNotFound, "No work that can be sent has that id.")
	case errors.Is(err, ErrNoFormat):
		api.Refuse(c, http.StatusConflict, "That connected app accepts no format this work can be written in.")
	case errors.Is(err, ErrCannotInstall):
		api.Refuse(c, http.StatusConflict, "That connected app does not install extensions from Illarin.")
	case errors.Is(err, ErrQueueFull):
		api.Refuse(c, http.StatusConflict, "That connected app already has as many sends waiting as it may hold.")
	case errors.Is(err, ErrSendNotFound):
		api.Refuse(c, http.StatusNotFound, "No send of yours has that id.")
	case errors.Is(err, ErrLibraryTooLarge):
		api.Refuse(c, http.StatusRequestEntityTooLarge, "Report fewer installed works in one request.")
	case errors.Is(err, ErrLibraryReport):
		api.Refuse(c, http.StatusBadRequest, "That report is not valid.")
	case errors.Is(err, ErrLibraryVersion):
		api.Refuse(c, http.StatusBadRequest, "The app version must be printable text of at most 64 characters.")
	case errors.Is(err, ErrAcknowledgement):
		api.Refuse(c, http.StatusBadRequest, "Acknowledge at most 32 sends in one request.")
	default:
		api.Refuse(c, http.StatusInternalServerError, "Could not complete the request.")
	}
}

const collectorsBusySeconds = 30

func toLibraryReport(request LibraryReport) ReportedLibrary {
	entries := make([]ReportedEntry, 0, len(request.Entries))
	for _, entry := range request.Entries {
		entries = append(entries, ReportedEntry{
			WorkID: entry.WorkId, VersionNumber: entry.VersionNumber,
		})
	}
	var removed []uuid.UUID
	if request.Removed != nil {
		removed = *request.Removed
	}
	report := ReportedLibrary{
		Snapshot: request.Snapshot, Entries: entries, Removed: removed,
	}
	if request.AppVersion != nil {
		report.AppVersion = *request.AppVersion
	}
	return report
}

func toAPICollectedSend(released Work) CollectedSend {
	files := make([]SendFile, 0, len(released.Files))
	for _, file := range released.Files {
		item := SendFile{Type: SendFileType(file.Type), Url: file.URL}
		if file.MediaID != nil {
			mediaID := *file.MediaID
			role, isCover := file.Role, file.IsCover
			item.MediaId, item.Role, item.IsCover = &mediaID, &role, &isCover
		}
		files = append(files, item)
	}
	return CollectedSend{
		Id: released.ID, WorkId: released.WorkID,
		VersionNumber: released.VersionNumber, Type: released.Type,
		Name: released.Name, Format: released.Format, Label: released.Label,
		QueuedAt: released.QueuedAt, LeaseExpiresAt: released.LeaseExpiresAt,
		Files: files,
	}
}

func toAPIQueuedSend(queued Send) QueuedSend {
	item := QueuedSend{
		Id: queued.ID, ConnectedAppId: queued.ConnectedAppID,
		WorkId: queued.WorkID, State: QueuedSendState(queued.State),
		QueuedAt: queued.QueuedAt, SettledAt: queued.SettledAt, ExpiresAt: queued.ExpiresAt,
		UpdatesInstall: queued.UpdatesInstall,
	}
	if queued.Reason != "" {
		reason := QueuedSendReason(queued.Reason)
		item.Reason = &reason
	}
	return item
}

func toAPIWorkConnectedApp(state AppState) WorkConnectedApp {
	item := WorkConnectedApp{
		ConnectedAppId: state.ConnectedAppID, AppName: state.AppName,
		Name: state.Name, LastSeenAt: state.LastSeenAt,
		CanReceive: state.CanReceive, ReportsLibrary: state.ReportsLibrary,
		InstalledVersion: state.InstalledVersion,
		UpdateAvailable:  state.UpdateAvailable,
	}
	if state.Send != nil {
		queued := toAPIQueuedSend(*state.Send)
		item.Send = &queued
	}
	return item
}

func toAPIWithheldNotices(notices []WithheldWork) []WithheldNotice {
	items := make([]WithheldNotice, 0, len(notices))
	for _, notice := range notices {
		items = append(items, WithheldNotice{
			WorkId: notice.WorkID, Name: notice.Name, WithheldAt: notice.WithheldAt,
		})
	}
	return items
}
