package upload_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
)

func TestStartingAWorkStillTakesTheOldNameForItsType(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)

	request := httptest.NewRequest(http.MethodPost, "/v1/assets", strings.NewReader(`{"kind":"character"}`))
	request.Header.Set("Content-Type", "application/json")
	started := apitest.Send(t, r, apitest.Authorized(request, session))

	if started.Code != http.StatusCreated {
		t.Fatalf("start status = %d, want 201: %s", started.Code, started.Body.String())
	}
	var page map[string]any
	if err := json.Unmarshal(started.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode the new page: %v", err)
	}
	if page["kind"] != "character" || page["type"] != "character" {
		t.Fatalf("new page kind = %v, type = %v; want character", page["kind"], page["type"])
	}
}

func TestAnUploadStillTakesTheOldNameForItsVisibility(t *testing.T) {
	t.Parallel()
	router, session, works := harness.NewVerifiedIngestRouter(t, format.NewRegistry())
	metadata := apitest.ExampleMetadata("A quiet draft")
	metadata["filename"] = "quiet-draft.lumitheme"
	delete(metadata, "visibility")
	metadata["discovery"] = "unlisted"

	finished := apitest.UploadAndFinish(t, router, session, works, metadata, []byte("theme"))

	var operation struct {
		Work *struct {
			Visibility string `json:"discovery"`
		} `json:"asset"`
	}
	if err := json.Unmarshal(finished.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode the upload: %v", err)
	}
	if operation.Work == nil || operation.Work.Visibility != "unlisted" {
		t.Fatalf("upload answered %s, want an unlisted work under its old names", finished.Body.String())
	}
}
