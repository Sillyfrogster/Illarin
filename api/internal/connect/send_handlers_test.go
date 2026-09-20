package connect_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAWaitWithNothingQueuedAnswersWithNoContent(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})

	rec := apitest.Collect(t, router, credentials.AccessToken, nil)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("empty wait status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	apitest.AssertNoStore(t, rec)
}

func TestSendingAWorkReleasesItInTheFormatTheConnectedAppAccepts(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)

	queued := apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID)
	if queued.Code != http.StatusAccepted {
		t.Fatalf("send status = %d, want 202: %s", queued.Code, queued.Body.String())
	}
	waiting := apitest.DecodeResponse[apitest.QueuedSend](t, queued)
	if waiting.State != "queued" || waiting.WorkID != workID {
		t.Fatalf("queued send = %+v, want a queued send for the work", waiting)
	}

	rec := apitest.Collect(t, router, credentials.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("collect status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	released := apitest.DecodeResponse[apitest.CollectedSends](t, rec)
	if len(released.Sends) != 1 {
		t.Fatalf("released %d sends, want 1", len(released.Sends))
	}
	work := released.Sends[0]
	if work.ID != waiting.ID || work.WorkID != workID || work.Format != "test_opaque" ||
		work.Type != "character" || work.Name == "" || work.VersionNumber < 1 {
		t.Fatalf("released work = %+v, want the queued work written as test_opaque", work)
	}
	if len(work.Files) == 0 || work.Files[0].Type != "export" {
		t.Fatalf("files = %+v, want an export first", work.Files)
	}
	if !strings.Contains(work.Files[0].URL, "signature=") {
		t.Fatalf("export address %q carries no signature", work.Files[0].URL)
	}
	fetched := apitest.FetchSigned(t, router, work.Files[0].URL)
	if fetched.Code != http.StatusOK {
		t.Fatalf("fetch status = %d, want 200: %s", fetched.Code, fetched.Body.String())
	}
	var originalFileMissing bool
	if err := pool.QueryRow(context.Background(), `
		select original_file_id is null
		  from download_events
		 where work_id = $1 and authorization_class = 'linked_instance'
	`, workID).Scan(&originalFileMissing); err != nil {
		t.Fatalf("read connected-app downloads event: %v", err)
	}
	if !originalFileMissing {
		t.Fatal("work made in Illarin recorded an original file")
	}
}

func TestQueueingRecordsNoDownloadAndFetchingTheCreatorsOwnFileRecordsOne(t *testing.T) {
	t.Parallel()
	router, session, works, pool := harness.NewVerifiedUploadRouterWithPool(t, format.NewRegistry())
	workID := apitest.UploadVisibilityTestWork(t, router, session, works, work.VisibilityListed)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"invented_by_the_client"})

	if queued := apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID); queued.Code != http.StatusAccepted {
		t.Fatalf("send status = %d, want 202: %s", queued.Code, queued.Body.String())
	}
	if before := apitest.DownloadEventCount(t, pool, "linked_instance"); before != 0 {
		t.Fatalf("queueing wrote %d download events, want 0", before)
	}

	rec := apitest.Collect(t, router, credentials.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("collect status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	work := apitest.DecodeResponse[apitest.CollectedSends](t, rec).Sends[0]
	if work.Format != format.Raw {
		t.Fatalf("format = %q, want the creator's own file as raw", work.Format)
	}
	fetched := apitest.FetchSigned(t, router, work.Files[0].URL)

	if fetched.Code != http.StatusOK || fetched.Header().Get("X-Accel-Redirect") == "" {
		t.Fatalf("fetch = %d, headers %v", fetched.Code, fetched.Header())
	}
	if got := apitest.DownloadEventCount(t, pool, "linked_instance"); got != 1 {
		t.Fatalf("recorded %d connected-app downloads, want 1", got)
	}
}

func TestATamperedOrUnsignedSendAddressIsRefused(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID)
	work := apitest.DecodeResponse[apitest.CollectedSends](t, apitest.Collect(t, router, credentials.AccessToken, nil)).Sends[0]

	address, err := url.Parse(work.Files[0].URL)
	if err != nil {
		t.Fatalf("parse the export address: %v", err)
	}
	query := address.Query()
	query.Set("signature", strings.Repeat("a", len(query.Get("signature"))))
	address.RawQuery = query.Encode()

	tampered := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, address.RequestURI(), nil))
	if tampered.Code != http.StatusNotFound {
		t.Fatalf("tampered address status = %d, want 404: %s", tampered.Code, tampered.Body.String())
	}
	unsigned := apitest.Send(t, router, httptest.NewRequest(
		http.MethodGet, address.Path+"?expires=0&signature=", nil))
	if unsigned.Code != http.StatusNotFound {
		t.Fatalf("unsigned address status = %d, want 404", unsigned.Code)
	}
}

func TestAnAcknowledgedSendLeavesTheQueueAndStaysOnRecordAsDelivered(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID)
	work := apitest.DecodeResponse[apitest.CollectedSends](t, apitest.Collect(t, router, credentials.AccessToken, nil)).Sends[0]

	again := apitest.Collect(t, router, credentials.AccessToken, []string{work.ID})

	if again.Code != http.StatusNoContent {
		t.Fatalf("wait after acknowledgement status = %d, want 204: %s", again.Code, again.Body.String())
	}
	delivered := apitest.WorkConnectedApps(t, router, session, workID).Items[0].Send
	if delivered == nil || delivered.State != "delivered" || delivered.SettledAt == nil || delivered.UpdatesInstall {
		t.Fatalf("send = %+v, want it on record as delivered", delivered)
	}
	if fetched := apitest.FetchSigned(t, router, work.Files[0].URL); fetched.Code != http.StatusNotFound {
		t.Fatalf("the export address still answers %d after acknowledgement", fetched.Code)
	}

	resent := apitest.DecodeResponse[apitest.QueuedSend](t, apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID))
	if resent.ID == work.ID || resent.State != "queued" {
		t.Fatalf("sending again = %+v, want a new queued send", resent)
	}
	dismissed := apitest.Send(t, router, apitest.BrowserRequest(t, http.MethodDelete, "/v1/sends/"+resent.ID, nil, session))
	if dismissed.Code != http.StatusNoContent {
		t.Fatalf("dismiss status = %d, want 204: %s", dismissed.Code, dismissed.Body.String())
	}
	if after := apitest.WorkConnectedApps(t, router, session, workID).Items[0].Send; after == nil || after.ID != work.ID {
		t.Fatalf("send = %+v, want the delivered record back once the new one is dismissed", after)
	}
}

func TestSendingTheSameWorkTwiceQueuesItOnce(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)

	first := apitest.DecodeResponse[apitest.QueuedSend](t, apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID))
	second := apitest.DecodeResponse[apitest.QueuedSend](t, apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID))

	if first.ID != second.ID {
		t.Fatalf("two sends made two sends, %s and %s", first.ID, second.ID)
	}
	work := apitest.DecodeResponse[apitest.CollectedSends](t, apitest.Collect(t, router, credentials.AccessToken, nil))
	if len(work.Sends) != 1 {
		t.Fatalf("released %d sends, want 1", len(work.Sends))
	}
}

func TestAnWorkWithdrawnAfterQueueingIsRefusedAtCollection(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID)
	if _, err := pool.Exec(context.Background(), `
		update works work
		   set withheld_at = now(), withheld_by = work.owner_id,
		       withheld_reason = 'Copyright report under review'
		 where work.id = $1
	`, workID); err != nil {
		t.Fatalf("withhold the work: %v", err)
	}

	rec := apitest.Collect(t, router, credentials.AccessToken, nil)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("collect status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	var state, reason string
	if err := pool.QueryRow(context.Background(),
		`select state, coalesce(settled_reason, '') from sends where work_id = $1`,
		workID,
	).Scan(&state, &reason); err != nil {
		t.Fatalf("read the send: %v", err)
	}
	if state != "failed" || reason != "withdrawn" {
		t.Fatalf("send = %s/%s, want failed/withdrawn", state, reason)
	}
}

func TestAConnectedAppThatAcceptsNoFormatWeCanWriteIsRefusedAtSend(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"invented_by_the_client"})
	workID := apitest.PublishedCharacter(t, router, session)

	rec := apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID)

	if rec.Code != http.StatusConflict {
		t.Fatalf("send status = %d, want 409: %s", rec.Code, rec.Body.String())
	}
}

func TestCollectingNeedsTheReceivePermission(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Reader", "desk", []string{"library:sync"})

	rec := apitest.Collect(t, router, credentials.AccessToken, nil)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("collect without the permission status = %d, want 403: %s", rec.Code, rec.Body.String())
	}
}

func TestSendingNeedsAConnectedAppOfYourOwnThatCanReceive(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	stranger := apitest.AddVerifiedUser(t, router, pool, "stranger@example.com", "stranger.creator")

	rec := apitest.SendToApp(t, router, stranger, workID, credentials.ConnectedApp.ID)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("send to another creator's connected app status = %d, want 404: %s",
			rec.Code, rec.Body.String())
	}
}

func TestARevokedConnectedAppLosesItsQueueAndMirrorAndAnotherKeepsBoth(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewConnectRouter(t)
	cut := apitest.ConnectApp(t, router, session, "Paper Lantern", "cut", []string{apitest.ReceivePermission, "library:sync"})
	kept := apitest.ConnectApp(t, router, session, "Paper Lantern", "kept", []string{apitest.ReceivePermission, "library:sync"})
	apitest.DeclareFormats(t, router, cut.AccessToken, []string{"test_opaque"})
	apitest.DeclareFormats(t, router, kept.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToApp(t, router, session, workID, cut.ConnectedApp.ID)
	apitest.SendToApp(t, router, session, workID, kept.ConnectedApp.ID)
	syncLibrary(t, router, cut.AccessToken, false, []map[string]any{{"workId": workID}}, nil)
	syncLibrary(t, router, kept.AccessToken, false, []map[string]any{{"workId": workID}}, nil)

	revoked := apitest.Send(t, router, apitest.BrowserRequest(
		t, http.MethodDelete, "/v1/connected-apps/"+cut.ConnectedApp.ID, nil, session))
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d, want 204: %s", revoked.Code, revoked.Body.String())
	}

	if got := rowCount(t, pool, `select count(*) from sends where connected_app_id = $1`, cut.ConnectedApp.ID); got != 0 {
		t.Fatalf("the revoked connected app kept %d sends", got)
	}
	if got := rowCount(t, pool, `select count(*) from app_library_entries where connected_app_id = $1`, cut.ConnectedApp.ID); got != 0 {
		t.Fatalf("the revoked connected app kept %d library entries", got)
	}
	if got := rowCount(t, pool, `select count(*) from sends where connected_app_id = $1`, kept.ConnectedApp.ID); got != 1 {
		t.Fatalf("the other connected app has %d sends, want 1", got)
	}
	if got := rowCount(t, pool, `select count(*) from app_library_entries where connected_app_id = $1`, kept.ConnectedApp.ID); got != 1 {
		t.Fatalf("the other connected app has %d library entries, want 1", got)
	}
}

func rowCount(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return count
}

func syncLibrary(
	t *testing.T,
	r *gin.Engine,
	token string,
	snapshot bool,
	entries []map[string]any,
	removed []string,
) apitest.LibraryResult {
	t.Helper()
	body := map[string]any{"snapshot": snapshot, "entries": entries}
	if removed != nil {
		body["removed"] = removed
	}
	rec := apitest.Send(t, r, apitest.AsApp(t, http.MethodPost, "/v1/library/sync", token, body))
	if rec.Code != http.StatusOK {
		t.Fatalf("library sync status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	return apitest.DecodeResponse[apitest.LibraryResult](t, rec)
}

func TestAnInstallWithNoVersionNumberStaysCurrentAfterPrivateEdits(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk",
		[]string{apitest.ReceivePermission, "library:sync"})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)

	result := syncLibrary(t, router, credentials.AccessToken, true,
		[]map[string]any{{"workId": workID}}, nil)
	if result.Accepted != 1 {
		t.Fatalf("library sync accepted %d, want 1", result.Accepted)
	}
	current := apitest.WorkConnectedApps(t, router, session, workID)
	if current.Items[0].InstalledVersion == nil || current.Items[0].UpdateAvailable {
		t.Fatalf("state = %+v, want installed and current", current.Items[0])
	}

	changeTheWork(t, router, session, workID)

	stale := apitest.WorkConnectedApps(t, router, session, workID)
	if stale.Items[0].UpdateAvailable ||
		*stale.Items[0].InstalledVersion != stale.VersionNumber {
		t.Fatalf("state = %+v at version %d, want the published version number unchanged",
			stale.Items[0], stale.VersionNumber)
	}
}

func TestASnapshotReplacesTheWholeMirrorForThatConnectedApp(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{"library:sync"})
	first := apitest.PublishedCharacter(t, router, session)
	second := apitest.PublishedCharacter(t, router, session)
	syncLibrary(t, router, credentials.AccessToken, false, []map[string]any{
		{"workId": first, "versionNumber": 1},
		{"workId": second, "versionNumber": 1},
	}, nil)

	result := syncLibrary(t, router, credentials.AccessToken, true, []map[string]any{
		{"workId": second, "versionNumber": 1},
	}, nil)

	if result.Accepted != 1 || result.Removed != 1 {
		t.Fatalf("snapshot = %+v, want one kept and one removed", result)
	}
	gone := apitest.WorkConnectedApps(t, router, session, first)
	if gone.Items[0].InstalledVersion != nil {
		t.Fatalf("the first work is still installed after a snapshot without it")
	}
}

func TestASnapshotMayNotAlsoCarryRemovals(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{"library:sync"})
	workID := apitest.PublishedCharacter(t, router, session)

	rec := apitest.Send(t, router, apitest.AsApp(t, http.MethodPost, "/v1/library/sync", credentials.AccessToken,
		map[string]any{
			"snapshot": true,
			"entries":  []map[string]any{{"workId": workID}},
			"removed":  []string{workID},
		}))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("contradictory snapshot status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestTheWorkPageOffersTheMostRecentlySeenConnectedAppFirst(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	older := apitest.ConnectApp(t, router, session, "Paper Lantern", "older", []string{apitest.ReceivePermission})
	newer := apitest.ConnectApp(t, router, session, "Paper Lantern", "newer", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, older.AccessToken, []string{"test_opaque"})
	apitest.DeclareFormats(t, router, newer.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)

	state := apitest.WorkConnectedApps(t, router, session, workID)

	if len(state.Items) != 2 {
		t.Fatalf("listed %d connected apps, want 2", len(state.Items))
	}
	if state.Items[0].ConnectedAppID != newer.ConnectedApp.ID {
		t.Fatalf("first connected app = %s, want the most recently seen %s",
			state.Items[0].Name, newer.ConnectedApp.Name)
	}
	if !state.Items[0].CanReceive || state.Items[0].ReportsLibrary {
		t.Fatalf("permissions on the picker = %+v, want receive only", state.Items[0])
	}
}

func TestAWorkNobodyMaySendHasNoConnectedAppState(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	draft := apitest.StartCharacter(t, router, session)

	rec := apitest.Send(t, router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/works/"+draft.ID+"/connected-apps", nil), session))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("draft connected app state status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
}

func TestALeaseThatRanOutBringsTheSendBack(t *testing.T) {
	t.Parallel()
	settings := apitest.SendSettings()
	settings.Lease = time.Millisecond
	router, session, _ := harness.NewConnectRouterWith(t, settings)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID)
	first := apitest.DecodeResponse[apitest.CollectedSends](t, apitest.Collect(t, router, credentials.AccessToken, nil))

	time.Sleep(10 * time.Millisecond)
	second := apitest.DecodeResponse[apitest.CollectedSends](t, apitest.Collect(t, router, credentials.AccessToken, nil))

	if len(second.Sends) != 1 || second.Sends[0].ID != first.Sends[0].ID {
		t.Fatalf("second collect = %+v, want the same send back", second.Sends)
	}
}

func TestASendTakenTooManyTimesWithoutAcknowledgementStops(t *testing.T) {
	t.Parallel()
	settings := apitest.SendSettings()
	settings.Lease = time.Millisecond
	settings.MaxAttempts = 2
	router, session, pool := harness.NewConnectRouterWith(t, settings)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID)

	for attempt := 0; attempt < 3; attempt++ {
		apitest.Collect(t, router, credentials.AccessToken, nil)
		time.Sleep(5 * time.Millisecond)
	}
	apitest.Collect(t, router, credentials.AccessToken, nil)

	var state, reason string
	if err := pool.QueryRow(context.Background(),
		`select state, coalesce(settled_reason, '') from sends where work_id = $1`,
		workID,
	).Scan(&state, &reason); err != nil {
		t.Fatalf("read the send: %v", err)
	}
	if state != "failed" || reason != "abandoned" {
		t.Fatalf("send = %s/%s, want failed/abandoned", state, reason)
	}
}

func TestAnExpiredSendIsSweptAway(t *testing.T) {
	t.Parallel()
	settings := apitest.SendSettings()
	settings.Retention = 250 * time.Millisecond
	router, session, pool := harness.NewConnectRouterWith(t, settings)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	if queued := apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID); queued.Code != http.StatusAccepted {
		t.Fatalf("queue the send status = %d, want 202: %s", queued.Code, queued.Body.String())
	}

	time.Sleep(settings.Retention)
	if rec := apitest.Collect(t, router, credentials.AccessToken, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("collect an expired send status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	swept := sweepSends(t, pool)

	if swept != 1 {
		t.Fatalf("swept %d sends, want 1", swept)
	}
}

func sweepSends(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	settings := apitest.SendSettings()
	apps := apitest.NewAppsService(pool)
	service := connect.NewSends(pool, nil, apps, settings)
	swept, err := service.Sweep(context.Background())
	if err != nil {
		t.Fatalf("sweep sends: %v", err)
	}
	return swept
}

func changeTheWork(t *testing.T, r *gin.Engine, session *http.Cookie, workID string) {
	t.Helper()
	rec := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/works/"+workID, nil), session))
	if rec.Code != http.StatusOK {
		t.Fatalf("read the work to change: %d %s", rec.Code, rec.Body.String())
	}
	var page apitest.StartedWork
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode the work to change: %v", err)
	}
	core := apitest.BlockNamed(t, page.Blocks, "character_core")
	edited := apitest.EditableBlock(core)
	edited.Elements[0].Content = json.RawMessage(
		`{"text":"She keeps the books, and one of them keeps her."}`)
	if saved := apitest.SaveBlock(t, r, session, workID, core.ID, edited); saved.Code != http.StatusOK {
		t.Fatalf("change the work: %d %s", saved.Code, saved.Body.String())
	}
}

func TestAConnectedAppThatLostTheReceivePermissionReleasesNothing(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk",
		[]string{apitest.ReceivePermission, "library:sync"})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID)
	if before := claimable(t, pool, credentials.ConnectedApp.ID); before != 1 {
		t.Fatalf("%d sends were claimable before the permission went, want 1", before)
	}

	if _, err := pool.Exec(context.Background(),
		`update connected_apps set permissions = array['library:sync'] where id = $1`,
		credentials.ConnectedApp.ID,
	); err != nil {
		t.Fatalf("narrow the connected app permissions: %v", err)
	}

	if after := claimable(t, pool, credentials.ConnectedApp.ID); after != 0 {
		t.Fatalf("%d sends were released to a connected app that cannot receive them", after)
	}
}

func TestARevokedConnectedAppReleasesNothingEvenWithRowsLeftBehind(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID)

	if _, err := pool.Exec(context.Background(),
		`update connected_apps
		    set revoked_at = now(), refresh_token_hash = null,
		        app_version = null, protocol_version = null,
		        capabilities = '{}', accepted_formats = '{}'
		  where id = $1`,
		credentials.ConnectedApp.ID,
	); err != nil {
		t.Fatalf("revoke the connected app without clearing its queue: %v", err)
	}

	if after := claimable(t, pool, credentials.ConnectedApp.ID); after != 0 {
		t.Fatalf("%d sends were released to a revoked connected app", after)
	}
}

func claimable(t *testing.T, pool *pgxpool.Pool, appID string) int {
	t.Helper()
	parsed, err := uuid.Parse(appID)
	if err != nil {
		t.Fatalf("parse the connected app id: %v", err)
	}
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin a claim: %v", err)
	}
	defer tx.Rollback(context.Background())
	claimed, err := db.New(tx).ClaimSends(context.Background(), db.ClaimSendsParams{
		LeaseExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Minute), Valid: true},
		ConnectedAppID: pgtype.UUID{Bytes: parsed, Valid: true},
		MaxAttempts:    5,
		BatchSize:      10,
	})
	if err != nil {
		t.Fatalf("claim sends: %v", err)
	}
	return len(claimed)
}
