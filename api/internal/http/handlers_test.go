package http

import (
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/gin-gonic/gin"
)

var harness = apitest.Harness{Register: register}

func register(r *gin.Engine, s apitest.Services, d api.Deadlines) error {
	handlers := NewHandlers(
		s.Assets, s.Works, s.Blocks, s.Versions, s.Uploads, s.Accounts, s.Links, s.Deliveries, s.Publications,
		s.UpdateDestinations, s.Notifications, s.MaxUploadBytes,
	)
	return Register(r, handlers, d, apitest.Ready)
}
