package asset

import (
	"context"
	"encoding/json"
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
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound         = errors.New("asset not found")
	ErrIngestNotFound   = errors.New("ingest operation not found")
	ErrInvalidDiscovery = errors.New("invalid discovery state")
	ErrAssetFrozen      = errors.New("asset is frozen")
	ErrInvalidBlock     = block.ErrInvalid
	ErrStorageCap       = errors.New("account storage cap exceeded")
	ErrAssetIsDraft     = errors.New("the asset is still a draft")
	ErrKindNotBuildable = errors.New("that kind cannot be built yet")
	ErrAppNotAnswered   = errors.New("that kind needs to know which app it is for")
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

func (s *Service) AcceptIngest(ctx context.Context, in IngestInput) (IngestOperation, error) {
	stored, err := s.store.Put(ctx, in.File)
	if err != nil {
		return IngestOperation{}, fmt.Errorf("store upload: %w", err)
	}

	id := uuid.New()
	var tags []string
	if in.Tags != nil {
		tags = *in.Tags
	}
	discovery := in.Discovery
	if discovery == "" {
		discovery = DiscoveryListed
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return IngestOperation{}, fmt.Errorf("begin ingest acceptance: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := s.ensureAccountStorage(ctx, tx, in.OwnerID, []uuid.UUID{stored.ID}); err != nil {
		return IngestOperation{}, err
	}
	_, err = tx.Exec(ctx, `
		insert into ingest_operations
			(id, owner_id, blob_id, filename, status, name, blurb, tags, is_nsfw, discovery)
		values ($1, $2, $3, $4, 'pending', $5, $6, $7, $8, $9)
	`, id, in.OwnerID, stored.ID, in.Filename, in.Name, in.Blurb, tags, in.IsNSFW, discovery)
	if err != nil {
		return IngestOperation{}, fmt.Errorf("record ingest: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return IngestOperation{}, fmt.Errorf("commit ingest acceptance: %w", err)
	}
	return IngestOperation{ID: id, Status: IngestPending}, nil
}

func (s *Service) ensureAccountStorage(
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

func (s *Service) GetIngest(ctx context.Context, ownerID, id uuid.UUID) (IngestOperation, error) {
	var status IngestStatus
	var assetID pgtype.UUID
	var failureReason pgtype.Text
	var failureMessage pgtype.Text
	var replacementPreview []byte
	err := s.pool.QueryRow(ctx, `
		select status, asset_id, failure_reason, failure_message, replacement_preview
		  from ingest_operations where id = $1 and owner_id = $2
	`, id, ownerID).Scan(&status, &assetID, &failureReason, &failureMessage, &replacementPreview)
	if errors.Is(err, pgx.ErrNoRows) {
		return IngestOperation{}, ErrIngestNotFound
	}
	if err != nil {
		return IngestOperation{}, fmt.Errorf("read ingest: %w", err)
	}
	operation := IngestOperation{ID: id, Status: status}
	if status == IngestFailed && failureReason.Valid {
		message := s.ingestFailureMessage(failureReason.String)
		if failureMessage.Valid {
			message = failureMessage.String
		}
		operation.Failure = &IngestFailure{
			Reason:  failureReason.String,
			Message: message,
		}
	}
	if status == IngestPreview {
		var staged stagedReplacement
		if err := json.Unmarshal(replacementPreview, &staged); err != nil {
			return IngestOperation{}, fmt.Errorf("read replacement preview: %w", err)
		}
		operation.Preview = &staged.Preview
	}
	if assetID.Valid {
		created, err := assetByID(ctx, s.pool, uuidFromPgtype(assetID))
		if err != nil {
			return IngestOperation{}, err
		}
		operation.Asset = &created
	}
	return operation, nil
}

func (s *Service) ingestFailureMessage(reason string) string {
	switch reason {
	case "malformed_input":
		return "The file is malformed and could not be read."
	case "unsupported_format":
		return "No supported format recognised this file. Illarin can read " +
			joinReadable(s.reg.ReadableLabels()) +
			". If yours is not one of those, start from nothing and build it here."
	case "unsupported_version":
		return "The file uses a version Illarin cannot read safely."
	case "safety_violation":
		return "The file breaks an archive safety rule."
	case "limit_exceeded":
		return "The file is over a content limit."
	case "wrong_kind":
		return "This file is a different kind of thing than the asset it would update."
	default:
		return "Illarin could not finish this upload. Please try again."
	}
}

func (s *Service) StartFromNothing(
	ctx context.Context,
	ownerID uuid.UUID,
	kind string,
	app string,
) (uuid.UUID, error) {
	if _, ok := block.Catalog(kind); !ok || !s.reg.BuildsFromNothing(kind) {
		return uuid.Nil, ErrKindNotBuildable
	}
	seeded, err := seedElements(kind, app)
	if err != nil {
		return uuid.Nil, err
	}
	blocks, err := block.Place(kind, seeded)
	if err != nil {
		return uuid.Nil, ErrKindNotBuildable
	}

	a := Asset{
		ID: uuid.New(), Kind: kind, Tags: []string{},
		Discovery: DiscoveryListed, Lifecycle: LifecycleDraft,
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := insertAsset(ctx, tx, a, ownerID, nil); err != nil {
		return uuid.Nil, err
	}
	if err := block.Insert(ctx, tx, a.ID, blocks); err != nil {
		return uuid.Nil, err
	}
	if err := s.writeProjections(ctx, tx, a.ID); err != nil {
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return a.ID, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Asset, error) {
	assetID := uuid.New()
	revisionID := uuid.New()

	stored, err := s.store.Put(ctx, in.File)
	if err != nil {
		return Asset{}, fmt.Errorf("store upload: %w", err)
	}

	inspected, err := format.Inspect(ctx, s.store, stored.ID, stored.ByteSize, in.Filename)
	if err != nil {
		return Asset{}, fmt.Errorf("probe upload: %w", err)
	}
	read, err := s.readImport(ctx, inspected, "")
	if err != nil {
		return Asset{}, fmt.Errorf("read upload: %w", err)
	}
	if read, err = s.seedFromReadme(ctx, inspected, read); err != nil {
		return Asset{}, fmt.Errorf("seed the page from the README: %w", err)
	}
	parsed := read.Parsed
	kind := parsed.Kind
	discovery := in.Discovery
	if discovery == "" {
		discovery = DiscoveryListed
	}

	a := Asset{
		ID: assetID, Kind: kind, Format: parsed.Format, OriginFormat: &parsed.Format,
		AssetVersion: parsed.Header.AssetVersion, CreditedAuthor: parsed.Header.CreditedAuthor,
		Nickname:  parsed.Header.Nickname,
		Name:      orElse(in.Name, parsed.Header.Name),
		Blurb:     orElse(in.Blurb, parsed.Header.Blurb),
		Tags:      in.Tags,
		IsNSFW:    &in.IsNSFW,
		Discovery: discovery,
		Lifecycle: LifecyclePublished,
	}
	if len(a.Tags) == 0 {
		a.Tags = parsed.Tags
	}
	if a.Tags == nil {
		a.Tags = []string{}
	}
	extractedMedia := read.Media
	blocks := read.Blocks

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Asset{}, err
	}
	defer tx.Rollback(ctx)

	made, err := insertAsset(ctx, tx, a, in.OwnerID, firstDate(in.CreatedAt, parsed.CreatedAt))
	if err != nil {
		return Asset{}, err
	}
	a.CreatedAt = made
	if err := insertRevision(ctx, tx, revisionID, a.ID, revisionRow{
		Revision: 1, BlobID: stored.ID, MediaType: "application/octet-stream", Format: a.Format,
	}); err != nil {
		return Asset{}, err
	}
	if err := insertAssetMedia(ctx, tx, a.ID, extractedMedia); err != nil {
		return Asset{}, err
	}
	if err := block.Insert(ctx, tx, a.ID, blocks); err != nil {
		return Asset{}, err
	}
	if err := insertVaultPictures(ctx, tx, a.ID, read.Vault); err != nil {
		return Asset{}, err
	}
	if err := replacePreservedData(ctx, tx, a.ID, parsed.Remainder); err != nil {
		return Asset{}, err
	}
	if err := setCurrentRevision(ctx, tx, a.ID, revisionID); err != nil {
		return Asset{}, err
	}
	if err := setCoverMedia(ctx, tx, a.ID, avatarMedia(extractedMedia)); err != nil {
		return Asset{}, err
	}
	if err := s.writeProjections(ctx, tx, a.ID); err != nil {
		return Asset{}, err
	}
	if _, err := tx.Exec(ctx, `select record_initial_asset_snapshot($1, false)`, a.ID); err != nil {
		return Asset{}, fmt.Errorf("record initial publication: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Asset{}, err
	}

	a.CurrentRevisionID = revisionID
	return a, nil
}

func orElse(preferred, fallback string) string {
	if preferred != "" {
		return preferred
	}
	return fallback
}

func firstDate(preferred, fallback *time.Time) *time.Time {
	if preferred != nil {
		return preferred
	}
	return fallback
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

func joinReadable(labels []string) string {
	switch len(labels) {
	case 0:
		return "nothing yet"
	case 1:
		return labels[0]
	default:
		return strings.Join(labels[:len(labels)-1], ", ") + " and " + labels[len(labels)-1]
	}
}

func (s *Service) Now() time.Time {
	return s.now()
}
func assetByID(ctx context.Context, q db.DBTX, id uuid.UUID) (Asset, error) {
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
