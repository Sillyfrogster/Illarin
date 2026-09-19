package version

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrUnlistedConsentRequired says the creator has not agreed to send an unlisted work's direct link
var ErrUnlistedConsentRequired = errors.New(
	"announcing an unlisted work sends its direct link, which needs explicit consent",
)

var ErrIntegrationIneligible = errors.New("choose only your own verified, active integrations")

type Announcement struct {
	IntegrationIDs   *[]uuid.UUID
	AnnounceUnlisted bool
	Notify           bool
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
