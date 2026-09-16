package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWatchersAndInstallersHearAboutAnUpdateAndTheOwnerDoesNot(t *testing.T) {
	t.Parallel()
	s := newUpdateInboxStack(t)
	watcher := s.reader(t, "watcher@example.com", "moon.watcher")
	s.watch(t, watcher)
	installer := s.reader(t, "installer@example.com", "moon.installer")
	s.install(t, installer, "Reading desk")
	stopped := s.reader(t, "stopped@example.com", "moon.stopped")
	s.install(t, stopped, "Travel laptop")
	s.stopWatching(t, stopped)
	bystander := s.reader(t, "bystander@example.com", "moon.bystander")

	s.describe(t, "She keeps the books that remember themselves.")
	s.publishUpdate(t, `{"summary":"Rewrote her opening","versionLabel":"v2"}`)
	s.fanOut(t, time.Now())

	for _, hearer := range []struct {
		name    string
		session *http.Cookie
	}{{"watcher", watcher}, {"installer", installer}} {
		page := s.inbox(t, hearer.session, "")
		if len(page.Items) != 1 {
			t.Fatalf("the %s has %d entries, want 1: %+v", hearer.name, len(page.Items), page.Items)
		}
		entry := page.Items[0]
		if entry.Type != "asset_updated" || entry.Asset == nil || entry.Asset.ID != s.assetID ||
			entry.Asset.Name != "Ilse of the west shelf" || entry.Update == nil ||
			entry.Update.Number != 2 || entry.Update.VersionLabel != "v2" ||
			entry.Update.Summary != "Rewrote her opening" || entry.ReadAt != nil || entry.Reason != "" {
			t.Fatalf("the %s's entry = %+v", hearer.name, entry)
		}
		if got := s.unread(t, hearer.session); got != 1 {
			t.Fatalf("the %s has %d unread, want 1", hearer.name, got)
		}
	}
	for _, silent := range []struct {
		name    string
		session *http.Cookie
	}{{"owner", s.creator}, {"stopped watcher", stopped}, {"bystander", bystander}} {
		if page := s.inbox(t, silent.session, ""); len(page.Items) != 0 {
			t.Errorf("the %s has %d entries, want none: %+v", silent.name, len(page.Items), page.Items)
		}
	}
}

func TestAQuietOrContentFreeUpdateTellsNoOne(t *testing.T) {
	t.Parallel()
	s := newUpdateInboxStack(t)
	watcher := s.reader(t, "watcher@example.com", "moon.watcher")
	s.watch(t, watcher)

	s.describe(t, "A private save that nobody hears about.")
	s.fanOut(t, time.Now())
	if page := s.inbox(t, watcher, ""); len(page.Items) != 0 {
		t.Fatalf("a private save reached the watcher: %+v", page.Items)
	}

	s.publishUpdate(t, `{"summary":"Kept quiet","notify":false}`)
	s.fanOut(t, time.Now())
	if page := s.inbox(t, watcher, ""); len(page.Items) != 0 {
		t.Fatalf("a quiet update reached the watcher: %+v", page.Items)
	}

	s.resizeMessages(t)
	s.publishUpdate(t, `{"summary":"Tidied the page"}`)
	s.fanOut(t, time.Now())
	if page := s.inbox(t, watcher, ""); len(page.Items) != 0 {
		t.Fatalf("resizing a block reached the watcher: %+v", page.Items)
	}

	s.describe(t, "A change everyone should hear about.")
	s.publishUpdate(t, `{"summary":"No destinations, still told","destinationIds":[]}`)
	s.fanOut(t, time.Now())
	page := s.inbox(t, watcher, "")
	if len(page.Items) != 1 || page.Items[0].Update == nil || page.Items[0].Update.Number != 4 ||
		page.Items[0].Update.Summary != "No destinations, still told" {
		t.Fatalf("an update with no destinations gave the watcher %+v, want update 4", page.Items)
	}
}

func TestAnUnlistedAssetStillTellsItsWatchers(t *testing.T) {
	t.Parallel()
	s := newUpdateInboxStack(t)
	unlisted := send(t, s.router, authorizedJSONRequest(
		t, http.MethodPut, "/v1/assets/"+s.assetID+"/discovery", `{"discovery":"unlisted"}`, s.creator,
	))
	if unlisted.Code != http.StatusNoContent {
		t.Fatalf("unlist status = %d, want 204: %s", unlisted.Code, unlisted.Body.String())
	}
	watcher := s.reader(t, "watcher@example.com", "moon.watcher")
	s.watch(t, watcher)

	s.describe(t, "Unlisted, and still updated.")
	s.publishUpdate(t, `{"summary":"Unlisted update"}`)
	s.fanOut(t, time.Now())

	page := s.inbox(t, watcher, "")
	if len(page.Items) != 1 || page.Items[0].Type != "asset_updated" ||
		page.Items[0].Update == nil || page.Items[0].Update.Summary != "Unlisted update" {
		t.Fatalf("the watcher of an unlisted asset has %+v, want one update entry", page.Items)
	}
}

type updateInboxStack struct {
	inboxStack
	assetID string
}

func newUpdateInboxStack(t *testing.T) updateInboxStack {
	t.Helper()
	s := newInboxStack(t)
	return updateInboxStack{inboxStack: s, assetID: publishedTestAsset(t, s.router, s.creator)}
}

func (s updateInboxStack) reader(t *testing.T, email, handle string) *http.Cookie {
	t.Helper()
	return verifiedSignUp(t, s.router, s.outbox, email, handle)
}

func (s updateInboxStack) watch(t *testing.T, session *http.Cookie) {
	t.Helper()
	s.setWatch(t, session, http.MethodPut)
}

func (s updateInboxStack) stopWatching(t *testing.T, session *http.Cookie) {
	t.Helper()
	s.setWatch(t, session, http.MethodDelete)
}

func (s updateInboxStack) setWatch(t *testing.T, session *http.Cookie, method string) {
	t.Helper()
	response := send(t, s.router, authorized(httptest.NewRequest(method, "/v1/assets/"+s.assetID+"/watch", nil), session))
	if response.Code != http.StatusOK {
		t.Fatalf("%s watch status = %d, want 200: %s", method, response.Code, response.Body.String())
	}
}

func (s updateInboxStack) install(t *testing.T, session *http.Cookie, instance string) {
	t.Helper()
	grant := linkDeviceInstance(t, s.router, session, "Lumiverse", instance, []string{receiveScope, librarySyncScope})
	reportInstalled(t, s.router, grant.AccessToken, "", s.assetID)
}

func (s updateInboxStack) describe(t *testing.T, text string) {
	t.Helper()
	started := fetchStartedAsset(t, s.router, s.creator, s.assetID)
	coreBlock := blockNamed(t, started.Blocks, "character_core")
	core := editableBlock(coreBlock)
	core.Elements[0].Content = json.RawMessage(fmt.Sprintf(`{"text":%q}`, text))
	if got := saveBlock(t, s.router, s.creator, s.assetID, coreBlock.ID, core); got.Code != http.StatusOK {
		t.Fatalf("save the description = %d, want 200: %s", got.Code, got.Body.String())
	}
}

func (s updateInboxStack) resizeMessages(t *testing.T) {
	t.Helper()
	started := fetchStartedAsset(t, s.router, s.creator, s.assetID)
	arranged := make([]arrangedBlock, 0, len(started.Blocks))
	for _, block := range started.Blocks {
		width := block.Width
		if block.Definition == "messages" {
			width = map[string]string{"full": "half", "half": "full"}[width]
		}
		arranged = append(arranged, arrangedBlock{ID: block.ID, Hidden: block.Hidden, Width: width})
	}
	if got := arrangeBlocks(t, s.router, s.creator, s.assetID, arranged); got.Code != http.StatusOK {
		t.Fatalf("resize the messages block = %d, want 200: %s", got.Code, got.Body.String())
	}
}

func (s updateInboxStack) publishUpdate(t *testing.T, body string) {
	t.Helper()
	if got := publishAssetUpdate(t, s.router, s.creator, s.assetID, body); got.Code != http.StatusOK {
		t.Fatalf("publish an update = %d, want 200: %s", got.Code, got.Body.String())
	}
}
