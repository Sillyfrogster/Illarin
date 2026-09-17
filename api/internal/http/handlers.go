package http

import (
	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/block/edit"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/Sillyfrogster/Illarin/api/internal/download"
	"github.com/Sillyfrogster/Illarin/api/internal/integration"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/Sillyfrogster/Illarin/api/internal/upload"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
)

type Handlers struct {
	assets             *asset.Service
	works              *work.Service
	blocks             *edit.Service
	versions           *version.Service
	uploads            *upload.Service
	downloads          *download.Handlers
	accounts           *account.Service
	links              *connect.Apps
	deliveries         *connect.Sends
	publications       *publication.Service
	updateDestinations *integration.Service
	notifications      *notify.Service
	maxUploadBytes     int64
}

func NewHandlers(
	assets *asset.Service,
	works *work.Service,
	blocks *edit.Service,
	versions *version.Service,
	uploads *upload.Service,
	downloads *download.Service,
	accounts *account.Service,
	links *connect.Apps,
	deliveries *connect.Sends,
	publications *publication.Service,
	updateDestinations *integration.Service,
	notifications *notify.Service,
	maxUploadBytes int64,
) *Handlers {
	return &Handlers{
		assets:             assets,
		works:              works,
		blocks:             blocks,
		versions:           versions,
		uploads:            uploads,
		downloads:          download.NewHandlers(downloads, accounts),
		accounts:           accounts,
		links:              links,
		deliveries:         deliveries,
		publications:       publications,
		updateDestinations: updateDestinations,
		notifications:      notifications,
		maxUploadBytes:     maxUploadBytes,
	}
}
