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

func withholdAsAdmin(t *testing.T, r http.Handler, pool *pgxpool.Pool, session *http.Cookie, handle, workID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`update users set role = 'admin' where username = $1`, handle); err != nil {
		t.Fatalf("make %s an admin: %v", handle, err)
	}
	rec := apitest.Send(t, r, apitest.AuthorizedJSONRequest(t, http.MethodPut, "/v1/works/"+workID+"/withhold",
		`{"reason":"Report under review"}`, session))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("withhold status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
}

func clearWithhold(t *testing.T, r http.Handler, session *http.Cookie, workID string) {
	t.Helper()
	rec := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/works/"+workID+"/withhold", nil), session))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("clear withhold status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
}

func TestTheNextLibrarySyncFromAConnectedAppReportingAWithheldExtensionCarriesANoticeNamingIt(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	installs := linkInstallations(t, r, session, 2)
	holding, other := installs[0], installs[1]
	apitest.ReportLibrary(t, r, holding.AccessToken, "1.2.0", workID)

	withholdAsAdmin(t, r, pool, session, "verified.creator", workID)

	noticed := apitest.ReportLibrary(t, r, holding.AccessToken, "1.2.0")
	if len(noticed.Withheld) != 1 || noticed.Withheld[0].WorkID != workID ||
		noticed.Withheld[0].Name != "Quiet Toolbox" || noticed.Withheld[0].WithheldAt.IsZero() {
		t.Fatalf("withheld = %+v, want one notice naming Quiet Toolbox", noticed.Withheld)
	}
	if again := apitest.ReportLibrary(t, r, holding.AccessToken, "1.2.0"); len(again.Withheld) != 0 {
		t.Fatalf("withheld = %+v on the sync after the notice, want none", again.Withheld)
	}
	if elsewhere := apitest.ReportLibrary(t, r, other.AccessToken, "1.2.0"); len(elsewhere.Withheld) != 0 {
		t.Fatalf("a connected app without the extension was told %+v", elsewhere.Withheld)
	}
}

func TestTheNextCollectFromAConnectedAppReportingAWithheldExtensionCarriesTheNoticeOnce(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	install := linkInstallations(t, r, session, 1)[0]
	apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0", workID)

	withholdAsAdmin(t, r, pool, session, "verified.creator", workID)

	rec := apitest.Collect(t, r, install.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("collect status = %d, want 200 carrying the notice: %s", rec.Code, rec.Body.String())
	}
	waited := apitest.DecodeResponse[apitest.CollectedSends](t, rec)
	if len(waited.Sends) != 0 || len(waited.Withheld) != 1 ||
		waited.Withheld[0].WorkID != workID || waited.Withheld[0].Name != "Quiet Toolbox" {
		t.Fatalf("wait = %+v, want no work and one notice naming Quiet Toolbox", waited)
	}
	if again := apitest.Collect(t, r, install.AccessToken, nil); again.Code != http.StatusNoContent {
		t.Fatalf("the next wait = %d, want 204 once the notice was carried: %s", again.Code, again.Body.String())
	}
	if synced := apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0"); len(synced.Withheld) != 0 {
		t.Fatalf("a library sync repeated the notice the wait carried: %+v", synced.Withheld)
	}
}

func TestAnExtensionWithheldAgainAfterBeingClearedIsNoticedAgain(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	install := linkInstallations(t, r, session, 1)[0]
	apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0", workID)
	withholdAsAdmin(t, r, pool, session, "verified.creator", workID)
	if first := apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0"); len(first.Withheld) != 1 {
		t.Fatalf("withheld = %+v, want the first notice", first.Withheld)
	}

	clearWithhold(t, r, session, workID)
	if cleared := apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0"); len(cleared.Withheld) != 0 {
		t.Fatalf("withheld = %+v after the withhold was cleared, want none", cleared.Withheld)
	}
	withholdAsAdmin(t, r, pool, session, "verified.creator", workID)

	if second := apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0"); len(second.Withheld) != 1 {
		t.Fatalf("withheld = %+v after a second withhold, want a new notice", second.Withheld)
	}
}

func TestOnlyAWithheldExtensionIsNoticed(t *testing.T) {
	t.Parallel()
	r, session, pool := harness.NewConnectRouter(t)
	workID := apitest.PublishedCharacter(t, r, session)
	install := linkInstallations(t, r, session, 1)[0]
	apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0", workID)

	withholdAsAdmin(t, r, pool, session, "connect.creator", workID)

	if synced := apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0"); len(synced.Withheld) != 0 {
		t.Fatalf("withholding a character told the connected app %+v", synced.Withheld)
	}
	if rec := apitest.Collect(t, r, install.AccessToken, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("collect = %d after a character was withheld, want 204: %s", rec.Code, rec.Body.String())
	}
}

func TestWithholdingAnExtensionStopsItsQueuedSendsAsWithdrawn(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	install := linkInstallations(t, r, session, 1)[0]
	apitest.DeclareCapabilities(t, r, install.AccessToken, []string{apitest.LumiverseInstalls}, []string{extension.SpindleID})
	if queued := apitest.SendToApp(t, r, session, workID, install.ConnectedApp.ID); queued.Code != http.StatusAccepted {
		t.Fatalf("send = %d: %s", queued.Code, queued.Body.String())
	}

	withholdAsAdmin(t, r, pool, session, "verified.creator", workID)
	clearWithhold(t, r, session, workID)

	stopped := apitest.WorkConnectedApps(t, r, session, workID).Items[0].Send
	if stopped == nil || stopped.State != "failed" || stopped.Reason == nil || *stopped.Reason != "withdrawn" {
		t.Fatalf("send = %+v, want it stopped as withdrawn when the extension was withheld", stopped)
	}
	if rec := apitest.Collect(t, r, install.AccessToken, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("collect = %d, want nothing released: %s", rec.Code, rec.Body.String())
	}
}
