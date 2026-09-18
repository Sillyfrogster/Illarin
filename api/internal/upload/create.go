package upload

import (
	"context"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

func (s *Service) Create(ctx context.Context, in CreateInput) (work.Asset, error) {
	assetID := uuid.New()

	stored, err := s.store.Put(ctx, in.File)
	if err != nil {
		return work.Asset{}, fmt.Errorf("store upload: %w", err)
	}

	inspected, err := format.Inspect(ctx, s.store, stored.ID, stored.ByteSize, in.Filename)
	if err != nil {
		return work.Asset{}, fmt.Errorf("probe upload: %w", err)
	}
	read, err := s.readImport(ctx, inspected, "")
	if err != nil {
		return work.Asset{}, fmt.Errorf("read upload: %w", err)
	}
	if read, err = s.seedFromReadme(ctx, inspected, read); err != nil {
		return work.Asset{}, fmt.Errorf("seed the page from the README: %w", err)
	}
	parsed := read.Parsed
	kind := parsed.Kind
	discovery := in.Discovery
	if discovery == "" {
		discovery = work.DiscoveryListed
	}

	a := work.Asset{
		ID: assetID, Kind: kind, Format: parsed.Format, OriginFormat: &parsed.Format,
		AssetVersion: parsed.Header.AssetVersion, CreditedAuthor: parsed.Header.CreditedAuthor,
		Nickname:  parsed.Header.Nickname,
		Name:      orElse(in.Name, parsed.Header.Name),
		Blurb:     orElse(in.Blurb, parsed.Header.Blurb),
		Tags:      in.Tags,
		IsNSFW:    &in.IsNSFW,
		Discovery: discovery,
		Lifecycle: work.LifecyclePublished,
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
		return work.Asset{}, err
	}
	defer tx.Rollback(ctx)

	made, err := work.InsertAsset(ctx, tx, a, in.OwnerID, firstDate(in.CreatedAt, parsed.CreatedAt))
	if err != nil {
		return work.Asset{}, err
	}
	a.CreatedAt = made
	revisionID, err := work.RecordRevision(ctx, tx, work.Revision{
		AssetID: a.ID, Number: 1, BlobID: stored.ID, MediaType: "application/octet-stream",
		Format: a.Format, Media: extractedMedia,
	})
	if err != nil {
		return work.Asset{}, err
	}
	if err := block.Insert(ctx, tx, a.ID, blocks); err != nil {
		return work.Asset{}, err
	}
	if err := insertVaultPictures(ctx, tx, a.ID, read.Vault); err != nil {
		return work.Asset{}, err
	}
	if err := replacePreservedData(ctx, tx, a.ID, parsed.Remainder); err != nil {
		return work.Asset{}, err
	}
	if err := s.writeSummary(ctx, tx, a.ID); err != nil {
		return work.Asset{}, err
	}
	if _, err := tx.Exec(ctx, `select record_initial_asset_snapshot($1, false)`, a.ID); err != nil {
		return work.Asset{}, fmt.Errorf("record initial publication: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return work.Asset{}, err
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
