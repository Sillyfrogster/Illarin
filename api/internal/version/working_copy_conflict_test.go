package version_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
)

func TestWorkingCopySaveRequiresAReviewedVersion(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartPreset(t, r, session, "lumiverse")
	req := httptest.NewRequest(http.MethodPut, "/v1/assets/"+started.ID+"/identity", strings.NewReader(`{"name":"Unreviewed edit","blurb":"","isNsfw":false}`))
	req.Header.Set("Content-Type", "application/json")
	req = apitest.Authorized(req, session)
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("save without a version = %d, want 400: %s", response.Code, response.Body.String())
	}
}

func TestConcurrentWorkingCopyRequestsKeepOnlyTheWinningCandidate(t *testing.T) {
	t.Parallel()
	for _, publish := range []bool{false, true} {
		name := "two editors"
		if publish {
			name = "save against publication"
		}
		t.Run(name, func(t *testing.T) {
			r, session := harness.NewVerifiedRouter(t)
			started := apitest.StartCharacter(t, r, session)
			apitest.WriteCharacterFloor(t, r, session, started)
			first := apitest.Authorized(httptest.NewRequest(http.MethodPut, "/v1/assets/"+started.ID+"/identity", strings.NewReader(`{"name":"First editor","blurb":"First pitch","isNsfw":false}`)), session)
			apitest.WithReviewedVersion(t, r, first)
			second := apitest.Authorized(httptest.NewRequest(http.MethodPut, "/v1/assets/"+started.ID+"/identity", strings.NewReader(`{"name":"Second editor","blurb":"Second pitch","isNsfw":false}`)), session)
			if publish {
				second = apitest.Authorized(httptest.NewRequest(http.MethodPost, "/v1/assets/"+started.ID+"/publish", nil), session)
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
			var conflict work.CandidateConflict
			if err := json.Unmarshal(responses[loser].Body.Bytes(), &conflict); err != nil {
				t.Fatal(err)
			}
			if conflict.Code != "working_copy_conflict" || conflict.CurrentVersion == nil || strconv.FormatInt(*conflict.CurrentVersion, 10) != responses[winner].Header().Get("X-Working-Copy-Version") {
				t.Fatalf("conflict lacks the winning version: %+v", conflict)
			}
			page := apitest.FetchStartedAsset(t, r, session, started.ID)
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
