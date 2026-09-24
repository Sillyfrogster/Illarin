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

func (s *Service) Create(ctx context.Context, in CreateInput) (work.Work, error) {
	workID := uuid.New()

	stored, err := s.store.Put(ctx, in.File)
	if err != nil {
		return work.Work{}, fmt.Errorf("store upload: %w", err)
	}

	inspected, err := format.Inspect(ctx, s.store, stored.ID, stored.ByteSize, in.Filename)
	if err != nil {
		return work.Work{}, fmt.Errorf("probe upload: %w", err)
	}
	read, err := s.readImport(ctx, inspected, "")
	if err != nil {
		return work.Work{}, fmt.Errorf("read upload: %w", err)
	}
	if read, err = s.seedFromReadme(ctx, inspected, read); err != nil {
		return work.Work{}, fmt.Errorf("seed the page from the README: %w", err)
	}
	parsed := read.Parsed
	workType := parsed.Type
	visibility := in.Visibility
	if visibility == "" {
		visibility = work.VisibilityListed
	}

	a := work.Work{
		ID: workID, Type: workType, Format: parsed.Format, OriginalFormat: &parsed.Format,
		WorkVersion: parsed.Header.WorkVersion, CreditedAuthor: parsed.Header.CreditedAuthor,
		Nickname:   parsed.Header.Nickname,
		Name:       orElse(in.Name, parsed.Header.Name),
		Blurb:      orElse(in.Blurb, parsed.Header.Blurb),
		Tags:       in.Tags,
		IsNSFW:     &in.IsNSFW,
		Visibility: visibility,
		Lifecycle:  work.LifecyclePublished,
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
		return work.Work{}, err
	}
	defer tx.Rollback(ctx)

	made, err := work.InsertWork(ctx, tx, a, in.OwnerID, firstDate(in.CreatedAt, parsed.CreatedAt))
	if err != nil {
		return work.Work{}, err
	}
	a.CreatedAt = made
	originalFileID, err := work.RecordOriginalFile(ctx, tx, work.OriginalFile{
		WorkID: a.ID, Number: 1, BlobID: stored.ID, MediaType: "application/octet-stream",
		Format: a.Format, Filename: in.Filename, Media: extractedMedia,
	})
	if err != nil {
		return work.Work{}, err
	}
	if err := block.Insert(ctx, tx, a.ID, blocks); err != nil {
		return work.Work{}, err
	}
	if err := insertShelf(ctx, tx, a.ID, ShelfSourceReadme, "", read.Shelf); err != nil {
		return work.Work{}, err
	}
	if err := replacePreservedData(ctx, tx, a.ID, parsed.Remainder); err != nil {
		return work.Work{}, err
	}
	if err := s.writeSummary(ctx, tx, a.ID); err != nil {
		return work.Work{}, err
	}
	if _, err := tx.Exec(ctx, `select record_initial_work_version($1, false)`, a.ID); err != nil {
		return work.Work{}, fmt.Errorf("record initial publication: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return work.Work{}, err
	}

	a.OriginalFileID = originalFileID
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
