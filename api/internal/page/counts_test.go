package page_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
