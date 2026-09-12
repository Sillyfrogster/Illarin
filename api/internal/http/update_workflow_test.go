package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
)

type workingCopyStanding struct {
	UnpublishedChanges bool `json:"unpublishedChanges"`
}

func readWorkingCopyStanding(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	assetID string,
) workingCopyStanding {
	t.Helper()
	response := send(t, r, authorized(httptest.NewRequest(
		http.MethodGet, "/v1/assets/"+assetID+"?workingCopy=true", nil), session))
	if response.Code != http.StatusOK {
		t.Fatalf("read the working copy = %d: %s", response.Code, response.Body.String())
	}
	var standing workingCopyStanding
	if err := json.Unmarshal(response.Body.Bytes(), &standing); err != nil {
		t.Fatalf("decode the working copy: %v", err)
	}
	return standing
}

func TestAWorkingCopySaysWhetherReadersHaveSeenItYet(t *testing.T) {
	r, session := newVerifiedTestRouter(t)
	started := startCharacter(t, r, session)
	writeCharacterFloor(t, r, session, started)
	if got := publishAsset(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200: %s", got.Code, got.Body.String())
	}
	if readWorkingCopyStanding(t, r, session, started.ID).UnpublishedChanges {
		t.Error("a freshly published asset reads as having changes readers cannot see")
	}

	coreBlock := blockNamed(t, started.Blocks, "character_core")
	core := editableBlock(coreBlock)
	core.Elements[0].Content = json.RawMessage(`{"text":"She has moved to the east shelf."}`)
	if got := saveBlock(t, r, session, started.ID, coreBlock.ID, core); got.Code != http.StatusOK {
		t.Fatalf("save the description status = %d, want 200: %s", got.Code, got.Body.String())
	}
	if !readWorkingCopyStanding(t, r, session, started.ID).UnpublishedChanges {
		t.Error("a private edit reads as though readers already have it")
	}

	update := publishAssetUpdate(t, r, session, started.ID,
		`{"summary":"Moved her to the east shelf"}`)
	if update.Code != http.StatusOK {
		t.Fatalf("publish an update status = %d, want 200: %s", update.Code, update.Body.String())
	}
	if readWorkingCopyStanding(t, r, session, started.ID).UnpublishedChanges {
		t.Error("a published update left the working copy reading as unpublished")
	}
}

func TestAReplacementWaitingForReviewIsFoundFromTheAssetItTargets(t *testing.T) {
	r, session, assets := newVerifiedIngestRouter(t, format.NewRegistry())
	metadata := exampleMetadata("Evening Theme")
	metadata["filename"] = "evening.lumitheme"
	upload := send(t, r, authorized(uploadRequest(t, metadata, []byte("first bytes")), session))
	if _, err := assets.ProcessNextIngest(context.Background()); err != nil {
		t.Fatalf("process ingest: %v", err)
	}
	created := pollIngestAsset(t, r, session, upload.Header().Get("Location"))
	published := send(t, r, authorized(httptest.NewRequest(
		http.MethodPost, "/v1/assets/"+created.ID+"/publish", nil), session))
	if published.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200: %s", published.Code, published.Body.String())
	}

	quiet := send(t, r, authorized(httptest.NewRequest(
		http.MethodGet, "/v1/assets/"+created.ID+"/revisions", nil), session))
	if quiet.Code != http.StatusOK || strings.TrimSpace(quiet.Body.String()) != "null" {
		t.Fatalf("an asset with no replacement = %d %s, want 200 null",
			quiet.Code, quiet.Body.String())
	}

	revision := send(t, r, authorized(
		revisionRequest(t, created.ID, "evening.lumitheme", []byte("second bytes")), session))
	if revision.Code != http.StatusAccepted {
		t.Fatalf("upload a replacement = %d, want 202: %s", revision.Code, revision.Body.String())
	}
	if _, err := assets.ProcessNextIngest(context.Background()); err != nil {
		t.Fatalf("process the replacement: %v", err)
	}

	found := send(t, r, authorized(httptest.NewRequest(
		http.MethodGet, "/v1/assets/"+created.ID+"/revisions", nil), session))
	if found.Code != http.StatusOK {
		t.Fatalf("read the waiting replacement = %d, want 200: %s", found.Code, found.Body.String())
	}
	var operation struct {
		ID      string `json:"id"`
		Status  string `json:"status"`
		Preview *struct {
			Format string `json:"format"`
		} `json:"preview"`
	}
	if err := json.Unmarshal(found.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode the waiting replacement: %v", err)
	}
	if operation.Status != "preview" || operation.Preview == nil {
		t.Fatalf("the waiting replacement = %+v", operation)
	}
	if "/v1/ingests/"+operation.ID != revision.Header().Get("Location") {
		t.Fatalf("found operation %s, want the uploaded %s",
			operation.ID, revision.Header().Get("Location"))
	}

	cancelled := send(t, r, authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/assets/"+created.ID+"/revisions/"+operation.ID, nil), session))
	if cancelled.Code != http.StatusNoContent {
		t.Fatalf("cancel the replacement = %d, want 204: %s", cancelled.Code, cancelled.Body.String())
	}
	after := send(t, r, authorized(httptest.NewRequest(
		http.MethodGet, "/v1/assets/"+created.ID+"/revisions", nil), session))
	if strings.TrimSpace(after.Body.String()) != "null" {
		t.Fatalf("a cancelled replacement still answers %s, want null", after.Body.String())
	}
}
