package testdb

import (
	"testing"

	"github.com/google/uuid"
)

func TestAWatchGoesWithItsAccountOrItsAsset(t *testing.T) {
	t.Parallel()
	pool := Connect(t)
	ctx := t.Context()
	leaving, staying := uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `
		insert into users (id, username) values ($1, 'leaving.reader'), ($2, 'staying.reader')
	`, leaving, staying); err != nil {
		t.Fatalf("insert accounts: %v", err)
	}
	kept, removed := uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `
		insert into assets (id, kind, name, lifecycle)
		values ($1, 'character', 'Quiet Shelf', 'published'), ($2, 'character', 'Loud Shelf', 'published')
	`, kept, removed); err != nil {
		t.Fatalf("insert assets: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into asset_watches (account_id, asset_id, state)
		values ($1, $3, 'watching'), ($2, $3, 'watching'), ($2, $4, 'stopped')
	`, leaving, staying, kept, removed); err != nil {
		t.Fatalf("insert watches: %v", err)
	}

	if _, err := pool.Exec(ctx, `delete from users where id = $1`, leaving); err != nil {
		t.Fatalf("delete account: %v", err)
	}
	if _, err := pool.Exec(ctx, `delete from assets where id = $1`, removed); err != nil {
		t.Fatalf("delete asset: %v", err)
	}

	var left, stayed int
	if err := pool.QueryRow(ctx, `
		select count(*), count(*) filter (where account_id = $1 and asset_id = $2) from asset_watches
	`, staying, kept).Scan(&left, &stayed); err != nil {
		t.Fatalf("count watches: %v", err)
	}
	if left != 1 || stayed != 1 {
		t.Fatalf("%d watches left, want only the staying reader's watch on the kept asset", left)
	}
}

func TestAWatchIsEitherWatchingOrStopped(t *testing.T) {
	t.Parallel()
	pool := Connect(t)
	ctx := t.Context()
	account, asset := uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `insert into users (id, username) values ($1, 'muted.reader')`, account); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into assets (id, kind, name, lifecycle) values ($1, 'character', 'Quiet Shelf', 'published')
	`, asset); err != nil {
		t.Fatalf("insert asset: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into asset_watches (account_id, asset_id, state) values ($1, $2, 'muted')
	`, account, asset); err == nil {
		t.Fatal("a watch outside watching and stopped was accepted")
	}
}
