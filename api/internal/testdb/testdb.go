package testdb

import (
	"context"
	"os"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

var immutableTables = []string{"download_events", "migration_legacy_counters"}

const seeded = `
	truncate publication_apps, publication_categories cascade;

	insert into publication_apps (id, slug, name, home_url, position)
	values ('9d3f1c00-0000-4000-8000-000000000001', 'illarin', 'Illarin',
	        'https://illarin.xyz', 0);

	insert into publication_categories (id, slug, label, position)
	values ('9d3f1c00-0000-4000-8000-000000000011', 'announcement', 'Announcement', 0),
	       ('9d3f1c00-0000-4000-8000-000000000012', 'release', 'Release', 1),
	       ('9d3f1c00-0000-4000-8000-000000000013', 'article', 'Article', 2);
`

func Connect(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return ConnectWith(t, nil)
}

func ConnectWith(t *testing.T, tune func(*postgres.Settings)) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	settings := postgres.DefaultSettings(url)
	if tune != nil {
		tune(&settings)
	}

	pool, err := postgres.NewPool(context.Background(), settings)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	for _, frozen := range immutableTables {
		if _, err = pool.Exec(context.Background(),
			"alter table "+frozen+" disable trigger user"); err != nil {
			t.Fatalf("allow %s test reset: %v", frozen, err)
		}
	}
	_, truncateErr := pool.Exec(context.Background(),
		`truncate migration_exceptions, download_events, ingest_operations,
		          assets, asset_revisions, asset_media,
		          migration_legacy_counters, migration_preserved_records,
		          blob_tombstones, blob_sweep_marks, blobs,
		          link_rate_limits, instance_access_tokens, instance_refresh_history,
		          link_authorizations, link_requests, linked_instances,
		          password_reset_tokens, oauth_states, oauth_identities, sessions,
		          email_verification_tokens,
		          retired_handles, users cascade`)
	var enableErr error
	for _, frozen := range immutableTables {
		if _, err := pool.Exec(context.Background(),
			"alter table "+frozen+" enable trigger user"); err != nil && enableErr == nil {
			enableErr = err
		}
	}
	if truncateErr != nil {
		t.Fatalf("reset test database: %v", truncateErr)
	}
	if enableErr != nil {
		t.Fatalf("restore table immutability: %v", enableErr)
	}
	if _, err := pool.Exec(context.Background(), seeded); err != nil {
		t.Fatalf("restore the seeded publication rows: %v", err)
	}

	return pool
}
