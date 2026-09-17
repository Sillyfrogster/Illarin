package http

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/delivery"
	"github.com/Sillyfrogster/Illarin/api/internal/notification"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handlers) ListNotifications(c *gin.Context) {
	q := readQuery(c)
	params := ListNotificationsParams{
		Limit:    queryNumber(q, "limit"),
		Before:   queryTime(q, "before"),
		BeforeId: queryID(q, "beforeId"),
	}
	if q.refused(c) {
		return
	}
	current, ok := h.signedInAccount(c, "reading your notifications")
	if !ok {
		return
	}
	if (params.Before == nil) != (params.BeforeId == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send before and beforeId together, or neither."})
		return
	}
	limit := notification.DefaultPageSize
	if params.Limit != nil {
		limit = *params.Limit
	}
	var after *notification.Cursor
	if params.Before != nil {
		after = &notification.Cursor{Before: *params.Before, BeforeID: uuid.UUID(*params.BeforeId)}
	}
	page, err := h.notifications.Inbox(c.Request.Context(), current.ID, after, limit)
	if errors.Is(err, notification.ErrPageSize) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ask for between 1 and 50 notifications."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read your notifications."})
		return
	}
	sends, err := h.sendTargetsFor(c, current.ID, page.Entries)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read your notifications."})
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
	current, ok := h.signedInAccount(c, "reading your notifications")
	if !ok {
		return
	}
	count, err := h.notifications.Unread(c.Request.Context(), current.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not count your notifications."})
		return
	}
	c.JSON(http.StatusOK, UnreadNotifications{Count: count})
}

func (h *Handlers) MarkNotificationRead(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	current, ok := h.signedInAccount(c, "marking a notification read")
	if !ok {
		return
	}
	found, err := h.notifications.MarkRead(c.Request.Context(), current.ID, id)
	switch {
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not mark the notification read."})
	case !found:
		c.JSON(http.StatusNotFound, gin.H{"error": "no such notification"})
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) MarkAllNotificationsRead(c *gin.Context) {
	current, ok := h.signedInAccount(c, "marking your notifications read")
	if !ok {
		return
	}
	if err := h.notifications.MarkAllRead(c.Request.Context(), current.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not mark your notifications read."})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) RemoveNotification(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	current, ok := h.signedInAccount(c, "removing a notification")
	if !ok {
		return
	}
	found, err := h.notifications.Remove(c.Request.Context(), current.ID, id)
	switch {
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not remove the notification."})
	case !found:
		c.JSON(http.StatusNotFound, gin.H{"error": "no such notification"})
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) ClearNotifications(c *gin.Context) {
	current, ok := h.signedInAccount(c, "clearing your notifications")
	if !ok {
		return
	}
	if err := h.notifications.Clear(c.Request.Context(), current.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not clear your notifications."})
		return
	}
	c.Status(http.StatusNoContent)
}

// sendTargetsFor gathers the instances that can take each update entry on one page.
func (h *Handlers) sendTargetsFor(
	c *gin.Context,
	account uuid.UUID,
	entries []notification.Entry,
) (map[uuid.UUID][]NotificationSendTarget, error) {
	updated := make([]uuid.UUID, 0, len(entries))
	seen := make(map[uuid.UUID]bool, len(entries))
	for _, entry := range entries {
		if entry.Type != notification.AssetUpdated || entry.Asset == nil || seen[*entry.Asset] {
			continue
		}
		seen[*entry.Asset] = true
		updated = append(updated, *entry.Asset)
	}
	holders, err := h.deliveries.UpdatableInstances(c.Request.Context(), account, updated)
	if err != nil {
		return nil, err
	}
	offered := make(map[uuid.UUID][]NotificationSendTarget, len(holders))
	for assetID, instances := range holders {
		for _, state := range instances {
			offered[assetID] = append(offered[assetID], NotificationSendTarget{
				InstanceId:      state.InstanceID,
				InstanceName:    state.InstanceName,
				ApplicationName: state.ApplicationName,
				Waiting:         waitingToCollect(state),
			})
		}
		slices.SortFunc(offered[assetID], byInstanceName)
	}
	return offered, nil
}

// waitingToCollect says whether a delivery of the asset is already waiting for the instance.
func waitingToCollect(state delivery.InstanceState) bool {
	if state.Delivery == nil {
		return false
	}
	return state.Delivery.State == delivery.StateQueued || state.Delivery.State == delivery.StateReleased
}

// byInstanceName keeps the sends an entry offers in the order a reader would read them.
func byInstanceName(first, second NotificationSendTarget) int {
	if named := strings.Compare(first.ApplicationName, second.ApplicationName); named != 0 {
		return named
	}
	return strings.Compare(first.InstanceName, second.InstanceName)
}

func toAPINotification(
	entry notification.Entry,
	sends map[uuid.UUID][]NotificationSendTarget,
) Notification {
	shown := Notification{
		Id: entry.ID, Type: NotificationType(entry.Type), CreatedAt: entry.CreatedAt, ReadAt: entry.ReadAt,
	}
	if entry.Asset != nil {
		shown.Asset = &NotificationAsset{Id: *entry.Asset, Name: entry.Words.AssetName}
	}
	if entry.Words.Reason != "" {
		reason := entry.Words.Reason
		shown.Reason = &reason
	}
	if entry.Type == notification.AssetUpdated {
		shown.Update = &NotificationUpdate{
			Number: entry.Words.UpdateNumber, Summary: entry.Words.Summary, Count: entry.Count,
		}
		if entry.Words.VersionLabel != "" {
			label := entry.Words.VersionLabel
			shown.Update.VersionLabel = &label
		}
		if entry.Asset == nil {
			return shown
		}
		if offered := sends[*entry.Asset]; len(offered) > 0 {
			shown.SendTargets = &offered
		}
	}
	return shown
}
