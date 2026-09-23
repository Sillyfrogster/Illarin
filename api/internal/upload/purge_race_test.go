package upload

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

func TestPurgeAndUploadFinalizationSerializeOnTheDigest(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool := testdb.Connect(t)
	store, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	service := NewService(pool, work.NewService(pool, registryWithModule(t, opaqueTestModule{}), store))
	ownerID := uuid.New()
	actorID := uuid.New()
	if _, err := pool.Exec(ctx, `
		insert into users (id, username) values ($1, 'race.owner'), ($2, 'race.actor')
	`, ownerID, actorID); err != nil {
		t.Fatalf("insert accounts: %v", err)
	}
	operation, err := service.AcceptUpload(ctx, UploadInput{
		OwnerID: ownerID, Filename: "race.lumitheme", File: bytes.NewReader([]byte("racing bytes")),
	})
	if err != nil {
		t.Fatalf("accept upload: %v", err)
	}
	job, ok, err := service.leaseNextUpload(ctx)
	if err != nil || !ok || job.ID != operation.ID {
		t.Fatalf("lease upload = %+v, %v, %v", job, ok, err)
	}
	var digestBytes []byte
	if err := pool.QueryRow(ctx, `select sha256 from blobs where id = $1`, job.BlobID).Scan(&digestBytes); err != nil {
		t.Fatalf("read digest: %v", err)
	}
	var digest [32]byte
	copy(digest[:], digestBytes)
	prepared := preparedUpload{
		Type: "character", Format: "unknown", Name: "Race", Tags: []string{},
		Visibility: work.VisibilityListed, MediaType: "application/octet-stream",
	}
	prepared.Blocks, err = block.Place(prepared.Type, nil)
	if err != nil {
		t.Fatalf("place fixture: %v", err)
	}

	start := make(chan struct{})
	finalized := make(chan error, 1)
	purged := make(chan error, 1)
	go func() {
		<-start
		finalized <- service.finalizeUpload(ctx, job, prepared)
	}()
	go func() {
		<-start
		purged <- cleanup(service.works).Purge(ctx, digest, "legal_order", actorID)
	}()
	close(start)
	if err := <-purged; err != nil {
		t.Fatalf("purge: %v", err)
	}
	if err := <-finalized; err != nil && !errors.Is(err, errUploadLeaseLost) {
		t.Fatalf("finalize: %v", err)
	}

	var references int
	if err := pool.QueryRow(ctx, `
		select (select count(*) from work_original_files where blob_id is not null)
		     + (select count(*) from upload_operations where blob_id is not null)
	`).Scan(&references); err != nil {
		t.Fatalf("count surviving references: %v", err)
	}
	if references != 0 {
		t.Fatalf("surviving references = %d, want 0", references)
	}
	var tombstones int
	if err := pool.QueryRow(ctx,
		`select count(*) from blob_tombstones where sha256 = $1`, digest[:],
	).Scan(&tombstones); err != nil {
		t.Fatalf("count tombstones: %v", err)
	}
	if tombstones != 1 {
		t.Fatalf("tombstones = %d, want 1", tombstones)
	}
}

// cleanup cleans up blobs the way the server's background cleanup does
func cleanup(works *work.Service) *storage.Cleanup {
	return storage.NewCleanup(works.Pool(), works.Store())
}
