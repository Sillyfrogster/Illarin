package upload_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
)

func TestAReplacementWaitingForReviewIsFoundFromTheWorkItTargets(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewVerifiedIngestRouter(t, format.NewRegistry())
	metadata := apitest.ExampleMetadata("Evening Theme")
	metadata["filename"] = "evening.lumitheme"
	upload := apitest.Send(t, r, apitest.Authorized(apitest.UploadRequest(t, metadata, []byte("first bytes")), session))
	if _, err := apitest.Uploads(works).ProcessNextIngest(context.Background()); err != nil {
		t.Fatalf("process ingest: %v", err)
	}
	created := apitest.PollIngestWork(t, r, session, upload.Header().Get("Location"))
	published := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodPost, "/v1/works/"+created.ID+"/publish", nil), session))
	if published.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200: %s", published.Code, published.Body.String())
	}

	quiet := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/works/"+created.ID+"/original-file", nil), session))
	if quiet.Code != http.StatusOK || strings.TrimSpace(quiet.Body.String()) != "null" {
		t.Fatalf("a work with no replacement = %d %s, want 200 null",
			quiet.Code, quiet.Body.String())
	}

	uploaded := apitest.Send(t, r, apitest.Authorized(
		apitest.OriginalFileRequest(t, created.ID, "evening.lumitheme", []byte("second bytes")), session))
	if uploaded.Code != http.StatusAccepted {
		t.Fatalf("upload a replacement = %d, want 202: %s", uploaded.Code, uploaded.Body.String())
	}
	if _, err := apitest.Uploads(works).ProcessNextIngest(context.Background()); err != nil {
		t.Fatalf("process the replacement: %v", err)
	}

	found := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/works/"+created.ID+"/original-file", nil), session))
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
	if "/v1/ingests/"+operation.ID != uploaded.Header().Get("Location") {
		t.Fatalf("found operation %s, want the uploaded %s",
			operation.ID, uploaded.Header().Get("Location"))
	}

	cancelled := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/works/"+created.ID+"/original-file/"+operation.ID, nil), session))
	if cancelled.Code != http.StatusNoContent {
		t.Fatalf("cancel the replacement = %d, want 204: %s", cancelled.Code, cancelled.Body.String())
	}
	after := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/works/"+created.ID+"/original-file", nil), session))
	if strings.TrimSpace(after.Body.String()) != "null" {
		t.Fatalf("a cancelled replacement still answers %s, want null", after.Body.String())
	}
}
