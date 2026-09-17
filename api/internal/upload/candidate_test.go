package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/staff"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
)

func TestRevocationRejectsAnUploadWaitingForCandidateAcceptance(t *testing.T) {
	t.Parallel()
	svc, pool := newTestService(t)
	owner := revisionOwner(t, svc, "candidate.owner")
	id, err := svc.StartFromNothing(context.Background(), owner, "character", "")
	if err != nil {
		t.Fatal(err)
	}
	candidate := currentCandidate(t, svc, id)
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	done := make(chan error, 1)
	go func() {
		_, err := svc.AcceptRevision(context.Background(), RevisionInput{
			OwnerID: owner, AssetID: id, Filename: "candidate.json", File: reader,
		}, candidate)
		done <- err
	}()
	if _, err := writer.Write([]byte(`{"candidate":"private"}`)); err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(context.Background(), `update assets set withheld_at = now(), withheld_by = owner_id, withheld_reason = 'Review' where id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, asset.ErrAssetFrozen) {
		t.Fatalf("revoked acceptance = %v, want frozen", err)
	}
	if err := staff.NewService(pool).ClearWithhold(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	_, err = svc.AcceptRevision(context.Background(), RevisionInput{
		OwnerID: owner, AssetID: id, Filename: "candidate.json", File: bytes.NewBufferString(`{"candidate":"private"}`),
	}, candidate)
	var conflict *asset.VersionConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("acceptance after revocation cleared = %v, want stale", err)
	}
	var count int
	if err := pool.QueryRow(context.Background(), `select count(*) from ingest_operations where target_asset_id = $1`, id).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("revoked acceptance queued %d uploads", count)
	}
}

func TestQueuedRevisionCannotOverwriteANewerWorkingCopy(t *testing.T) {
	t.Parallel()
	registry := registryWithModule(t, kindModule{id: "as_character", kind: "character"})
	svc, _ := newTestServiceWithRegistry(t, registry)
	owner := revisionOwner(t, svc, "queued.owner")
	created := ingestOne(t, svc, owner, "card.json", []byte(`{"spec":"as_character"}`))
	candidate := currentCandidate(t, svc, created.ID)
	operation, err := svc.AcceptRevision(context.Background(), RevisionInput{
		OwnerID: owner, AssetID: created.ID, Filename: "card.json", File: bytes.NewBufferString(`{"spec":"as_character"}`),
	}, candidate)
	if err != nil {
		t.Fatal(err)
	}
	adult := false
	if err := works(svc).SetIdentity(context.Background(), work.Identity{OwnerID: owner, AssetID: created.ID, Name: "Newer work", IsNSFW: &adult}, candidate); err != nil {
		t.Fatal(err)
	}
	if processed, err := svc.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("process stale upload = %v, %v", processed, err)
	}
	finished, err := svc.GetIngest(context.Background(), owner, operation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != IngestFailed || finished.Failure == nil || finished.Failure.Reason != "working_copy_conflict" {
		t.Fatalf("stale upload = %+v", finished)
	}
	page, err := works(svc).WorkingCopy(context.Background(), created.ID, &owner, asset.ContentShown)
	if err != nil {
		t.Fatal(err)
	}
	if page.Name != "Newer work" || page.WorkingCopyVersion == nil || *page.WorkingCopyVersion != candidate.SavedVersion {
		t.Fatalf("stale upload changed the candidate: %+v", page)
	}
}
