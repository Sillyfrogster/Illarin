package account

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/google/uuid"
)

const recommendationLimit = 4

func (s *Service) AddRecommendation(ctx context.Context, owner api.Account, workID uuid.UUID) (PublicProfile, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PublicProfile{}, fmt.Errorf("begin recommendation: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := lockProfileForEdit(ctx, tx, owner.ID); err != nil {
		return PublicProfile{}, err
	}
	var eligible bool
	err = tx.QueryRow(ctx, `
		select exists (select 1 from works
		 where id = $2 and owner_id <> $1
		   and lifecycle = 'published' and visibility = 'listed'
		   and deleted_at is null and taken_down_at is null)
	`, owner.ID, workID).Scan(&eligible)
	if err != nil {
		return PublicProfile{}, fmt.Errorf("check the work to recommend: %w", err)
	}
	if !eligible {
		return PublicProfile{}, FieldError{Field: "workId", Message: "Choose another creator's published work that is in Browse."}
	}
	if _, err := tx.Exec(ctx, `
		delete from profile_recommendations recommendation using works work
		 where recommendation.user_id = $1 and recommendation.work_id = work.id
		   and (work.lifecycle <> 'published' or work.visibility <> 'listed'
		        or work.deleted_at is not null or work.taken_down_at is not null)
	`, owner.ID); err != nil {
		return PublicProfile{}, fmt.Errorf("clear unavailable recommendations: %w", err)
	}
	var count int
	var alreadyAdded bool
	err = tx.QueryRow(ctx, `
		select count(*), coalesce(bool_or(work_id = $2), false)
		  from profile_recommendations where user_id = $1
	`, owner.ID, workID).Scan(&count, &alreadyAdded)
	if err != nil {
		return PublicProfile{}, fmt.Errorf("count recommendations: %w", err)
	}
	if !alreadyAdded && count >= recommendationLimit {
		return PublicProfile{}, FieldError{Field: "workId", Message: "You can recommend up to four works. Remove one to add another."}
	}
	if _, err := tx.Exec(ctx, `
		insert into profile_recommendations (user_id, work_id) values ($1, $2)
		on conflict (user_id, work_id) do nothing
	`, owner.ID, workID); err != nil {
		return PublicProfile{}, fmt.Errorf("add recommendation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return PublicProfile{}, fmt.Errorf("commit recommendation: %w", err)
	}
	return s.PublicProfile(ctx, owner.Handle)
}

func (s *Service) RemoveRecommendation(ctx context.Context, owner api.Account, workID uuid.UUID) (PublicProfile, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PublicProfile{}, fmt.Errorf("begin recommendation removal: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := lockProfileForEdit(ctx, tx, owner.ID); err != nil {
		return PublicProfile{}, err
	}
	if _, err := tx.Exec(ctx, `delete from profile_recommendations where user_id = $1 and work_id = $2`, owner.ID, workID); err != nil {
		return PublicProfile{}, fmt.Errorf("remove recommendation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return PublicProfile{}, fmt.Errorf("commit recommendation removal: %w", err)
	}
	return s.PublicProfile(ctx, owner.Handle)
}
