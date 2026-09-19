package connect_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestAnAppStillReportsItsLibraryUnderTheOldNameForAWork(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.LibrarySyncScope})
	workID := apitest.PublishedCharacter(t, router, session)

	result := syncLibrary(t, router, grant.AccessToken, true,
		[]map[string]any{{"assetId": workID, "contentGeneration": 1}}, nil)

	if result.Accepted != 1 {
		t.Fatalf("library sync accepted %d, want 1", result.Accepted)
	}
}

func TestASendStillCarriesTheOldNamesForTheWorkAndItsType(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceiveScope})
	apitest.DeclareTargets(t, router, grant.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToInstance(t, router, session, workID, grant.Instance.ID)

	collected := apitest.Collect(t, router, grant.AccessToken, nil)

	var list struct {
		Deliveries []map[string]any `json:"deliveries"`
	}
	if err := json.Unmarshal(collected.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode the send: %v", err)
	}
	if len(list.Deliveries) != 1 {
		t.Fatalf("collected %s, want one send", collected.Body.String())
	}
	sent := list.Deliveries[0]
	if sent["assetId"] != workID || sent["workId"] != workID || sent["kind"] != sent["type"] {
		t.Fatalf("send = %v, want workId and assetId both %s and kind matching type", sent, workID)
	}
	if sent["contentGeneration"] != sent["versionNumber"] || sent["versionNumber"] != float64(1) {
		t.Fatalf("send = %v, want contentGeneration repeating versionNumber 1", sent)
	}
	files, _ := json.Marshal(sent["files"])
	artifacts, _ := json.Marshal(sent["artifacts"])
	if string(files) != string(artifacts) || string(files) == "null" {
		t.Fatalf("send = %v, want artifacts repeating files", sent)
	}
}

func TestAnAppStillReadsTheInstalledVersionUnderTheOldName(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(t, router, session, "Paper Lantern", "desk", []string{apitest.LibrarySyncScope})
	workID := apitest.PublishedCharacter(t, router, session)
	syncLibrary(t, router, grant.AccessToken, true, []map[string]any{{"workId": workID, "versionNumber": 1}}, nil)

	answered := apitest.Send(t, router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/works/"+workID+"/instances", nil), session))
	var list struct {
		ContentGeneration int `json:"contentGeneration"`
		VersionNumber     int `json:"versionNumber"`
		Items             []struct {
			InstalledGeneration *int `json:"installedGeneration"`
			InstalledVersion    *int `json:"installedVersion"`
		} `json:"items"`
	}
	if err := json.Unmarshal(answered.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode the instances: %v", err)
	}
	if list.ContentGeneration != 1 || list.VersionNumber != 1 || len(list.Items) != 1 ||
		list.Items[0].InstalledGeneration == nil || *list.Items[0].InstalledGeneration != *list.Items[0].InstalledVersion {
		t.Fatalf("instances = %s, want contentGeneration and installedGeneration repeating the new names", answered.Body.String())
	}
}
