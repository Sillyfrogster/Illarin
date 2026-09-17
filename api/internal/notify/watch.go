package notify

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// WatchState says whether an account hears about an asset's updates, and why.
type WatchState string

const (
	NotWatching       WatchState = "none"
	Watching          WatchState = "watching"
	WatchingInstalled WatchState = "installed"
	StoppedWatching   WatchState = "stopped"
)

var (
	ErrNothingToWatch = errors.New("no published asset has that id")
	ErrOwnAsset       = errors.New("an asset's owner cannot watch it")
)

// Watch is an account's watch on one asset and the linked instances that report having it installed.
type Watch struct {
	State       WatchState
	InstalledOn []string
}

// StartWatching watches a published asset the account does not own.
func (s *Service) StartWatching(ctx context.Context, account, asset uuid.UUID) (Watch, error) {
	return s.setWatch(ctx, account, asset, Watching)
}

// StopWatching stops a watch and keeps it stopped through later installs until the account watches again.
func (s *Service) StopWatching(ctx context.Context, account, asset uuid.UUID) (Watch, error) {
	return s.setWatch(ctx, account, asset, StoppedWatching)
}

func (s *Service) setWatch(ctx context.Context, account, asset uuid.UUID, state WatchState) (Watch, error) {
	var isOwner bool
	err := s.pool.QueryRow(ctx, `
		with target as (
			select id, owner_id is not distinct from $1 as is_owner
			  from assets
			 where id = $2 and lifecycle = 'published' and deleted_at is null and withheld_at is null
		), written as (
			insert into asset_watches (account_id, asset_id, state, set_at)
			select $1, id, $3, $4 from target where not is_owner
			on conflict (account_id, asset_id) do update set state = excluded.state, set_at = excluded.set_at
		)
		select is_owner from target
	`, account, asset, state, s.now()).Scan(&isOwner)
	if errors.Is(err, pgx.ErrNoRows) {
		return Watch{}, ErrNothingToWatch
	}
	if err != nil {
		return Watch{}, fmt.Errorf("set a watch: %w", err)
	}
	if isOwner {
		return Watch{}, ErrOwnAsset
	}
	return s.WatchOf(ctx, account, asset)
}

// WatchOf reads an account's watch on an asset, where an install on one of its linked instances counts as watching.
func (s *Service) WatchOf(ctx context.Context, account, asset uuid.UUID) (Watch, error) {
	var chosen *WatchState
	watch := Watch{State: NotWatching}
	if err := s.pool.QueryRow(ctx, `
		select (select state from asset_watches where account_id = $1 and asset_id = $2),
		       array(select instance.instance_name
		               from instance_library_entries entry
		               join linked_instances instance on instance.id = entry.instance_id
		              where entry.asset_id = $2 and instance.user_id = $1 and instance.revoked_at is null
		              order by instance.instance_name, instance.id)
	`, account, asset).Scan(&chosen, &watch.InstalledOn); err != nil {
		return Watch{}, fmt.Errorf("read a watch: %w", err)
	}
	switch {
	case chosen != nil:
		watch.State = *chosen
	case len(watch.InstalledOn) > 0:
		watch.State = WatchingInstalled
	}
	return watch, nil
}
