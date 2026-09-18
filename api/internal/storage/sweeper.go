package storage

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Sweeper deletes what nothing refers to any more and purges the blobs a decision removes
type Sweeper struct {
	pool  *pgxpool.Pool
	store Store
	now   func() time.Time
}

func NewSweeper(pool *pgxpool.Pool, store Store) *Sweeper {
	return NewSweeperWithClock(pool, store, time.Now)
}

// NewSweeperWithClock reads the time from the given clock rather than the wall clock
func NewSweeperWithClock(pool *pgxpool.Pool, store Store, now func() time.Time) *Sweeper {
	return &Sweeper{pool: pool, store: store, now: now}
}
