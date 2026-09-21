package notify

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrOwnProfile = errors.New("a creator cannot follow themselves")

// CreatorFollow is how many accounts follow a creator, and whether the viewer is one of them.
type CreatorFollow struct {
	Followers int
	Following *bool
}

// FollowCreator asks to hear when the creator publishes a new work.
func (s *Service) FollowCreator(ctx context.Context, account, creator uuid.UUID) (CreatorFollow, error) {
	if account == creator {
		return CreatorFollow{}, ErrOwnProfile
	}
	if _, err := s.pool.Exec(ctx, `
		insert into creator_follows (account_id, creator_id, created_at)
		values ($1, $2, $3)
		on conflict do nothing
	`, account, creator, s.now()); err != nil {
		return CreatorFollow{}, fmt.Errorf("follow a creator: %w", err)
	}
	return s.CreatorFollowOf(ctx, &account, creator)
}

func (s *Service) StopFollowingCreator(ctx context.Context, account, creator uuid.UUID) (CreatorFollow, error) {
	if account == creator {
		return CreatorFollow{}, ErrOwnProfile
	}
	if _, err := s.pool.Exec(ctx, `
		delete from creator_follows where account_id = $1 and creator_id = $2
	`, account, creator); err != nil {
		return CreatorFollow{}, fmt.Errorf("stop following a creator: %w", err)
	}
	return s.CreatorFollowOf(ctx, &account, creator)
}

// CreatorFollowOf counts a creator's followers and, for a signed-in reader who is not the creator, says whether they follow.
func (s *Service) CreatorFollowOf(ctx context.Context, viewer *uuid.UUID, creator uuid.UUID) (CreatorFollow, error) {
	var follow CreatorFollow
	var following bool
	if err := s.pool.QueryRow(ctx, `
		select count(*), coalesce(bool_or(account_id = $2), false)
		  from creator_follows where creator_id = $1
	`, creator, viewer).Scan(&follow.Followers, &following); err != nil {
		return CreatorFollow{}, fmt.Errorf("read a creator's followers: %w", err)
	}
	if viewer != nil && *viewer != creator {
		follow.Following = &following
	}
	return follow, nil
}
