package work_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
	"github.com/Sillyfrogster/Illarin/api/internal/upload"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

func TestSweepRecordsACanonicalFileLeftBeforeItsBlobTransactionCommitted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.Connect(t)
	root := t.TempDir()
	store, err := storage.NewStore(pool, root)
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	content := []byte("installed before a crashed transaction")
	digest := sha256.Sum256(content)
	encoded := hex.EncodeToString(digest[:])
	path := filepath.Join(root, "blobs", encoded[:2], encoded)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create orphan directory: %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write canonical orphan: %v", err)
	}

	result, err := storage.NewSweeper(pool, store).Sweep(ctx)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if result.Marked != 1 {
		t.Fatalf("sweep = %+v, want the recovered orphan marked", result)
	}
	var recorded bool
	if err := pool.QueryRow(ctx,
		`select exists (select 1 from blobs where sha256 = $1)`, digest[:],
	).Scan(&recorded); err != nil {
		t.Fatalf("find recovered orphan: %v", err)
	}
	if !recorded {
		t.Fatal("canonical orphan was not recorded for sweeping")
	}
}

func TestSweepRemovesATombstonedCanonicalOrphanInsteadOfRecordingIt(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.Connect(t)
	root := t.TempDir()
	store, err := storage.NewStore(pool, root)
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	stored, err := store.Put(ctx, bytes.NewReader([]byte("tombstoned orphan")))
	if err != nil {
		t.Fatalf("put blob: %v", err)
	}
	actorID := uuid.New()
	if _, err := pool.Exec(ctx,
		`insert into users (id, username) values ($1, 'orphan.actor')`, actorID,
	); err != nil {
		t.Fatalf("insert purge actor: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into blob_tombstones (sha256, reason_code, purged_at, actor_id)
		values ($1, 'legal_order', now(), $2)
	`, stored.Digest[:], actorID); err != nil {
		t.Fatalf("insert tombstone: %v", err)
	}
	if _, err := pool.Exec(ctx, `delete from blobs where id = $1`, stored.ID); err != nil {
		t.Fatalf("leave canonical orphan: %v", err)
	}

	result, err := storage.NewSweeper(pool, store).Sweep(ctx)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if result.Marked != 0 {
		t.Fatalf("sweep = %+v, want no tombstoned orphan recorded", result)
	}
	encoded := hex.EncodeToString(stored.Digest[:])
	if _, err := os.Stat(filepath.Join(root, "blobs", encoded[:2], encoded)); !os.IsNotExist(err) {
		t.Fatalf("tombstoned orphan stat error = %v, want not found", err)
	}
}

func TestSweepCommitsExpiredReferenceRemovalBeforeDeletingBytes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.Connect(t)
	store, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	service := work.NewServiceWithClock(pool, apitest.RegistryWith(t, apitest.OpaqueModule{}), store, clock)
	ownerID := uuid.New()
	if _, err := pool.Exec(ctx, `insert into users (id, username) values ($1, 'sweep.durable')`, ownerID); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	created, err := apitest.Uploads(service).Create(ctx, upload.CreateInput{
		OwnerID: ownerID, Type: "theme", Filename: "expired.lumitheme",
		File: bytes.NewReader([]byte("expired but durable")), Name: "Expired",
	})
	if err != nil {
		t.Fatalf("create work: %v", err)
	}
	if err := apitest.Pages(service).Delete(ctx, ownerID, created.ID); err != nil {
		t.Fatalf("delete work: %v", err)
	}
	now = now.Add(page.RecoveryWindow + time.Second)
	if _, err := storage.NewSweeperWithClock(pool, store, clock).Sweep(ctx); err != nil {
		t.Fatalf("mark expired work: %v", err)
	}

	now = now.Add(storage.SweepDelay + time.Second)
	if _, err := storage.NewSweeperWithClock(pool, failingDeleteStore{Store: store}, clock).
		Sweep(ctx); err == nil {
		t.Fatal("sweep succeeded despite the storage failure")
	}
	var references int
	if err := pool.QueryRow(ctx,
		`select count(*) from work_original_files where work_id = $1 and blob_id is not null`, created.ID,
	).Scan(&references); err != nil {
		t.Fatalf("count expired references: %v", err)
	}
	if references != 0 {
		t.Fatalf("sweep failure left %d expired references", references)
	}
}

func TestSweepMarksThenDeletesOnlyBlobsWithoutLiveOrRecoverableReferences(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.Connect(t)
	store, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	service := work.NewServiceWithClock(pool, apitest.RegistryWith(t, apitest.OpaqueModule{}), store, clock)
	ownerID := uuid.New()
	if _, err := pool.Exec(ctx, `insert into users (id, username) values ($1, 'sweep.owner')`, ownerID); err != nil {
		t.Fatalf("insert owner: %v", err)
	}

	recoverable, err := apitest.Uploads(service).Create(ctx, upload.CreateInput{
		OwnerID: ownerID, Type: "theme", Filename: "recoverable.lumitheme",
		File: bytes.NewReader([]byte("recoverable")), Name: "Recoverable",
	})
	if err != nil {
		t.Fatalf("create recoverable work: %v", err)
	}
	if err := apitest.Pages(service).Delete(ctx, ownerID, recoverable.ID); err != nil {
		t.Fatalf("delete recoverable work: %v", err)
	}
	var recoverableBlob uuid.UUID
	if err := pool.QueryRow(ctx,
		`select blob_id from work_original_files where work_id = $1`, recoverable.ID,
	).Scan(&recoverableBlob); err != nil {
		t.Fatalf("read recoverable blob: %v", err)
	}

	orphan, err := store.Put(ctx, bytes.NewReader([]byte("worker crashed before recording a reference")))
	if err != nil {
		t.Fatalf("put orphan: %v", err)
	}
	rejected, err := apitest.Uploads(service).AcceptIngest(ctx, upload.IngestInput{
		OwnerID: ownerID, Filename: "rejected.bin", File: bytes.NewReader([]byte("rejected")),
	})
	if err != nil {
		t.Fatalf("accept rejected ingest: %v", err)
	}
	var rejectedBlob uuid.UUID
	if err := pool.QueryRow(ctx,
		`select blob_id from ingest_operations where id = $1`, rejected.ID,
	).Scan(&rejectedBlob); err != nil {
		t.Fatalf("read rejected blob: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		update ingest_operations
		   set status = 'failed', failure_reason = 'malformed_input', blob_id = null
		 where id = $1
	`, rejected.ID); err != nil {
		t.Fatalf("reject ingest: %v", err)
	}
	first, err := storage.NewSweeperWithClock(pool, store, clock).Sweep(ctx)
	if err != nil {
		t.Fatalf("first sweep: %v", err)
	}
	if first.Marked != 2 || first.Deleted != 0 {
		t.Fatalf("first sweep = %+v, want two marked and none deleted", first)
	}
	for _, id := range []uuid.UUID{recoverableBlob, orphan.ID, rejectedBlob} {
		opened, err := store.Open(ctx, id)
		if err != nil {
			t.Fatalf("blob %s disappeared on the mark pass: %v", id, err)
		}
		opened.Close()
	}

	now = now.Add(storage.SweepDelay + time.Second)
	second, err := storage.NewSweeperWithClock(pool, store, clock).Sweep(ctx)
	if err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if second.Deleted != 2 {
		t.Fatalf("second sweep = %+v, want two deleted", second)
	}
	if _, err := store.Open(ctx, recoverableBlob); err != nil {
		t.Fatalf("recoverable blob was swept: %v", err)
	}
	for _, id := range []uuid.UUID{orphan.ID, rejectedBlob} {
		if _, err := store.Open(ctx, id); !errors.Is(err, storage.ErrBlobNotFound) {
			t.Fatalf("collected blob %s error = %v, want ErrBlobNotFound", id, err)
		}
	}
}

func TestSweepRechecksReferencesImmediatelyBeforeDeleting(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.Connect(t)
	store, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	ownerID := uuid.New()
	if _, err := pool.Exec(ctx, `insert into users (id, username) values ($1, 'race.owner')`, ownerID); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	stored, err := store.Put(ctx, bytes.NewReader([]byte("converged bytes")))
	if err != nil {
		t.Fatalf("put blob: %v", err)
	}
	if _, err := storage.NewSweeperWithClock(pool, store, clock).Sweep(ctx); err != nil {
		t.Fatalf("mark sweep: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into ingest_operations (id, owner_id, blob_id, filename, status)
		values ($1, $2, $3, 'converged.bin', 'pending')
	`, uuid.New(), ownerID, stored.ID); err != nil {
		t.Fatalf("add concurrent reference: %v", err)
	}

	now = now.Add(storage.SweepDelay + time.Second)
	result, err := storage.NewSweeperWithClock(pool, store, clock).Sweep(ctx)
	if err != nil {
		t.Fatalf("delete sweep: %v", err)
	}
	if result.Deleted != 0 {
		t.Fatalf("delete sweep = %+v, want the new reference to cancel deletion", result)
	}
	opened, err := store.Open(ctx, stored.ID)
	if err != nil {
		t.Fatalf("newly referenced blob was swept: %v", err)
	}
	opened.Close()
}

func TestConcurrentConvergenceClearsAnOldSweepMark(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.Connect(t)
	store, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	stored, err := store.Put(ctx, bytes.NewReader([]byte("converging upload")))
	if err != nil {
		t.Fatalf("put orphan: %v", err)
	}
	if _, err := storage.NewSweeperWithClock(pool, store, clock).Sweep(ctx); err != nil {
		t.Fatalf("mark orphan: %v", err)
	}
	now = now.Add(storage.SweepDelay + time.Second)

	converged, err := store.Put(ctx, bytes.NewReader([]byte("converging upload")))
	if err != nil {
		t.Fatalf("converge upload: %v", err)
	}
	if converged.ID != stored.ID {
		t.Fatalf("converged blob = %s, want %s", converged.ID, stored.ID)
	}
	result, err := storage.NewSweeperWithClock(pool, store, clock).Sweep(ctx)
	if err != nil {
		t.Fatalf("sweep after convergence: %v", err)
	}
	if result.Deleted != 0 {
		t.Fatalf("sweep after convergence = %+v, want no deletion", result)
	}
	opened, err := store.Open(ctx, stored.ID)
	if err != nil {
		t.Fatalf("converging upload lost its bytes: %v", err)
	}
	opened.Close()
}

func TestSweepCollectsAnWorkAfterItsRecoveryWindow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.Connect(t)
	store, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	service := work.NewServiceWithClock(pool, apitest.RegistryWith(t, apitest.OpaqueModule{}), store, clock)
	ownerID := uuid.New()
	if _, err := pool.Exec(ctx, `insert into users (id, username) values ($1, 'expired.owner')`, ownerID); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	created, err := apitest.Uploads(service).Create(ctx, upload.CreateInput{
		OwnerID: ownerID, Type: "theme", Filename: "expired.lumitheme",
		File: bytes.NewReader([]byte("expired source")), Name: "Expired",
	})
	if err != nil {
		t.Fatalf("create work: %v", err)
	}
	if err := apitest.Pages(service).Delete(ctx, ownerID, created.ID); err != nil {
		t.Fatalf("delete work: %v", err)
	}
	var blobID uuid.UUID
	if err := pool.QueryRow(ctx,
		`select blob_id from work_original_files where work_id = $1`, created.ID,
	).Scan(&blobID); err != nil {
		t.Fatalf("read blob id: %v", err)
	}

	now = now.Add(page.RecoveryWindow + time.Second)
	marked, err := storage.NewSweeperWithClock(pool, store, clock).Sweep(ctx)
	if err != nil || marked.Marked != 1 || marked.Deleted != 0 {
		t.Fatalf("mark expired work = %+v, %v; want one mark", marked, err)
	}
	now = now.Add(storage.SweepDelay + time.Second)
	deleted, err := storage.NewSweeperWithClock(pool, store, clock).Sweep(ctx)
	if err != nil || deleted.Deleted != 1 {
		t.Fatalf("collect expired work = %+v, %v; want one deletion", deleted, err)
	}
	if _, err := store.Open(ctx, blobID); !errors.Is(err, storage.ErrBlobNotFound) {
		t.Fatalf("expired work blob error = %v, want ErrBlobNotFound", err)
	}
	if err := apitest.Pages(service).Restore(ctx, ownerID, created.ID); !errors.Is(err, work.ErrNotFound) {
		t.Fatalf("restore after recovery error = %v, want ErrNotFound", err)
	}
}

func TestPostPicturesLiveWhileAnEditionStillRefersToThem(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.Connect(t)
	store, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	now := time.Date(2026, 8, 29, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }

	authorID := uuid.New()
	if _, err := pool.Exec(ctx,
		`insert into users (id, username) values ($1, 'sweep.author')`, authorID,
	); err != nil {
		t.Fatalf("insert author: %v", err)
	}
	var categoryID uuid.UUID
	err = pool.QueryRow(ctx,
		`select id from blog_categories where slug = 'announcement'`,
	).Scan(&categoryID)
	if err != nil {
		t.Fatalf("read a category: %v", err)
	}
	postID := uuid.New()
	if _, err := pool.Exec(ctx, `
		insert into posts (id, author_id, category_id, title, document, body_version)
		values ($1, $2, $3, 'A post with pictures', '{"version":2,"content":[]}', 2)
	`, postID, authorID, categoryID); err != nil {
		t.Fatalf("insert post: %v", err)
	}

	kept, err := store.Put(ctx, bytes.NewReader([]byte("the picture a revision carries")))
	if err != nil {
		t.Fatalf("put the kept picture: %v", err)
	}
	dropped, err := store.Put(ctx, bytes.NewReader([]byte("the picture nobody placed")))
	if err != nil {
		t.Fatalf("put the dropped picture: %v", err)
	}
	keptMedia, droppedMedia := uuid.New(), uuid.New()
	for id, blobID := range map[uuid.UUID]uuid.UUID{keptMedia: kept.ID, droppedMedia: dropped.ID} {
		if _, err := pool.Exec(ctx, `
			insert into post_media (id, post_id, blob_id, purpose, width, height)
			values ($1, $2, $3, 'document', 10, 10)
		`, id, postID, blobID); err != nil {
			t.Fatalf("insert post media: %v", err)
		}
	}
	originalFileID := uuid.New()
	if _, err := pool.Exec(ctx, `
		insert into post_revisions (id, post_id, number, title, summary, slug, category_id,
		                            document, body_version, captured_for)
		values ($1, $2, 1, 'A post with pictures', 'A summary.', 'a-post-with-pictures', $3,
		        '{"version":2,"content":[]}', 2, 'publication')
	`, originalFileID, postID, categoryID); err != nil {
		t.Fatalf("insert revision: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into post_media_uses (media_id, post_id, revision_id) values ($1, $2, $3)
	`, keptMedia, postID, originalFileID); err != nil {
		t.Fatalf("record what the revision refers to: %v", err)
	}

	if _, err := storage.NewSweeperWithClock(pool, store, clock).Sweep(ctx); err != nil {
		t.Fatalf("mark sweep: %v", err)
	}
	now = now.Add(storage.SweepDelay + time.Second)
	if _, err := storage.NewSweeperWithClock(pool, store, clock).Sweep(ctx); err != nil {
		t.Fatalf("delete sweep: %v", err)
	}

	opened, err := store.Open(ctx, kept.ID)
	if err != nil {
		t.Fatalf("a picture a revision refers to was swept: %v", err)
	}
	opened.Close()
	if _, err := store.Open(ctx, dropped.ID); !errors.Is(err, storage.ErrBlobNotFound) {
		t.Fatalf("opening the unplaced picture = %v, want it collected", err)
	}
	var blobID *uuid.UUID
	if err := pool.QueryRow(ctx,
		`select blob_id from post_media where id = $1`, droppedMedia,
	).Scan(&blobID); err != nil {
		t.Fatalf("read the unplaced picture: %v", err)
	}
	if blobID != nil {
		t.Error("the unplaced picture still points at bytes that are gone")
	}
}
