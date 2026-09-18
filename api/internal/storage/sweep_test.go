package storage

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
)

func TestSweeperRunsWithoutAnExternalCaller(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool := testdb.Connect(t)
	store, err := NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	sweeper := NewSweeper(pool, store)
	stored, err := store.Put(ctx, bytes.NewReader([]byte("background orphan")))
	if err != nil {
		t.Fatalf("put orphan: %v", err)
	}
	done := make(chan struct{})
	go func() {
		sweeper.runSweeper(ctx, time.Millisecond, nil)
		close(done)
	}()

	deadline := time.Now().Add(time.Second)
	for {
		var marked bool
		if err := pool.QueryRow(ctx,
			`select exists (select 1 from blob_sweep_marks where blob_id = $1)`, stored.ID,
		).Scan(&marked); err != nil {
			t.Fatalf("read sweep mark: %v", err)
		}
		if marked {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("background sweeper did not run")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	<-done
}
