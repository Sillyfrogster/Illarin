package http

import (
	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/delivery"
	"github.com/Sillyfrogster/Illarin/api/internal/linking"
	"github.com/Sillyfrogster/Illarin/api/internal/publication"
)

type Handlers struct {
	assets         *asset.Service
	accounts       *account.Service
	links          *linking.Service
	deliveries     *delivery.Service
	publications   *publication.Service
	maxUploadBytes int64
}

func NewHandlers(
	assets *asset.Service,
	accounts *account.Service,
	links *linking.Service,
	deliveries *delivery.Service,
	publications *publication.Service,
	maxUploadBytes int64,
) *Handlers {
	return &Handlers{
		assets:         assets,
		accounts:       accounts,
		links:          links,
		deliveries:     deliveries,
		publications:   publications,
		maxUploadBytes: maxUploadBytes,
	}
}
