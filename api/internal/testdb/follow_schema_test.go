package testdb

import (
	"testing"

	"github.com/google/uuid"
)

func TestAFollowGoesWithItsAccountOrItsWork(t *testing.T) {
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
		insert into works (id, type, name, lifecycle)
		values ($1, 'character', 'Quiet Shelf', 'published'), ($2, 'character', 'Loud Shelf', 'published')
	`, kept, removed); err != nil {
		t.Fatalf("insert works: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into work_follows (account_id, work_id, state)
		values ($1, $3, 'watching'), ($2, $3, 'watching'), ($2, $4, 'stopped')
	`, leaving, staying, kept, removed); err != nil {
		t.Fatalf("insert follows: %v", err)
	}

	if _, err := pool.Exec(ctx, `delete from users where id = $1`, leaving); err != nil {
		t.Fatalf("delete account: %v", err)
	}
	if _, err := pool.Exec(ctx, `delete from works where id = $1`, removed); err != nil {
		t.Fatalf("delete work: %v", err)
	}

	var left, stayed int
	if err := pool.QueryRow(ctx, `
		select count(*), count(*) filter (where account_id = $1 and work_id = $2) from work_follows
	`, staying, kept).Scan(&left, &stayed); err != nil {
		t.Fatalf("count follows: %v", err)
	}
	if left != 1 || stayed != 1 {
		t.Fatalf("%d follows left, want only the staying reader's follow on the kept work", left)
	}
}

func TestAFollowIsEitherFollowingOrStopped(t *testing.T) {
	t.Parallel()
	pool := Connect(t)
	ctx := t.Context()
	account, work := uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `insert into users (id, username) values ($1, 'muted.reader')`, account); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into works (id, type, name, lifecycle) values ($1, 'character', 'Quiet Shelf', 'published')
	`, work); err != nil {
		t.Fatalf("insert work: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into work_follows (account_id, work_id, state) values ($1, $2, 'muted')
	`, account, work); err == nil {
		t.Fatal("a follow outside following and stopped was accepted")
	}
}
