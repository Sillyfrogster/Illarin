package work

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/summary"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound          = errors.New("work not found")
	ErrInvalidVisibility = errors.New("invalid visibility state")
	ErrWorkFrozen        = errors.New("work is frozen")
	ErrInvalidBlock      = block.ErrInvalid
	ErrStorageCap        = errors.New("account storage cap exceeded")
	ErrWorkIsDraft       = errors.New("the work is still a draft")
)

type Service struct {
	pool    *pgxpool.Pool
	reg     *format.Registry
	store   storage.Store
	media   *mediaproc.Library
	ingest  IngestSettings
	signer  dispatch.Key
	now     func() time.Time
	siteURL string
}

func (s *Service) BeginReadSnapshot(ctx context.Context) (pgx.Tx, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `set local search_path = work_public, public`); err != nil {
		tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}

type IngestSettings struct {
	ProbeLimits            format.Limits
	LeaseDuration          time.Duration
	RetryBase              time.Duration
	MaxAttempts            int
	MediaWorkers           int
	AccountStorageCapBytes int64
}

type MediaProcessor = mediaproc.Renderer

func DefaultIngestSettings() IngestSettings {
	return IngestSettings{
		ProbeLimits:   format.DefaultLimits(),
		LeaseDuration: 30 * time.Second,
		RetryBase:     time.Second,
		MaxAttempts:   3,
		MediaWorkers:  2,
	}
}

func NewService(pool *pgxpool.Pool, reg *format.Registry, store storage.Store) *Service {
	return NewServiceWithIngestSettings(pool, reg, store, DefaultIngestSettings())
}

func NewServiceWithProbeLimits(
	pool *pgxpool.Pool,
	reg *format.Registry,
	store storage.Store,
	limits format.Limits,
) *Service {
	settings := DefaultIngestSettings()
	settings.ProbeLimits = limits
	return NewServiceWithIngestSettings(pool, reg, store, settings)
}

func NewServiceForSite(
	pool *pgxpool.Pool,
	reg *format.Registry,
	store storage.Store,
	limits format.Limits,
	siteURL string,
	accountStorageCapBytes int64,
) *Service {
	settings := DefaultIngestSettings()
	settings.ProbeLimits = limits
	settings.AccountStorageCapBytes = accountStorageCapBytes
	service := NewServiceWithIngestSettings(pool, reg, store, settings)
	service.siteURL = strings.TrimRight(siteURL, "/")
	return service
}

func NewServiceWithIngestSettings(
	pool *pgxpool.Pool,
	reg *format.Registry,
	store storage.Store,
	settings IngestSettings,
) *Service {
	return NewServiceWithMediaProcessor(
		pool, reg, store, settings,
		mediaproc.NewProcessor(mediaproc.DefaultLimits()),
	)
}

// NewServiceWithClock reads the time from the given clock rather than the wall clock
func NewServiceWithClock(
	pool *pgxpool.Pool,
	reg *format.Registry,
	store storage.Store,
	now func() time.Time,
) *Service {
	service := NewService(pool, reg, store)
	service.now = now
	return service
}

func NewServiceWithMediaProcessor(
	pool *pgxpool.Pool,
	reg *format.Registry,
	store storage.Store,
	settings IngestSettings,
	processor MediaProcessor,
) *Service {
	workers := settings.MediaWorkers
	if workers < 1 {
		workers = 1
	}
	return &Service{
		pool: pool, reg: reg, store: store,
		media:  mediaproc.NewLibrary(store, processor, workers),
		ingest: settings, signer: dispatch.NewKey(), now: time.Now,
	}
}

func (s *Service) EnsureAccountStorage(
	ctx context.Context,
	tx pgx.Tx,
	ownerID uuid.UUID,
	candidates []uuid.UUID,
) error {
	capBytes := s.ingest.AccountStorageCapBytes
	if capBytes <= 0 || len(candidates) == 0 {
		return nil
	}
	var locked uuid.UUID
	if err := tx.QueryRow(ctx, `select id from users where id = $1 for update`, ownerID).Scan(&locked); err != nil {
		return fmt.Errorf("lock account storage: %w", err)
	}
	var usedBytes, additionalBytes int64
	err := tx.QueryRow(ctx, `
		with account_blobs as (
			select operation.blob_id
			  from ingest_operations operation
			 where operation.owner_id = $1
			   and operation.blob_id is not null
			   and operation.status in ('pending', 'processing')
			union
			select revision.blob_id
			  from work_revisions revision
			  join works work on work.id = revision.work_id
			 where work.owner_id = $1
			   and revision.blob_id is not null
			   and (work.deleted_at is null or work.recoverable_until > $3)
			union
			select media.blob_id
			  from work_media media
			  join works work on work.id = media.work_id
			 where work.owner_id = $1
			   and media.blob_id is not null
			   and (work.deleted_at is null or work.recoverable_until > $3)
		), candidate_blobs as (
			select distinct unnest($2::uuid[]) as blob_id
		)
		select
			coalesce((
				select sum(blob.byte_size)
				  from account_blobs account_blob
				  join blobs blob on blob.id = account_blob.blob_id
			), 0),
			coalesce((
				select sum(blob.byte_size)
				  from candidate_blobs candidate
				  join blobs blob on blob.id = candidate.blob_id
				 where not exists (
					select 1 from account_blobs account_blob
					 where account_blob.blob_id = candidate.blob_id
				 )
			), 0)
	`, ownerID, candidates, s.now()).Scan(&usedBytes, &additionalBytes)
	if err != nil {
		return fmt.Errorf("measure account storage: %w", err)
	}
	if additionalBytes > 0 && (additionalBytes > capBytes || usedBytes > capBytes-additionalBytes) {
		return ErrStorageCap
	}
	return nil
}

func (s *Service) OpenSource(ctx context.Context, workID uuid.UUID) (io.ReadCloser, error) {
	location, err := CurrentRevisionLocation(ctx, s.pool, workID, nil)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find current revision: %w", err)
	}

	rc, err := s.store.Open(ctx, location.BlobID)
	if err != nil {
		return nil, fmt.Errorf("open stored file: %w", err)
	}
	return rc, nil
}

func (s *Service) Now() time.Time {
	return s.now()
}
func WorkByID(ctx context.Context, q db.DBTX, id uuid.UUID) (Work, error) {
	row, err := db.New(q).WorkByID(ctx, uuidToPgtype(id))
	if err != nil {
		return Work{}, fmt.Errorf("read work: %w", err)
	}
	return Work{
		ID: uuidFromPgtype(row.ID), Type: row.Type, Format: row.Format,
		OriginFormat: textToPointer(row.OriginFormat), WorkVersion: row.WorkVersion,
		CreditedAuthor: row.CreditedAuthor, Nickname: row.Nickname,
		Name: row.Name, Blurb: row.Blurb, Tags: row.Tags,
		IsNSFW: &row.IsNsfw, Visibility: Visibility(row.Visibility), Lifecycle: Lifecycle(row.Lifecycle),
		CurrentRevisionID: uuidFromPgtype(row.CurrentRevisionID),
		CreatedAt:         timeFromPgtype(row.CreatedAt),
	}, nil
}

// missingWork says the work is gone in this package's words
func missingWork(err error) error {
	if errors.Is(err, summary.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func (s *Service) Registry() *format.Registry {
	return s.reg
}

func (s *Service) Store() storage.Store {
	return s.store
}

func (s *Service) IngestSettings() IngestSettings {
	return s.ingest
}

func (s *Service) Pool() *pgxpool.Pool {
	return s.pool
}
