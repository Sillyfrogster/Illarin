package connect_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestLibrarySyncCarriesTheLatestPublishedVersionAndNotes(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	app := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{"library:sync"})
	workID := apitest.PublishedCharacter(t, router, session)

	current := syncLibrary(t, router, app.AccessToken, false,
		[]map[string]any{{"workId": workID, "versionNumber": 1}}, nil)
	want := apitest.LibraryVersion{WorkID: workID, VersionNumber: 1, Notes: ""}
	if len(current.Entries) != 1 || current.Entries[0] != want {
		t.Fatalf("current copy = %+v, want %+v", current.Entries, want)
	}

	changeTheWork(t, router, session, workID)
	privateEdits := syncLibrary(t, router, app.AccessToken, false, nil, nil)
	if len(privateEdits.Entries) != 1 || privateEdits.Entries[0] != want {
		t.Fatalf("drafted changes reached the app: %+v", privateEdits.Entries)
	}
	if published := apitest.PublishWorkVersion(t, router, session, workID,
		`{"summary":"Revised description","notes":"Keeps your existing chats.\nAdds a new detail."}`); published.Code != http.StatusOK {
		t.Fatalf("publish update = %d: %s", published.Code, published.Body.String())
	}

	want.VersionNumber = 2
	want.Notes = "Keeps your existing chats.\nAdds a new detail."
	behind := syncLibrary(t, router, app.AccessToken, false, nil, nil)
	if len(behind.Entries) != 1 || behind.Entries[0] != want {
		t.Fatalf("outdated copy = %+v, want %+v", behind.Entries, want)
	}
	state := apitest.WorkConnectedApps(t, router, session, workID).Items[0]
	if state.InstalledVersion == nil || *state.InstalledVersion != 1 || !state.UpdateAvailable {
		t.Fatalf("sync changed the installed copy: %+v", state)
	}

	current = syncLibrary(t, router, app.AccessToken, false,
		[]map[string]any{{"workId": workID, "versionNumber": 2}}, nil)
	if len(current.Entries) != 1 || current.Entries[0] != want {
		t.Fatalf("updated copy = %+v, want %+v", current.Entries, want)
	}
	removed := syncLibrary(t, router, app.AccessToken, false, nil, []string{workID})
	if removed.Removed != 1 || removed.Entries == nil || len(removed.Entries) != 0 {
		t.Fatalf("removed copy = %+v, want one removal and an empty entries array", removed)
	}
}

func TestLibrarySyncHidesWorksTheReaderCanNoLongerOpen(t *testing.T) {
	for _, state := range []string{"taken down", "deleted", "draft"} {
		t.Run(state, func(t *testing.T) {
			t.Parallel()
			router, session, pool := harness.NewConnectRouter(t)
			app := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{"library:sync"})
			workID := apitest.PublishedCharacter(t, router, session)
			before := syncLibrary(t, router, app.AccessToken, false,
				[]map[string]any{{"workId": workID, "versionNumber": 1}}, nil)
			if len(before.Entries) != 1 {
				t.Fatalf("installed work missing before access changed: %+v", before)
			}

			switch state {
			case "taken down":
				takeDownAsAdmin(t, router, pool, session, "connect.creator", workID)
			case "deleted":
				deleted := apitest.Send(t, router, apitest.BrowserRequest(t, http.MethodDelete,
					"/v1/works/"+workID, nil, session))
				if deleted.Code != http.StatusNoContent {
					t.Fatalf("delete = %d: %s", deleted.Code, deleted.Body.String())
				}
			case "draft":
				if _, err := pool.Exec(context.Background(),
					`update works set lifecycle = 'draft' where id = $1`, workID); err != nil {
					t.Fatal(err)
				}
			}

			for _, entries := range [][]map[string]any{nil, {{"workId": workID, "versionNumber": 1}}} {
				after := syncLibrary(t, router, app.AccessToken, false, entries, nil)
				if after.Entries == nil || len(after.Entries) != 0 || len(after.Takedowns) != 0 {
					t.Fatalf("inaccessible work revealed by sync: %+v", after)
				}
			}
		})
	}
}

func TestLibrarySyncReturnsOnlyThisInstallationsRemainingWorks(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	app := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{"library:sync"})
	other := apitest.ConnectApp(t, router, session, "Paper Lantern", "phone", []string{"library:sync"})
	first := apitest.PublishedCharacter(t, router, session)
	second := apitest.PublishedCharacter(t, router, session)
	apitest.ReportLibrary(t, router, other.AccessToken, "1.0.0", second)

	unlisted := apitest.Send(t, router, apitest.AuthorizedJSONRequest(t, http.MethodPut,
		"/v1/works/"+first+"/visibility", `{"visibility":"unlisted"}`, session))
	if unlisted.Code != http.StatusNoContent {
		t.Fatalf("unlist = %d: %s", unlisted.Code, unlisted.Body.String())
	}
	installed := apitest.ReportLibrary(t, router, app.AccessToken, "1.0.0", first)
	if len(installed.Entries) != 1 || installed.Entries[0].WorkID != first {
		t.Fatalf("library = %+v, want only this installation's unlisted work", installed)
	}

	cleared := syncLibrary(t, router, app.AccessToken, true, nil, nil)
	if cleared.Removed != 1 || cleared.Entries == nil || len(cleared.Entries) != 0 {
		t.Fatalf("empty snapshot = %+v, want one removal and no versions", cleared)
	}
	kept := apitest.ReportLibrary(t, router, other.AccessToken, "1.0.0")
	if len(kept.Entries) != 1 || kept.Entries[0].WorkID != second {
		t.Fatalf("snapshot changed another installation's library: %+v", kept)
	}
}
