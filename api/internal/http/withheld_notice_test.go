package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/format/extension"
	"github.com/jackc/pgx/v5/pgxpool"
)

type withheldNotice struct {
	AssetID    string    `json:"assetId"`
	Name       string    `json:"name"`
	WithheldAt time.Time `json:"withheldAt"`
}

func withholdAsAdmin(t *testing.T, r http.Handler, pool *pgxpool.Pool, session *http.Cookie, handle, assetID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`update users set role = 'admin' where username = $1`, handle); err != nil {
		t.Fatalf("make %s an admin: %v", handle, err)
	}
	rec := send(t, r, authorizedJSONRequest(t, http.MethodPut, "/v1/assets/"+assetID+"/withhold",
		`{"reason":"Report under review"}`, session))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("withhold status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
}

func clearWithhold(t *testing.T, r http.Handler, session *http.Cookie, assetID string) {
	t.Helper()
	rec := send(t, r, authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/assets/"+assetID+"/withhold", nil), session))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("clear withhold status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
}

func TestTheNextLibrarySyncFromAnInstanceReportingAWithheldExtensionCarriesANoticeNamingIt(t *testing.T) {
	r, session, assets, pool := newExtensionRouter(t)
	assetID := publishedSpindleExtension(t, r, session, assets)
	installs := linkInstallations(t, r, session, 2)
	holding, other := installs[0], installs[1]
	reportInstalled(t, r, holding.AccessToken, "1.2.0", assetID)

	withholdAsAdmin(t, r, pool, session, "verified.creator", assetID)

	noticed := reportInstalled(t, r, holding.AccessToken, "1.2.0")
	if len(noticed.Withheld) != 1 || noticed.Withheld[0].AssetID != assetID ||
		noticed.Withheld[0].Name != "Quiet Toolbox" || noticed.Withheld[0].WithheldAt.IsZero() {
		t.Fatalf("withheld = %+v, want one notice naming Quiet Toolbox", noticed.Withheld)
	}
	if again := reportInstalled(t, r, holding.AccessToken, "1.2.0"); len(again.Withheld) != 0 {
		t.Fatalf("withheld = %+v on the sync after the notice, want none", again.Withheld)
	}
	if elsewhere := reportInstalled(t, r, other.AccessToken, "1.2.0"); len(elsewhere.Withheld) != 0 {
		t.Fatalf("an instance without the extension was told %+v", elsewhere.Withheld)
	}
}

func TestTheNextDeliveryWaitFromAnInstanceReportingAWithheldExtensionCarriesTheNoticeOnce(t *testing.T) {
	r, session, assets, pool := newExtensionRouter(t)
	assetID := publishedSpindleExtension(t, r, session, assets)
	install := linkInstallations(t, r, session, 1)[0]
	reportInstalled(t, r, install.AccessToken, "1.2.0", assetID)

	withholdAsAdmin(t, r, pool, session, "verified.creator", assetID)

	rec := collect(t, r, install.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("collect status = %d, want 200 carrying the notice: %s", rec.Code, rec.Body.String())
	}
	waited := decodeResponse[deliveryWorkList](t, rec)
	if len(waited.Deliveries) != 0 || len(waited.Withheld) != 1 ||
		waited.Withheld[0].AssetID != assetID || waited.Withheld[0].Name != "Quiet Toolbox" {
		t.Fatalf("wait = %+v, want no work and one notice naming Quiet Toolbox", waited)
	}
	if again := collect(t, r, install.AccessToken, nil); again.Code != http.StatusNoContent {
		t.Fatalf("the next wait = %d, want 204 once the notice was carried: %s", again.Code, again.Body.String())
	}
	if synced := reportInstalled(t, r, install.AccessToken, "1.2.0"); len(synced.Withheld) != 0 {
		t.Fatalf("a library sync repeated the notice the wait carried: %+v", synced.Withheld)
	}
}

func TestAnExtensionWithheldAgainAfterBeingClearedIsNoticedAgain(t *testing.T) {
	r, session, assets, pool := newExtensionRouter(t)
	assetID := publishedSpindleExtension(t, r, session, assets)
	install := linkInstallations(t, r, session, 1)[0]
	reportInstalled(t, r, install.AccessToken, "1.2.0", assetID)
	withholdAsAdmin(t, r, pool, session, "verified.creator", assetID)
	if first := reportInstalled(t, r, install.AccessToken, "1.2.0"); len(first.Withheld) != 1 {
		t.Fatalf("withheld = %+v, want the first notice", first.Withheld)
	}

	clearWithhold(t, r, session, assetID)
	if cleared := reportInstalled(t, r, install.AccessToken, "1.2.0"); len(cleared.Withheld) != 0 {
		t.Fatalf("withheld = %+v after the withhold was cleared, want none", cleared.Withheld)
	}
	withholdAsAdmin(t, r, pool, session, "verified.creator", assetID)

	if second := reportInstalled(t, r, install.AccessToken, "1.2.0"); len(second.Withheld) != 1 {
		t.Fatalf("withheld = %+v after a second withhold, want a new notice", second.Withheld)
	}
}

func TestOnlyAWithheldExtensionIsNoticed(t *testing.T) {
	r, session, pool := newLinkingRouter(t)
	assetID := publishedTestAsset(t, r, session)
	install := linkInstallations(t, r, session, 1)[0]
	reportInstalled(t, r, install.AccessToken, "1.2.0", assetID)

	withholdAsAdmin(t, r, pool, session, "linking.creator", assetID)

	if synced := reportInstalled(t, r, install.AccessToken, "1.2.0"); len(synced.Withheld) != 0 {
		t.Fatalf("withholding a character told the instance %+v", synced.Withheld)
	}
	if rec := collect(t, r, install.AccessToken, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("collect = %d after a character was withheld, want 204: %s", rec.Code, rec.Body.String())
	}
}

func TestWithholdingAnExtensionStopsItsQueuedDeliveriesAsWithdrawn(t *testing.T) {
	r, session, assets, pool := newExtensionRouter(t)
	assetID := publishedSpindleExtension(t, r, session, assets)
	install := linkInstallations(t, r, session, 1)[0]
	declare(t, r, install.AccessToken, []string{lumiverseInstalls}, []string{extension.SpindleID})
	if queued := sendToInstance(t, r, session, assetID, install.Instance.ID); queued.Code != http.StatusAccepted {
		t.Fatalf("send = %d: %s", queued.Code, queued.Body.String())
	}

	withholdAsAdmin(t, r, pool, session, "verified.creator", assetID)
	clearWithhold(t, r, session, assetID)

	stopped := assetInstances(t, r, session, assetID).Items[0].Delivery
	if stopped == nil || stopped.State != "failed" || stopped.Reason == nil || *stopped.Reason != "withdrawn" {
		t.Fatalf("delivery = %+v, want it stopped as withdrawn when the extension was withheld", stopped)
	}
	if rec := collect(t, r, install.AccessToken, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("collect = %d, want nothing released: %s", rec.Code, rec.Body.String())
	}
}
