package http

import (
	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/assetdestination"
	"github.com/Sillyfrogster/Illarin/api/internal/delivery"
	"github.com/Sillyfrogster/Illarin/api/internal/linking"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
)

type Handlers struct {
	assets             *asset.Service
	works              *work.Service
	accounts           *account.Service
	links              *linking.Service
	deliveries         *delivery.Service
	publications       *publication.Service
	updateDestinations *assetdestination.Service
	notifications      *notify.Service
	maxUploadBytes     int64
}

func NewHandlers(
	assets *asset.Service,
	works *work.Service,
	accounts *account.Service,
	links *linking.Service,
	deliveries *delivery.Service,
	publications *publication.Service,
	updateDestinations *assetdestination.Service,
	notifications *notify.Service,
	maxUploadBytes int64,
) *Handlers {
	return &Handlers{
		assets:             assets,
		works:              works,
		accounts:           accounts,
		links:              links,
		deliveries:         deliveries,
		publications:       publications,
		updateDestinations: updateDestinations,
		notifications:      notifications,
		maxUploadBytes:     maxUploadBytes,
	}
}
