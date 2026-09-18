package notify_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/gin-gonic/gin"
)

func TestAReaderFollowesAnWorkAndStopsFollowingIt(t *testing.T) {
	t.Parallel()
	s := newFollowStack(t)
	if got := s.followState(t, s.reader); got == nil || got.State != "none" {
		t.Fatalf("follow before following = %+v, want none", got)
	}

	if got := s.setFollow(t, s.reader, http.MethodPut); got.State != "following" {
		t.Fatalf("following answered %+v, want following", got)
	}
	if got := s.followState(t, s.reader); got == nil || got.State != "following" {
		t.Fatalf("follow after following = %+v, want following", got)
	}

	if got := s.setFollow(t, s.reader, http.MethodDelete); got.State != "stopped" {
		t.Fatalf("stopping answered %+v, want stopped", got)
	}
	if got := s.followState(t, s.reader); got == nil || got.State != "stopped" {
		t.Fatalf("follow after stopping = %+v, want stopped", got)
	}

	if got := s.setFollow(t, s.reader, http.MethodPut); got.State != "following" {
		t.Fatalf("following again answered %+v, want following", got)
	}
}

func TestAnOwnerNeitherFollowesNorLearnsWhoFollowesTheirWork(t *testing.T) {
	t.Parallel()
	s := newFollowStack(t)
	s.setFollow(t, s.reader, http.MethodPut)

	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		if got := s.followAt(t, s.creator, method, s.workID); got.Code != http.StatusForbidden {
			t.Errorf("%s a follow on your own work = %d, want 403: %s", method, got.Code, got.Body.String())
		}
	}
	page := apitest.Send(t, s.router, apitest.Authorized(httptest.NewRequest(http.MethodGet, "/v1/works/"+s.workID, nil), s.creator))
	if page.Code != http.StatusOK {
		t.Fatalf("owner's work page = %d, want 200: %s", page.Code, page.Body.String())
	}
	if body := page.Body.String(); strings.Contains(body, `"watch"`) || strings.Contains(body, readerHandle) {
		t.Fatalf("the owner's work page says something about follows: %s", body)
	}
}

func TestASignedOutReaderHasNoFollow(t *testing.T) {
	t.Parallel()
	s := newFollowStack(t)
	if got := s.followState(t, nil); got != nil {
		t.Fatalf("signed-out follow = %+v, want none returned", got)
	}
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		response := apitest.Send(t, s.router, apitest.BrowserMutation(httptest.NewRequest(method, "/v1/works/"+s.workID+"/follow", nil)))
		if response.Code != http.StatusUnauthorized {
			t.Errorf("signed-out %s follow = %d, want 401", method, response.Code)
		}
	}
}

func TestADraftCannotBeFollowedButAnUnlistedWorkCan(t *testing.T) {
	t.Parallel()
	s := newFollowStack(t)
	draft := apitest.StartCharacter(t, s.router, s.creator)
	for _, workID := range []string{draft.ID, "44444444-4444-4444-8444-444444444444"} {
		if got := s.followAt(t, s.reader, http.MethodPut, workID); got.Code != http.StatusNotFound {
			t.Errorf("following %s = %d, want 404: %s", workID, got.Code, got.Body.String())
		}
	}

	unlisted := apitest.Send(t, s.router, apitest.AuthorizedJSONRequest(
		t, http.MethodPut, "/v1/works/"+s.workID+"/visibility", `{"visibility":"unlisted"}`, s.creator,
	))
	if unlisted.Code != http.StatusNoContent {
		t.Fatalf("unlist status = %d, want 204: %s", unlisted.Code, unlisted.Body.String())
	}
	if got := s.setFollow(t, s.reader, http.MethodPut); got.State != "following" {
		t.Fatalf("following an unlisted work answered %+v, want following", got)
	}
}

func TestAnInstallReportedByLibrarySyncCountsAsFollowingUntilTheReaderStops(t *testing.T) {
	t.Parallel()
	s := newFollowStack(t)
	studio := apitest.LinkDeviceInstance(t, s.router, s.creator, "Lumiverse", "Studio", []string{apitest.ReceiveScope, apitest.LibrarySyncScope})
	apitest.ReportInstalled(t, s.router, studio.AccessToken, "", s.workID)
	if got := s.followState(t, s.reader); got == nil || got.State != "none" || len(got.InstalledOn) != 0 {
		t.Fatalf("another account's install gave the reader %+v, want none", got)
	}

	desk := apitest.LinkDeviceInstance(t, s.router, s.reader, "Lumiverse", "Reading desk", []string{apitest.ReceiveScope, apitest.LibrarySyncScope})
	apitest.ReportInstalled(t, s.router, desk.AccessToken, "", s.workID)
	if got := s.followState(t, s.reader); got == nil || got.State != "installed" ||
		!slices.Equal(got.InstalledOn, []string{"Reading desk"}) {
		t.Fatalf("follow with the work installed = %+v, want installed on Reading desk", got)
	}

	if got := s.setFollow(t, s.reader, http.MethodDelete); got.State != "stopped" {
		t.Fatalf("stopping an installed work answered %+v, want stopped", got)
	}
	apitest.ReportInstalled(t, s.router, desk.AccessToken, "", s.workID)
	if got := s.followState(t, s.reader); got == nil || got.State != "stopped" ||
		!slices.Equal(got.InstalledOn, []string{"Reading desk"}) {
		t.Fatalf("follow after the next library sync = %+v, want it still stopped", got)
	}

	if got := s.setFollow(t, s.reader, http.MethodPut); got.State != "following" {
		t.Fatalf("following again answered %+v, want following", got)
	}
}

func TestARevokedInstanceNoLongerCountsAsAFollow(t *testing.T) {
	t.Parallel()
	s := newFollowStack(t)
	desk := apitest.LinkDeviceInstance(t, s.router, s.reader, "Lumiverse", "Reading desk", []string{apitest.ReceiveScope, apitest.LibrarySyncScope})
	laptop := apitest.LinkDeviceInstance(t, s.router, s.reader, "Lumiverse", "Travel laptop", []string{apitest.ReceiveScope, apitest.LibrarySyncScope})
	apitest.ReportInstalled(t, s.router, desk.AccessToken, "", s.workID)
	apitest.ReportInstalled(t, s.router, laptop.AccessToken, "", s.workID)
	if got := s.followState(t, s.reader); got == nil || got.State != "installed" ||
		!slices.Equal(got.InstalledOn, []string{"Reading desk", "Travel laptop"}) {
		t.Fatalf("follow with two installs = %+v, want installed on both", got)
	}

	revoke := func(grant apitest.TokenGrant) {
		t.Helper()
		revoked := apitest.Send(t, s.router, apitest.BrowserRequest(t, http.MethodDelete, "/v1/instances/"+grant.Instance.ID, nil, s.reader))
		if revoked.Code != http.StatusNoContent {
			t.Fatalf("revoke status = %d, want 204: %s", revoked.Code, revoked.Body.String())
		}
	}
	revoke(desk)
	if got := s.followState(t, s.reader); got == nil || got.State != "installed" ||
		!slices.Equal(got.InstalledOn, []string{"Travel laptop"}) {
		t.Fatalf("follow after revoking the desk = %+v, want installed on the laptop", got)
	}
	revoke(laptop)
	if got := s.followState(t, s.reader); got == nil || got.State != "none" || len(got.InstalledOn) != 0 {
		t.Fatalf("follow after revoking both = %+v, want none", got)
	}
}

const readerHandle = "moon.reader"

type followStack struct {
	router  *gin.Engine
	creator *http.Cookie
	reader  *http.Cookie
	workID  string
}

type workFollow struct {
	State       string   `json:"state"`
	InstalledOn []string `json:"installedOn"`
}

func newFollowStack(t *testing.T) followStack {
	t.Helper()
	outbox := &apitest.VerificationOutbox{}
	router, _, _ := harness.NewRouterWithSenderPoolAndServices(t, 1<<20, api.DefaultDeadlines(), outbox)
	creator := apitest.VerifiedSignUp(t, router, outbox, "creator@example.com", apitest.CreatorHandle)
	reader := apitest.VerifiedSignUp(t, router, outbox, "reader@example.com", readerHandle)
	return followStack{
		router: router, creator: creator, reader: reader,
		workID: apitest.PublishedCharacter(t, router, creator),
	}
}

func (s followStack) followState(t *testing.T, session *http.Cookie) *workFollow {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/works/"+s.workID, nil)
	if session != nil {
		request = apitest.Authorized(request, session)
	}
	response := apitest.Send(t, s.router, request)
	if response.Code != http.StatusOK {
		t.Fatalf("work page status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var page struct {
		Follow *workFollow `json:"watch"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode work page: %v", err)
	}
	return page.Follow
}

func (s followStack) followAt(t *testing.T, session *http.Cookie, method, workID string) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(httptest.NewRequest(method, "/v1/works/"+workID+"/follow", nil), session))
}

func (s followStack) setFollow(t *testing.T, session *http.Cookie, method string) workFollow {
	t.Helper()
	response := s.followAt(t, session, method, s.workID)
	if response.Code != http.StatusOK {
		t.Fatalf("%s follow status = %d, want 200: %s", method, response.Code, response.Body.String())
	}
	var follow workFollow
	if err := json.Unmarshal(response.Body.Bytes(), &follow); err != nil {
		t.Fatalf("decode follow: %v", err)
	}
	return follow
}
