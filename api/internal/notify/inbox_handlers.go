package notify

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handlers) ListNotifications(c *gin.Context) {
	q := api.ReadQuery(c)
	params := ListNotificationsParams{
		Limit:    api.QueryNumber(q, "limit"),
		Before:   api.QueryTime(q, "before"),
		BeforeId: api.QueryID(q, "beforeId"),
	}
	if q.Refused(c) {
		return
	}
	current, ok := api.SignedIn(c, "reading your notifications")
	if !ok {
		return
	}
	if (params.Before == nil) != (params.BeforeId == nil) {
		api.Refuse(c, http.StatusBadRequest, "Send before and beforeId together, or neither.")
		return
	}
	limit := DefaultPageSize
	if params.Limit != nil {
		limit = *params.Limit
	}
	var after *Cursor
	if params.Before != nil {
		after = &Cursor{Before: *params.Before, BeforeID: uuid.UUID(*params.BeforeId)}
	}
	page, err := h.notifications.Inbox(c.Request.Context(), current.ID, after, limit)
	if errors.Is(err, ErrPageSize) {
		api.Refuse(c, http.StatusBadRequest, "Ask for between 1 and 50 notifications.")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read your notifications.")
		return
	}
	sends, err := h.sendAppsFor(c, current.ID, page.Entries)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read your notifications.")
		return
	}
	listed := NotificationList{Items: make([]Notification, 0, len(page.Entries))}
	for _, entry := range page.Entries {
		listed.Items = append(listed.Items, toAPINotification(entry, sends))
	}
	if page.Next != nil {
		listed.NextCursor = &NotificationCursor{Before: page.Next.Before, BeforeId: page.Next.BeforeID}
	}
	c.JSON(http.StatusOK, listed)
}

func (h *Handlers) CountUnreadNotifications(c *gin.Context) {
	current, ok := api.SignedIn(c, "reading your notifications")
	if !ok {
		return
	}
	count, err := h.notifications.Unread(c.Request.Context(), current.ID)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not count your notifications.")
		return
	}
	c.JSON(http.StatusOK, UnreadNotifications{Count: count})
}

func (h *Handlers) MarkNotificationRead(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	current, ok := api.SignedIn(c, "marking a notification read")
	if !ok {
		return
	}
	found, err := h.notifications.MarkRead(c.Request.Context(), current.ID, id)
	switch {
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not mark the notification read.")
	case !found:
		api.Refuse(c, http.StatusNotFound, "no such notification")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) MarkAllNotificationsRead(c *gin.Context) {
	current, ok := api.SignedIn(c, "marking your notifications read")
	if !ok {
		return
	}
	if err := h.notifications.MarkAllRead(c.Request.Context(), current.ID); err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not mark your notifications read.")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) RemoveNotification(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	current, ok := api.SignedIn(c, "removing a notification")
	if !ok {
		return
	}
	found, err := h.notifications.Remove(c.Request.Context(), current.ID, id)
	switch {
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not remove the notification.")
	case !found:
		api.Refuse(c, http.StatusNotFound, "no such notification")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) ClearNotifications(c *gin.Context) {
	current, ok := api.SignedIn(c, "clearing your notifications")
	if !ok {
		return
	}
	if err := h.notifications.Clear(c.Request.Context(), current.ID); err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not clear your notifications.")
		return
	}
	c.Status(http.StatusNoContent)
}

// sendAppsFor gathers the connected apps that can take each update entry on one page.
func (h *Handlers) sendAppsFor(
	c *gin.Context,
	account uuid.UUID,
	entries []Entry,
) (map[uuid.UUID][]NotificationSendApp, error) {
	updated := make([]uuid.UUID, 0, len(entries))
	seen := make(map[uuid.UUID]bool, len(entries))
	for _, entry := range entries {
		if entry.Type != WorkUpdated || entry.Work == nil || seen[*entry.Work] {
			continue
		}
		seen[*entry.Work] = true
		updated = append(updated, *entry.Work)
	}
	holders, err := h.sends.UpdatableApps(c.Request.Context(), account, updated)
	if err != nil {
		return nil, err
	}
	offered := make(map[uuid.UUID][]NotificationSendApp, len(holders))
	for workID, apps := range holders {
		for _, state := range apps {
			offered[workID] = append(offered[workID], NotificationSendApp{
				ConnectedAppId: state.ConnectedAppID,
				Name:           state.Name,
				AppName:        state.AppName,
				Waiting:        waitingToCollect(state),
			})
		}
		slices.SortFunc(offered[workID], byName)
	}
	return offered, nil
}

// waitingToCollect says whether a send of the work is already waiting for the connected app.
func waitingToCollect(state connect.AppState) bool {
	if state.Send == nil {
		return false
	}
	return state.Send.State == connect.StateQueued || state.Send.State == connect.StateReleased
}

// byName keeps the sends an entry offers in the order a reader would read them.
func byName(first, second NotificationSendApp) int {
	if named := strings.Compare(first.AppName, second.AppName); named != 0 {
		return named
	}
	return strings.Compare(first.Name, second.Name)
}

func toAPINotification(
	entry Entry,
	sends map[uuid.UUID][]NotificationSendApp,
) Notification {
	shown := Notification{
		Id: entry.ID, Type: NotificationType(entry.Type), CreatedAt: entry.CreatedAt, ReadAt: entry.ReadAt,
	}
	if entry.Work != nil {
		shown.Work = &NotificationWork{Id: *entry.Work, Name: entry.Words.WorkName}
	}
	if entry.Words.Reason != "" {
		reason := entry.Words.Reason
		shown.Reason = &reason
	}
	if entry.Words.Creator != "" {
		shown.Creator = &NotificationCreator{Handle: entry.Words.Creator, Name: entry.Words.CreatorName}
	}
	if entry.Type == WorkUpdated {
		shown.Update = &NotificationUpdate{
			Number: entry.Words.UpdateNumber, Summary: entry.Words.Summary, Count: entry.Count,
		}
		if entry.Words.VersionLabel != "" {
			label := entry.Words.VersionLabel
			shown.Update.VersionLabel = &label
		}
		if entry.Work == nil {
			return shown
		}
		if offered := sends[*entry.Work]; len(offered) > 0 {
			shown.SendTo = &offered
		}
	}
	return shown
}
