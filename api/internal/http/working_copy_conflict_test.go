package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func withReviewedVersion(t *testing.T, r http.Handler, req *http.Request) {
	t.Helper()
	parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	if req.Method == http.MethodGet || len(parts) < 4 || parts[0] != "v1" || parts[1] != "assets" || req.Header.Get("X-Working-Copy-Version") != "" {
		return
	}
	switch parts[3] {
	case "identity", "blocks", "publish", "updates", "preserved", "media", "revisions":
	default:
		return
	}
	read := httptest.NewRequest(http.MethodGet, "/v1/assets/"+parts[2]+"?workingCopy=true", nil)
	read.Header.Set("Cookie", req.Header.Get("Cookie"))
	response := httptest.NewRecorder()
	r.ServeHTTP(response, read)
	var page struct {
		WorkingCopyVersion int64 `json:"workingCopyVersion"`
	}
	if response.Code == http.StatusOK {
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
	}
	if page.WorkingCopyVersion == 0 {
		page.WorkingCopyVersion = 1
	}
	req.Header.Set("X-Working-Copy-Version", strconv.FormatInt(page.WorkingCopyVersion, 10))
}

func TestWorkingCopySaveRequiresAReviewedVersion(t *testing.T) {
	r, session := newVerifiedTestRouter(t)
	started := startPreset(t, r, session, "lumiverse")
	req := httptest.NewRequest(http.MethodPut, "/v1/assets/"+started.ID+"/identity", strings.NewReader(`{"name":"Unreviewed edit","isNsfw":false}`))
	req.Header.Set("Content-Type", "application/json")
	req = authorized(req, session)
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("save without a version = %d, want 400: %s", response.Code, response.Body.String())
	}
}

func TestConcurrentWorkingCopyRequestsKeepOnlyTheWinningCandidate(t *testing.T) {
	for _, publish := range []bool{false, true} {
		name := "two editors"
		if publish {
			name = "save against publication"
		}
		t.Run(name, func(t *testing.T) {
			r, session := newVerifiedTestRouter(t)
			started := startCharacter(t, r, session)
			writeCharacterFloor(t, r, session, started)
			first := authorized(httptest.NewRequest(http.MethodPut, "/v1/assets/"+started.ID+"/identity", strings.NewReader(`{"name":"First editor","blurb":"First pitch","isNsfw":false}`)), session)
			withReviewedVersion(t, r, first)
			second := authorized(httptest.NewRequest(http.MethodPut, "/v1/assets/"+started.ID+"/identity", strings.NewReader(`{"name":"Second editor","blurb":"Second pitch","isNsfw":false}`)), session)
			if publish {
				second = authorized(httptest.NewRequest(http.MethodPost, "/v1/assets/"+started.ID+"/publish", nil), session)
			}
			second.Header.Set("X-Working-Copy-Version", first.Header.Get("X-Working-Copy-Version"))
			responses := []*httptest.ResponseRecorder{httptest.NewRecorder(), httptest.NewRecorder()}
			start := make(chan struct{})
			done := make(chan struct{}, 2)
			for i, request := range []*http.Request{first, second} {
				go func() { <-start; r.ServeHTTP(responses[i], request); done <- struct{}{} }()
			}
			close(start)
			<-done
			<-done
			winner, loser := 0, 1
			if responses[0].Code == http.StatusConflict {
				winner, loser = 1, 0
			}
			want := http.StatusNoContent
			if publish && winner == 1 {
				want = http.StatusOK
			}
			if responses[winner].Code != want || responses[loser].Code != http.StatusConflict {
				t.Fatalf("concurrent results: %d %s; %d %s", responses[0].Code, responses[0].Body, responses[1].Code, responses[1].Body)
			}
			var conflict CandidateConflict
			if err := json.Unmarshal(responses[loser].Body.Bytes(), &conflict); err != nil {
				t.Fatal(err)
			}
			if conflict.Code != "working_copy_conflict" || conflict.CurrentVersion == nil || strconv.FormatInt(*conflict.CurrentVersion, 10) != responses[winner].Header().Get("X-Working-Copy-Version") {
				t.Fatalf("conflict lacks the winning version: %+v", conflict)
			}
			page := fetchStartedAsset(t, r, session, started.ID)
			wantName := "First editor"
			wantBlurb := "First pitch"
			if winner == 1 {
				wantName = "Second editor"
				wantBlurb = "Second pitch"
				if publish {
					wantName = "Ilse of the west shelf"
					wantBlurb = ""
				}
			}
			if page.Name != wantName || page.Blurb != wantBlurb {
				t.Fatalf("saved identity = %q / %q, want %q / %q", page.Name, page.Blurb, wantName, wantBlurb)
			}
			wantLifecycle := "draft"
			if publish && winner == 1 {
				wantLifecycle = "published"
			}
			if page.Lifecycle != wantLifecycle {
				t.Fatalf("lifecycle = %q, want %q", page.Lifecycle, wantLifecycle)
			}
		})
	}
}
