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

func TestAnExtensionGoesOnlyToAConnectedAppDeclaringItsAppsInstallCapability(t *testing.T) {
	t.Parallel()
	r, session, works, _ := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	credentials := apitest.ConnectApp(t, r, session, "Lumiverse", "desk", []string{apitest.ReceivePermission, apitest.LibrarySyncPermission})

	for name, capabilities := range map[string][]string{
		"no capability":         {},
		"another app's":         {apitest.SillyTavernInstalls},
		"an unknown capability": {"lumiverse:extension-install", "chat.lumiverse:extension-installer"},
	} {
		apitest.DeclareCapabilities(t, r, credentials.AccessToken, capabilities, []string{extension.SpindleID})
		if state := apitest.WorkConnectedApps(t, r, session, workID).Items[0]; state.CanReceive {
			t.Errorf("%s: the page offers the extension to the connected app", name)
		}
		refused := apitest.SendToApp(t, r, session, workID, credentials.ConnectedApp.ID)
		if refused.Code != http.StatusConflict || !strings.Contains(refused.Body.String(), "does not install extensions") {
			t.Errorf("%s: send = %d %s, want 409 naming the missing capability", name, refused.Code, refused.Body.String())
		}
	}

	apitest.DeclareCapabilities(t, r, credentials.AccessToken, []string{"chat.lumiverse:preset-install", apitest.LumiverseInstalls}, []string{extension.SpindleID})
	if state := apitest.WorkConnectedApps(t, r, session, workID).Items[0]; !state.CanReceive {
		t.Fatal("the page does not offer the extension to a connected app declaring the capability")
	}
	queued := apitest.SendToApp(t, r, session, workID, credentials.ConnectedApp.ID)
	if queued.Code != http.StatusAccepted {
		t.Fatalf("send = %d: %s", queued.Code, queued.Body.String())
	}
	if first := apitest.DecodeResponse[apitest.QueuedSend](t, queued); first.State != "queued" || first.UpdatesInstall {
		t.Fatalf("send = %+v, want a queued first install", first)
	}
}

func TestTheInstallTrackFollowsTheSendAndTheLibrary(t *testing.T) {
	t.Parallel()
	r, session, works, _ := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	credentials := apitest.ConnectApp(t, r, session, "Lumiverse", "desk", []string{apitest.ReceivePermission, apitest.LibrarySyncPermission})
	apitest.DeclareCapabilities(t, r, credentials.AccessToken, []string{apitest.LumiverseInstalls}, []string{extension.SpindleID})
	apitest.SendToApp(t, r, session, workID, credentials.ConnectedApp.ID)

	work := apitest.DecodeResponse[apitest.CollectedSends](t, apitest.Collect(t, r, credentials.AccessToken, nil)).Sends[0]
	if work.Format != extension.SpindleID || work.Type != "extension" {
		t.Fatalf("send = %+v, want the Spindle archive", work)
	}
	if picked := apitest.WorkConnectedApps(t, r, session, workID).Items[0].Send; picked == nil || picked.State != "released" {
		t.Fatalf("send = %+v, want it picked up", picked)
	}

	apitest.Collect(t, r, credentials.AccessToken, []string{work.ID})
	installed := apitest.WorkConnectedApps(t, r, session, workID).Items[0]
	if installed.Send == nil || installed.Send.State != "delivered" || installed.Send.SettledAt == nil {
		t.Fatalf("send = %+v, want it delivered", installed.Send)
	}
	if installed.InstalledVersion != nil {
		t.Fatalf("connected app = %+v, want no library word yet", installed)
	}

	syncLibrary(t, r, credentials.AccessToken, false, []map[string]any{
		{"workId": workID, "versionNumber": 1},
	}, nil)
	reported := apitest.WorkConnectedApps(t, r, session, workID).Items[0]
	if reported.InstalledVersion == nil || *reported.InstalledVersion != 1 {
		t.Fatalf("connected app = %+v, want version 1 installed", reported)
	}

	again := apitest.DecodeResponse[apitest.QueuedSend](t, apitest.SendToApp(t, r, session, workID, credentials.ConnectedApp.ID))
	if !again.UpdatesInstall || again.State != "queued" {
		t.Fatalf("second send = %+v, want one that updates the install", again)
	}
}

func TestAConnectedAppThatDropsTheInstallCapabilityStopsTheSendAsUnsupported(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	credentials := apitest.ConnectApp(t, r, session, "Lumiverse", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareCapabilities(t, r, credentials.AccessToken, []string{apitest.LumiverseInstalls}, []string{extension.SpindleID})
	if queued := apitest.SendToApp(t, r, session, workID, credentials.ConnectedApp.ID); queued.Code != http.StatusAccepted {
		t.Fatalf("send = %d: %s", queued.Code, queued.Body.String())
	}
	apitest.DeclareCapabilities(t, r, credentials.AccessToken, []string{}, []string{extension.SpindleID})

	if rec := apitest.Collect(t, r, credentials.AccessToken, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("collect = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	var state, reason string
	if err := pool.QueryRow(context.Background(),
		`select state, coalesce(settled_reason, '') from sends where work_id = $1`, workID,
	).Scan(&state, &reason); err != nil {
		t.Fatalf("read the send: %v", err)
	}
	if state != "failed" || reason != "unsupported" {
		t.Fatalf("send = %s/%s, want failed/unsupported", state, reason)
	}
	stopped := apitest.WorkConnectedApps(t, r, session, workID).Items[0]
	if stopped.CanReceive || stopped.Send == nil || stopped.Send.Reason == nil || *stopped.Send.Reason != "unsupported" {
		t.Fatalf("connected app = %+v, want the stop and its reason on the page", stopped)
	}
}

func TestALibraryEntryAddressMustBeAWebAddress(t *testing.T) {
	t.Parallel()
	r, session, works, _ := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	credentials := apitest.ConnectApp(t, r, session, "Lumiverse", "desk", []string{apitest.ReceivePermission, apitest.LibrarySyncPermission})

	for _, bad := range []string{"javascript:alert(1)", "lumiverse://extensions/x", "/extensions/x", "http://localhost/" + strings.Repeat("a", 600)} {
		rec := apitest.Send(t, r, apitest.AsApp(t, http.MethodPost, "/v1/library/sync", credentials.AccessToken, map[string]any{
			"snapshot": false,
			"entries":  []map[string]any{{"workId": workID, "address": bad}},
		}))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("address %q = %d, want 400: %s", bad, rec.Code, rec.Body.String())
		}
	}
	if state := apitest.WorkConnectedApps(t, r, session, workID).Items[0]; state.InstalledVersion != nil {
		t.Fatal("a refused report still recorded the install")
	}
}
