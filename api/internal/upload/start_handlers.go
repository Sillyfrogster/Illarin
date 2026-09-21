package upload

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) startWorkFromNothing(c *gin.Context, owner api.Account) {
	var request StartWorkRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the type to build as JSON.")
		return
	}
	app := ""
	if request.App != nil {
		app = *request.App
	}
	id, err := h.uploads.StartFromNothing(c.Request.Context(), owner.ID, request.Type, app)
	if errors.Is(err, ErrTypeNotBuildable) {
		api.Refuse(c, http.StatusBadRequest, "Illarin cannot build that type yet. Choose another.")
		return
	}
	if errors.Is(err, ErrAppNotAnswered) {
		api.Refuse(c, http.StatusBadRequest, appAnswerRefusal(request.Type))
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not start the work.")
		return
	}

	found, err := h.works.Detail(c.Request.Context(), id, &owner.ID, work.NSFWShown)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the new work.")
		return
	}
	page, err := page.ToPage(found, work.NSFWShown)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the new work.")
		return
	}
	c.Header("Location", "/v1/works/"+id.String())
	c.JSON(http.StatusCreated, page)
}

// GetBuildChoices answers which types can be built from nothing and which apps each asks for
func (h *Handlers) GetBuildChoices(c *gin.Context) {
	c.JSON(http.StatusOK, h.uploads.BuildChoices())
}

func appAnswerRefusal(workType string) string {
	apps := appsAsked(workType)
	if len(apps) == 0 {
		return "Nothing about this type depends on an app, so do not send one."
	}
	labels := make([]string, len(apps))
	for i, app := range apps {
		labels[i] = format.AppLabel(app)
	}
	return "Pick the app this " + workType + " is for: " + strings.Join(labels, " or ") + "."
}
