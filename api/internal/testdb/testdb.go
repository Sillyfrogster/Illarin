package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect gives the test a migrated database of its own, dropped when the test passes
func Connect(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return ConnectWith(t, nil)
}

// ConnectWith is Connect with the pool settings adjusted before the pool opens
func ConnectWith(t *testing.T, tune func(*postgres.Settings)) *pgxpool.Pool {
	t.Helper()

	address := os.Getenv("TEST_DATABASE_URL")
	if address == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()

	server, err := serverAt(ctx, address)
	if err != nil {
		t.Fatalf("prepare the test server: %v", err)
	}
	name := "illarin_test_" + randomSuffix()
	if err := server.create(ctx, name); err != nil {
		t.Fatalf("create the test database: %v", err)
	}
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("kept database %s for inspection", name)
			return
		}
		if err := server.drop(context.Background(), name); err != nil {
			t.Errorf("drop the test database: %v", err)
		}
	})

	settings := postgres.DefaultSettings(server.databaseURL(name))
	settings.MinConns = 0
	if tune != nil {
		tune(&settings)
	}
	pool, err := postgres.NewPool(ctx, settings)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func randomSuffix() string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(raw[:])
}
