package connect_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format/extension"
	"github.com/jackc/pgx/v5/pgxpool"
)

func takeDownAsAdmin(t *testing.T, r http.Handler, pool *pgxpool.Pool, session *http.Cookie, handle, workID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`update users set role = 'admin' where username = $1`, handle); err != nil {
		t.Fatalf("make %s an admin: %v", handle, err)
	}
	rec := apitest.Send(t, r, apitest.AuthorizedJSONRequest(t, http.MethodPut, "/v1/works/"+workID+"/takedown",
		`{"reason":"Report under review"}`, session))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("takedown status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
}

func clearTakeDown(t *testing.T, r http.Handler, session *http.Cookie, workID string) {
	t.Helper()
	rec := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/works/"+workID+"/takedown", nil), session))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("clear takedown status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
}

func TestTheNextLibrarySyncFromAConnectedAppReportingATakenDownExtensionCarriesANoticeNamingIt(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	installs := linkInstallations(t, r, session, 2)
	holding, other := installs[0], installs[1]
	apitest.ReportLibrary(t, r, holding.AccessToken, "1.2.0", workID)

	takeDownAsAdmin(t, r, pool, session, "verified.creator", workID)

	noticed := apitest.ReportLibrary(t, r, holding.AccessToken, "1.2.0")
	if len(noticed.Takedowns) != 1 || noticed.Takedowns[0].WorkID != workID ||
		noticed.Takedowns[0].Name != "Quiet Toolbox" || noticed.Takedowns[0].TakenDownAt.IsZero() {
		t.Fatalf("taken down = %+v, want one notice naming Quiet Toolbox", noticed.Takedowns)
	}
	if again := apitest.ReportLibrary(t, r, holding.AccessToken, "1.2.0"); len(again.Takedowns) != 0 {
		t.Fatalf("taken down = %+v on the sync after the notice, want none", again.Takedowns)
	}
	if elsewhere := apitest.ReportLibrary(t, r, other.AccessToken, "1.2.0"); len(elsewhere.Takedowns) != 0 {
		t.Fatalf("a connected app without the extension was told %+v", elsewhere.Takedowns)
	}
}

func TestTheNextCollectFromAConnectedAppReportingATakenDownExtensionCarriesTheNoticeOnce(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	install := linkInstallations(t, r, session, 1)[0]
	apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0", workID)

	takeDownAsAdmin(t, r, pool, session, "verified.creator", workID)

	rec := apitest.Collect(t, r, install.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("collect status = %d, want 200 carrying the notice: %s", rec.Code, rec.Body.String())
	}
	waited := apitest.DecodeResponse[apitest.CollectedSends](t, rec)
	if len(waited.Sends) != 0 || len(waited.Takedowns) != 1 ||
		waited.Takedowns[0].WorkID != workID || waited.Takedowns[0].Name != "Quiet Toolbox" {
		t.Fatalf("wait = %+v, want no work and one notice naming Quiet Toolbox", waited)
	}
	if again := apitest.Collect(t, r, install.AccessToken, nil); again.Code != http.StatusNoContent {
		t.Fatalf("the next wait = %d, want 204 once the notice was carried: %s", again.Code, again.Body.String())
	}
	if synced := apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0"); len(synced.Takedowns) != 0 {
		t.Fatalf("a library sync repeated the notice the wait carried: %+v", synced.Takedowns)
	}
}

func TestAnExtensionTakenDownAgainAfterBeingClearedIsNoticedAgain(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	install := linkInstallations(t, r, session, 1)[0]
	apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0", workID)
	takeDownAsAdmin(t, r, pool, session, "verified.creator", workID)
	if first := apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0"); len(first.Takedowns) != 1 {
		t.Fatalf("taken down = %+v, want the first notice", first.Takedowns)
	}

	clearTakeDown(t, r, session, workID)
	if cleared := apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0"); len(cleared.Takedowns) != 0 {
		t.Fatalf("taken down = %+v after the takedown was cleared, want none", cleared.Takedowns)
	}
	takeDownAsAdmin(t, r, pool, session, "verified.creator", workID)

	if second := apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0"); len(second.Takedowns) != 1 {
		t.Fatalf("taken down = %+v after a second takeDown, want a new notice", second.Takedowns)
	}
}

func TestOnlyATakenDownExtensionIsNoticed(t *testing.T) {
	t.Parallel()
	r, session, pool := harness.NewConnectRouter(t)
	workID := apitest.PublishedCharacter(t, r, session)
	install := linkInstallations(t, r, session, 1)[0]
	apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0", workID)

	takeDownAsAdmin(t, r, pool, session, "connect.creator", workID)

	if synced := apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0"); len(synced.Takedowns) != 0 {
		t.Fatalf("taking down a character told the connected app %+v", synced.Takedowns)
	}
	if rec := apitest.Collect(t, r, install.AccessToken, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("collect = %d after a character was takenDown, want 204: %s", rec.Code, rec.Body.String())
	}
}

func TestTakingDownAnExtensionStopsItsQueuedSendsAsWithdrawn(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	install := linkInstallations(t, r, session, 1)[0]
	apitest.DeclareCapabilities(t, r, install.AccessToken, []string{apitest.LumiverseInstalls}, []string{extension.SpindleID})
	if queued := apitest.SendToApp(t, r, session, workID, install.ConnectedApp.ID); queued.Code != http.StatusAccepted {
		t.Fatalf("send = %d: %s", queued.Code, queued.Body.String())
	}

	takeDownAsAdmin(t, r, pool, session, "verified.creator", workID)
	clearTakeDown(t, r, session, workID)

	stopped := apitest.WorkConnectedApps(t, r, session, workID).Items[0].Send
	if stopped == nil || stopped.State != "failed" || stopped.Reason == nil || *stopped.Reason != "withdrawn" {
		t.Fatalf("send = %+v, want it stopped as withdrawn when the extension was taken down", stopped)
	}
	if rec := apitest.Collect(t, r, install.AccessToken, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("collect = %d, want nothing released: %s", rec.Code, rec.Body.String())
	}
}
