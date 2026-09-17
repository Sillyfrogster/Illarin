package asset

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
	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/Sillyfrogster/Illarin/api/internal/protected"
	"github.com/Sillyfrogster/Illarin/api/internal/signing"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound         = errors.New("asset not found")
	ErrInvalidDiscovery = errors.New("invalid discovery state")
	ErrAssetFrozen      = errors.New("asset is frozen")
	ErrInvalidBlock     = block.ErrInvalid
	ErrStorageCap       = errors.New("account storage cap exceeded")
	ErrAssetIsDraft     = errors.New("the asset is still a draft")
)

type Service struct {
	pool            *pgxpool.Pool
	reg             *format.Registry
	store           storage.Store
	media           *mediaproc.Library
	ingest          IngestSettings
	signer          signing.Key
	now             func() time.Time
	siteURL         string
	updateListeners []UpdateListener
}

func (s *Service) BeginReadSnapshot(ctx context.Context) (pgx.Tx, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `set local search_path = asset_public, public`); err != nil {
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
		ingest: settings, signer: signing.NewKey(), now: time.Now,
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
			  from asset_revisions revision
			  join assets asset on asset.id = revision.asset_id
			 where asset.owner_id = $1
			   and revision.blob_id is not null
			   and (asset.deleted_at is null or asset.recoverable_until > $3)
			union
			select media.blob_id
			  from asset_media media
			  join assets asset on asset.id = media.asset_id
			 where asset.owner_id = $1
			   and media.blob_id is not null
			   and (asset.deleted_at is null or asset.recoverable_until > $3)
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

func (s *Service) OpenSource(ctx context.Context, assetID uuid.UUID) (io.ReadCloser, error) {
	location, err := currentRevisionLocation(ctx, s.pool, assetID, nil)
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

type SourceDownload struct {
	InternalRedirect string
	MediaType        string
	Inline           bool
	Event            DownloadEvent
}

func (s *Service) DownloadSource(
	ctx context.Context,
	assetID uuid.UUID,
	viewerID *uuid.UUID,
) (SourceDownload, error) {
	tx, err := s.BeginReadSnapshot(ctx)
	if err != nil {
		return SourceDownload{}, err
	}
	defer tx.Rollback(ctx)
	location, err := currentRevisionLocation(ctx, tx, assetID, viewerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SourceDownload{}, ErrNotFound
		}
		return SourceDownload{}, fmt.Errorf("find current revision: %w", err)
	}
	apps, err := protected.Apps(ctx, tx, assetID)
	if err != nil {
		return SourceDownload{}, err
	}
	blocks, err := block.Read(ctx, tx, assetID)
	if err != nil {
		return SourceDownload{}, err
	}
	if err := protected.ApplyPublishedPolicy(ctx, tx, assetID, blocks); err != nil {
		return SourceDownload{}, err
	}
	if (len(apps) > 0 || protected.HasPromptFragments(blocks)) && (viewerID == nil || location.OwnerID == nil || *viewerID != *location.OwnerID) {
		return SourceDownload{}, ErrLinkedInstallOnly
	}
	redirect, err := s.store.InternalRedirect(ctx, location.BlobID)
	if err != nil {
		return SourceDownload{}, fmt.Errorf("resolve stored file: %w", err)
	}
	revisionID := location.RevisionID
	return SourceDownload{
		InternalRedirect: redirect, MediaType: location.MediaType,
		Inline: format.IsInlineMediaType(location.MediaType),
		Event: downloadEvent(
			location.AssetID, &revisionID, RawDownloadTarget,
			location.OwnerID, viewerID,
		),
	}, nil
}

func (s *Service) DownloadExport(
	ctx context.Context,
	assetID uuid.UUID,
	viewerID *uuid.UUID,
	target string,
	gallery *GallerySelection,
) (Export, error) {
	return s.OpenExport(ctx, assetID, viewerID, target, gallery)
}

func (s *Service) DownloadRecordedExport(
	ctx context.Context,
	assetID uuid.UUID,
	viewerID *uuid.UUID,
	number int,
	target string,
	gallery *GallerySelection,
) (Export, error) {
	return s.OpenRecordedExport(ctx, assetID, viewerID, number, target, gallery)
}

func (s *Service) DownloadExportForLinkedInstance(
	ctx context.Context,
	assetID uuid.UUID,
	target string,
) (Export, error) {
	download, err := s.OpenExportForLinkedInstance(ctx, assetID, target)
	if err != nil {
		return Export{}, err
	}
	if download.Event != nil {
		download.Event.AuthorizationClass = AuthorizationLinkedInstance
	}
	return download, nil
}

func (s *Service) Now() time.Time {
	return s.now()
}
func AssetByID(ctx context.Context, q db.DBTX, id uuid.UUID) (Asset, error) {
	row, err := db.New(q).AssetByID(ctx, uuidToPgtype(id))
	if err != nil {
		return Asset{}, fmt.Errorf("read asset: %w", err)
	}
	return Asset{
		ID: uuidFromPgtype(row.ID), Kind: row.Kind, Format: row.Format,
		OriginFormat: textToPointer(row.OriginFormat), AssetVersion: row.AssetVersion,
		CreditedAuthor: row.CreditedAuthor, Nickname: row.Nickname,
		Name: row.Name, Blurb: row.Blurb, Tags: row.Tags,
		IsNSFW: &row.IsNsfw, Discovery: Discovery(row.Discovery), Lifecycle: Lifecycle(row.Lifecycle),
		CurrentRevisionID: uuidFromPgtype(row.CurrentRevisionID),
		CreatedAt:         timeFromPgtype(row.CreatedAt),
	}, nil
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
