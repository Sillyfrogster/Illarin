package notify_test

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

func TestFollowersAndInstallersHearAboutAnUpdateAndTheOwnerDoesNot(t *testing.T) {
	t.Parallel()
	s := newUpdateInboxStack(t)
	follower := s.reader(t, "follower@example.com", "moon.follower")
	s.follow(t, follower)
	installer := s.reader(t, "installer@example.com", "moon.installer")
	s.install(t, installer, "Reading desk")
	stopped := s.reader(t, "stopped@example.com", "moon.stopped")
	s.install(t, stopped, "Travel laptop")
	s.stopFollowing(t, stopped)
	bystander := s.reader(t, "bystander@example.com", "moon.bystander")

	s.describe(t, "She keeps the books that remember themselves.")
	s.publishUpdate(t, `{"summary":"Rewrote her opening","versionLabel":"v2"}`)
	s.fanOut(t, time.Now())

	for _, hearer := range []struct {
		name    string
		session *http.Cookie
	}{{"follower", follower}, {"installer", installer}} {
		page := s.inbox(t, hearer.session, "")
		if len(page.Items) != 1 {
			t.Fatalf("the %s has %d entries, want 1: %+v", hearer.name, len(page.Items), page.Items)
		}
		entry := page.Items[0]
		if entry.Type != "work_updated" || entry.Work == nil || entry.Work.ID != s.workID ||
			entry.Work.Name != "Ilse of the west shelf" || entry.Update == nil ||
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
	}{{"owner", s.creator}, {"stopped follower", stopped}, {"bystander", bystander}} {
		if page := s.inbox(t, silent.session, ""); len(page.Items) != 0 {
			t.Errorf("the %s has %d entries, want none: %+v", silent.name, len(page.Items), page.Items)
		}
	}
}

func TestAQuietOrContentFreeUpdateTellsNoOne(t *testing.T) {
	t.Parallel()
	s := newUpdateInboxStack(t)
	follower := s.reader(t, "follower@example.com", "moon.follower")
	s.follow(t, follower)

	s.describe(t, "A private save that nobody hears about.")
	s.fanOut(t, time.Now())
	if page := s.inbox(t, follower, ""); len(page.Items) != 0 {
		t.Fatalf("a private save reached the follower: %+v", page.Items)
	}

	s.publishUpdate(t, `{"summary":"Kept quiet","notify":false}`)
	s.fanOut(t, time.Now())
	if page := s.inbox(t, follower, ""); len(page.Items) != 0 {
		t.Fatalf("a quiet update reached the follower: %+v", page.Items)
	}

	s.resizeMessages(t)
	s.publishUpdate(t, `{"summary":"Tidied the page"}`)
	s.fanOut(t, time.Now())
	if page := s.inbox(t, follower, ""); len(page.Items) != 0 {
		t.Fatalf("resizing a block reached the follower: %+v", page.Items)
	}

	s.describe(t, "A change everyone should hear about.")
	s.publishUpdate(t, `{"summary":"No integrations, still told","integrationIds":[]}`)
	s.fanOut(t, time.Now())
	page := s.inbox(t, follower, "")
	if len(page.Items) != 1 || page.Items[0].Update == nil || page.Items[0].Update.Number != 4 ||
		page.Items[0].Update.Summary != "No integrations, still told" {
		t.Fatalf("an update with no integrations gave the follower %+v, want update 4", page.Items)
	}
}

func TestAnUnlistedWorkStillTellsItsFollowers(t *testing.T) {
	t.Parallel()
	s := newUpdateInboxStack(t)
	unlisted := apitest.Send(t, s.router, apitest.AuthorizedJSONRequest(
		t, http.MethodPut, "/v1/works/"+s.workID+"/visibility", `{"visibility":"unlisted"}`, s.creator,
	))
	if unlisted.Code != http.StatusNoContent {
		t.Fatalf("unlist status = %d, want 204: %s", unlisted.Code, unlisted.Body.String())
	}
	follower := s.reader(t, "follower@example.com", "moon.follower")
	s.follow(t, follower)

	s.describe(t, "Unlisted, and still updated.")
	s.publishUpdate(t, `{"summary":"Unlisted update"}`)
	s.fanOut(t, time.Now())

	page := s.inbox(t, follower, "")
	if len(page.Items) != 1 || page.Items[0].Type != "work_updated" ||
		page.Items[0].Update == nil || page.Items[0].Update.Summary != "Unlisted update" {
		t.Fatalf("the follower of an unlisted work has %+v, want one update entry", page.Items)
	}
}

func TestUpdatesFoldIntoOneUnreadEntryThatTheNextReadUnfolds(t *testing.T) {
	t.Parallel()
	s := newUpdateInboxStack(t)
	follower := s.reader(t, "follower@example.com", "moon.follower")
	s.follow(t, follower)

	for _, summary := range []string{"Rewrote her opening", "Softened her temper", "Added a greeting"} {
		s.describe(t, summary)
		s.publishUpdate(t, fmt.Sprintf(`{"summary":%q}`, summary))
		s.fanOut(t, time.Now())
	}

	page := s.inbox(t, follower, "")
	if len(page.Items) != 1 {
		t.Fatalf("three updates left %d entries, want one: %+v", len(page.Items), page.Items)
	}
	folded := page.Items[0]
	if folded.Update == nil || folded.Update.Count != 3 || folded.Update.Number != 4 ||
		folded.Update.Summary != "Added a greeting" {
		t.Fatalf("folded entry = %+v, want three updates showing the latest", folded.Update)
	}
	if got := s.unread(t, follower); got != 1 {
		t.Fatalf("unread = %d, want 1", got)
	}

	if got := s.markRead(t, follower, folded.ID); got != http.StatusNoContent {
		t.Fatalf("mark read = %d, want 204", got)
	}
	s.describe(t, "One more change")
	s.publishUpdate(t, `{"summary":"One more change"}`)
	s.fanOut(t, time.Now())

	after := s.inbox(t, follower, "")
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
	other := s.with(apitest.PublishedCharacter(t, s.router, s.creator))
	follower := s.reader(t, "follower@example.com", "moon.follower")
	s.follow(t, follower)
	other.follow(t, follower)

	s.describe(t, "The first change")
	s.publishUpdate(t, `{"summary":"The first change"}`)
	other.describe(t, "A change to the other asset")
	other.publishUpdate(t, `{"summary":"A change to the other asset"}`)
	s.fanOut(t, time.Now())
	if top := s.inbox(t, follower, "").Items[0]; top.Work.ID != other.workID {
		t.Fatalf("the newer work is not on top: %+v", top)
	}

	s.describe(t, "The second change")
	s.publishUpdate(t, `{"summary":"The second change"}`)
	s.fanOut(t, time.Now())

	page := s.inbox(t, follower, "")
	if len(page.Items) != 2 || page.Items[0].Work.ID != s.workID ||
		page.Items[0].Update.Count != 2 || page.Items[0].Update.Summary != "The second change" {
		t.Fatalf("inbox = %+v, want the folded entry back on top", page.Items)
	}
}

func TestAFoldedEntrysNinetyDaysRunFromTheUpdateItLastAbsorbed(t *testing.T) {
	t.Parallel()
	s := newUpdateInboxStack(t)
	follower := s.reader(t, "follower@example.com", "moon.follower")
	s.follow(t, follower)

	s.describe(t, "The first change")
	s.publishUpdate(t, `{"summary":"The first change"}`)
	s.fanOut(t, time.Now())
	arrived := s.inbox(t, follower, "").Items[0].CreatedAt

	s.describe(t, "The second change")
	s.publishUpdate(t, `{"summary":"The second change"}`)
	s.fanOut(t, time.Now())
	absorbed := s.inbox(t, follower, "").Items[0].CreatedAt

	waited := absorbed.Sub(arrived)
	if waited <= 0 {
		t.Fatalf("the folded entry still reads as arriving at %s", arrived)
	}
	s.cleanup(t, arrived.Add(notify.Retention).Add(waited/2))
	if kept := s.inbox(t, follower, ""); len(kept.Items) != 1 {
		t.Fatalf("the cleanup counted the folded entry from the update it replaced")
	}
	s.cleanup(t, absorbed.Add(notify.Retention).Add(waited))
	if emptied := s.inbox(t, follower, ""); len(emptied.Items) != 0 {
		t.Fatalf("the folded entry outlived its ninety days: %+v", emptied.Items)
	}
}

func TestAnEntryAboutATakenDownOrDeletedWorkLeavesEveryInboxButTheOwners(t *testing.T) {
	t.Parallel()
	s := newUpdateInboxStack(t)
	follower := s.reader(t, "follower@example.com", "moon.follower")
	s.follow(t, follower)
	s.describe(t, "A change everyone should hear about")
	s.publishUpdate(t, `{"summary":"A change everyone should hear about"}`)
	s.fanOut(t, time.Now())
	if page := s.inbox(t, follower, ""); len(page.Items) != 1 {
		t.Fatalf("the follower has %d entries before the takeDown, want one", len(page.Items))
	}

	s.takeDown(t, s.workID, "Copyright report under review")
	s.fanOut(t, time.Now())
	if page := s.inbox(t, follower, ""); len(page.Items) != 0 {
		t.Fatalf("a taken down work left %+v in the follower's inbox", page.Items)
	}
	if got := s.unread(t, follower); got != 0 {
		t.Fatalf("a taken down work counts %d unread for the follower, want 0", got)
	}
	owner := s.inbox(t, s.creator, "")
	if len(owner.Items) != 1 || owner.Items[0].Type != "work_taken_down" {
		t.Fatalf("the owner's inbox = %+v, want the taken down entry", owner.Items)
	}

	s.restore(t, s.workID)
	s.fanOut(t, time.Now())
	if page := s.inbox(t, follower, ""); len(page.Items) != 1 || s.unread(t, follower) != 1 {
		t.Fatalf("restoring the work left the follower %+v", page.Items)
	}

	s.deleteWork(t)
	if page := s.inbox(t, follower, ""); len(page.Items) != 0 || s.unread(t, follower) != 0 {
		t.Fatalf("a deleted work left %+v in the follower's inbox", page.Items)
	}
	s.restoreWork(t)
	if page := s.inbox(t, follower, ""); len(page.Items) != 1 || s.unread(t, follower) != 1 {
		t.Fatalf("recovering the work left the follower %+v", page.Items)
	}
}

func TestAnUpdateEntryOffersASendToEachConnectedAppHoldingAnOlderCopy(t *testing.T) {
	t.Parallel()
	s := newUpdateInboxStack(t)
	reader := s.reader(t, "reader@example.com", "moon.reader")
	desk := s.install(t, reader, "Reading desk")
	laptop := s.install(t, reader, "Travel laptop")
	tablet := apitest.ConnectApp(t, s.router, reader, "Lumiverse", "Old tablet", []string{apitest.LibrarySyncPermission})
	apitest.ReportLibrary(t, s.router, tablet.AccessToken, "", s.workID)

	s.describe(t, "A change worth sending on")
	s.publishUpdate(t, `{"summary":"A change worth sending on"}`)
	current := s.install(t, reader, "Up-to-date shelf")
	s.fanOut(t, time.Now())

	entry := s.inbox(t, reader, "").Items[0]
	var offered []string
	for _, app := range entry.SendTo {
		offered = append(offered, app.AppName+" — "+app.Name)
	}
	want := []string{"Lumiverse — Reading desk", "Lumiverse — Travel laptop"}
	if strings.Join(offered, ", ") != strings.Join(want, ", ") {
		t.Fatalf("the entry offers %q, want %q", offered, want)
	}
	raw := apitest.Send(t, s.router, apitest.Authorized(httptest.NewRequest(http.MethodGet, "/v1/notifications", nil), reader)).Body.String()
	if !strings.Contains(raw, `"sendTargets":[{`) || !strings.Contains(raw, `"instanceId":"`+desk.ConnectedApp.ID+`"`) {
		t.Fatalf("the entry = %s, want sendTargets and instanceId repeating the new names", raw)
	}
	for _, sent := range entry.SendTo {
		if sent.ConnectedAppID != desk.ConnectedApp.ID && sent.ConnectedAppID != laptop.ConnectedApp.ID {
			t.Fatalf("the entry offers a connected app it should not: %+v", sent)
		}
		if queued := apitest.SendToApp(t, s.router, reader, s.workID, sent.ConnectedAppID); queued.Code != http.StatusAccepted {
			t.Fatalf("sending from the entry = %d, want 202: %s", queued.Code, queued.Body.String())
		}
	}
	if current.ConnectedApp.ID == "" {
		t.Fatal("the up-to-date connected app was never connected")
	}

	owner := s.inbox(t, s.creator, "")
	if len(owner.Items) != 0 {
		t.Fatalf("the owner heard about their own update: %+v", owner.Items)
	}
}

type updateInboxStack struct {
	inboxStack
	workID string
}

func newUpdateInboxStack(t *testing.T) updateInboxStack {
	t.Helper()
	s := newInboxStack(t)
	return updateInboxStack{inboxStack: s, workID: apitest.PublishedCharacter(t, s.router, s.creator)}
}

func (s updateInboxStack) reader(t *testing.T, email, handle string) *http.Cookie {
	t.Helper()
	return apitest.VerifiedSignUp(t, s.router, s.outbox, email, handle)
}

func (s updateInboxStack) follow(t *testing.T, session *http.Cookie) {
	t.Helper()
	s.setFollow(t, session, http.MethodPut)
}

func (s updateInboxStack) stopFollowing(t *testing.T, session *http.Cookie) {
	t.Helper()
	s.setFollow(t, session, http.MethodDelete)
}

func (s updateInboxStack) setFollow(t *testing.T, session *http.Cookie, method string) {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(httptest.NewRequest(method, "/v1/works/"+s.workID+"/follow", nil), session))
	if response.Code != http.StatusOK {
		t.Fatalf("%s follow status = %d, want 200: %s", method, response.Code, response.Body.String())
	}
}

func (s updateInboxStack) install(t *testing.T, session *http.Cookie, name string) apitest.AppCredentials {
	t.Helper()
	credentials := apitest.ConnectApp(t, s.router, session, "Lumiverse", name, []string{apitest.ReceivePermission, apitest.LibrarySyncPermission})
	apitest.DeclareFormats(t, s.router, credentials.AccessToken, []string{"test_opaque"})
	apitest.ReportLibrary(t, s.router, credentials.AccessToken, "", s.workID)
	return credentials
}

// with points the same stack at another of the creator's works.
func (s updateInboxStack) with(workID string) updateInboxStack {
	return updateInboxStack{inboxStack: s.inboxStack, workID: workID}
}

func (s updateInboxStack) deleteWork(t *testing.T) {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.BrowserRequest(
		t, http.MethodDelete, "/v1/works/"+s.workID, nil, s.creator,
	))
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204: %s", response.Code, response.Body.String())
	}
}

func (s updateInboxStack) restoreWork(t *testing.T) {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.BrowserRequest(
		t, http.MethodPost, "/v1/works/"+s.workID+"/restore", nil, s.creator,
	))
	if response.Code != http.StatusNoContent {
		t.Fatalf("restore status = %d, want 204: %s", response.Code, response.Body.String())
	}
}

func (s updateInboxStack) describe(t *testing.T, text string) {
	t.Helper()
	started := apitest.FetchStartedWork(t, s.router, s.creator, s.workID)
	coreBlock := apitest.BlockNamed(t, started.Blocks, "character_core")
	core := apitest.EditableBlock(coreBlock)
	core.Elements[0].Content = json.RawMessage(fmt.Sprintf(`{"text":%q}`, text))
	if got := apitest.SaveBlock(t, s.router, s.creator, s.workID, coreBlock.ID, core); got.Code != http.StatusOK {
		t.Fatalf("save the description = %d, want 200: %s", got.Code, got.Body.String())
	}
}

func (s updateInboxStack) resizeMessages(t *testing.T) {
	t.Helper()
	started := apitest.FetchStartedWork(t, s.router, s.creator, s.workID)
	arranged := make([]apitest.ArrangedBlock, 0, len(started.Blocks))
	for _, block := range started.Blocks {
		width := block.Width
		if block.Definition == "messages" {
			width = map[string]string{"full": "half", "half": "full"}[width]
		}
		arranged = append(arranged, apitest.ArrangedBlock{ID: block.ID, Hidden: block.Hidden, Width: width})
	}
	if got := apitest.ArrangeBlocks(t, s.router, s.creator, s.workID, arranged); got.Code != http.StatusOK {
		t.Fatalf("resize the messages block = %d, want 200: %s", got.Code, got.Body.String())
	}
}

func (s updateInboxStack) publishUpdate(t *testing.T, body string) {
	t.Helper()
	if got := apitest.PublishWorkVersion(t, s.router, s.creator, s.workID, body); got.Code != http.StatusOK {
		t.Fatalf("publish an update = %d, want 200: %s", got.Code, got.Body.String())
	}
}
