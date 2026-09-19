package connect_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format/extension"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
)

func publishedSpindleExtension(t *testing.T, r http.Handler, session *http.Cookie, works *work.Service) string {
	t.Helper()
	upload := apitest.ExtensionZip(t, map[string]string{
		"spindle.json": apitest.ToolboxManifest, "dist/frontend.js": "export default {}",
	})
	return apitest.PublishExtension(t, r, session, works, "Quiet Toolbox", upload)
}

func TestAnExtensionGoesOnlyToAnInstanceDeclaringItsAppsInstallCapability(t *testing.T) {
	t.Parallel()
	r, session, works, _ := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	grant := apitest.LinkDeviceInstance(t, r, session, "Lumiverse", "desk", []string{apitest.ReceiveScope, apitest.LibrarySyncScope})

	for name, capabilities := range map[string][]string{
		"no capability":         {},
		"another app's":         {apitest.SillyTavernInstalls},
		"an unknown capability": {"lumiverse:extension-install", "chat.lumiverse:extension-installer"},
	} {
		apitest.Declare(t, r, grant.AccessToken, capabilities, []string{extension.SpindleID})
		if state := apitest.WorkInstances(t, r, session, workID).Items[0]; state.CanReceive {
			t.Errorf("%s: the page offers the extension to the instance", name)
		}
		refused := apitest.SendToInstance(t, r, session, workID, grant.Instance.ID)
		if refused.Code != http.StatusConflict || !strings.Contains(refused.Body.String(), "does not install extensions") {
			t.Errorf("%s: send = %d %s, want 409 naming the missing capability", name, refused.Code, refused.Body.String())
		}
	}

	apitest.Declare(t, r, grant.AccessToken, []string{"chat.lumiverse:preset-install", apitest.LumiverseInstalls}, []string{extension.SpindleID})
	if state := apitest.WorkInstances(t, r, session, workID).Items[0]; !state.CanReceive {
		t.Fatal("the page does not offer the extension to an instance declaring the capability")
	}
	queued := apitest.SendToInstance(t, r, session, workID, grant.Instance.ID)
	if queued.Code != http.StatusAccepted {
		t.Fatalf("send = %d: %s", queued.Code, queued.Body.String())
	}
	if first := apitest.DecodeResponse[apitest.QueuedDelivery](t, queued); first.State != "queued" || first.UpdatesInstall {
		t.Fatalf("delivery = %+v, want a queued first install", first)
	}
}

func TestTheInstallTrackFollowsTheDeliveryAndTheLibrary(t *testing.T) {
	t.Parallel()
	r, session, works, _ := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	grant := apitest.LinkDeviceInstance(t, r, session, "Lumiverse", "desk", []string{apitest.ReceiveScope, apitest.LibrarySyncScope})
	apitest.Declare(t, r, grant.AccessToken, []string{apitest.LumiverseInstalls}, []string{extension.SpindleID})
	apitest.SendToInstance(t, r, session, workID, grant.Instance.ID)

	work := apitest.DecodeResponse[apitest.DeliveryWorkList](t, apitest.Collect(t, r, grant.AccessToken, nil)).Deliveries[0]
	if work.Format != extension.SpindleID || work.Type != "extension" {
		t.Fatalf("delivery = %+v, want the Spindle archive", work)
	}
	if picked := apitest.WorkInstances(t, r, session, workID).Items[0].Delivery; picked == nil || picked.State != "released" {
		t.Fatalf("delivery = %+v, want it picked up", picked)
	}

	apitest.Collect(t, r, grant.AccessToken, []string{work.ID})
	installed := apitest.WorkInstances(t, r, session, workID).Items[0]
	if installed.Delivery == nil || installed.Delivery.State != "delivered" || installed.Delivery.SettledAt == nil {
		t.Fatalf("delivery = %+v, want it delivered", installed.Delivery)
	}
	if installed.InstalledVersion != nil {
		t.Fatalf("instance = %+v, want no library word yet", installed)
	}

	syncLibrary(t, r, grant.AccessToken, false, []map[string]any{
		{"workId": workID, "versionNumber": 1},
	}, nil)
	reported := apitest.WorkInstances(t, r, session, workID).Items[0]
	if reported.InstalledVersion == nil || *reported.InstalledVersion != 1 {
		t.Fatalf("instance = %+v, want version 1 installed", reported)
	}

	again := apitest.DecodeResponse[apitest.QueuedDelivery](t, apitest.SendToInstance(t, r, session, workID, grant.Instance.ID))
	if !again.UpdatesInstall || again.State != "queued" {
		t.Fatalf("second delivery = %+v, want one that updates the install", again)
	}
}

func TestAnInstanceThatDropsTheInstallCapabilityStopsTheDeliveryAsUnsupported(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	grant := apitest.LinkDeviceInstance(t, r, session, "Lumiverse", "desk", []string{apitest.ReceiveScope})
	apitest.Declare(t, r, grant.AccessToken, []string{apitest.LumiverseInstalls}, []string{extension.SpindleID})
	if queued := apitest.SendToInstance(t, r, session, workID, grant.Instance.ID); queued.Code != http.StatusAccepted {
		t.Fatalf("send = %d: %s", queued.Code, queued.Body.String())
	}
	apitest.Declare(t, r, grant.AccessToken, []string{}, []string{extension.SpindleID})

	if rec := apitest.Collect(t, r, grant.AccessToken, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("collect = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	var state, reason string
	if err := pool.QueryRow(context.Background(),
		`select state, coalesce(settled_reason, '') from instance_deliveries where work_id = $1`, workID,
	).Scan(&state, &reason); err != nil {
		t.Fatalf("read the delivery: %v", err)
	}
	if state != "failed" || reason != "unsupported" {
		t.Fatalf("delivery = %s/%s, want failed/unsupported", state, reason)
	}
	stopped := apitest.WorkInstances(t, r, session, workID).Items[0]
	if stopped.CanReceive || stopped.Delivery == nil || stopped.Delivery.Reason == nil || *stopped.Delivery.Reason != "unsupported" {
		t.Fatalf("instance = %+v, want the stop and its reason on the page", stopped)
	}
}

func TestALibraryEntryAddressMustBeAWebAddress(t *testing.T) {
	t.Parallel()
	r, session, works, _ := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	grant := apitest.LinkDeviceInstance(t, r, session, "Lumiverse", "desk", []string{apitest.ReceiveScope, apitest.LibrarySyncScope})

	for _, bad := range []string{"javascript:alert(1)", "lumiverse://extensions/x", "/extensions/x", "http://localhost/" + strings.Repeat("a", 600)} {
		rec := apitest.Send(t, r, apitest.AsInstance(t, http.MethodPost, "/v1/library/sync", grant.AccessToken, map[string]any{
			"snapshot": false,
			"entries":  []map[string]any{{"workId": workID, "address": bad}},
		}))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("address %q = %d, want 400: %s", bad, rec.Code, rec.Body.String())
		}
	}
	if state := apitest.WorkInstances(t, r, session, workID).Items[0]; state.InstalledVersion != nil {
		t.Fatal("a refused report still recorded the install")
	}
}
