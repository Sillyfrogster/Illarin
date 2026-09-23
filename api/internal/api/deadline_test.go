package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest/full"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

const alreadyPast = time.Nanosecond

func deadlines(json time.Duration) api.Deadlines {
	return api.Deadlines{
		JSON: json, Upload: time.Minute, Download: time.Minute,
		Collect: time.Minute, Verify: time.Minute,
	}
}

func TestAListingPastItsDeadlineFailsRatherThanAnswers(t *testing.T) {
	t.Parallel()
	answered := list(t, harness.NewRouterWith(t, 1<<20, deadlines(5*time.Second)))
	if answered.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 with time to spare. body: %s",
			answered.Code, answered.Body.String())
	}

	gaveUp := list(t, harness.NewRouterWith(t, 1<<20, deadlines(alreadyPast)))
	if gaveUp.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 once the deadline has gone. body: %s",
			gaveUp.Code, gaveUp.Body.String())
	}
}

func list(t *testing.T, r *gin.Engine) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/works", nil))
	return rec
}

func TestARouteWithNoDeadlineIsRefused(t *testing.T) {
	t.Parallel()
	err := full.Register(
		gin.New(),
		apitest.Services{Works: &work.Service{}, Apps: &connect.Apps{}},
		api.Deadlines{Upload: time.Minute, Download: time.Minute, Collect: time.Minute},
	)

	if err == nil {
		t.Fatal("registered a listing route that may run for as long as it likes")
	}
}

func TestAnUploadIsNotHeldToTheListingDeadline(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouterWith(t, 1<<20, deadlines(alreadyPast))

	rec := apitest.Send(t, r, apitest.Authorized(
		apitest.UploadRequest(t, apitest.ExampleMetadata("Patient"), []byte("bytes")), session,
	))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202. body: %s", rec.Code, rec.Body.String())
	}
}

func TestADownloadIsNotHeldToTheListingDeadline(t *testing.T) {
	t.Parallel()
	setup, r, session, works := harness.NewVerifiedRoutersWithService(t, 1<<20, deadlines(alreadyPast))

	file := []byte("bytes worth waiting for")
	metadata := apitest.ExampleMetadata("Roomy")
	metadata["filename"] = "roomy.lumitheme"
	rec := apitest.UploadAndFinish(t, setup, session, works, metadata, file)

	var created struct {
		Work *struct {
			ID string `json:"id"`
		} `json:"work"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if created.Work == nil {
		t.Fatal("completed upload has no work")
	}
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/download/"+created.Work.ID, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Accel-Redirect") == "" {
		t.Fatal("download was not handed to nginx")
	}
}
