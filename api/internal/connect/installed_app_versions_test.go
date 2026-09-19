package connect_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format/extension"
	"github.com/gin-gonic/gin"
)

func linkInstallations(t *testing.T, r *gin.Engine, session *http.Cookie, count int) []apitest.AppCredentials {
	t.Helper()
	connected := make([]apitest.AppCredentials, 0, count)
	for index := range count {
		credentials := apitest.ConnectApp(
			t, r, session, "Lumiverse", fmt.Sprintf("desk %d", index+1),
			[]string{apitest.ReceivePermission, apitest.LibrarySyncPermission},
		)
		apitest.DeclareCapabilities(t, r, credentials.AccessToken, []string{apitest.LumiverseInstalls}, []string{extension.SpindleID})
		connected = append(connected, credentials)
	}
	return connected
}

func installedAppVersions(t *testing.T, r http.Handler, workID string) []string {
	t.Helper()
	return apitest.ReadExtensionPage(t, r, nil, workID).InstalledAppVersions
}

func TestAnExtensionPageListsAnAppVersionOnlyOnceFiveInstallationsReportIt(t *testing.T) {
	t.Parallel()
	r, session, works, _ := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	installs := linkInstallations(t, r, session, 5)

	for _, install := range installs[:4] {
		apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0", workID)
	}
	if versions := installedAppVersions(t, r, workID); len(versions) != 0 {
		t.Fatalf("installed app versions = %v with four installations, want none", versions)
	}

	apitest.ReportLibrary(t, r, installs[4].AccessToken, "1.2.0", workID)
	if versions := installedAppVersions(t, r, workID); !slices.Equal(versions, []string{"1.2.0"}) {
		t.Fatalf("installed app versions = %v with five installations, want [1.2.0]", versions)
	}
}

func TestRevokingAnInstallationLeavesNoAppVersionOrNoticeBehind(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	installs := linkInstallations(t, r, session, 5)
	for _, install := range installs {
		apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0", workID)
	}
	withholdAsAdmin(t, r, pool, session, "verified.creator", workID)
	revokedID := installs[0].ConnectedApp.ID

	revoked := apitest.Send(t, r, apitest.BrowserRequest(t, http.MethodDelete, "/v1/connected-apps/"+revokedID, nil, session))
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d, want 204: %s", revoked.Code, revoked.Body.String())
	}

	if versions := apitest.ReadExtensionPage(t, r, session, workID).InstalledAppVersions; len(versions) != 0 {
		t.Fatalf("installed app versions = %v after one of five installations was revoked, want none", versions)
	}
	if kept := rowCount(t, pool, `select count(*) from connected_apps
		where id = $1 and library_app_version is not null`, revokedID); kept != 0 {
		t.Fatal("the revoked connected app kept the app version it reported")
	}
	if kept := rowCount(t, pool,
		`select count(*) from app_library_entries where connected_app_id = $1`, revokedID); kept != 0 {
		t.Fatalf("the revoked connected app kept %d library entries and the notices waiting on them", kept)
	}
}

func TestOnlyAnExtensionPageListsInstalledAppVersions(t *testing.T) {
	t.Parallel()
	r, session, _ := harness.NewConnectRouter(t)
	workID := apitest.PublishedCharacter(t, r, session)
	for _, install := range linkInstallations(t, r, session, 5) {
		apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0", workID)
	}

	if versions := installedAppVersions(t, r, workID); len(versions) != 0 {
		t.Fatalf("a character page lists installed app versions %v, want none", versions)
	}
}

func TestAnAppVersionCountsOnlyFromInstallationsThatCanInstallTheExtensionsApp(t *testing.T) {
	t.Parallel()
	r, session, works, _ := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	for _, install := range linkInstallations(t, r, session, 5) {
		apitest.DeclareCapabilities(t, r, install.AccessToken, []string{apitest.SillyTavernInstalls}, []string{extension.SillyTavernID})
		apitest.ReportLibrary(t, r, install.AccessToken, "1.2.0", workID)
	}

	if versions := installedAppVersions(t, r, workID); len(versions) != 0 {
		t.Fatalf("a Lumiverse extension page lists %v reported by SillyTavern installations, want none", versions)
	}
}

func declareApplicationVersion(t *testing.T, r http.Handler, token, version string) {
	t.Helper()
	rec := apitest.Send(t, r, apitest.AsApp(t, http.MethodPut, "/v1/connected-apps/me", token, map[string]any{
		"appVersion":      version,
		"protocolVersion": 1,
		"capabilities":    []string{apitest.LumiverseInstalls},
		"acceptedFormats": []string{extension.SpindleID},
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("declare status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestAnInstallationThatLeavesOutItsAppVersionCountsUnderTheVersionItDeclared(t *testing.T) {
	t.Parallel()
	r, session, works, _ := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	installs := linkInstallations(t, r, session, 5)

	for _, install := range installs {
		declareApplicationVersion(t, r, install.AccessToken, "1.1.6")
		apitest.ReportLibrary(t, r, install.AccessToken, "", workID)
	}
	if versions := installedAppVersions(t, r, workID); !slices.Equal(versions, []string{"1.1.6"}) {
		t.Fatalf("installed app versions = %v, want the declared [1.1.6]", versions)
	}

	apitest.ReportLibrary(t, r, installs[0].AccessToken, "1.2.0", workID)
	if versions := installedAppVersions(t, r, workID); len(versions) != 0 {
		t.Fatalf("installed app versions = %v after one installation reported 1.2.0, want none", versions)
	}
}

func TestALibraryReportRefusesAnAppVersionThatIsNotShortPrintableText(t *testing.T) {
	t.Parallel()
	r, session, works, _ := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	install := linkInstallations(t, r, session, 1)[0]

	for _, bad := range []string{"1.2.0\a", strings.Repeat("9", 65)} {
		rec := apitest.Send(t, r, apitest.AsApp(t, http.MethodPost, "/v1/library/sync", install.AccessToken, map[string]any{
			"snapshot":   false,
			"appVersion": bad,
			"entries":    []map[string]any{{"workId": workID}},
		}))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("application version %q = %d, want 400: %s", bad, rec.Code, rec.Body.String())
		}
	}
	if state := apitest.WorkConnectedApps(t, r, session, workID).Items[0]; state.InstalledVersion != nil {
		t.Fatal("a refused report still recorded the install")
	}
}
