package page_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/staff"
)

const browserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0"

func TestAWorkPageCountsReaderViewsAndPublicDownloadsOnly(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	unlisted := apitest.Send(t, router, apitest.AuthorizedJSONRequest(
		t, http.MethodPut, "/v1/works/"+workID+"/visibility", `{"visibility":"unlisted"}`, session))
	if unlisted.Code != http.StatusNoContent {
		t.Fatalf("unlist status = %d, want 204: %s", unlisted.Code, unlisted.Body.String())
	}
	draft := apitest.StartCharacter(t, router, session)

	viewWork(t, router, workID, browserAgent, nil)
	viewWork(t, router, workID, browserAgent, nil)
	viewWork(t, router, workID, browserAgent, session)
	viewWork(t, router, workID, "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", nil)
	viewWork(t, router, workID, "", nil)
	viewWork(t, router, draft.ID, browserAgent, nil)

	reader := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/download/"+workID+"/test_opaque", nil))
	owner := apitest.Send(t, router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/download/"+workID+"/test_opaque", nil), session))
	if reader.Code != http.StatusOK || owner.Code != http.StatusOK {
		t.Fatalf("download statuses = %d and %d, want 200", reader.Code, owner.Code)
	}
	if queued := apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID); queued.Code != http.StatusAccepted {
		t.Fatalf("send status = %d, want 202: %s", queued.Code, queued.Body.String())
	}
	collected := apitest.DecodeResponse[apitest.CollectedSends](t, apitest.Collect(t, router, credentials.AccessToken, nil))
	if fetched := apitest.FetchSigned(t, router, collected.Sends[0].Files[0].URL); fetched.Code != http.StatusOK {
		t.Fatalf("app fetch status = %d, want 200: %s", fetched.Code, fetched.Body.String())
	}

	page := apitest.FetchWorkPage(t, router, "/v1/works/"+workID)
	if page.ViewCount != 2 || page.DownloadCount != 1 {
		t.Fatalf("counts = %d views and %d downloads, want 2 reader views and 1 public download", page.ViewCount, page.DownloadCount)
	}
	var draftViews int
	if err := pool.QueryRow(t.Context(), `select count(*) from events where work_id = $1`, draft.ID).Scan(&draftViews); err != nil {
		t.Fatalf("count draft events: %v", err)
	}
	if draftViews != 0 {
		t.Fatalf("draft events = %d, want none", draftViews)
	}
}

func TestViewCountsSurviveTheNightlyRollup(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewConnectRouter(t)
	workID := apitest.PublishedCharacter(t, router, session)
	today := time.Now().UTC()
	if _, err := pool.Exec(t.Context(), `
		insert into events (kind, work_id, day) values
			('view', $1, $2::date - 40), ('view', $1, $2::date - 40), ('view', $1, $2::date - 1)
	`, workID, today.Format(time.DateOnly)); err != nil {
		t.Fatalf("insert past views: %v", err)
	}
	viewWork(t, router, workID, browserAgent, nil)

	works := apitest.WorksOver(t, pool, format.NewRegistry())
	if err := staff.NewService(works).Rollup(t.Context(), today); err != nil {
		t.Fatalf("roll up: %v", err)
	}

	if page := apitest.FetchWorkPage(t, router, "/v1/works/"+workID); page.ViewCount != 4 {
		t.Fatalf("views = %d, want the 3 rolled up and today's 1", page.ViewCount)
	}
}

func viewWork(t *testing.T, router http.Handler, workID, agent string, session *http.Cookie) {
	t.Helper()
	request := apitest.BrowserMutation(httptest.NewRequest(http.MethodPost, "/v1/works/"+workID+"/views", nil))
	request.Header.Set("User-Agent", agent)
	if session != nil {
		request = apitest.Authorized(request, session)
	}
	if viewed := apitest.Send(t, router, request); viewed.Code != http.StatusNoContent {
		t.Fatalf("view status = %d, want 204: %s", viewed.Code, viewed.Body.String())
	}
}

func TestAReadersAcknowledgedSendCountsOnceAsAPublicDownloadAndTheCreatorsOwnSendNever(t *testing.T) {
	t.Parallel()
	router, creator, pool := harness.NewConnectRouter(t)
	reader := apitest.AddVerifiedUser(t, router, pool, "reader@example.com", "quiet.reader")
	desk := apitest.ConnectApp(t, router, reader, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, desk.AccessToken, []string{"test_opaque"})
	studio := apitest.ConnectApp(t, router, creator, "Paper Lantern", "studio", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, studio.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, creator)
	expireLeases := func(attempts int) {
		t.Helper()
		if _, err := pool.Exec(t.Context(), `
			update sends set lease_expires_at = now() - interval '1 second', attempts = greatest(attempts, $1)
			 where state = 'released'
		`, attempts); err != nil {
			t.Fatalf("expire the send leases: %v", err)
		}
	}
	collect := func(token string) apitest.CollectedSends {
		t.Helper()
		return apitest.DecodeResponse[apitest.CollectedSends](t, apitest.Collect(t, router, token, nil))
	}
	acknowledge := func(token, id string) {
		t.Helper()
		apitest.Collect(t, router, token, []string{id})
	}

	apitest.SendToApp(t, router, reader, workID, desk.ConnectedApp.ID)
	first := collect(desk.AccessToken).Sends[0].ID
	expireLeases(0)
	if retried := collect(desk.AccessToken); len(retried.Sends) != 1 || retried.Sends[0].ID != first {
		t.Fatalf("retried collection = %+v, want the same send again", retried.Sends)
	}
	if page := apitest.FetchWorkPage(t, router, "/v1/works/"+workID); page.SendCount != 0 {
		t.Fatalf("send count before acknowledgement = %d, want 0", page.SendCount)
	}
	acknowledge(desk.AccessToken, first)
	acknowledge(desk.AccessToken, first)

	apitest.SendToApp(t, router, reader, workID, desk.ConnectedApp.ID)
	second := collect(desk.AccessToken).Sends[0].ID
	expireLeases(apitest.SendSettings().MaxAttempts)
	apitest.Collect(t, router, desk.AccessToken, nil)
	acknowledge(desk.AccessToken, second)

	apitest.SendToApp(t, router, creator, workID, studio.ConnectedApp.ID)
	acknowledge(studio.AccessToken, collect(studio.AccessToken).Sends[0].ID)
	if got := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/download/"+workID+"/test_opaque", nil)); got.Code != http.StatusOK {
		t.Fatalf("download status = %d, want 200: %s", got.Code, got.Body.String())
	}

	page := apitest.FetchWorkPage(t, router, "/v1/works/"+workID)
	if page.DownloadCount != 2 || page.SendCount != 1 {
		t.Fatalf("counts = %d downloads with %d sent, want the public download plus the reader's one acknowledged send",
			page.DownloadCount, page.SendCount)
	}
	var sendEvents int
	if err := pool.QueryRow(t.Context(), `select count(*) from events where kind = 'send' and work_id = $1`, workID).
		Scan(&sendEvents); err != nil {
		t.Fatalf("count send events: %v", err)
	}
	if sendEvents != 1 {
		t.Fatalf("send events = %d, want only the reader's", sendEvents)
	}
}

func TestAWorkThatCanOnlyBeSentRanksByItsSendsInMostDownloaded(t *testing.T) {
	t.Parallel()
	router, creator, pool := harness.NewConnectRouter(t)
	sealed := apitest.PublishPrivatePromptPreset(t, router, creator, "Sealed preset", "Private for allowed apps only.")
	newer := apitest.PublishPrivatePromptPreset(t, router, creator, "Newer preset", "Also private.")
	reader := apitest.AddVerifiedUser(t, router, pool, "reader@example.com", "quiet.reader")
	desk := apitest.ConnectApp(t, router, reader, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, desk.AccessToken, []string{"preset_lumiverse"})
	if queued := apitest.SendToApp(t, router, reader, sealed, desk.ConnectedApp.ID); queued.Code != http.StatusAccepted {
		t.Fatalf("send status = %d, want 202: %s", queued.Code, queued.Body.String())
	}
	collected := apitest.DecodeResponse[apitest.CollectedSends](t, apitest.Collect(t, router, desk.AccessToken, nil))
	apitest.Collect(t, router, desk.AccessToken, []string{collected.Sends[0].ID})

	works := apitest.WorksOver(t, pool, format.NewRegistry())
	if err := staff.NewService(works).Rollup(t.Context(), time.Now().UTC().AddDate(0, 0, 1)); err != nil {
		t.Fatalf("roll up: %v", err)
	}

	listed := apitest.ListItems(t, router, "/v1/works?sort=downloads&type=preset")
	assertOrder(t, listed, sealed, newer)
	if listed[0].DownloadCount != 1 {
		t.Fatalf("card downloads = %d, want the one send", listed[0].DownloadCount)
	}
}

func TestTheCreatorSeesEachFollowerOnceAndRealChangesAreCountedWithoutWhoMadeThem(t *testing.T) {
	t.Parallel()
	outbox := &apitest.VerificationOutbox{}
	router, pool, _ := harness.NewRouterWithSenderPoolAndServices(t, 1<<20, api.DefaultDeadlines(), outbox)
	creator := apitest.VerifiedSignUp(t, router, outbox, "creator@example.com", apitest.CreatorHandle)
	reader := apitest.VerifiedSignUp(t, router, outbox, "reader@example.com", "quiet.reader")
	other := apitest.VerifiedSignUp(t, router, outbox, "other@example.com", "other.reader")
	workID := apitest.PublishedCharacter(t, router, creator)
	follow := func(session *http.Cookie, method string) {
		t.Helper()
		if got := apitest.Send(t, router, apitest.Authorized(
			httptest.NewRequest(method, "/v1/works/"+workID+"/follow", nil), session)); got.Code != http.StatusOK {
			t.Fatalf("%s follow status = %d, want 200: %s", method, got.Code, got.Body.String())
		}
	}
	followers := func() *int {
		t.Helper()
		return apitest.DecodeResponse[apitest.WorkPageResponse](t, apitest.Send(t, router, apitest.Authorized(
			httptest.NewRequest(http.MethodGet, "/v1/works/"+workID, nil), creator))).FollowerCount
	}
	library := []string{apitest.ReceivePermission, apitest.LibrarySyncPermission}

	follow(reader, http.MethodPut)
	follow(reader, http.MethodPut)
	desk := apitest.ConnectApp(t, router, reader, "Lumiverse", "Reading desk", library)
	apitest.ReportLibrary(t, router, desk.AccessToken, "", workID)
	studio := apitest.ConnectApp(t, router, creator, "Lumiverse", "Studio", library)
	apitest.ReportLibrary(t, router, studio.AccessToken, "", workID)
	if count := followers(); count == nil || *count != 1 {
		t.Fatalf("followers = %v, want the reader once", count)
	}

	follow(reader, http.MethodDelete)
	follow(reader, http.MethodPut)
	shelf := apitest.ConnectApp(t, router, other, "Lumiverse", "Shelf", library)
	apitest.ReportLibrary(t, router, shelf.AccessToken, "", workID)
	apitest.ReportLibrary(t, router, shelf.AccessToken, "", workID)
	if count := followers(); count == nil || *count != 2 {
		t.Fatalf("followers = %v, want the reader and the app-library follower", count)
	}
	removed := apitest.Send(t, router, apitest.AsApp(t, http.MethodPost, "/v1/library/sync", shelf.AccessToken,
		map[string]any{"snapshot": false, "entries": []any{}, "removed": []string{workID}}))
	if removed.Code != http.StatusOK {
		t.Fatalf("library removal status = %d, want 200: %s", removed.Code, removed.Body.String())
	}
	if count := followers(); count == nil || *count != 1 {
		t.Fatalf("followers after the library removal = %v, want 1", count)
	}
	if page := apitest.FetchWorkPage(t, router, "/v1/works/"+workID); page.FollowerCount != nil {
		t.Fatalf("reader sees follower count %d, want none", *page.FollowerCount)
	}

	works := apitest.WorksOver(t, pool, format.NewRegistry())
	if err := staff.NewService(works).Rollup(t.Context(), time.Now().UTC().AddDate(0, 0, staff.EventRetentionDays+1)); err != nil {
		t.Fatalf("roll up: %v", err)
	}
	var follows, unfollows, left int
	if err := pool.QueryRow(t.Context(), `
		select coalesce(sum(count) filter (where kind = 'follow'), 0),
		       coalesce(sum(count) filter (where kind = 'unfollow'), 0),
		       (select count(*) from events where kind in ('follow', 'unfollow'))
		  from daily_totals where work_id = $1
	`, workID).Scan(&follows, &unfollows, &left); err != nil {
		t.Fatalf("read the follow totals: %v", err)
	}
	if follows != 3 || unfollows != 2 || left != 0 {
		t.Fatalf("totals = %d follows and %d unfollows with %d events left, want 3, 2 and 0", follows, unfollows, left)
	}
}
