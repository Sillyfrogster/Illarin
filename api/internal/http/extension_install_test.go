package http

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/format/extension"
)

const (
	lumiverseInstalls   = "chat.lumiverse:extension-install"
	sillyTavernInstalls = "app.sillytavern:extension-install"
	librarySyncScope    = "library:sync"
)

func publishedSpindleExtension(t *testing.T, r http.Handler, session *http.Cookie, assets *asset.Service) string {
	t.Helper()
	upload := extensionZip(t, map[string]string{
		"spindle.json": toolboxManifest, "dist/frontend.js": "export default {}",
	})
	return publishExtension(t, r, session, assets, "Quiet Toolbox", upload)
}

func TestAnExtensionGoesOnlyToAnInstanceDeclaringItsAppsInstallCapability(t *testing.T) {
	t.Parallel()
	r, session, assets, _ := newExtensionRouter(t)
	assetID := publishedSpindleExtension(t, r, session, assets)
	grant := linkDeviceInstance(t, r, session, "Lumiverse", "desk", []string{receiveScope, librarySyncScope})

	for name, capabilities := range map[string][]string{
		"no capability":         {},
		"another app's":         {sillyTavernInstalls},
		"an unknown capability": {"lumiverse:extension-install", "chat.lumiverse:extension-installer"},
	} {
		declare(t, r, grant.AccessToken, capabilities, []string{extension.SpindleID})
		if state := assetInstances(t, r, session, assetID).Items[0]; state.CanReceive {
			t.Errorf("%s: the page offers the extension to the instance", name)
		}
		refused := sendToInstance(t, r, session, assetID, grant.Instance.ID)
		if refused.Code != http.StatusConflict || !strings.Contains(refused.Body.String(), "does not install extensions") {
			t.Errorf("%s: send = %d %s, want 409 naming the missing capability", name, refused.Code, refused.Body.String())
		}
	}

	declare(t, r, grant.AccessToken, []string{"chat.lumiverse:preset-install", lumiverseInstalls}, []string{extension.SpindleID})
	if state := assetInstances(t, r, session, assetID).Items[0]; !state.CanReceive {
		t.Fatal("the page does not offer the extension to an instance declaring the capability")
	}
	queued := sendToInstance(t, r, session, assetID, grant.Instance.ID)
	if queued.Code != http.StatusAccepted {
		t.Fatalf("send = %d: %s", queued.Code, queued.Body.String())
	}
	if first := decodeResponse[queuedDelivery](t, queued); first.State != "queued" || first.UpdatesInstall {
		t.Fatalf("delivery = %+v, want a queued first install", first)
	}
}

func TestTheInstallTrackFollowsTheDeliveryAndTheLibrary(t *testing.T) {
	t.Parallel()
	r, session, assets, _ := newExtensionRouter(t)
	assetID := publishedSpindleExtension(t, r, session, assets)
	grant := linkDeviceInstance(t, r, session, "Lumiverse", "desk", []string{receiveScope, librarySyncScope})
	declare(t, r, grant.AccessToken, []string{lumiverseInstalls}, []string{extension.SpindleID})
	sendToInstance(t, r, session, assetID, grant.Instance.ID)

	work := decodeResponse[deliveryWorkList](t, collect(t, r, grant.AccessToken, nil)).Deliveries[0]
	if work.Format != extension.SpindleID || work.Kind != "extension" {
		t.Fatalf("delivery = %+v, want the Spindle archive", work)
	}
	if picked := assetInstances(t, r, session, assetID).Items[0].Delivery; picked == nil || picked.State != "released" {
		t.Fatalf("delivery = %+v, want it picked up", picked)
	}

	collect(t, r, grant.AccessToken, []string{work.ID})
	installed := assetInstances(t, r, session, assetID).Items[0]
	if installed.Delivery == nil || installed.Delivery.State != "delivered" || installed.Delivery.SettledAt == nil {
		t.Fatalf("delivery = %+v, want it delivered", installed.Delivery)
	}
	if installed.InstalledGeneration != nil {
		t.Fatalf("instance = %+v, want no library word yet", installed)
	}

	syncLibrary(t, r, grant.AccessToken, false, []map[string]any{
		{"assetId": assetID, "contentGeneration": 1},
	}, nil)
	reported := assetInstances(t, r, session, assetID).Items[0]
	if reported.InstalledGeneration == nil || *reported.InstalledGeneration != 1 {
		t.Fatalf("instance = %+v, want generation 1 installed", reported)
	}

	again := decodeResponse[queuedDelivery](t, sendToInstance(t, r, session, assetID, grant.Instance.ID))
	if !again.UpdatesInstall || again.State != "queued" {
		t.Fatalf("second delivery = %+v, want one that updates the install", again)
	}
}

func TestAnInstanceThatDropsTheInstallCapabilityStopsTheDeliveryAsUnsupported(t *testing.T) {
	t.Parallel()
	r, session, assets, pool := newExtensionRouter(t)
	assetID := publishedSpindleExtension(t, r, session, assets)
	grant := linkDeviceInstance(t, r, session, "Lumiverse", "desk", []string{receiveScope})
	declare(t, r, grant.AccessToken, []string{lumiverseInstalls}, []string{extension.SpindleID})
	if queued := sendToInstance(t, r, session, assetID, grant.Instance.ID); queued.Code != http.StatusAccepted {
		t.Fatalf("send = %d: %s", queued.Code, queued.Body.String())
	}
	declare(t, r, grant.AccessToken, []string{}, []string{extension.SpindleID})

	if rec := collect(t, r, grant.AccessToken, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("collect = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	var state, reason string
	if err := pool.QueryRow(context.Background(),
		`select state, coalesce(settled_reason, '') from instance_deliveries where asset_id = $1`, assetID,
	).Scan(&state, &reason); err != nil {
		t.Fatalf("read the delivery: %v", err)
	}
	if state != "failed" || reason != "unsupported" {
		t.Fatalf("delivery = %s/%s, want failed/unsupported", state, reason)
	}
	stopped := assetInstances(t, r, session, assetID).Items[0]
	if stopped.CanReceive || stopped.Delivery == nil || stopped.Delivery.Reason == nil || *stopped.Delivery.Reason != "unsupported" {
		t.Fatalf("instance = %+v, want the stop and its reason on the page", stopped)
	}
}

func TestALibraryEntryAddressMustBeAWebAddress(t *testing.T) {
	t.Parallel()
	r, session, assets, _ := newExtensionRouter(t)
	assetID := publishedSpindleExtension(t, r, session, assets)
	grant := linkDeviceInstance(t, r, session, "Lumiverse", "desk", []string{receiveScope, librarySyncScope})

	for _, bad := range []string{"javascript:alert(1)", "lumiverse://extensions/x", "/extensions/x", "http://localhost/" + strings.Repeat("a", 600)} {
		rec := send(t, r, asInstance(t, http.MethodPost, "/v1/library/sync", grant.AccessToken, map[string]any{
			"snapshot": false,
			"entries":  []map[string]any{{"assetId": assetID, "address": bad}},
		}))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("address %q = %d, want 400: %s", bad, rec.Code, rec.Body.String())
		}
	}
	if state := assetInstances(t, r, session, assetID).Items[0]; state.InstalledGeneration != nil {
		t.Fatal("a refused report still recorded the install")
	}
}
