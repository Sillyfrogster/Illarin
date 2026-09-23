package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/staff"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
)

func TestRevocationRejectsAnUploadWaitingForCandidateAcceptance(t *testing.T) {
	t.Parallel()
	svc, pool := newTestService(t)
	owner := originalFileOwner(t, svc, "candidate.owner")
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
		_, err := svc.AcceptOriginalFile(context.Background(), OriginalFileInput{
			OwnerID: owner, WorkID: id, Filename: "candidate.json", File: reader,
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
	if _, err := tx.Exec(context.Background(), `update works set taken_down_at = now(), taken_down_by = owner_id, taken_down_reason = 'Review' where id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, work.ErrWorkFrozen) {
		t.Fatalf("revoked acceptance = %v, want frozen", err)
	}
	if err := staff.NewService(svc.works).LiftTakedown(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	_, err = svc.AcceptOriginalFile(context.Background(), OriginalFileInput{
		OwnerID: owner, WorkID: id, Filename: "candidate.json", File: bytes.NewBufferString(`{"candidate":"private"}`),
	}, candidate)
	var conflict *work.VersionConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("acceptance after revocation cleared = %v, want stale", err)
	}
	var count int
	if err := pool.QueryRow(context.Background(), `select count(*) from upload_operations where target_work_id = $1`, id).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("revoked acceptance queued %d uploads", count)
	}
}

func TestAQueuedOriginalFileCannotOverwriteNewerDraftedChanges(t *testing.T) {
	t.Parallel()
	registry := registryWithModule(t, typeModule{id: "as_character", workType: "character"})
	svc, _ := newTestServiceWithRegistry(t, registry)
	owner := originalFileOwner(t, svc, "queued.owner")
	created := uploadOne(t, svc, owner, "card.json", []byte(`{"spec":"as_character"}`))
	candidate := currentCandidate(t, svc, created.ID)
	operation, err := svc.AcceptOriginalFile(context.Background(), OriginalFileInput{
		OwnerID: owner, WorkID: created.ID, Filename: "card.json", File: bytes.NewBufferString(`{"spec":"as_character"}`),
	}, candidate)
	if err != nil {
		t.Fatal(err)
	}
	nsfw := false
	if err := works(svc).SetDetails(context.Background(), page.Details{OwnerID: owner, WorkID: created.ID, Name: "Newer work", IsNSFW: &nsfw}, candidate); err != nil {
		t.Fatal(err)
	}
	if processed, err := svc.ProcessNextUpload(context.Background()); err != nil || !processed {
		t.Fatalf("process stale upload = %v, %v", processed, err)
	}
	finished, err := svc.GetUpload(context.Background(), owner, operation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != UploadFailed || finished.Failure == nil || finished.Failure.Reason != "drafted_changes_conflict" {
		t.Fatalf("stale upload = %+v", finished)
	}
	page, err := works(svc).DraftedChanges(context.Background(), created.ID, &owner, work.NSFWShown)
	if err != nil {
		t.Fatal(err)
	}
	if page.Name != "Newer work" || page.DraftedChangesVersion == nil || *page.DraftedChangesVersion != candidate.SavedVersion {
		t.Fatalf("stale upload changed the candidate: %+v", page)
	}
}
