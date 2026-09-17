package http

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

func linkInstallations(t *testing.T, r *gin.Engine, session *http.Cookie, count int) []apitest.TokenGrant {
	t.Helper()
	grants := make([]apitest.TokenGrant, 0, count)
	for index := range count {
		grant := apitest.LinkDeviceInstance(
			t, r, session, "Lumiverse", fmt.Sprintf("desk %d", index+1),
			[]string{apitest.ReceiveScope, apitest.LibrarySyncScope},
		)
		declare(t, r, grant.AccessToken, []string{lumiverseInstalls}, []string{extension.SpindleID})
		grants = append(grants, grant)
	}
	return grants
}

func installedAppVersions(t *testing.T, r http.Handler, assetID string) []string {
	t.Helper()
	return readExtensionPage(t, r, nil, assetID).InstalledAppVersions
}

func TestAnExtensionPageListsAnAppVersionOnlyOnceFiveInstallationsReportIt(t *testing.T) {
	t.Parallel()
	r, session, assets, _ := harness.NewExtensionRouter(t)
	assetID := publishedSpindleExtension(t, r, session, assets)
	installs := linkInstallations(t, r, session, 5)

	for _, install := range installs[:4] {
		apitest.ReportInstalled(t, r, install.AccessToken, "1.2.0", assetID)
	}
	if versions := installedAppVersions(t, r, assetID); len(versions) != 0 {
		t.Fatalf("installed app versions = %v with four installations, want none", versions)
	}

	apitest.ReportInstalled(t, r, installs[4].AccessToken, "1.2.0", assetID)
	if versions := installedAppVersions(t, r, assetID); !slices.Equal(versions, []string{"1.2.0"}) {
		t.Fatalf("installed app versions = %v with five installations, want [1.2.0]", versions)
	}
}

func TestRevokingAnInstallationLeavesNoAppVersionOrNoticeBehind(t *testing.T) {
	t.Parallel()
	r, session, assets, pool := harness.NewExtensionRouter(t)
	assetID := publishedSpindleExtension(t, r, session, assets)
	installs := linkInstallations(t, r, session, 5)
	for _, install := range installs {
		apitest.ReportInstalled(t, r, install.AccessToken, "1.2.0", assetID)
	}
	withholdAsAdmin(t, r, pool, session, "verified.creator", assetID)
	revokedID := installs[0].Instance.ID

	revoked := apitest.Send(t, r, apitest.BrowserRequest(t, http.MethodDelete, "/v1/instances/"+revokedID, nil, session))
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d, want 204: %s", revoked.Code, revoked.Body.String())
	}

	if versions := readExtensionPage(t, r, session, assetID).InstalledAppVersions; len(versions) != 0 {
		t.Fatalf("installed app versions = %v after one of five installations was revoked, want none", versions)
	}
	if kept := rowCount(t, pool, `select count(*) from linked_instances
		where id = $1 and library_application_version is not null`, revokedID); kept != 0 {
		t.Fatal("the revoked instance kept the app version it reported")
	}
	if kept := rowCount(t, pool,
		`select count(*) from instance_library_entries where instance_id = $1`, revokedID); kept != 0 {
		t.Fatalf("the revoked instance kept %d library entries and the notices waiting on them", kept)
	}
}

func TestOnlyAnExtensionPageListsInstalledAppVersions(t *testing.T) {
	t.Parallel()
	r, session, _ := newLinkingRouter(t)
	assetID := apitest.PublishedAsset(t, r, session)
	for _, install := range linkInstallations(t, r, session, 5) {
		apitest.ReportInstalled(t, r, install.AccessToken, "1.2.0", assetID)
	}

	if versions := installedAppVersions(t, r, assetID); len(versions) != 0 {
		t.Fatalf("a character page lists installed app versions %v, want none", versions)
	}
}

func TestAnAppVersionCountsOnlyFromInstallationsThatCanInstallTheExtensionsApp(t *testing.T) {
	t.Parallel()
	r, session, assets, _ := harness.NewExtensionRouter(t)
	assetID := publishedSpindleExtension(t, r, session, assets)
	for _, install := range linkInstallations(t, r, session, 5) {
		declare(t, r, install.AccessToken, []string{sillyTavernInstalls}, []string{extension.SillyTavernID})
		apitest.ReportInstalled(t, r, install.AccessToken, "1.2.0", assetID)
	}

	if versions := installedAppVersions(t, r, assetID); len(versions) != 0 {
		t.Fatalf("a Lumiverse extension page lists %v reported by SillyTavern installations, want none", versions)
	}
}

func declareApplicationVersion(t *testing.T, r http.Handler, token, version string) {
	t.Helper()
	rec := apitest.Send(t, r, apitest.AsInstance(t, http.MethodPut, "/v1/instances/me", token, map[string]any{
		"applicationVersion": version,
		"protocolVersion":    1,
		"capabilities":       []string{lumiverseInstalls},
		"acceptedTargets":    []string{extension.SpindleID},
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("declare status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestAnInstallationThatLeavesOutItsAppVersionCountsUnderTheVersionItDeclared(t *testing.T) {
	t.Parallel()
	r, session, assets, _ := harness.NewExtensionRouter(t)
	assetID := publishedSpindleExtension(t, r, session, assets)
	installs := linkInstallations(t, r, session, 5)

	for _, install := range installs {
		declareApplicationVersion(t, r, install.AccessToken, "1.1.6")
		apitest.ReportInstalled(t, r, install.AccessToken, "", assetID)
	}
	if versions := installedAppVersions(t, r, assetID); !slices.Equal(versions, []string{"1.1.6"}) {
		t.Fatalf("installed app versions = %v, want the declared [1.1.6]", versions)
	}

	apitest.ReportInstalled(t, r, installs[0].AccessToken, "1.2.0", assetID)
	if versions := installedAppVersions(t, r, assetID); len(versions) != 0 {
		t.Fatalf("installed app versions = %v after one installation reported 1.2.0, want none", versions)
	}
}

func TestALibraryReportRefusesAnAppVersionThatIsNotShortPrintableText(t *testing.T) {
	t.Parallel()
	r, session, assets, _ := harness.NewExtensionRouter(t)
	assetID := publishedSpindleExtension(t, r, session, assets)
	install := linkInstallations(t, r, session, 1)[0]

	for _, bad := range []string{"1.2.0\a", strings.Repeat("9", 65)} {
		rec := apitest.Send(t, r, apitest.AsInstance(t, http.MethodPost, "/v1/library/sync", install.AccessToken, map[string]any{
			"snapshot":           false,
			"applicationVersion": bad,
			"entries":            []map[string]any{{"assetId": assetID}},
		}))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("application version %q = %d, want 400: %s", bad, rec.Code, rec.Body.String())
		}
	}
	if state := assetInstances(t, r, session, assetID).Items[0]; state.InstalledGeneration != nil {
		t.Fatal("a refused report still recorded the install")
	}
}
