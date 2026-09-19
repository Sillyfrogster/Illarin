package download

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Source struct {
	InternalRedirect string
	MediaType        string
	Inline           bool
	Event            Event
}

// Source hands over the main file of a work
func (s *Service) Source(
	ctx context.Context,
	workID uuid.UUID,
	viewerID *uuid.UUID,
) (Source, error) {
	tx, err := s.works.BeginReadSnapshot(ctx)
	if err != nil {
		return Source{}, err
	}
	defer tx.Rollback(ctx)
	location, err := work.LocateOriginalFile(ctx, tx, workID, viewerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Source{}, work.ErrNotFound
		}
		return Source{}, fmt.Errorf("find the original file: %w", err)
	}
	apps, err := private.Apps(ctx, tx, workID)
	if err != nil {
		return Source{}, err
	}
	blocks, err := block.Read(ctx, tx, workID)
	if err != nil {
		return Source{}, err
	}
	if err := private.ApplyPublishedPolicy(ctx, tx, workID, blocks); err != nil {
		return Source{}, err
	}
	if (len(apps) > 0 || private.HasPromptFragments(blocks)) && (viewerID == nil || location.OwnerID == nil || *viewerID != *location.OwnerID) {
		return Source{}, ErrLinkedInstallOnly
	}
	redirect, err := s.store.InternalRedirect(ctx, location.BlobID)
	if err != nil {
		return Source{}, fmt.Errorf("resolve stored file: %w", err)
	}
	originalFileID := location.OriginalFileID
	return Source{
		InternalRedirect: redirect, MediaType: location.MediaType,
		Inline: format.IsInlineMediaType(location.MediaType),
		Event: newEvent(
			location.WorkID, &originalFileID, format.Raw,
			location.OwnerID, viewerID,
		),
	}, nil
}

func (s *Service) SourceForSend(ctx context.Context, workID uuid.UUID) (Source, error) {
	download, err := s.Source(ctx, workID, nil)
	if err != nil {
		return Source{}, err
	}
	download.Event.AuthorizationClass = AuthorizationLinkedInstance
	return download, nil
}
