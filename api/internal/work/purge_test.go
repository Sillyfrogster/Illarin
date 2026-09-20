package work_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
	"github.com/Sillyfrogster/Illarin/api/internal/upload"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type failingDeleteStore struct {
	storage.Store
}

func (failingDeleteStore) Delete(context.Context, uuid.UUID) error {
	return errors.New("storage unavailable")
}

func TestPurgeCommitsTheTombstoneAndBrokenReferencesBeforeDeletingBytes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.Connect(t)
	store, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	service := work.NewService(pool, apitest.RegistryWith(t, apitest.OpaqueModule{}), failingDeleteStore{Store: store})
	ownerID := uuid.New()
	actorID := uuid.New()
	if _, err := pool.Exec(ctx, `
		insert into users (id, username) values ($1, 'durable.owner'), ($2, 'durable.actor')
	`, ownerID, actorID); err != nil {
		t.Fatalf("insert accounts: %v", err)
	}
	created, err := apitest.Uploads(service).Create(ctx, upload.CreateInput{
		OwnerID: ownerID, Type: "theme", Filename: "durable.lumitheme",
		File: bytes.NewReader([]byte("durably purged bytes")), Name: "Durable purge",
	})
	if err != nil {
		t.Fatalf("create work: %v", err)
	}
	var digestBytes []byte
	if err := pool.QueryRow(ctx, `
		select blob.sha256 from work_original_files revision
		join blobs blob on blob.id = revision.blob_id
		where revision.work_id = $1
	`, created.ID).Scan(&digestBytes); err != nil {
		t.Fatalf("read digest: %v", err)
	}
	var digest [32]byte
	copy(digest[:], digestBytes)

	if err := storage.NewSweeper(pool, failingDeleteStore{Store: store}).
		Purge(ctx, digest, "legal_order", actorID); err == nil {
		t.Fatal("purge succeeded despite the storage failure")
	}
	var tombstoned bool
	if err := pool.QueryRow(ctx,
		`select exists (select 1 from blob_tombstones where sha256 = $1)`, digest[:],
	).Scan(&tombstoned); err != nil {
		t.Fatalf("read tombstone: %v", err)
	}
	if !tombstoned {
		t.Fatal("purge failure rolled back the tombstone")
	}
	var references int
	if err := pool.QueryRow(ctx,
		`select count(*) from work_original_files where work_id = $1 and blob_id is not null`, created.ID,
	).Scan(&references); err != nil {
		t.Fatalf("count references: %v", err)
	}
	if references != 0 {
		t.Fatalf("purge failure left %d live references", references)
	}

	if _, err := storage.NewSweeper(pool, store).Sweep(ctx); err != nil {
		t.Fatalf("resume purge from the sweeper: %v", err)
	}
	var blobID uuid.UUID
	if err := pool.QueryRow(ctx,
		`select id from blobs where sha256 = $1`, digest[:],
	).Scan(&blobID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("interrupted purge blob lookup = %v, want no row", err)
	}
}

func TestPurgeDeletesSharedBytesBreaksReferencesAndRecordsATombstone(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.Connect(t)
	store, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	now := time.Date(2026, 8, 14, 15, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	service := work.NewServiceWithClock(pool, apitest.RegistryWith(t, apitest.OpaqueModule{}), store, clock)
	ownerID := uuid.New()
	actorID := uuid.New()
	if _, err := pool.Exec(ctx, `
		insert into users (id, username) values ($1, 'purge.owner'), ($2, 'purge.actor')
	`, ownerID, actorID); err != nil {
		t.Fatalf("insert accounts: %v", err)
	}

	var blobID uuid.UUID
	var digest []byte
	for _, name := range []string{"First copy", "Second copy"} {
		created, err := apitest.Uploads(service).Create(ctx, upload.CreateInput{
			OwnerID: ownerID, Type: "theme", Filename: name + ".lumitheme",
			File: bytes.NewReader([]byte("shared forbidden bytes")), Name: name,
		})
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if err := pool.QueryRow(ctx, `
			select blob.id, blob.sha256
			  from work_original_files revision
			  join blobs blob on blob.id = revision.blob_id
			 where revision.work_id = $1
		`, created.ID).Scan(&blobID, &digest); err != nil {
			t.Fatalf("read %s blob: %v", name, err)
		}
	}
	var contentDigest [32]byte
	copy(contentDigest[:], digest)

	if err := storage.NewSweeperWithClock(pool, store, clock).Purge(ctx, contentDigest, "legal_order", actorID); err != nil {
		t.Fatalf("purge: %v", err)
	}
	if _, err := store.Open(ctx, blobID); !errors.Is(err, storage.ErrBlobNotFound) {
		t.Fatalf("purged blob error = %v, want ErrBlobNotFound", err)
	}
	var liveReferences int
	if err := pool.QueryRow(ctx,
		`select count(*) from work_original_files where blob_id is not null`,
	).Scan(&liveReferences); err != nil {
		t.Fatalf("count revision references: %v", err)
	}
	if liveReferences != 0 {
		t.Fatalf("live revision references = %d, want 0", liveReferences)
	}
	var reason string
	var purgedAt time.Time
	var actor pgtype.UUID
	if err := pool.QueryRow(ctx, `
		select reason_code, purged_at, actor_id from blob_tombstones where sha256 = $1
	`, digest).Scan(&reason, &purgedAt, &actor); err != nil {
		t.Fatalf("read tombstone: %v", err)
	}
	if reason != "legal_order" || !purgedAt.Equal(now) || !actor.Valid || uuid.UUID(actor.Bytes) != actorID {
		t.Fatalf("tombstone = reason %q, at %v, actor %v", reason, purgedAt, actor)
	}
}

func TestPurgedBytesCannotBeStoredAgain(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.Connect(t)
	store, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	service := work.NewService(pool, apitest.RegistryWith(t, apitest.OpaqueModule{}), store)
	ownerID := uuid.New()
	actorID := uuid.New()
	if _, err := pool.Exec(ctx, `
		insert into users (id, username) values ($1, 'upload.owner'), ($2, 'upload.actor')
	`, ownerID, actorID); err != nil {
		t.Fatalf("insert accounts: %v", err)
	}
	stored, err := store.Put(ctx, bytes.NewReader([]byte("bytes that stay gone")))
	if err != nil {
		t.Fatalf("put blob: %v", err)
	}
	if err := storage.NewSweeper(pool, store).Purge(ctx, stored.Digest, "illegal_content", actorID); err != nil {
		t.Fatalf("purge: %v", err)
	}
	_, err = apitest.Uploads(service).AcceptUpload(ctx, upload.UploadInput{
		OwnerID: ownerID, Filename: "same.bin", File: bytes.NewReader([]byte("bytes that stay gone")),
	})
	if !errors.Is(err, storage.ErrTombstoned) {
		t.Fatalf("re-upload error = %v, want ErrTombstoned", err)
	}
}
