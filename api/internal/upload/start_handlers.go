package upload

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) startAssetFromNothing(c *gin.Context, owner api.Account) {
	var request StartAssetRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the kind to build as JSON.")
		return
	}
	app := ""
	if request.App != nil {
		app = string(*request.App)
	}
	id, err := h.uploads.StartFromNothing(c.Request.Context(), owner.ID, request.Kind, app)
	if errors.Is(err, ErrKindNotBuildable) {
		api.Refuse(c, http.StatusBadRequest, "Illarin cannot build that kind yet. Choose another.")
		return
	}
	if errors.Is(err, ErrAppNotAnswered) {
		api.Refuse(c, http.StatusBadRequest, appAnswerRefusal(request.Kind))
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not start the asset.")
		return
	}

	found, err := h.works.Detail(c.Request.Context(), id, &owner.ID, asset.ContentShown)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the new asset.")
		return
	}
	page, err := work.ToPage(found, asset.ContentShown)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the new asset.")
		return
	}
	c.Header("Location", "/v1/assets/"+id.String())
	c.JSON(http.StatusCreated, page)
}

func appAnswerRefusal(kind string) string {
	apps := Apps(kind)
	if len(apps) == 0 {
		return "Nothing about this kind depends on an app, so do not send one."
	}
	return "Say which app this is for: " + strings.Join(apps, " or ") + "."
}
