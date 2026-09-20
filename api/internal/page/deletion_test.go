package page_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

func TestCreatorCanDeleteAndRestoreAnWorkDuringItsRecoveryWindow(t *testing.T) {
	t.Parallel()
	router, session, works := harness.NewVerifiedUploadRouter(t, format.NewRegistry())
	workID := apitest.WorkIDFromUpload(t, apitest.UploadAndFinish(
		t, router, session, works,
		withFilename(apitest.ExampleMetadata("Recoverable garden"), "recoverable-garden"),
		[]byte("the retained source"),
	))

	deleted := apitest.Send(t, router, apitest.Authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/works/"+workID, nil), session,
	))
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204: %s", deleted.Code, deleted.Body.String())
	}

	page := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/works/"+workID, nil))
	if page.Code != http.StatusNotFound {
		t.Fatalf("deleted work page status = %d, want 404: %s", page.Code, page.Body.String())
	}
	browse := readProfileListing(t, router, "/v1/works?creator=verified.creator", session)
	if len(browse.Items) != 0 {
		t.Fatalf("active owner listing after delete = %+v, want empty", browse.Items)
	}

	listed := apitest.Send(t, router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/profiles/verified.creator/deleted", nil), session,
	))
	if listed.Code != http.StatusOK {
		t.Fatalf("deleted listing status = %d, want 200: %s", listed.Code, listed.Body.String())
	}
	var recovery struct {
		Items []struct {
			ID               string    `json:"id"`
			Name             string    `json:"name"`
			Type             string    `json:"type"`
			DeletedAt        time.Time `json:"deletedAt"`
			RecoverableUntil time.Time `json:"recoverableUntil"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &recovery); err != nil {
		t.Fatalf("decode deleted listing: %v", err)
	}
	if len(recovery.Items) != 1 || recovery.Items[0].ID != workID ||
		recovery.Items[0].Name != "Recoverable garden" || recovery.Items[0].Type != "character" {
		t.Fatalf("deleted listing = %+v, want the deleted work", recovery.Items)
	}
	if !recovery.Items[0].RecoverableUntil.After(recovery.Items[0].DeletedAt) {
		t.Fatalf("recovery deadline = %v, deleted at %v", recovery.Items[0].RecoverableUntil, recovery.Items[0].DeletedAt)
	}

	restored := apitest.Send(t, router, apitest.Authorized(
		httptest.NewRequest(http.MethodPost, "/v1/works/"+workID+"/restore", nil), session,
	))
	if restored.Code != http.StatusNoContent {
		t.Fatalf("restore status = %d, want 204: %s", restored.Code, restored.Body.String())
	}
	if got := apitest.FetchWorkPage(t, router, "/v1/works/"+workID); got.ID != workID {
		t.Fatalf("restored work id = %q, want %q", got.ID, workID)
	}
	download := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/download/"+workID, nil))
	if download.Code != http.StatusOK || download.Header().Get("X-Accel-Redirect") == "" {
		t.Fatalf("restored download = %d, headers %v", download.Code, download.Header())
	}
}

func TestPrivatePromptsSurviveRecoveryAndLeaveAfterItExpires(t *testing.T) {
	t.Parallel()
	_, router, session, works, pool := harness.NewVerifiedRoutersWithPool(t, 1<<20, api.DefaultDeadlines())
	started := apitest.StartPreset(t, router, session, "lumiverse")
	coreBlock := apitest.BlockNamed(t, started.Blocks, "preset_core")
	core := apitest.EditableBlock(coreBlock)
	const privateText = "Recover this exact private prompt."
	core.Elements[0].Content = json.RawMessage(`{"groups":[],"fragments":[
		{"name":"Recoverable","role":"system","text":"` + privateText + `","private":true,"enabled":true}
	]}`)
	apps := []string{"lumiverse"}
	core.AllowedApps = &apps
	if response := apitest.SaveBlock(t, router, session, started.ID, coreBlock.ID, core); response.Code != http.StatusOK {
		t.Fatalf("save private prompt: %d %s", response.Code, response.Body.String())
	}

	deleteWork := func() {
		t.Helper()
		response := apitest.Send(t, router, apitest.Authorized(
			httptest.NewRequest(http.MethodDelete, "/v1/works/"+started.ID, nil), session,
		))
		if response.Code != http.StatusNoContent {
			t.Fatalf("delete status = %d, want 204: %s", response.Code, response.Body.String())
		}
	}
	deleteWork()
	if payloads, policies := apitest.PrivatePromptCounts(t, pool, started.ID); payloads != 1 || policies != 1 {
		t.Fatalf("during recovery: %d payloads and %d policy rows, want 1 and 1", payloads, policies)
	}

	restored := apitest.Send(t, router, apitest.Authorized(
		httptest.NewRequest(http.MethodPost, "/v1/works/"+started.ID+"/restore", nil), session,
	))
	if restored.Code != http.StatusNoContent {
		t.Fatalf("restore status = %d, want 204: %s", restored.Code, restored.Body.String())
	}
	owner := apitest.FetchStartedWork(t, router, session, started.ID)
	if !strings.Contains(string(owner.Blocks[0].Elements[0].Content), privateText) ||
		!owner.HasPrivatePrompts || len(owner.AllowedApps) != 1 || owner.AllowedApps[0].ID != "lumiverse" {
		t.Fatalf("restored work with private prompts lost its prompt or policy: %+v", owner)
	}

	deleteWork()
	if _, err := pool.Exec(t.Context(), `
		update works set recoverable_until = now() - interval '1 second' where id = $1
	`, started.ID); err != nil {
		t.Fatalf("expire recovery window: %v", err)
	}
	if _, err := cleanup(works).Cleanup(t.Context()); err != nil {
		t.Fatalf("cleanup expired work: %v", err)
	}
	if payloads, policies := apitest.PrivatePromptCounts(t, pool, started.ID); payloads != 0 || policies != 0 {
		t.Fatalf("after recovery expired: %d payloads and %d policy rows, want none", payloads, policies)
	}
}

func TestUploadRefusesBytesNamedByAPurgeTombstone(t *testing.T) {
	t.Parallel()
	router, session, works, pool := harness.NewVerifiedUploadRouterWithPool(t, format.NewRegistry())
	file := []byte("bytes that cannot return")
	workID := apitest.WorkIDFromUpload(t, apitest.UploadAndFinish(
		t, router, session, works,
		withFilename(apitest.ExampleMetadata("Gone for good"), "gone-for-good"), file,
	))
	var digest []byte
	if err := pool.QueryRow(context.Background(), `
		select blob.sha256
		  from work_original_files original
		  join blobs blob on blob.id = original.blob_id
		 where original.work_id = $1
	`, workID).Scan(&digest); err != nil {
		t.Fatalf("read work digest: %v", err)
	}
	var contentDigest [32]byte
	copy(contentDigest[:], digest)
	actorID := uuid.New()
	if _, err := pool.Exec(context.Background(),
		`insert into users (id, username) values ($1, 'purge.actor')`, actorID,
	); err != nil {
		t.Fatalf("insert purge actor: %v", err)
	}
	if err := cleanup(works).Purge(context.Background(), contentDigest, "legal_order", actorID); err != nil {
		t.Fatalf("purge: %v", err)
	}

	reupload := apitest.Send(t, router, apitest.Authorized(
		apitest.UploadRequest(t, withFilename(apitest.ExampleMetadata("Attempted return"), "attempted-return"), file),
		session,
	))
	if reupload.Code != http.StatusUnprocessableEntity {
		t.Fatalf("purged re-upload status = %d, want 422: %s", reupload.Code, reupload.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(reupload.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode refusal: %v", err)
	}
	if body["error"] != "This file cannot be accepted." {
		t.Fatalf("purged re-upload error = %q", body["error"])
	}
}

func TestDeletedListingBelongsOnlyToItsOwner(t *testing.T) {
	t.Parallel()
	router, session, works := harness.NewVerifiedUploadRouter(t, format.NewRegistry())
	workID := apitest.WorkIDFromUpload(t, apitest.UploadAndFinish(
		t, router, session, works,
		withFilename(apitest.ExampleMetadata("Private recovery"), "private-recovery"), []byte("source"),
	))
	response := apitest.Send(t, router, apitest.Authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/works/"+workID, nil), session,
	))
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d: %s", response.Code, response.Body.String())
	}

	stranger := apitest.Send(t, router, httptest.NewRequest(
		http.MethodGet, "/v1/profiles/verified.creator/deleted", nil,
	))
	if stranger.Code != http.StatusUnauthorized {
		t.Fatalf("stranger deleted listing status = %d, want 401", stranger.Code)
	}
}

// cleanup cleans up blobs the way the server's background cleanup does
func cleanup(works *work.Service) *storage.Cleanup {
	return storage.NewCleanup(works.Pool(), works.Store())
}
