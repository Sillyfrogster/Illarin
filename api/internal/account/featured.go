package account

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/google/uuid"
)

const featuredLimit = 4

// SaveFeatured records the works the owner shows first on their profile, in the order given.
func (s *Service) SaveFeatured(ctx context.Context, owner api.Account, workIDs []uuid.UUID) (PublicProfile, error) {
	if len(workIDs) > featuredLimit {
		return PublicProfile{}, FieldError{
			Field:   "workIds",
			Message: fmt.Sprintf("A profile can feature up to %d works.", featuredLimit),
		}
	}
	seen := make(map[uuid.UUID]bool, len(workIDs))
	for _, workID := range workIDs {
		if seen[workID] {
			return PublicProfile{}, FieldError{Field: "workIds", Message: "Each work can be featured once."}
		}
		seen[workID] = true
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PublicProfile{}, fmt.Errorf("begin featured save: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := lockProfileForEdit(ctx, tx, owner.ID); err != nil {
		return PublicProfile{}, err
	}
	var listed int
	err = tx.QueryRow(ctx, `
		select count(*) from works
		 where id = any($2) and owner_id = $1
		   and lifecycle = 'published' and visibility = 'listed'
		   and deleted_at is null and taken_down_at is null
	`, owner.ID, workIDs).Scan(&listed)
	if err != nil {
		return PublicProfile{}, fmt.Errorf("check the works to feature: %w", err)
	}
	if listed != len(workIDs) {
		return PublicProfile{}, FieldError{Field: "workIds", Message: "Feature only your own published works that are in Browse."}
	}
	if _, err := tx.Exec(ctx, `delete from profile_featured_works where user_id = $1`, owner.ID); err != nil {
		return PublicProfile{}, fmt.Errorf("clear featured works: %w", err)
	}
	for position, workID := range workIDs {
		if _, err := tx.Exec(ctx, `
			insert into profile_featured_works (user_id, position, work_id) values ($1, $2, $3)
		`, owner.ID, position, workID); err != nil {
			return PublicProfile{}, fmt.Errorf("save a featured work: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return PublicProfile{}, fmt.Errorf("commit featured save: %w", err)
	}
	return s.PublicProfile(ctx, owner.Handle)
}
