package testdb

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAnAccountsNotificationsGoWithIt(t *testing.T) {
	t.Parallel()
	pool := Connect(t)
	ctx := t.Context()
	accountID := uuid.New()
	if _, err := pool.Exec(ctx, `insert into users (id, username) values ($1, 'leaving.reader')`, accountID); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into notification_events (id, type, account_id, words)
		values ($1, 'asset_restored', $2, '{"assetName":"Quiet Shelf"}')
	`, uuid.New(), accountID); err != nil {
		t.Fatalf("insert waiting event: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into notifications (id, account_id, type, words, created_at)
		values ($1, $2, 'asset_restored', '{"assetName":"Quiet Shelf"}', $3)
	`, uuid.New(), accountID, time.Now()); err != nil {
		t.Fatalf("insert entry: %v", err)
	}

	if _, err := pool.Exec(ctx, `delete from users where id = $1`, accountID); err != nil {
		t.Fatalf("delete account: %v", err)
	}

	var events, entries int
	if err := pool.QueryRow(ctx, `
		select (select count(*) from notification_events where account_id = $1),
		       (select count(*) from notifications where account_id = $1)
	`, accountID).Scan(&events, &entries); err != nil {
		t.Fatalf("count what the account left behind: %v", err)
	}
	if events != 0 || entries != 0 {
		t.Fatalf("deleting the account left %d waiting events and %d entries", events, entries)
	}
}
