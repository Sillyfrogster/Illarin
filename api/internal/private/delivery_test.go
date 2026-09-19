package private_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ordinaryPreset = `{
	"schemaVersion": 1,
	"name": "Ordinary preset",
	"blocks": [
		{"id":"public","name":"Public","role":"system","content":"Visible prompt.","enabled":true}
	]
}`

func settledDelivery(t *testing.T, pool *pgxpool.Pool, workID string) (string, string) {
	t.Helper()
	var state, reason string
	if err := pool.QueryRow(context.Background(),
		`select state, coalesce(settled_reason, '') from instance_deliveries where work_id = $1`,
		workID,
	).Scan(&state, &reason); err != nil {
		t.Fatalf("read the delivery: %v", err)
	}
	return state, reason
}

func TestSealingAPromptStopsAQueuedDeliveryTheAppCanNoLongerReceive(t *testing.T) {
	t.Parallel()
	router, session, works, pool := harness.NewVerifiedIngestRouterWithPool(t, apitest.Registry(t))
	metadata := apitest.ExampleMetadata("Ordinary preset")
	metadata["filename"] = "ordinary.json"
	workID := apitest.WorkIDFromIngest(
		t, apitest.UploadAndFinish(t, router, session, works, metadata, []byte(ordinaryPreset)),
	)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"invented_by_the_client"})
	if queued := apitest.SendToInstance(t, router, session, workID, grant.Instance.ID); queued.Code != http.StatusAccepted {
		t.Fatalf("queue status = %d, want 202: %s", queued.Code, queued.Body.String())
	}

	page := apitest.FetchStartedWork(t, router, session, workID)
	core := apitest.BlockNamed(t, page.Blocks, "preset_core")
	sealed := apitest.SealEveryFragment(t, apitest.EditableBlock(core), []string{"lumiverse"})
	if got := apitest.SaveBlock(t, router, session, workID, core.ID, sealed); got.Code != http.StatusOK {
		t.Fatalf("seal the prompt status = %d, want 200: %s", got.Code, got.Body.String())
	}

	rec := apitest.Collect(t, router, grant.AccessToken, nil)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("collect status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	state, reason := settledDelivery(t, pool, workID)
	if state != "failed" || reason != "unsupported" {
		t.Fatalf("delivery = %s/%s, want failed/unsupported", state, reason)
	}
	if got := apitest.DownloadEventCount(t, pool, "linked_instance"); got != 0 {
		t.Fatalf("a stopped delivery recorded %d downloads, want 0", got)
	}
}

func TestAnArtifactAddressSignedBeforeSealingHandsOverNoBytesAfterwards(t *testing.T) {
	t.Parallel()
	router, session, works, pool := harness.NewVerifiedIngestRouterWithPool(t, apitest.Registry(t))
	metadata := apitest.ExampleMetadata("Ordinary preset")
	metadata["filename"] = "ordinary.json"
	workID := apitest.WorkIDFromIngest(
		t, apitest.UploadAndFinish(t, router, session, works, metadata, []byte(ordinaryPreset)),
	)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"invented_by_the_client"})
	apitest.SendToInstance(t, router, session, workID, grant.Instance.ID)
	work := apitest.DecodeResponse[apitest.DeliveryWorkList](t, apitest.Collect(t, router, grant.AccessToken, nil)).Deliveries[0]

	page := apitest.FetchStartedWork(t, router, session, workID)
	core := apitest.BlockNamed(t, page.Blocks, "preset_core")
	sealed := apitest.SealEveryFragment(t, apitest.EditableBlock(core), []string{"lumiverse"})
	if got := apitest.SaveBlock(t, router, session, workID, core.ID, sealed); got.Code != http.StatusOK {
		t.Fatalf("seal the prompt status = %d, want 200: %s", got.Code, got.Body.String())
	}

	fetched := apitest.FetchSigned(t, router, work.Files[0].URL)

	if fetched.Code != http.StatusNotFound {
		t.Fatalf("fetch after sealing = %d, want 404: %s", fetched.Code, fetched.Body.String())
	}
	if fetched.Header().Get("X-Accel-Redirect") != "" {
		t.Fatalf("a refused artifact still pointed at %q", fetched.Header().Get("X-Accel-Redirect"))
	}
	if got := apitest.DownloadEventCount(t, pool, "linked_instance"); got != 0 {
		t.Fatalf("a refused artifact recorded %d downloads, want 0", got)
	}
}

func TestAnInstancesApplicationNameGrantsNoProtectedDelivery(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewLinkingRouter(t)
	workID := apitest.PublishSealedPreset(t, router, session, "Named app preset", "Sealed for allowed apps only.")
	borrowedName := apitest.LinkDeviceInstance(t, router, session, "Lumiverse", "desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, borrowedName.AccessToken, []string{"invented_by_the_client"})
	otherName := apitest.LinkDeviceInstance(t, router, session, "Some Other App", "tablet", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, otherName.AccessToken, []string{"preset_lumiverse"})

	offered := map[string]bool{}
	for _, state := range apitest.WorkInstances(t, router, session, workID).Items {
		offered[state.InstanceID] = state.CanReceive
	}

	if offered[borrowedName.Instance.ID] {
		t.Fatal("an instance calling itself Lumiverse was offered a sealed preset")
	}
	if !offered[otherName.Instance.ID] {
		t.Fatal("an instance accepting the allowed target was not offered a sealed preset")
	}
	if got := apitest.SendToInstance(t, router, session, workID, borrowedName.Instance.ID); got.Code != http.StatusConflict {
		t.Fatalf("queue by borrowed name = %d, want 409: %s", got.Code, got.Body.String())
	}
	if got := apitest.SendToInstance(t, router, session, workID, otherName.Instance.ID); got.Code != http.StatusAccepted {
		t.Fatalf("queue by accepted target = %d, want 202: %s", got.Code, got.Body.String())
	}
	work := apitest.DecodeResponse[apitest.DeliveryWorkList](t, apitest.Collect(t, router, otherName.AccessToken, nil)).Deliveries[0]
	if work.Format != "preset_lumiverse" {
		t.Fatalf("released format = %q, want preset_lumiverse", work.Format)
	}
}

func TestAnyReadersAllowedInstanceReceivesTheCompleteProtectedPreset(t *testing.T) {
	t.Parallel()
	router, session, works, pool := harness.NewVerifiedIngestRouterWithPool(t, apitest.Registry(t))
	metadata := apitest.ExampleMetadata("Keyed sealed preset")
	metadata["filename"] = "keyed.json"
	workID := apitest.WorkIDFromIngest(
		t, apitest.UploadAndFinish(t, router, session, works, metadata, []byte(apitest.KeyedSealedPreset)),
	)
	reader := apitest.AddVerifiedLinkingUser(t, router, pool, "reader@example.com", "reader.creator")
	grant := apitest.LinkDeviceInstance(t, router, reader, "Lumiverse", "reader desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"preset_lumiverse"})

	if queued := apitest.SendToInstance(t, router, reader, workID, grant.Instance.ID); queued.Code != http.StatusAccepted {
		t.Fatalf("a reader could not queue the preset: %d %s", queued.Code, queued.Body.String())
	}
	if before := apitest.DownloadEventCount(t, pool, "linked_instance"); before != 0 {
		t.Fatalf("queueing recorded %d downloads, want 0", before)
	}
	work := apitest.DecodeResponse[apitest.DeliveryWorkList](t, apitest.Collect(t, router, grant.AccessToken, nil)).Deliveries[0]
	artifact := apitest.FetchSigned(t, router, work.Files[0].URL)

	if artifact.Code != http.StatusOK {
		t.Fatalf("artifact status = %d, want 200: %s", artifact.Code, artifact.Body.String())
	}
	if !strings.Contains(artifact.Body.String(), "Exact private prompt.") {
		t.Fatal("a reader's allowed instance did not receive the complete preset")
	}
	if got := apitest.DownloadEventCount(t, pool, "linked_instance"); got != 1 {
		t.Fatalf("the handoff recorded %d downloads, want 1", got)
	}
	if got := apitest.Collect(t, router, grant.AccessToken, []string{work.ID}); got.Code != http.StatusNoContent {
		t.Fatalf("acknowledge status = %d, want 204: %s", got.Code, got.Body.String())
	}
}
