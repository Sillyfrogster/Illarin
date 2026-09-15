package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/notification"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

func (h *Handlers) ListNotifications(c *gin.Context, params ListNotificationsParams) {
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
	listed := NotificationList{Items: make([]Notification, 0, len(page.Entries))}
	for _, entry := range page.Entries {
		listed.Items = append(listed.Items, toAPINotification(entry))
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

func (h *Handlers) MarkNotificationRead(c *gin.Context, id types.UUID) {
	current, ok := h.signedInAccount(c, "marking a notification read")
	if !ok {
		return
	}
	found, err := h.notifications.MarkRead(c.Request.Context(), current.ID, uuid.UUID(id))
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

func toAPINotification(entry notification.Entry) Notification {
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
	return shown
}
