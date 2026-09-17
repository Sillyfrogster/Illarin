// Package full registers every route the server has, for tests outside the http package
package full

import (
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	apihttp "github.com/Sillyfrogster/Illarin/api/internal/http"
	"github.com/gin-gonic/gin"
)

// Harness builds test routers that serve every route
var Harness = apitest.Harness{Register: Register}

func Register(r *gin.Engine, s apitest.Services, d api.Deadlines) error {
	handlers := apihttp.NewHandlers(
		s.Assets, s.Works, s.Blocks, s.Versions, s.Uploads, s.Downloads, s.Accounts, s.Links, s.Deliveries, s.Publications,
		s.UpdateDestinations, s.Notifications, s.MaxUploadBytes,
	)
	return apihttp.Register(r, handlers, d, apitest.Ready)
}
