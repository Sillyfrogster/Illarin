package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/notification"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) WatchAsset(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	current, ok := h.signedInAccount(c, "watching an asset")
	if !ok {
		return
	}
	watch, err := h.notifications.StartWatching(c.Request.Context(), current.ID, id)
	answerWatch(c, watch, err)
}

func (h *Handlers) StopWatchingAsset(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	current, ok := h.signedInAccount(c, "watching an asset")
	if !ok {
		return
	}
	watch, err := h.notifications.StopWatching(c.Request.Context(), current.ID, id)
	answerWatch(c, watch, err)
}

func answerWatch(c *gin.Context, watch notification.Watch, err error) {
	switch {
	case errors.Is(err, notification.ErrNothingToWatch):
		c.JSON(http.StatusNotFound, gin.H{"error": "no such asset"})
	case errors.Is(err, notification.ErrOwnAsset):
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot watch your own asset."})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not change your watch on this asset. Try again."})
	default:
		c.JSON(http.StatusOK, toAPIWatch(watch))
	}
}

func toAPIWatch(watch notification.Watch) AssetWatch {
	installedOn := watch.InstalledOn
	if installedOn == nil {
		installedOn = []string{}
	}
	return AssetWatch{State: AssetWatchState(watch.State), InstalledOn: installedOn}
}
