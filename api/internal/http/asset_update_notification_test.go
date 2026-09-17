package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
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
	unlisted := apitest.Send(t, s.router, apitest.AuthorizedJSONRequest(
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

func TestUpdatesFoldIntoOneUnreadEntryThatTheNextReadUnfolds(t *testing.T) {
	t.Parallel()
	s := newUpdateInboxStack(t)
	watcher := s.reader(t, "watcher@example.com", "moon.watcher")
	s.watch(t, watcher)

	for _, summary := range []string{"Rewrote her opening", "Softened her temper", "Added a greeting"} {
		s.describe(t, summary)
		s.publishUpdate(t, fmt.Sprintf(`{"summary":%q}`, summary))
		s.fanOut(t, time.Now())
	}

	page := s.inbox(t, watcher, "")
	if len(page.Items) != 1 {
		t.Fatalf("three updates left %d entries, want one: %+v", len(page.Items), page.Items)
	}
	folded := page.Items[0]
	if folded.Update == nil || folded.Update.Count != 3 || folded.Update.Number != 4 ||
		folded.Update.Summary != "Added a greeting" {
		t.Fatalf("folded entry = %+v, want three updates showing the latest", folded.Update)
	}
	if got := s.unread(t, watcher); got != 1 {
		t.Fatalf("unread = %d, want 1", got)
	}

	if got := s.markRead(t, watcher, folded.ID); got != http.StatusNoContent {
		t.Fatalf("mark read = %d, want 204", got)
	}
	s.describe(t, "One more change")
	s.publishUpdate(t, `{"summary":"One more change"}`)
	s.fanOut(t, time.Now())

	after := s.inbox(t, watcher, "")
	if len(after.Items) != 2 {
		t.Fatalf("after reading the folded entry the inbox holds %d entries, want two", len(after.Items))
	}
	if newest := after.Items[0]; newest.Update == nil || newest.Update.Count != 1 ||
		newest.Update.Summary != "One more change" || newest.ReadAt != nil {
		t.Fatalf("the update after a read entry = %+v, want a new unread entry", newest)
	}
	if kept := after.Items[1]; kept.ID != folded.ID || kept.Update.Count != 3 {
		t.Fatalf("the read entry changed: %+v", kept)
	}
}

func TestAFoldedEntryMovesBackToTheTopOfTheInbox(t *testing.T) {
	t.Parallel()
	s := newUpdateInboxStack(t)
	other := s.with(apitest.PublishedAsset(t, s.router, s.creator))
	watcher := s.reader(t, "watcher@example.com", "moon.watcher")
	s.watch(t, watcher)
	other.watch(t, watcher)

	s.describe(t, "The first change")
	s.publishUpdate(t, `{"summary":"The first change"}`)
	other.describe(t, "A change to the other asset")
	other.publishUpdate(t, `{"summary":"A change to the other asset"}`)
	s.fanOut(t, time.Now())
	if top := s.inbox(t, watcher, "").Items[0]; top.Asset.ID != other.assetID {
		t.Fatalf("the newer asset is not on top: %+v", top)
	}

	s.describe(t, "The second change")
	s.publishUpdate(t, `{"summary":"The second change"}`)
	s.fanOut(t, time.Now())

	page := s.inbox(t, watcher, "")
	if len(page.Items) != 2 || page.Items[0].Asset.ID != s.assetID ||
		page.Items[0].Update.Count != 2 || page.Items[0].Update.Summary != "The second change" {
		t.Fatalf("inbox = %+v, want the folded entry back on top", page.Items)
	}
}

func TestAFoldedEntrysNinetyDaysRunFromTheUpdateItLastAbsorbed(t *testing.T) {
	t.Parallel()
	s := newUpdateInboxStack(t)
	watcher := s.reader(t, "watcher@example.com", "moon.watcher")
	s.watch(t, watcher)

	s.describe(t, "The first change")
	s.publishUpdate(t, `{"summary":"The first change"}`)
	s.fanOut(t, time.Now())
	arrived := s.inbox(t, watcher, "").Items[0].CreatedAt

	s.describe(t, "The second change")
	s.publishUpdate(t, `{"summary":"The second change"}`)
	s.fanOut(t, time.Now())
	absorbed := s.inbox(t, watcher, "").Items[0].CreatedAt

	waited := absorbed.Sub(arrived)
	if waited <= 0 {
		t.Fatalf("the folded entry still reads as arriving at %s", arrived)
	}
	s.sweep(t, arrived.Add(notify.Retention).Add(waited/2))
	if kept := s.inbox(t, watcher, ""); len(kept.Items) != 1 {
		t.Fatalf("the sweeper counted the folded entry from the update it replaced")
	}
	s.sweep(t, absorbed.Add(notify.Retention).Add(waited))
	if swept := s.inbox(t, watcher, ""); len(swept.Items) != 0 {
		t.Fatalf("the folded entry outlived its ninety days: %+v", swept.Items)
	}
}

func TestAnEntryAboutAWithheldOrDeletedAssetLeavesEveryInboxButTheOwners(t *testing.T) {
	t.Parallel()
	s := newUpdateInboxStack(t)
	watcher := s.reader(t, "watcher@example.com", "moon.watcher")
	s.watch(t, watcher)
	s.describe(t, "A change everyone should hear about")
	s.publishUpdate(t, `{"summary":"A change everyone should hear about"}`)
	s.fanOut(t, time.Now())
	if page := s.inbox(t, watcher, ""); len(page.Items) != 1 {
		t.Fatalf("the watcher has %d entries before the withhold, want one", len(page.Items))
	}

	s.withhold(t, s.assetID, "Copyright report under review")
	s.fanOut(t, time.Now())
	if page := s.inbox(t, watcher, ""); len(page.Items) != 0 {
		t.Fatalf("a withheld asset left %+v in the watcher's inbox", page.Items)
	}
	if got := s.unread(t, watcher); got != 0 {
		t.Fatalf("a withheld asset counts %d unread for the watcher, want 0", got)
	}
	owner := s.inbox(t, s.creator, "")
	if len(owner.Items) != 1 || owner.Items[0].Type != "asset_withheld" {
		t.Fatalf("the owner's inbox = %+v, want the withheld entry", owner.Items)
	}

	s.restore(t, s.assetID)
	s.fanOut(t, time.Now())
	if page := s.inbox(t, watcher, ""); len(page.Items) != 1 || s.unread(t, watcher) != 1 {
		t.Fatalf("restoring the asset left the watcher %+v", page.Items)
	}

	s.deleteAsset(t)
	if page := s.inbox(t, watcher, ""); len(page.Items) != 0 || s.unread(t, watcher) != 0 {
		t.Fatalf("a deleted asset left %+v in the watcher's inbox", page.Items)
	}
	s.restoreAsset(t)
	if page := s.inbox(t, watcher, ""); len(page.Items) != 1 || s.unread(t, watcher) != 1 {
		t.Fatalf("recovering the asset left the watcher %+v", page.Items)
	}
}

func TestAnUpdateEntryOffersASendToEachInstanceHoldingAnOlderCopy(t *testing.T) {
	t.Parallel()
	s := newUpdateInboxStack(t)
	reader := s.reader(t, "reader@example.com", "moon.reader")
	desk := s.install(t, reader, "Reading desk")
	laptop := s.install(t, reader, "Travel laptop")
	tablet := apitest.LinkDeviceInstance(t, s.router, reader, "Lumiverse", "Old tablet", []string{apitest.LibrarySyncScope})
	apitest.ReportInstalled(t, s.router, tablet.AccessToken, "", s.assetID)

	s.describe(t, "A change worth sending on")
	s.publishUpdate(t, `{"summary":"A change worth sending on"}`)
	current := s.install(t, reader, "Up-to-date shelf")
	s.fanOut(t, time.Now())

	entry := s.inbox(t, reader, "").Items[0]
	var offered []string
	for _, target := range entry.SendTargets {
		offered = append(offered, target.ApplicationName+" — "+target.InstanceName)
	}
	want := []string{"Lumiverse — Reading desk", "Lumiverse — Travel laptop"}
	if strings.Join(offered, ", ") != strings.Join(want, ", ") {
		t.Fatalf("the entry offers %q, want %q", offered, want)
	}
	for _, sent := range entry.SendTargets {
		if sent.InstanceID != desk.Instance.ID && sent.InstanceID != laptop.Instance.ID {
			t.Fatalf("the entry offers an instance it should not: %+v", sent)
		}
		if queued := sendToInstance(t, s.router, reader, s.assetID, sent.InstanceID); queued.Code != http.StatusAccepted {
			t.Fatalf("sending from the entry = %d, want 202: %s", queued.Code, queued.Body.String())
		}
	}
	if current.Instance.ID == "" {
		t.Fatal("the up-to-date instance was never linked")
	}

	owner := s.inbox(t, s.creator, "")
	if len(owner.Items) != 0 {
		t.Fatalf("the owner heard about their own update: %+v", owner.Items)
	}
}

type updateInboxStack struct {
	inboxStack
	assetID string
}

func newUpdateInboxStack(t *testing.T) updateInboxStack {
	t.Helper()
	s := newInboxStack(t)
	return updateInboxStack{inboxStack: s, assetID: apitest.PublishedAsset(t, s.router, s.creator)}
}

func (s updateInboxStack) reader(t *testing.T, email, handle string) *http.Cookie {
	t.Helper()
	return apitest.VerifiedSignUp(t, s.router, s.outbox, email, handle)
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
	response := apitest.Send(t, s.router, apitest.Authorized(httptest.NewRequest(method, "/v1/assets/"+s.assetID+"/watch", nil), session))
	if response.Code != http.StatusOK {
		t.Fatalf("%s watch status = %d, want 200: %s", method, response.Code, response.Body.String())
	}
}

func (s updateInboxStack) install(t *testing.T, session *http.Cookie, instance string) apitest.TokenGrant {
	t.Helper()
	grant := apitest.LinkDeviceInstance(t, s.router, session, "Lumiverse", instance, []string{apitest.ReceiveScope, apitest.LibrarySyncScope})
	declareTargets(t, s.router, grant.AccessToken, []string{"test_opaque"})
	apitest.ReportInstalled(t, s.router, grant.AccessToken, "", s.assetID)
	return grant
}

// with points the same stack at another of the creator's assets.
func (s updateInboxStack) with(assetID string) updateInboxStack {
	return updateInboxStack{inboxStack: s.inboxStack, assetID: assetID}
}

func (s updateInboxStack) deleteAsset(t *testing.T) {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.BrowserRequest(
		t, http.MethodDelete, "/v1/assets/"+s.assetID, nil, s.creator,
	))
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204: %s", response.Code, response.Body.String())
	}
}

func (s updateInboxStack) restoreAsset(t *testing.T) {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.BrowserRequest(
		t, http.MethodPost, "/v1/assets/"+s.assetID+"/restore", nil, s.creator,
	))
	if response.Code != http.StatusNoContent {
		t.Fatalf("restore status = %d, want 204: %s", response.Code, response.Body.String())
	}
}

func (s updateInboxStack) describe(t *testing.T, text string) {
	t.Helper()
	started := fetchStartedAsset(t, s.router, s.creator, s.assetID)
	coreBlock := apitest.BlockNamed(t, started.Blocks, "character_core")
	core := apitest.EditableBlock(coreBlock)
	core.Elements[0].Content = json.RawMessage(fmt.Sprintf(`{"text":%q}`, text))
	if got := apitest.SaveBlock(t, s.router, s.creator, s.assetID, coreBlock.ID, core); got.Code != http.StatusOK {
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
