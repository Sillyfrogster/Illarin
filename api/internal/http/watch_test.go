package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAReaderWatchesAnAssetAndStopsWatchingIt(t *testing.T) {
	t.Parallel()
	s := newWatchStack(t)
	if got := s.watchState(t, s.reader); got == nil || got.State != "none" {
		t.Fatalf("watch before watching = %+v, want none", got)
	}

	if got := s.setWatch(t, s.reader, http.MethodPut); got.State != "watching" {
		t.Fatalf("watching answered %+v, want watching", got)
	}
	if got := s.watchState(t, s.reader); got == nil || got.State != "watching" {
		t.Fatalf("watch after watching = %+v, want watching", got)
	}

	if got := s.setWatch(t, s.reader, http.MethodDelete); got.State != "stopped" {
		t.Fatalf("stopping answered %+v, want stopped", got)
	}
	if got := s.watchState(t, s.reader); got == nil || got.State != "stopped" {
		t.Fatalf("watch after stopping = %+v, want stopped", got)
	}

	if got := s.setWatch(t, s.reader, http.MethodPut); got.State != "watching" {
		t.Fatalf("watching again answered %+v, want watching", got)
	}
}

func TestAnOwnerNeitherWatchesNorLearnsWhoWatchesTheirAsset(t *testing.T) {
	t.Parallel()
	s := newWatchStack(t)
	s.setWatch(t, s.reader, http.MethodPut)

	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		if got := s.watchAt(t, s.creator, method, s.assetID); got.Code != http.StatusForbidden {
			t.Errorf("%s a watch on your own asset = %d, want 403: %s", method, got.Code, got.Body.String())
		}
	}
	page := send(t, s.router, authorized(httptest.NewRequest(http.MethodGet, "/v1/assets/"+s.assetID, nil), s.creator))
	if page.Code != http.StatusOK {
		t.Fatalf("owner's asset page = %d, want 200: %s", page.Code, page.Body.String())
	}
	if body := page.Body.String(); strings.Contains(body, `"watch"`) || strings.Contains(body, readerHandle) {
		t.Fatalf("the owner's asset page says something about watches: %s", body)
	}
}

func TestASignedOutReaderHasNoWatch(t *testing.T) {
	t.Parallel()
	s := newWatchStack(t)
	if got := s.watchState(t, nil); got != nil {
		t.Fatalf("signed-out watch = %+v, want none returned", got)
	}
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		response := send(t, s.router, browserMutation(httptest.NewRequest(method, "/v1/assets/"+s.assetID+"/watch", nil)))
		if response.Code != http.StatusUnauthorized {
			t.Errorf("signed-out %s watch = %d, want 401", method, response.Code)
		}
	}
}

func TestADraftCannotBeWatchedButAnUnlistedAssetCan(t *testing.T) {
	t.Parallel()
	s := newWatchStack(t)
	draft := startCharacter(t, s.router, s.creator)
	for _, assetID := range []string{draft.ID, "44444444-4444-4444-8444-444444444444"} {
		if got := s.watchAt(t, s.reader, http.MethodPut, assetID); got.Code != http.StatusNotFound {
			t.Errorf("watching %s = %d, want 404: %s", assetID, got.Code, got.Body.String())
		}
	}

	unlisted := send(t, s.router, authorizedJSONRequest(
		t, http.MethodPut, "/v1/assets/"+s.assetID+"/discovery", `{"discovery":"unlisted"}`, s.creator,
	))
	if unlisted.Code != http.StatusNoContent {
		t.Fatalf("unlist status = %d, want 204: %s", unlisted.Code, unlisted.Body.String())
	}
	if got := s.setWatch(t, s.reader, http.MethodPut); got.State != "watching" {
		t.Fatalf("watching an unlisted asset answered %+v, want watching", got)
	}
}

func TestAnInstallReportedByLibrarySyncCountsAsWatchingUntilTheReaderStops(t *testing.T) {
	t.Parallel()
	s := newWatchStack(t)
	studio := linkDeviceInstance(t, s.router, s.creator, "Lumiverse", "Studio", []string{receiveScope, librarySyncScope})
	reportInstalled(t, s.router, studio.AccessToken, "", s.assetID)
	if got := s.watchState(t, s.reader); got == nil || got.State != "none" || len(got.InstalledOn) != 0 {
		t.Fatalf("another account's install gave the reader %+v, want none", got)
	}

	desk := linkDeviceInstance(t, s.router, s.reader, "Lumiverse", "Reading desk", []string{receiveScope, librarySyncScope})
	reportInstalled(t, s.router, desk.AccessToken, "", s.assetID)
	if got := s.watchState(t, s.reader); got == nil || got.State != "installed" ||
		!slices.Equal(got.InstalledOn, []string{"Reading desk"}) {
		t.Fatalf("watch with the asset installed = %+v, want installed on Reading desk", got)
	}

	if got := s.setWatch(t, s.reader, http.MethodDelete); got.State != "stopped" {
		t.Fatalf("stopping an installed asset answered %+v, want stopped", got)
	}
	reportInstalled(t, s.router, desk.AccessToken, "", s.assetID)
	if got := s.watchState(t, s.reader); got == nil || got.State != "stopped" ||
		!slices.Equal(got.InstalledOn, []string{"Reading desk"}) {
		t.Fatalf("watch after the next library sync = %+v, want it still stopped", got)
	}

	if got := s.setWatch(t, s.reader, http.MethodPut); got.State != "watching" {
		t.Fatalf("watching again answered %+v, want watching", got)
	}
}

func TestARevokedInstanceNoLongerCountsAsAWatch(t *testing.T) {
	t.Parallel()
	s := newWatchStack(t)
	desk := linkDeviceInstance(t, s.router, s.reader, "Lumiverse", "Reading desk", []string{receiveScope, librarySyncScope})
	laptop := linkDeviceInstance(t, s.router, s.reader, "Lumiverse", "Travel laptop", []string{receiveScope, librarySyncScope})
	reportInstalled(t, s.router, desk.AccessToken, "", s.assetID)
	reportInstalled(t, s.router, laptop.AccessToken, "", s.assetID)
	if got := s.watchState(t, s.reader); got == nil || got.State != "installed" ||
		!slices.Equal(got.InstalledOn, []string{"Reading desk", "Travel laptop"}) {
		t.Fatalf("watch with two installs = %+v, want installed on both", got)
	}

	revoke := func(grant tokenGrant) {
		t.Helper()
		revoked := send(t, s.router, browserRequest(t, http.MethodDelete, "/v1/instances/"+grant.Instance.ID, nil, s.reader))
		if revoked.Code != http.StatusNoContent {
			t.Fatalf("revoke status = %d, want 204: %s", revoked.Code, revoked.Body.String())
		}
	}
	revoke(desk)
	if got := s.watchState(t, s.reader); got == nil || got.State != "installed" ||
		!slices.Equal(got.InstalledOn, []string{"Travel laptop"}) {
		t.Fatalf("watch after revoking the desk = %+v, want installed on the laptop", got)
	}
	revoke(laptop)
	if got := s.watchState(t, s.reader); got == nil || got.State != "none" || len(got.InstalledOn) != 0 {
		t.Fatalf("watch after revoking both = %+v, want none", got)
	}
}

const readerHandle = "moon.reader"

type watchStack struct {
	router  *gin.Engine
	creator *http.Cookie
	reader  *http.Cookie
	assetID string
}

type assetWatch struct {
	State       string   `json:"state"`
	InstalledOn []string `json:"installedOn"`
}

func newWatchStack(t *testing.T) watchStack {
	t.Helper()
	outbox := &verificationOutbox{}
	router, _, _ := newTestRouterWithSenderPoolAndHandlers(t, 1<<20, DefaultDeadlines(), outbox)
	creator := verifiedSignUp(t, router, outbox, "creator@example.com", creatorHandle)
	reader := verifiedSignUp(t, router, outbox, "reader@example.com", readerHandle)
	return watchStack{
		router: router, creator: creator, reader: reader,
		assetID: publishedTestAsset(t, router, creator),
	}
}

func (s watchStack) watchState(t *testing.T, session *http.Cookie) *assetWatch {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/assets/"+s.assetID, nil)
	if session != nil {
		request = authorized(request, session)
	}
	response := send(t, s.router, request)
	if response.Code != http.StatusOK {
		t.Fatalf("asset page status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var page struct {
		Watch *assetWatch `json:"watch"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode asset page: %v", err)
	}
	return page.Watch
}

func (s watchStack) watchAt(t *testing.T, session *http.Cookie, method, assetID string) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, authorized(httptest.NewRequest(method, "/v1/assets/"+assetID+"/watch", nil), session))
}

func (s watchStack) setWatch(t *testing.T, session *http.Cookie, method string) assetWatch {
	t.Helper()
	response := s.watchAt(t, session, method, s.assetID)
	if response.Code != http.StatusOK {
		t.Fatalf("%s watch status = %d, want 200: %s", method, response.Code, response.Body.String())
	}
	var watch assetWatch
	if err := json.Unmarshal(response.Body.Bytes(), &watch); err != nil {
		t.Fatalf("decode watch: %v", err)
	}
	return watch
}
