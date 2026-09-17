package upload

import (
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool     *pgxpool.Pool
	reg      *format.Registry
	store    storage.Store
	settings asset.IngestSettings
	assets   *asset.Service
	now      func() time.Time
}

func NewService(pool *pgxpool.Pool, assets *asset.Service) *Service {
	return &Service{
		pool: pool, reg: assets.Registry(), store: assets.Store(),
		settings: assets.IngestSettings(), assets: assets, now: assets.Now,
	}
}
