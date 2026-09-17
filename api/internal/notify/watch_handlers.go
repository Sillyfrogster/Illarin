package notify

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) WatchAsset(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	current, ok := api.SignedIn(c, "watching an asset")
	if !ok {
		return
	}
	watch, err := h.notifications.StartWatching(c.Request.Context(), current.ID, id)
	answerWatch(c, watch, err)
}

func (h *Handlers) StopWatchingAsset(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	current, ok := api.SignedIn(c, "watching an asset")
	if !ok {
		return
	}
	watch, err := h.notifications.StopWatching(c.Request.Context(), current.ID, id)
	answerWatch(c, watch, err)
}

func answerWatch(c *gin.Context, watch Watch, err error) {
	switch {
	case errors.Is(err, ErrNothingToWatch):
		api.Refuse(c, http.StatusNotFound, "no such asset")
	case errors.Is(err, ErrOwnAsset):
		api.Refuse(c, http.StatusForbidden, "You cannot watch your own asset.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not change your watch on this asset. Try again.")
	default:
		c.JSON(http.StatusOK, ToAPIWatch(watch))
	}
}

// ToAPIWatch writes a watch as the site reads it
func ToAPIWatch(watch Watch) AssetWatch {
	installedOn := watch.InstalledOn
	if installedOn == nil {
		installedOn = []string{}
	}
	return AssetWatch{State: AssetWatchState(watch.State), InstalledOn: installedOn}
}
