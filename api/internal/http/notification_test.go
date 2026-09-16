package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/notification"
	"github.com/gin-gonic/gin"
)

func TestWithholdingAnAssetTellsItsOwnerWhyOnceTheFanOutRuns(t *testing.T) {
	t.Parallel()
	s := newInboxStack(t)
	assetID := s.upload(t, "Moonlit Archive")
	s.withhold(t, assetID, "Copyright report under review")

	if waiting := s.inbox(t, s.creator, ""); len(waiting.Items) != 0 {
		t.Fatalf("recording put %d entries in the inbox before the fan-out ran", len(waiting.Items))
	}
	s.fanOut(t, time.Now())

	response := send(t, s.router, authorized(httptest.NewRequest(http.MethodGet, "/v1/notifications", nil), s.creator))
	if response.Code != http.StatusOK {
		t.Fatalf("inbox status = %d, want 200: %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), staffHandle) {
		t.Fatalf("the owner's inbox names the staff member: %s", response.Body.String())
	}
	var page inboxPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode inbox: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("inbox has %d entries, want 1: %s", len(page.Items), response.Body.String())
	}
	entry := page.Items[0]
	if entry.Type != "asset_withheld" || entry.Asset == nil || entry.Asset.ID != assetID ||
		entry.Asset.Name != "Moonlit Archive" || entry.Reason != "Copyright report under review" ||
		entry.ReadAt != nil || entry.CreatedAt.IsZero() {
		t.Fatalf("withheld entry = %+v", entry)
	}
	if staffInbox := s.inbox(t, s.staff, ""); len(staffInbox.Items) != 0 {
		t.Fatalf("the staff member who acted has %d entries, want none", len(staffInbox.Items))
	}
}

func TestRestoringAWithheldAssetTellsItsOwnerItIsBack(t *testing.T) {
	t.Parallel()
	s := newInboxStack(t)
	assetID := s.upload(t, "Moonlit Archive")
	s.withhold(t, assetID, "Copyright report under review")
	s.restore(t, assetID)
	s.fanOut(t, time.Now())

	page := s.inbox(t, s.creator, "")
	if len(page.Items) != 2 || page.Items[0].Type != "asset_restored" || page.Items[1].Type != "asset_withheld" {
		t.Fatalf("inbox = %+v, want the restore above the withhold", page.Items)
	}
	restored := page.Items[0]
	if restored.Asset == nil || restored.Asset.ID != assetID || restored.Asset.Name != "Moonlit Archive" ||
		restored.Reason != "" {
		t.Fatalf("restored entry = %+v", restored)
	}
	if got := s.unread(t, s.creator); got != 2 {
		t.Fatalf("unread = %d, want 2", got)
	}
}

func TestRestrictingAProfileTellsItsOwnerWhyOnceTheFanOutRuns(t *testing.T) {
	t.Parallel()
	s := newInboxStack(t)
	s.restrictProfile(t, "Impersonating another creator")

	if waiting := s.inbox(t, s.creator, ""); len(waiting.Items) != 0 {
		t.Fatalf("recording put %d entries in the inbox before the fan-out ran", len(waiting.Items))
	}
	s.fanOut(t, time.Now())

	page := s.inbox(t, s.creator, "")
	if len(page.Items) != 1 {
		t.Fatalf("inbox has %d entries, want 1", len(page.Items))
	}
	entry := page.Items[0]
	if entry.Type != "profile_restricted" || entry.Asset != nil ||
		entry.Reason != "Impersonating another creator" || entry.ReadAt != nil || entry.CreatedAt.IsZero() {
		t.Fatalf("restricted entry = %+v", entry)
	}
	if staffInbox := s.inbox(t, s.staff, ""); len(staffInbox.Items) != 0 {
		t.Fatalf("the staff member who acted has %d entries, want none", len(staffInbox.Items))
	}
}

func TestRestoringAProfileTellsItsOwnerItIsTheirsToEditAgain(t *testing.T) {
	t.Parallel()
	s := newInboxStack(t)
	s.restrictProfile(t, "Impersonating another creator")
	if got := s.restoreProfile(t); got != http.StatusNoContent {
		t.Fatalf("restore status = %d, want 204", got)
	}
	if got := s.restoreProfile(t); got != http.StatusNotFound {
		t.Fatalf("restoring an unrestricted profile = %d, want 404", got)
	}
	s.fanOut(t, time.Now())

	page := s.inbox(t, s.creator, "")
	if len(page.Items) != 2 || page.Items[0].Type != "profile_restored" || page.Items[1].Type != "profile_restricted" {
		t.Fatalf("inbox = %+v, want one restore above the restriction", page.Items)
	}
	if restored := page.Items[0]; restored.Asset != nil || restored.Reason != "" {
		t.Fatalf("restored entry = %+v", restored)
	}
	if got := s.unread(t, s.creator); got != 2 {
		t.Fatalf("unread = %d, want 2", got)
	}
}

func TestNothingTheOwnerReadsNamesTheStaffMemberWhoActed(t *testing.T) {
	t.Parallel()
	s := newInboxStack(t)
	assetID := s.upload(t, "Moonlit Archive")
	s.withhold(t, assetID, "Copyright report under review")
	s.restrictProfile(t, "Impersonating another creator")
	s.fanOut(t, time.Now())

	for _, path := range []string{
		"/v1/notifications",
		"/v1/assets/" + assetID,
		"/v1/assets?creator=" + creatorHandle,
		"/v1/profiles/" + creatorHandle,
		"/v1/auth/session",
	} {
		response := send(t, s.router, authorized(httptest.NewRequest(http.MethodGet, path, nil), s.creator))
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200: %s", path, response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), staffHandle) {
			t.Errorf("GET %s names the staff member: %s", path, response.Body.String())
		}
	}

	record := send(t, s.router, authorized(httptest.NewRequest(
		http.MethodGet, "/v1/profiles/"+creatorHandle+"/restriction", nil,
	), s.staff))
	if record.Code != http.StatusOK || !strings.Contains(record.Body.String(), `"restrictedBy":"`+staffHandle+`"`) {
		t.Fatalf("the restriction record staff read = %d %s, want it to keep the actor", record.Code, record.Body.String())
	}
}

func TestAnEntryReadsAsItDidWhenTheChangeHappenedAfterARename(t *testing.T) {
	t.Parallel()
	s := newInboxStack(t)
	assetID := s.upload(t, "Moonlit Archive")
	s.withhold(t, assetID, "Copyright report under review")
	s.restore(t, assetID)
	if got := saveIdentity(t, s.router, s.creator, assetID,
		`{"name":"Sunlit Archive","blurb":"","isNsfw":false}`); got.Code != http.StatusNoContent {
		t.Fatalf("rename status = %d, want 204: %s", got.Code, got.Body.String())
	}
	if got := publishAssetUpdate(t, s.router, s.creator, assetID, `{"summary":"A new name"}`); got.Code != http.StatusOK {
		t.Fatalf("publish the rename = %d, want 200: %s", got.Code, got.Body.String())
	}
	s.fanOut(t, time.Now())
	s.withhold(t, assetID, "A second report")
	s.fanOut(t, time.Now())

	page := s.inbox(t, s.creator, "")
	var names []string
	for _, entry := range page.Items {
		names = append(names, entry.Type+" "+entry.Asset.Name)
	}
	want := []string{
		"asset_withheld Sunlit Archive", "asset_restored Moonlit Archive", "asset_withheld Moonlit Archive",
	}
	if strings.Join(names, ", ") != strings.Join(want, ", ") {
		t.Fatalf("inbox reads %q, want %q", names, want)
	}
}

func TestTheInboxPagesNewestFirstByCursor(t *testing.T) {
	t.Parallel()
	s := newInboxStack(t)
	assetID := s.upload(t, "Moonlit Archive")
	for round := range 2 {
		s.withhold(t, assetID, "Report "+string(rune('A'+round)))
		s.restore(t, assetID)
	}
	s.withhold(t, assetID, "Report C")
	s.fanOut(t, time.Now())

	var read []string
	query := "?limit=2"
	for pages := 0; ; pages++ {
		if pages == 3 {
			t.Fatal("the inbox kept offering another page")
		}
		page := s.inbox(t, s.creator, query)
		for _, entry := range page.Items {
			read = append(read, entry.Type+" "+entry.Reason)
		}
		if page.NextCursor == nil {
			break
		}
		query = "?limit=2&before=" + page.NextCursor.Before.Format(time.RFC3339Nano) +
			"&beforeId=" + page.NextCursor.BeforeID
	}
	want := []string{
		"asset_withheld Report C", "asset_restored ", "asset_withheld Report B",
		"asset_restored ", "asset_withheld Report A",
	}
	if strings.Join(read, "|") != strings.Join(want, "|") {
		t.Fatalf("paged inbox = %q, want %q", read, want)
	}

	for _, query := range []string{"?limit=0", "?limit=51", "?before=2026-09-14T12:00:00Z"} {
		response := send(t, s.router, authorized(httptest.NewRequest(http.MethodGet, "/v1/notifications"+query, nil), s.creator))
		if response.Code != http.StatusBadRequest {
			t.Errorf("GET /v1/notifications%s status = %d, want 400", query, response.Code)
		}
	}
}

func TestOpeningAnEntryOrMarkingAllReadClearsTheUnreadCount(t *testing.T) {
	t.Parallel()
	s := newInboxStack(t)
	assetID := s.upload(t, "Moonlit Archive")
	s.withhold(t, assetID, "Copyright report under review")
	s.fanOut(t, time.Now())
	entryID := s.inbox(t, s.creator, "").Items[0].ID

	if got := s.markRead(t, s.staff, entryID); got != http.StatusNotFound {
		t.Fatalf("another account marking the entry = %d, want 404", got)
	}
	if got := s.unread(t, s.creator); got != 1 {
		t.Fatalf("unread after another account tried = %d, want 1", got)
	}
	for range 2 {
		if got := s.markRead(t, s.creator, entryID); got != http.StatusNoContent {
			t.Fatalf("mark read = %d, want 204", got)
		}
	}
	if got := s.markRead(t, s.creator, "44444444-4444-4444-8444-444444444444"); got != http.StatusNotFound {
		t.Fatalf("marking a missing entry = %d, want 404", got)
	}
	if opened := s.inbox(t, s.creator, "").Items[0]; opened.ReadAt == nil || s.unread(t, s.creator) != 0 {
		t.Fatalf("opened entry = %+v with %d unread", opened, s.unread(t, s.creator))
	}

	s.restore(t, assetID)
	s.withhold(t, assetID, "A second report")
	s.fanOut(t, time.Now())
	if got := s.unread(t, s.creator); got != 2 {
		t.Fatalf("unread = %d, want 2", got)
	}
	everything := send(t, s.router, authorized(httptest.NewRequest(http.MethodPost, "/v1/notifications/read", nil), s.creator))
	if everything.Code != http.StatusNoContent {
		t.Fatalf("mark all read = %d, want 204: %s", everything.Code, everything.Body.String())
	}
	if got := s.unread(t, s.creator); got != 0 {
		t.Fatalf("unread after marking all read = %d, want 0", got)
	}
}

func TestAnAccountRemovesOneEntryOrClearsItsWholeInbox(t *testing.T) {
	t.Parallel()
	s := newInboxStack(t)
	assetID := s.upload(t, "Moonlit Archive")
	s.withhold(t, assetID, "Copyright report under review")
	s.restore(t, assetID)
	s.withhold(t, assetID, "A second report")
	s.fanOut(t, time.Now())
	entries := s.inbox(t, s.creator, "").Items
	if len(entries) != 3 {
		t.Fatalf("inbox has %d entries, want 3", len(entries))
	}
	s.markRead(t, s.creator, entries[1].ID)

	if got := s.remove(t, s.staff, entries[0].ID); got != http.StatusNotFound {
		t.Fatalf("another account removing the entry = %d, want 404", got)
	}
	if got := s.remove(t, s.creator, entries[0].ID); got != http.StatusNoContent {
		t.Fatalf("remove = %d, want 204", got)
	}
	if got := s.remove(t, s.creator, entries[0].ID); got != http.StatusNotFound {
		t.Fatalf("removing the same entry again = %d, want 404", got)
	}
	kept := s.inbox(t, s.creator, "").Items
	if len(kept) != 2 || kept[0].ID != entries[1].ID || kept[1].ID != entries[2].ID {
		t.Fatalf("after removing one entry the inbox holds %+v", kept)
	}
	if got := s.unread(t, s.creator); got != 1 {
		t.Fatalf("unread after removing an unread entry = %d, want 1", got)
	}

	cleared := send(t, s.router, authorized(httptest.NewRequest(http.MethodDelete, "/v1/notifications", nil), s.creator))
	if cleared.Code != http.StatusNoContent {
		t.Fatalf("clear = %d, want 204: %s", cleared.Code, cleared.Body.String())
	}
	if left := s.inbox(t, s.creator, ""); len(left.Items) != 0 || s.unread(t, s.creator) != 0 {
		t.Fatalf("after clearing the inbox holds %+v with %d unread", left.Items, s.unread(t, s.creator))
	}
	if staffInbox := s.inbox(t, s.staff, ""); len(staffInbox.Items) != 0 {
		t.Fatalf("clearing one inbox touched another: %+v", staffInbox.Items)
	}
}

func TestTheSweeperRemovesEntriesNinetyDaysAfterTheyArrived(t *testing.T) {
	t.Parallel()
	s := newInboxStack(t)
	assetID := s.upload(t, "Moonlit Archive")
	s.withhold(t, assetID, "Copyright report under review")
	arrived := time.Now()
	s.fanOut(t, arrived)

	s.sweep(t, arrived.Add(89*24*time.Hour))
	if kept := s.inbox(t, s.creator, ""); len(kept.Items) != 1 {
		t.Fatalf("after 89 days the inbox holds %d entries, want 1", len(kept.Items))
	}
	s.sweep(t, arrived.Add(91*24*time.Hour))
	if swept := s.inbox(t, s.creator, ""); len(swept.Items) != 0 || s.unread(t, s.creator) != 0 {
		t.Fatalf("after 91 days the inbox holds %d entries, want none", len(swept.Items))
	}
}

func TestTheInboxNeedsAnAccount(t *testing.T) {
	t.Parallel()
	s := newInboxStack(t)
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/v1/notifications", nil),
		httptest.NewRequest(http.MethodGet, "/v1/notifications/unread", nil),
		browserMutation(httptest.NewRequest(http.MethodPost, "/v1/notifications/read", nil)),
		browserMutation(httptest.NewRequest(
			http.MethodPost, "/v1/notifications/44444444-4444-4444-8444-444444444444/read", nil,
		)),
		browserMutation(httptest.NewRequest(http.MethodDelete, "/v1/notifications", nil)),
		browserMutation(httptest.NewRequest(
			http.MethodDelete, "/v1/notifications/44444444-4444-4444-8444-444444444444", nil,
		)),
	} {
		if response := send(t, s.router, request); response.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want 401", request.Method, request.URL.Path, response.Code)
		}
	}
}

const (
	creatorHandle = "moon.creator"
	staffHandle   = "night.staff"
)

type inboxStack struct {
	router        *gin.Engine
	assets        *asset.Service
	notifications *notification.Service
	outbox        *verificationOutbox
	creator       *http.Cookie
	staff         *http.Cookie
}

type inboxEntry struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	CreatedAt time.Time  `json:"createdAt"`
	ReadAt    *time.Time `json:"readAt"`
	Asset     *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"asset"`
	Reason string `json:"reason"`
	Update *struct {
		Number       int    `json:"number"`
		VersionLabel string `json:"versionLabel"`
		Summary      string `json:"summary"`
		Count        int    `json:"count"`
	} `json:"update"`
	SendTargets []struct {
		InstanceID      string `json:"instanceId"`
		InstanceName    string `json:"instanceName"`
		ApplicationName string `json:"applicationName"`
	} `json:"sendTargets"`
}

type inboxPage struct {
	Items      []inboxEntry `json:"items"`
	NextCursor *struct {
		Before   time.Time `json:"before"`
		BeforeID string    `json:"beforeId"`
	} `json:"nextCursor"`
}

func newInboxStack(t *testing.T) inboxStack {
	t.Helper()
	outbox := &verificationOutbox{}
	router, pool, handlers := newTestRouterWithSenderPoolAndHandlers(t, 1<<20, DefaultDeadlines(), outbox)
	creator := verifiedSignUp(t, router, outbox, "creator@example.com", creatorHandle)
	staff := verifiedSignUp(t, router, outbox, "staff@example.com", staffHandle)
	if _, err := pool.Exec(t.Context(), `update users set role = 'admin' where username = $1`, staffHandle); err != nil {
		t.Fatalf("make the staff account an admin: %v", err)
	}
	return inboxStack{
		router: router, assets: handlers.assets, notifications: handlers.notifications,
		outbox: outbox, creator: creator, staff: staff,
	}
}

func (s inboxStack) upload(t *testing.T, name string) string {
	t.Helper()
	metadata := exampleMetadata(name)
	metadata["filename"] = "archive.lumitheme"
	return assetIDFromIngest(t, uploadAndFinish(t, s.router, s.creator, s.assets, metadata, []byte(name)))
}

func (s inboxStack) withhold(t *testing.T, assetID, reason string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"reason": reason})
	if err != nil {
		t.Fatal(err)
	}
	response := send(t, s.router, authorizedJSONRequest(
		t, http.MethodPut, "/v1/assets/"+assetID+"/withhold", string(body), s.staff,
	))
	if response.Code != http.StatusNoContent {
		t.Fatalf("withhold status = %d, want 204: %s", response.Code, response.Body.String())
	}
}

func (s inboxStack) restore(t *testing.T, assetID string) {
	t.Helper()
	response := send(t, s.router, authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/assets/"+assetID+"/withhold", nil), s.staff,
	))
	if response.Code != http.StatusNoContent {
		t.Fatalf("restore status = %d, want 204: %s", response.Code, response.Body.String())
	}
}

func (s inboxStack) restrictProfile(t *testing.T, reason string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"reason": reason})
	if err != nil {
		t.Fatal(err)
	}
	response := send(t, s.router, authorizedJSONRequest(
		t, http.MethodPut, "/v1/profiles/"+creatorHandle+"/restriction", string(body), s.staff,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("restrict status = %d, want 200: %s", response.Code, response.Body.String())
	}
}

func (s inboxStack) restoreProfile(t *testing.T) int {
	t.Helper()
	return send(t, s.router, authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/profiles/"+creatorHandle+"/restriction", nil), s.staff,
	)).Code
}

func (s inboxStack) unread(t *testing.T, session *http.Cookie) int {
	t.Helper()
	response := send(t, s.router, authorized(httptest.NewRequest(http.MethodGet, "/v1/notifications/unread", nil), session))
	if response.Code != http.StatusOK {
		t.Fatalf("unread status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var counted struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &counted); err != nil {
		t.Fatalf("decode unread count: %v", err)
	}
	return counted.Count
}

func (s inboxStack) markRead(t *testing.T, session *http.Cookie, entryID string) int {
	t.Helper()
	return send(t, s.router, authorized(
		httptest.NewRequest(http.MethodPost, "/v1/notifications/"+entryID+"/read", nil), session,
	)).Code
}

func (s inboxStack) remove(t *testing.T, session *http.Cookie, entryID string) int {
	t.Helper()
	return send(t, s.router, authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/notifications/"+entryID, nil), session,
	)).Code
}

func browserMutation(request *http.Request) *http.Request {
	request.Header.Set("Origin", testBrowserOrigin)
	request.Header.Set(browserMutationHeader, "1")
	return request
}

func (s inboxStack) fanOut(t *testing.T, now time.Time) {
	t.Helper()
	if _, err := s.notifications.FanOut(t.Context(), now); err != nil {
		t.Fatalf("fan out: %v", err)
	}
}

func (s inboxStack) sweep(t *testing.T, now time.Time) {
	t.Helper()
	if _, err := s.notifications.Sweep(t.Context(), now); err != nil {
		t.Fatalf("sweep: %v", err)
	}
}

func (s inboxStack) inbox(t *testing.T, session *http.Cookie, query string) inboxPage {
	t.Helper()
	response := send(t, s.router, authorized(httptest.NewRequest(http.MethodGet, "/v1/notifications"+query, nil), session))
	if response.Code != http.StatusOK {
		t.Fatalf("inbox status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var page inboxPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode inbox: %v", err)
	}
	return page
}
