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

var ErrUpdateDestinationIneligible = errors.New("choose only your own verified, active update destinations")

type UpdateAnnouncement struct {
	DestinationIDs   *[]uuid.UUID
	AnnounceUnlisted bool
	Notify           bool
}

type Update struct {
	ID                uuid.UUID
	WorkID            uuid.UUID
	Number            int
	RecordedAt        time.Time
	VersionLabel      string
	Summary           string
	Notes             string
	ContentGeneration int
	ContentChanged    bool
}

// UpdateListener is called inside the publish transaction with every update, so what it records commits with the update.
type UpdateListener func(ctx context.Context, tx pgx.Tx, published Update, choice UpdateAnnouncement) error

func (s *Service) OnUpdatePublished(listeners ...UpdateListener) {
	s.updateListeners = append(s.updateListeners, listeners...)
}

func (s *Service) UpdateListeners() []UpdateListener {
	return s.updateListeners
}
