package blog

import (
	"context"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	"github.com/Sillyfrogster/Illarin/api/internal/secrets"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool    *pgxpool.Pool
	sealing secrets.Key
	sender  Sender
	ledger  dispatch.Ledger
	now     func() time.Time
	Summary func(context.Context, db.DBTX, uuid.UUID) (Event, error)
	Audit   func(context.Context, pgx.Tx, Change) error
}

func NewService(pool *pgxpool.Pool, sealing secrets.Key, sender Sender, summary func(context.Context, db.DBTX, uuid.UUID) (Event, error), audit func(context.Context, pgx.Tx, Change) error) *Service {
	return &Service{pool: pool, sealing: sealing, sender: sender, ledger: dispatch.NewLedger(pool, deliveryTables), now: time.Now, Summary: summary, Audit: audit}
}

type Change struct {
	Actor      uuid.UUID
	Credential string
	Action     string
	AppID      *uuid.UUID
	CategoryID *uuid.UUID
	GrantID    *uuid.UUID
	PostID     *uuid.UUID
	RevisionID *uuid.UUID
	ScheduleID *uuid.UUID
	SubjectID  *uuid.UUID

	DestinationID *uuid.UUID
	DeliveryID    *uuid.UUID
	Before        string
	After         string
}

const (
	EventPublished   = "publication.post.published.v1"
	EventUpdated     = "publication.post.updated.v1"
	EventWithdrawn   = "publication.post.withdrawn.v1"
	CredentialSystem = "system"
)

const CredentialSession = "session"
