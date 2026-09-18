package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// SweepDelay is how long a marked blob waits before its bytes go
const SweepDelay = 24 * time.Hour
const sweepInterval = time.Hour

type SweepResult struct {
	Marked  int64
	Deleted int
}

func (s *Sweeper) Sweep(ctx context.Context) (SweepResult, error) {
	now := s.now()
	if err := s.deleteExpiredSnapshots(ctx); err != nil {
		return SweepResult{}, err
	}
	if err := s.deleteExpiredProtectedContent(ctx, now); err != nil {
		return SweepResult{}, err
	}
	if _, err := s.store.RecordOrphans(ctx); err != nil {
		return SweepResult{}, fmt.Errorf("record filesystem orphans: %w", err)
	}
	if err := s.resumePurges(ctx); err != nil {
		return SweepResult{}, err
	}
	marked, err := s.pool.Exec(ctx, `
		insert into blob_sweep_marks (blob_id, marked_at)
		select blob.id, $1
		  from blobs blob
		 where not exists (select 1 from blob_sweep_marks mark where mark.blob_id = blob.id)
		   and not `+liveBlobReferenceExpression("blob.id", "$1"), now)
	if err != nil {
		return SweepResult{}, fmt.Errorf("mark unreferenced blobs: %w", err)
	}
	result := SweepResult{Marked: marked.RowsAffected()}

	rows, err := s.pool.Query(ctx, `
		select blob_id from blob_sweep_marks where marked_at <= $1 order by marked_at, blob_id
	`, now.Add(-SweepDelay))
	if err != nil {
		return SweepResult{}, fmt.Errorf("list marked blobs: %w", err)
	}
	var candidates []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return SweepResult{}, fmt.Errorf("read marked blob: %w", err)
		}
		candidates = append(candidates, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return SweepResult{}, fmt.Errorf("list marked blobs: %w", err)
	}
	rows.Close()

	for _, id := range candidates {
		deleted, err := s.deleteMarkedBlob(ctx, id, now)
		if err != nil {
			return result, err
		}
		if deleted {
			result.Deleted++
		}
	}
	return result, nil
}

func (s *Sweeper) deleteExpiredSnapshots(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin expired history cleanup: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		update works set published_snapshot_id = null
		 where deleted_at is not null and recoverable_until <= now()
		   and published_snapshot_id is not null
	`); err != nil {
		return fmt.Errorf("release expired published snapshots: %w", err)
	}
	if _, err := tx.Exec(ctx, `delete from work_snapshots s using works a
		where s.work_id = a.id and a.deleted_at is not null and a.recoverable_until <= now()`); err != nil {
		return fmt.Errorf("remove expired history: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Sweeper) deleteExpiredProtectedContent(ctx context.Context, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin protected content cleanup: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		select id from works
		 where deleted_at is not null and recoverable_until <= $1
		   and (exists (select 1 from protected_content where work_id = works.id)
		        or exists (select 1 from protected_delivery_apps where work_id = works.id))
		 for update
	`, now)
	if err != nil {
		return fmt.Errorf("lock expired works for protected content cleanup: %w", err)
	}
	var expired []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("read expired work for protected content cleanup: %w", err)
		}
		expired = append(expired, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("list expired works for protected content cleanup: %w", err)
	}
	rows.Close()
	if len(expired) == 0 {
		return tx.Commit(ctx)
	}
	if _, err := tx.Exec(ctx, `delete from protected_content where work_id = any($1)`, expired); err != nil {
		return fmt.Errorf("remove expired protected content: %w", err)
	}
	if _, err := tx.Exec(ctx, `delete from protected_delivery_apps where work_id = any($1)`, expired); err != nil {
		return fmt.Errorf("remove expired protected delivery policy: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit protected content cleanup: %w", err)
	}
	return nil
}

func (s *Sweeper) resumePurges(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `
		select blob.id, blob.sha256
		  from blobs blob
		  join blob_tombstones tombstone on tombstone.sha256 = blob.sha256
		 order by tombstone.purged_at, blob.id
	`)
	if err != nil {
		return fmt.Errorf("list interrupted purges: %w", err)
	}
	type interruptedPurge struct {
		blobID uuid.UUID
		digest [32]byte
	}
	var interrupted []interruptedPurge
	for rows.Next() {
		var purge interruptedPurge
		var digestBytes []byte
		if err := rows.Scan(&purge.blobID, &digestBytes); err != nil {
			rows.Close()
			return fmt.Errorf("read interrupted purge: %w", err)
		}
		copy(purge.digest[:], digestBytes)
		interrupted = append(interrupted, purge)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("list interrupted purges: %w", err)
	}
	rows.Close()
	for _, purge := range interrupted {
		if err := s.deletePurgedBlob(ctx, purge.blobID, purge.digest); err != nil {
			return fmt.Errorf("resume interrupted purge: %w", err)
		}
	}
	return nil
}

func (s *Sweeper) RunSweeper(ctx context.Context, report func(error)) {
	s.runSweeper(ctx, sweepInterval, report)
}

func (s *Sweeper) runSweeper(ctx context.Context, interval time.Duration, report func(error)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if _, err := s.Sweep(ctx); err != nil && !errors.Is(err, context.Canceled) && report != nil {
			report(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Sweeper) deleteMarkedBlob(ctx context.Context, id uuid.UUID, now time.Time) (bool, error) {
	ready, err := s.prepareMarkedBlob(ctx, id, now)
	if err != nil || !ready {
		return false, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin blob sweep: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := LockBlobDigest(ctx, tx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	var marked pgtype.Timestamptz
	if err := tx.QueryRow(ctx,
		`select marked_at from blob_sweep_marks where blob_id = $1 for update`, id,
	).Scan(&marked); errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("lock marked blob: %w", err)
	}
	if !marked.Valid || marked.Time.After(now.Add(-SweepDelay)) {
		return false, nil
	}

	referenced, err := blobHasLiveReference(ctx, tx, id, now)
	if err != nil {
		return false, err
	}
	if referenced {
		if _, err := tx.Exec(ctx, `delete from blob_sweep_marks where blob_id = $1`, id); err != nil {
			return false, fmt.Errorf("clear blob sweep mark: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("commit cancelled blob sweep: %w", err)
		}
		return false, nil
	}

	var digestBytes []byte
	if err := tx.QueryRow(ctx, `select sha256 from blobs where id = $1`, id).Scan(&digestBytes); err != nil {
		return false, fmt.Errorf("read swept blob digest: %w", err)
	}
	var digest [32]byte
	copy(digest[:], digestBytes)
	if err := postgres.LockBlobDeletionAgainstBackup(ctx, tx); err != nil {
		return false, fmt.Errorf("lock physical blob deletion against backup: %w", err)
	}
	if err := s.store.DeleteDerivatives(ctx, digest); err != nil {
		return false, fmt.Errorf("delete swept blob derivatives: %w", err)
	}
	if err := s.store.Delete(ctx, id); err != nil && !errors.Is(err, ErrBlobNotFound) {
		return false, fmt.Errorf("delete swept blob bytes: %w", err)
	}
	if _, err := tx.Exec(ctx, `delete from blobs where id = $1`, id); err != nil {
		return false, fmt.Errorf("delete swept blob record: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit blob sweep: %w", err)
	}
	return true, nil
}

func (s *Sweeper) prepareMarkedBlob(ctx context.Context, id uuid.UUID, now time.Time) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin blob sweep preparation: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := LockBlobDigest(ctx, tx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	var marked pgtype.Timestamptz
	if err := tx.QueryRow(ctx,
		`select marked_at from blob_sweep_marks where blob_id = $1 for update`, id,
	).Scan(&marked); errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("lock marked blob for preparation: %w", err)
	}
	if !marked.Valid || marked.Time.After(now.Add(-SweepDelay)) {
		return false, nil
	}
	referenced, err := blobHasLiveReference(ctx, tx, id, now)
	if err != nil {
		return false, err
	}
	if referenced {
		if _, err := tx.Exec(ctx, `delete from blob_sweep_marks where blob_id = $1`, id); err != nil {
			return false, fmt.Errorf("clear blob sweep mark: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("commit cancelled blob sweep: %w", err)
		}
		return false, nil
	}
	if err := releaseExpiredReferences(ctx, tx, id, now); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit blob sweep preparation: %w", err)
	}
	return true, nil
}

func LockBlobDigest(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	var digest []byte
	if err := tx.QueryRow(ctx, `select sha256 from blobs where id = $1`, id).Scan(&digest); err != nil {
		return err
	}
	return postgres.LockBlobDigest(ctx, tx, digest)
}

func blobHasLiveReference(ctx context.Context, tx pgx.Tx, id uuid.UUID, now time.Time) (bool, error) {
	var referenced bool
	err := tx.QueryRow(ctx, "select "+liveBlobReferenceExpression("$1", "$2"), id, now).Scan(&referenced)
	if err != nil {
		return false, fmt.Errorf("recheck blob references: %w", err)
	}
	return referenced, nil
}

func liveBlobReferenceExpression(blobID, at string) string {
	return `exists (
		select 1 from ingest_operations operation
		 where operation.blob_id = ` + blobID + `
		   and operation.status in ('pending', 'processing')
		union all
		select 1 from work_revisions revision
		  join works work on work.id = revision.work_id
		 where revision.blob_id = ` + blobID + `
		   and (work.deleted_at is null or work.recoverable_until > ` + at + `)
		union all
		select 1 from work_media media
		  join works work on work.id = media.work_id
		 where media.blob_id = ` + blobID + `
		   and (work.deleted_at is null or work.recoverable_until > ` + at + `)
		union all
		select 1 from profile_media media where media.blob_id = ` + blobID + `
		union all
		select 1 from publication_media media where media.blob_id = ` + blobID + `
		union all
		select 1 from post_media media
		  join post_media_uses use on use.media_id = media.id
		 where media.blob_id = ` + blobID + `
	)`
}

func releaseExpiredReferences(ctx context.Context, tx pgx.Tx, id uuid.UUID, now time.Time) error {
	statements := []string{
		`update work_revisions revision set blob_id = null
		   from works work
		  where revision.work_id = work.id and revision.blob_id = $1
		    and work.deleted_at is not null and work.recoverable_until <= $2`,
		`update work_media media set blob_id = null
		   from works work
		  where media.work_id = work.id and media.blob_id = $1
		    and work.deleted_at is not null and work.recoverable_until <= $2`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement, id, now); err != nil {
			return fmt.Errorf("release expired blob reference: %w", err)
		}
	}
	_, err := tx.Exec(ctx, `
		update post_media set blob_id = null
		 where blob_id = $1
		   and not exists (select 1 from post_media_uses use where use.media_id = post_media.id)
	`, id)
	if err != nil {
		return fmt.Errorf("release an unused post picture: %w", err)
	}
	return nil
}
