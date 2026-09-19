package notify

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// FollowState says whether an account hears about a work's updates, and why.
type FollowState string

const (
	NotFollowing       FollowState = "none"
	Following          FollowState = "following"
	FollowingInstalled FollowState = "installed"
	StoppedFollowing   FollowState = "stopped"
)

var (
	ErrNothingToFollow = errors.New("no published work has that id")
	ErrOwnWork         = errors.New("a work's owner cannot follow it")
)

// Follow is an account's follow on one work and the connected apps that report having it installed.
type Follow struct {
	State       FollowState
	InstalledOn []string
}

// StartFollowing follows a published work the account does not own.
func (s *Service) StartFollowing(ctx context.Context, account, work uuid.UUID) (Follow, error) {
	return s.setFollow(ctx, account, work, Following)
}

// StopFollowing stops a follow and keeps it stopped through later installs until the account follows again.
func (s *Service) StopFollowing(ctx context.Context, account, work uuid.UUID) (Follow, error) {
	return s.setFollow(ctx, account, work, StoppedFollowing)
}

func (s *Service) setFollow(ctx context.Context, account, work uuid.UUID, state FollowState) (Follow, error) {
	var isOwner bool
	err := s.pool.QueryRow(ctx, `
		with target as (
			select id, owner_id is not distinct from $1 as is_owner
			  from works
			 where id = $2 and lifecycle = 'published' and deleted_at is null and withheld_at is null
		), written as (
			insert into work_follows (account_id, work_id, state, set_at)
			select $1, id, $3, $4 from target where not is_owner
			on conflict (account_id, work_id) do update set state = excluded.state, set_at = excluded.set_at
		)
		select is_owner from target
	`, account, work, state, s.now()).Scan(&isOwner)
	if errors.Is(err, pgx.ErrNoRows) {
		return Follow{}, ErrNothingToFollow
	}
	if err != nil {
		return Follow{}, fmt.Errorf("set a follow: %w", err)
	}
	if isOwner {
		return Follow{}, ErrOwnWork
	}
	return s.FollowOf(ctx, account, work)
}

// FollowOf reads an account's follow on a work, where an install on one of its connected apps counts as following.
func (s *Service) FollowOf(ctx context.Context, account, work uuid.UUID) (Follow, error) {
	var chosen *FollowState
	follow := Follow{State: NotFollowing}
	if err := s.pool.QueryRow(ctx, `
		select (select state from work_follows where account_id = $1 and work_id = $2),
		       array(select app.name
		               from app_library_entries entry
		               join connected_apps app on app.id = entry.connected_app_id
		              where entry.work_id = $2 and app.user_id = $1 and app.revoked_at is null
		              order by app.name, app.id)
	`, account, work).Scan(&chosen, &follow.InstalledOn); err != nil {
		return Follow{}, fmt.Errorf("read a follow: %w", err)
	}
	switch {
	case chosen != nil:
		follow.State = *chosen
	case len(follow.InstalledOn) > 0:
		follow.State = FollowingInstalled
	}
	return follow, nil
}
