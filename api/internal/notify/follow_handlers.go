package notify

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) FollowWork(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	current, ok := api.SignedIn(c, "following a work")
	if !ok {
		return
	}
	follow, err := h.notifications.StartFollowing(c.Request.Context(), current.ID, id)
	answerFollow(c, follow, err)
}

func (h *Handlers) StopFollowingWork(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	current, ok := api.SignedIn(c, "following a work")
	if !ok {
		return
	}
	follow, err := h.notifications.StopFollowing(c.Request.Context(), current.ID, id)
	answerFollow(c, follow, err)
}

func answerFollow(c *gin.Context, follow Follow, err error) {
	switch {
	case errors.Is(err, ErrNothingToFollow):
		api.Refuse(c, http.StatusNotFound, "no such work")
	case errors.Is(err, ErrOwnWork):
		api.Refuse(c, http.StatusForbidden, "You cannot follow your own work.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not change whether you follow this work. Try again.")
	default:
		c.JSON(http.StatusOK, ToAPIFollow(follow))
	}
}

// ToAPIFollow writes a follow as the site reads it
func ToAPIFollow(follow Follow) WorkFollow {
	installedOn := follow.InstalledOn
	if installedOn == nil {
		installedOn = []string{}
	}
	return WorkFollow{State: WorkFollowState(follow.State), InstalledOn: installedOn}
}
