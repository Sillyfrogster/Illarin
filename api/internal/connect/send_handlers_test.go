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
	router, session, _ := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})

	rec := apitest.Collect(t, router, grant.AccessToken, nil)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("empty wait status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	apitest.AssertNoStore(t, rec)
}

func TestSendingAnWorkReleasesItInTheFormatTheInstanceAccepts(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)

	queued := apitest.SendToInstance(t, router, session, workID, grant.Instance.ID)
	if queued.Code != http.StatusAccepted {
		t.Fatalf("send status = %d, want 202: %s", queued.Code, queued.Body.String())
	}
	waiting := apitest.DecodeResponse[apitest.QueuedDelivery](t, queued)
	if waiting.State != "queued" || waiting.WorkID != workID {
		t.Fatalf("queued delivery = %+v, want a queued delivery for the work", waiting)
	}

	rec := apitest.Collect(t, router, grant.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("collect status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	released := apitest.DecodeResponse[apitest.DeliveryWorkList](t, rec)
	if len(released.Deliveries) != 1 {
		t.Fatalf("released %d deliveries, want 1", len(released.Deliveries))
	}
	work := released.Deliveries[0]
	if work.ID != waiting.ID || work.WorkID != workID || work.Format != "test_opaque" ||
		work.Type != "character" || work.Name == "" || work.ContentGeneration < 1 {
		t.Fatalf("released work = %+v, want the queued work written as test_opaque", work)
	}
	if len(work.Artifacts) == 0 || work.Artifacts[0].Type != "export" {
		t.Fatalf("artifacts = %+v, want an export first", work.Artifacts)
	}
	if !strings.Contains(work.Artifacts[0].URL, "signature=") {
		t.Fatalf("export address %q carries no signature", work.Artifacts[0].URL)
	}
	fetched := apitest.FetchSigned(t, router, work.Artifacts[0].URL)
	if fetched.Code != http.StatusOK {
		t.Fatalf("fetch status = %d, want 200: %s", fetched.Code, fetched.Body.String())
	}
	var revisionMissing bool
	if err := pool.QueryRow(context.Background(), `
		select revision_id is null
		  from download_events
		 where work_id = $1 and authorization_class = 'linked_instance'
	`, workID).Scan(&revisionMissing); err != nil {
		t.Fatalf("read linked-instance download event: %v", err)
	}
	if !revisionMissing {
		t.Fatal("work made in Illarin recorded a source revision")
	}
}

func TestQueueingRecordsNoDownloadAndFetchingTheCreatorsOwnFileRecordsOne(t *testing.T) {
	t.Parallel()
	router, session, works, pool := harness.NewVerifiedIngestRouterWithPool(t, format.NewRegistry())
	workID := apitest.UploadVisibilityTestWork(t, router, session, works, work.VisibilityListed)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"invented_by_the_client"})

	if queued := apitest.SendToInstance(t, router, session, workID, grant.Instance.ID); queued.Code != http.StatusAccepted {
		t.Fatalf("send status = %d, want 202: %s", queued.Code, queued.Body.String())
	}
	if before := apitest.DownloadEventCount(t, pool, "linked_instance"); before != 0 {
		t.Fatalf("queueing wrote %d download events, want 0", before)
	}

	rec := apitest.Collect(t, router, grant.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("collect status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	work := apitest.DecodeResponse[apitest.DeliveryWorkList](t, rec).Deliveries[0]
	if work.Format != format.RawTarget {
		t.Fatalf("format = %q, want the creator's own file as raw", work.Format)
	}
	fetched := apitest.FetchSigned(t, router, work.Artifacts[0].URL)

	if fetched.Code != http.StatusOK || fetched.Header().Get("X-Accel-Redirect") == "" {
		t.Fatalf("fetch = %d, headers %v", fetched.Code, fetched.Header())
	}
	if got := apitest.DownloadEventCount(t, pool, "linked_instance"); got != 1 {
		t.Fatalf("recorded %d linked-instance downloads, want 1", got)
	}
}

func TestATamperedOrUnsignedDeliveryAddressIsRefused(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToInstance(t, router, session, workID, grant.Instance.ID)
	work := apitest.DecodeResponse[apitest.DeliveryWorkList](t, apitest.Collect(t, router, grant.AccessToken, nil)).Deliveries[0]

	address, err := url.Parse(work.Artifacts[0].URL)
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

func TestAnAcknowledgedDeliveryLeavesTheQueueAndStaysOnRecordAsDelivered(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToInstance(t, router, session, workID, grant.Instance.ID)
	work := apitest.DecodeResponse[apitest.DeliveryWorkList](t, apitest.Collect(t, router, grant.AccessToken, nil)).Deliveries[0]

	again := apitest.Collect(t, router, grant.AccessToken, []string{work.ID})

	if again.Code != http.StatusNoContent {
		t.Fatalf("wait after acknowledgement status = %d, want 204: %s", again.Code, again.Body.String())
	}
	delivered := apitest.WorkInstances(t, router, session, workID).Items[0].Delivery
	if delivered == nil || delivered.State != "delivered" || delivered.SettledAt == nil || delivered.UpdatesInstall {
		t.Fatalf("delivery = %+v, want it on record as delivered", delivered)
	}
	if fetched := apitest.FetchSigned(t, router, work.Artifacts[0].URL); fetched.Code != http.StatusNotFound {
		t.Fatalf("the export address still answers %d after acknowledgement", fetched.Code)
	}

	resent := apitest.DecodeResponse[apitest.QueuedDelivery](t, apitest.SendToInstance(t, router, session, workID, grant.Instance.ID))
	if resent.ID == work.ID || resent.State != "queued" {
		t.Fatalf("sending again = %+v, want a new queued delivery", resent)
	}
	dismissed := apitest.Send(t, router, apitest.BrowserRequest(t, http.MethodDelete, "/v1/deliveries/"+resent.ID, nil, session))
	if dismissed.Code != http.StatusNoContent {
		t.Fatalf("dismiss status = %d, want 204: %s", dismissed.Code, dismissed.Body.String())
	}
	if after := apitest.WorkInstances(t, router, session, workID).Items[0].Delivery; after == nil || after.ID != work.ID {
		t.Fatalf("delivery = %+v, want the delivered record back once the new one is dismissed", after)
	}
}

func TestSendingTheSameWorkTwiceQueuesItOnce(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)

	first := apitest.DecodeResponse[apitest.QueuedDelivery](t, apitest.SendToInstance(t, router, session, workID, grant.Instance.ID))
	second := apitest.DecodeResponse[apitest.QueuedDelivery](t, apitest.SendToInstance(t, router, session, workID, grant.Instance.ID))

	if first.ID != second.ID {
		t.Fatalf("two sends made two deliveries, %s and %s", first.ID, second.ID)
	}
	work := apitest.DecodeResponse[apitest.DeliveryWorkList](t, apitest.Collect(t, router, grant.AccessToken, nil))
	if len(work.Deliveries) != 1 {
		t.Fatalf("released %d deliveries, want 1", len(work.Deliveries))
	}
}

func TestAnWorkWithdrawnAfterQueueingIsRefusedAtCollection(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToInstance(t, router, session, workID, grant.Instance.ID)
	if _, err := pool.Exec(context.Background(), `
		update works work
		   set withheld_at = now(), withheld_by = work.owner_id,
		       withheld_reason = 'Copyright report under review'
		 where work.id = $1
	`, workID); err != nil {
		t.Fatalf("withhold the work: %v", err)
	}

	rec := apitest.Collect(t, router, grant.AccessToken, nil)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("collect status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	var state, reason string
	if err := pool.QueryRow(context.Background(),
		`select state, coalesce(settled_reason, '') from instance_deliveries where work_id = $1`,
		workID,
	).Scan(&state, &reason); err != nil {
		t.Fatalf("read the delivery: %v", err)
	}
	if state != "failed" || reason != "withdrawn" {
		t.Fatalf("delivery = %s/%s, want failed/withdrawn", state, reason)
	}
}

func TestAnInstanceThatAcceptsNoFormatWeCanWriteIsRefusedAtSend(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"invented_by_the_client"})
	workID := apitest.PublishedCharacter(t, router, session)

	rec := apitest.SendToInstance(t, router, session, workID, grant.Instance.ID)

	if rec.Code != http.StatusConflict {
		t.Fatalf("send status = %d, want 409: %s", rec.Code, rec.Body.String())
	}
}

func TestCollectingNeedsTheReceiveScope(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Reader", "desk", []string{"library:sync"})

	rec := apitest.Collect(t, router, grant.AccessToken, nil)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("collect without the scope status = %d, want 403: %s", rec.Code, rec.Body.String())
	}
}

func TestSendingNeedsAnInstanceOfYourOwnThatCanReceive(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	stranger := apitest.AddVerifiedLinkingUser(t, router, pool, "stranger@example.com", "stranger.creator")

	rec := apitest.SendToInstance(t, router, stranger, workID, grant.Instance.ID)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("send to another creator's instance status = %d, want 404: %s",
			rec.Code, rec.Body.String())
	}
}

func TestARevokedInstanceLosesItsQueueAndMirrorAndAnotherKeepsBoth(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewLinkingRouter(t)
	cut := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "cut", []string{apitest.ReceiveScope, "library:sync"})
	kept := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "kept", []string{apitest.ReceiveScope, "library:sync"})
	apitest.DeclareTargets(t, router, cut.AccessToken, []string{"test_opaque"})
	apitest.DeclareTargets(t, router, kept.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToInstance(t, router, session, workID, cut.Instance.ID)
	apitest.SendToInstance(t, router, session, workID, kept.Instance.ID)
	syncLibrary(t, router, cut.AccessToken, false, []map[string]any{{"assetId": workID}}, nil)
	syncLibrary(t, router, kept.AccessToken, false, []map[string]any{{"assetId": workID}}, nil)

	revoked := apitest.Send(t, router, apitest.BrowserRequest(
		t, http.MethodDelete, "/v1/instances/"+cut.Instance.ID, nil, session))
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d, want 204: %s", revoked.Code, revoked.Body.String())
	}

	if got := rowCount(t, pool, `select count(*) from instance_deliveries where instance_id = $1`, cut.Instance.ID); got != 0 {
		t.Fatalf("the revoked instance kept %d deliveries", got)
	}
	if got := rowCount(t, pool, `select count(*) from instance_library_entries where instance_id = $1`, cut.Instance.ID); got != 0 {
		t.Fatalf("the revoked instance kept %d library entries", got)
	}
	if got := rowCount(t, pool, `select count(*) from instance_deliveries where instance_id = $1`, kept.Instance.ID); got != 1 {
		t.Fatalf("the other instance has %d deliveries, want 1", got)
	}
	if got := rowCount(t, pool, `select count(*) from instance_library_entries where instance_id = $1`, kept.Instance.ID); got != 1 {
		t.Fatalf("the other instance has %d library entries, want 1", got)
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
	rec := apitest.Send(t, r, apitest.AsInstance(t, http.MethodPost, "/v1/library/sync", token, body))
	if rec.Code != http.StatusOK {
		t.Fatalf("library sync status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	return apitest.DecodeResponse[apitest.LibraryResult](t, rec)
}

func TestAnInstallWithNoGenerationStaysCurrentAfterPrivateEdits(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk",
		[]string{apitest.ReceiveScope, "library:sync"})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)

	result := syncLibrary(t, router, grant.AccessToken, true,
		[]map[string]any{{"assetId": workID}}, nil)
	if result.Accepted != 1 {
		t.Fatalf("library sync accepted %d, want 1", result.Accepted)
	}
	current := apitest.WorkInstances(t, router, session, workID)
	if current.Items[0].InstalledGeneration == nil || current.Items[0].UpdateAvailable {
		t.Fatalf("state = %+v, want installed and current", current.Items[0])
	}

	changeTheWork(t, router, session, workID)

	stale := apitest.WorkInstances(t, router, session, workID)
	if stale.Items[0].UpdateAvailable ||
		*stale.Items[0].InstalledGeneration != stale.ContentGeneration {
		t.Fatalf("state = %+v at generation %d, want the published generation unchanged",
			stale.Items[0], stale.ContentGeneration)
	}
}

func TestASnapshotReplacesTheWholeMirrorForThatInstance(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{"library:sync"})
	first := apitest.PublishedCharacter(t, router, session)
	second := apitest.PublishedCharacter(t, router, session)
	syncLibrary(t, router, grant.AccessToken, false, []map[string]any{
		{"assetId": first, "contentGeneration": 1},
		{"assetId": second, "contentGeneration": 1},
	}, nil)

	result := syncLibrary(t, router, grant.AccessToken, true, []map[string]any{
		{"assetId": second, "contentGeneration": 1},
	}, nil)

	if result.Accepted != 1 || result.Removed != 1 {
		t.Fatalf("snapshot = %+v, want one kept and one removed", result)
	}
	gone := apitest.WorkInstances(t, router, session, first)
	if gone.Items[0].InstalledGeneration != nil {
		t.Fatalf("the first work is still installed after a snapshot without it")
	}
}

func TestASnapshotMayNotAlsoCarryRemovals(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{"library:sync"})
	workID := apitest.PublishedCharacter(t, router, session)

	rec := apitest.Send(t, router, apitest.AsInstance(t, http.MethodPost, "/v1/library/sync", grant.AccessToken,
		map[string]any{
			"snapshot": true,
			"entries":  []map[string]any{{"assetId": workID}},
			"removed":  []string{workID},
		}))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("contradictory snapshot status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestTheWorkPageOffersTheMostRecentlySeenInstanceFirst(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewLinkingRouter(t)
	older := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "older", []string{apitest.ReceiveScope})
	newer := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "newer", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, older.AccessToken, []string{"test_opaque"})
	apitest.DeclareTargets(t, router, newer.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)

	state := apitest.WorkInstances(t, router, session, workID)

	if len(state.Items) != 2 {
		t.Fatalf("listed %d instances, want 2", len(state.Items))
	}
	if state.Items[0].InstanceID != newer.Instance.ID {
		t.Fatalf("first instance = %s, want the most recently seen %s",
			state.Items[0].InstanceName, newer.Instance.InstanceName)
	}
	if !state.Items[0].CanReceive || state.Items[0].ReportsLibrary {
		t.Fatalf("scopes on the picker = %+v, want receive only", state.Items[0])
	}
}

func TestAnWorkNobodyMaySendHasNoInstanceState(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewLinkingRouter(t)
	apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})
	draft := apitest.StartCharacter(t, router, session)

	rec := apitest.Send(t, router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/assets/"+draft.ID+"/instances", nil), session))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("draft instance state status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
}

func TestALeaseThatRanOutBringsTheDeliveryBack(t *testing.T) {
	t.Parallel()
	settings := apitest.DeliverySettings()
	settings.Lease = time.Millisecond
	router, session, _ := harness.NewLinkingRouterWith(t, settings)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToInstance(t, router, session, workID, grant.Instance.ID)
	first := apitest.DecodeResponse[apitest.DeliveryWorkList](t, apitest.Collect(t, router, grant.AccessToken, nil))

	time.Sleep(10 * time.Millisecond)
	second := apitest.DecodeResponse[apitest.DeliveryWorkList](t, apitest.Collect(t, router, grant.AccessToken, nil))

	if len(second.Deliveries) != 1 || second.Deliveries[0].ID != first.Deliveries[0].ID {
		t.Fatalf("second collect = %+v, want the same delivery back", second.Deliveries)
	}
}

func TestADeliveryTakenTooManyTimesWithoutAcknowledgementStops(t *testing.T) {
	t.Parallel()
	settings := apitest.DeliverySettings()
	settings.Lease = time.Millisecond
	settings.MaxAttempts = 2
	router, session, pool := harness.NewLinkingRouterWith(t, settings)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToInstance(t, router, session, workID, grant.Instance.ID)

	for attempt := 0; attempt < 3; attempt++ {
		apitest.Collect(t, router, grant.AccessToken, nil)
		time.Sleep(5 * time.Millisecond)
	}
	apitest.Collect(t, router, grant.AccessToken, nil)

	var state, reason string
	if err := pool.QueryRow(context.Background(),
		`select state, coalesce(settled_reason, '') from instance_deliveries where work_id = $1`,
		workID,
	).Scan(&state, &reason); err != nil {
		t.Fatalf("read the delivery: %v", err)
	}
	if state != "failed" || reason != "abandoned" {
		t.Fatalf("delivery = %s/%s, want failed/abandoned", state, reason)
	}
}

func TestAnExpiredDeliveryIsSweptAway(t *testing.T) {
	t.Parallel()
	settings := apitest.DeliverySettings()
	settings.Retention = 250 * time.Millisecond
	router, session, pool := harness.NewLinkingRouterWith(t, settings)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	if queued := apitest.SendToInstance(t, router, session, workID, grant.Instance.ID); queued.Code != http.StatusAccepted {
		t.Fatalf("queue the delivery status = %d, want 202: %s", queued.Code, queued.Body.String())
	}

	time.Sleep(settings.Retention)
	if rec := apitest.Collect(t, router, grant.AccessToken, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("collect an expired delivery status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	swept := sweepDeliveries(t, pool)

	if swept != 1 {
		t.Fatalf("swept %d deliveries, want 1", swept)
	}
}

func sweepDeliveries(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	settings := apitest.DeliverySettings()
	links := apitest.NewLinkingService(pool)
	service := connect.NewSends(pool, nil, links, settings)
	swept, err := service.Sweep(context.Background())
	if err != nil {
		t.Fatalf("sweep deliveries: %v", err)
	}
	return swept
}

func changeTheWork(t *testing.T, r *gin.Engine, session *http.Cookie, workID string) {
	t.Helper()
	rec := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/assets/"+workID, nil), session))
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

func TestAnInstanceThatLostTheReceiveScopeReleasesNothing(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk",
		[]string{apitest.ReceiveScope, "library:sync"})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToInstance(t, router, session, workID, grant.Instance.ID)
	if before := claimable(t, pool, grant.Instance.ID); before != 1 {
		t.Fatalf("%d deliveries were claimable before the scope went, want 1", before)
	}

	if _, err := pool.Exec(context.Background(),
		`update linked_instances set scopes = array['library:sync'] where id = $1`,
		grant.Instance.ID,
	); err != nil {
		t.Fatalf("narrow the instance scopes: %v", err)
	}

	if after := claimable(t, pool, grant.Instance.ID); after != 0 {
		t.Fatalf("%d deliveries were released to an instance that cannot receive them", after)
	}
}

func TestARevokedInstanceReleasesNothingEvenWithRowsLeftBehind(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToInstance(t, router, session, workID, grant.Instance.ID)

	if _, err := pool.Exec(context.Background(),
		`update linked_instances
		    set revoked_at = now(), refresh_token_hash = null,
		        application_version = null, protocol_version = null,
		        capabilities = '{}', accepted_targets = '{}'
		  where id = $1`,
		grant.Instance.ID,
	); err != nil {
		t.Fatalf("revoke the instance without clearing its queue: %v", err)
	}

	if after := claimable(t, pool, grant.Instance.ID); after != 0 {
		t.Fatalf("%d deliveries were released to a revoked instance", after)
	}
}

func claimable(t *testing.T, pool *pgxpool.Pool, instanceID string) int {
	t.Helper()
	parsed, err := uuid.Parse(instanceID)
	if err != nil {
		t.Fatalf("parse the instance id: %v", err)
	}
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin a claim: %v", err)
	}
	defer tx.Rollback(context.Background())
	claimed, err := db.New(tx).ClaimDeliveries(context.Background(), db.ClaimDeliveriesParams{
		LeaseExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Minute), Valid: true},
		InstanceID:     pgtype.UUID{Bytes: parsed, Valid: true},
		MaxAttempts:    5,
		BatchSize:      10,
	})
	if err != nil {
		t.Fatalf("claim deliveries: %v", err)
	}
	return len(claimed)
}
