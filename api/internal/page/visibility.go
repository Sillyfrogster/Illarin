package page

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidVisibility = errors.New("invalid visibility state")
	ErrAlreadyPublished  = errors.New("the work is already published")
)

func (s *Service) SetVisibility(
	ctx context.Context,
	ownerID uuid.UUID,
	id uuid.UUID,
	visibility work.Visibility,
) error {
	if !visibility.Valid() {
		return ErrInvalidVisibility
	}

	queries := db.New(s.pool)
	changed, err := queries.SetWorkVisibility(ctx, db.SetWorkVisibilityParams{
		ID:         uuidToPgtype(id),
		OwnerID:    uuidToPgtype(ownerID),
		Visibility: string(visibility),
	})
	if err != nil {
		return fmt.Errorf("set work visibility: %w", err)
	}
	if changed == 1 {
		return nil
	}

	state, err := queries.WorkStateForOwner(ctx, db.WorkStateForOwnerParams{
		ID:      uuidToPgtype(id),
		OwnerID: uuidToPgtype(ownerID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return work.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("check work visibility: %w", err)
	}
	if state.TakenDownAt.Valid {
		return work.ErrWorkFrozen
	}
	if work.Lifecycle(state.Lifecycle) == work.LifecycleDraft {
		return work.ErrWorkIsDraft
	}
	return work.ErrNotFound
}

// Publish makes a draft public once it clears the publish floor, and records its first version
func (s *Service) Publish(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
	candidate *work.Candidate,
) ([]work.ReadinessItem, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := candidate.Lock(ctx, tx, ownerID, workID); err != nil {
		return nil, err
	}

	var workType, name, lifecycle, visibility, creator, creatorName string
	var isNSFW *bool
	err = tx.QueryRow(ctx, `
		select owned.type, owned.name, owned.is_nsfw, owned.lifecycle, owned.visibility,
		       owner.username, coalesce(profile.display_name, '')
		  from works owned
		  join users owner on owner.id = owned.owner_id
		  left join public_profiles profile on profile.user_id = owner.id
		 where owned.id = $1
	`, workID).Scan(&workType, &name, &isNSFW, &lifecycle, &visibility, &creator, &creatorName)
	if err != nil {
		return nil, fmt.Errorf("read work to publish: %w", err)
	}

	if work.Lifecycle(lifecycle) != work.LifecycleDraft {
		return nil, ErrAlreadyPublished
	}

	blocks, err := block.Read(ctx, tx, workID)
	if err != nil {
		return nil, err
	}
	items, err := s.works.CandidateReadiness(ctx, tx, workID, workType, name, isNSFW, blocks)
	if err != nil {
		return nil, err
	}
	if !work.Ready(items) {
		return items, work.ErrPublishFloor
	}

	if _, err := tx.Exec(ctx, `
		update works set lifecycle = 'published', updated_at = now()
		 where id = $1
	`, workID); err != nil {
		return nil, fmt.Errorf("publish work: %w", err)
	}
	if _, err := tx.Exec(ctx, `select record_initial_work_version($1, false)`, workID); err != nil {
		return nil, fmt.Errorf("record initial publication: %w", err)
	}
	if work.Visibility(visibility) == work.VisibilityListed {
		if err := notify.Record(ctx, tx, notify.Event{
			Type: notify.WorkPublished, Work: &workID,
			Words: notify.Words{WorkName: name, Creator: creator, CreatorName: creatorName},
		}); err != nil {
			return nil, err
		}
	}
	if err := candidate.Commit(ctx, tx, workID); err != nil {
		return nil, err
	}
	return items, nil
}
