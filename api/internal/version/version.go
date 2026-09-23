package version

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Announcement says who hears about a version: followers, and the creator's Discord channel
type Announcement struct {
	Notify  bool
	Discord bool
}

type Version struct {
	ID             uuid.UUID
	WorkID         uuid.UUID
	Number         int
	RecordedAt     time.Time
	VersionLabel   string
	Summary        string
	Notes          string
	ContentChanged bool
}

// Listener is called inside the publish transaction with every version, so what it records commits with the version.
type Listener func(ctx context.Context, tx pgx.Tx, published Version, choice Announcement) error

func (s *Service) OnPublished(listeners ...Listener) {
	s.listeners = append(s.listeners, listeners...)
}

func (s *Service) Listeners() []Listener {
	return s.listeners
}
