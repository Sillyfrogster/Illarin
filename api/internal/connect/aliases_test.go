package connect_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format/extension"
)

func TestAnAppStillReportsItsLibraryUnderTheOldNameForAWork(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.LibrarySyncPermission})
	workID := apitest.PublishedCharacter(t, router, session)

	result := syncLibrary(t, router, credentials.AccessToken, true,
		[]map[string]any{{"assetId": workID, "contentGeneration": 1}}, nil)

	if result.Accepted != 1 {
		t.Fatalf("library sync accepted %d, want 1", result.Accepted)
	}
}

func TestASendStillCarriesTheOldNamesForTheWorkAndItsType(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID)

	collected := apitest.Collect(t, router, credentials.AccessToken, nil)

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
	router, session, _ := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.LibrarySyncPermission})
	workID := apitest.PublishedCharacter(t, router, session)
	syncLibrary(t, router, credentials.AccessToken, true, []map[string]any{{"workId": workID, "versionNumber": 1}}, nil)

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
		t.Fatalf("decode the connected apps: %v", err)
	}
	if list.ContentGeneration != 1 || list.VersionNumber != 1 || len(list.Items) != 1 ||
		list.Items[0].InstalledGeneration == nil || *list.Items[0].InstalledGeneration != *list.Items[0].InstalledVersion {
		t.Fatalf("connected apps = %s, want contentGeneration and installedGeneration repeating the new names", answered.Body.String())
	}
}

func TestAnAppStillConnectsUnderTheOldPathsFieldsAndPermission(t *testing.T) {
	t.Parallel()
	r, session, _ := harness.NewConnectRouter(t)
	started := apitest.SendJSON(t, r, http.MethodPost, "/v1/link/requests", apitest.JSONText(t, map[string]any{
		"applicationName": "Paper Lantern", "instanceName": "desk", "applicationVersion": "1.0.0",
		"protocolVersion": 1, "capabilities": []string{}, "acceptedTargets": []string{"portable-card-v1"},
		"scopes": []string{"asset:receive"},
	}))
	if started.Code != http.StatusCreated {
		t.Fatalf("start under the old names = %d: %s", started.Code, started.Body.String())
	}
	request := apitest.DecodeResponse[apitest.StartedConnection](t, started)

	reviewed := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/link/requests/"+request.UserCode, nil), session))
	var pending struct {
		ApplicationName string   `json:"applicationName"`
		InstanceName    string   `json:"instanceName"`
		AcceptedTargets []string `json:"acceptedTargets"`
		Scopes          []string `json:"scopes"`
		Permissions     []string `json:"permissions"`
		ApprovalToken   string   `json:"approvalToken"`
	}
	if err := json.Unmarshal(reviewed.Body.Bytes(), &pending); err != nil {
		t.Fatalf("decode the review: %v", err)
	}
	if pending.ApplicationName != "Paper Lantern" || pending.InstanceName != "desk" ||
		!slices.Equal(pending.AcceptedTargets, []string{"portable-card-v1"}) ||
		!slices.Equal(pending.Scopes, []string{"asset:receive"}) || !slices.Equal(pending.Permissions, []string{"work:receive"}) {
		t.Fatalf("review = %s, want the old names beside the new ones", reviewed.Body.String())
	}
	apitest.Send(t, r, apitest.BrowserRequest(t, http.MethodPost, "/v1/link/requests/"+request.UserCode+"/approve",
		map[string]any{"approvalToken": pending.ApprovalToken, "permissions": []string{"work:receive"}}, session))

	polled := apitest.SendJSON(t, r, http.MethodPost, "/v1/link/poll",
		apitest.JSONText(t, map[string]string{"deviceCode": request.DeviceCode}))
	var answer struct {
		Status   string `json:"status"`
		Instance struct {
			InstanceName string   `json:"instanceName"`
			Scopes       []string `json:"scopes"`
			LinkedAt     string   `json:"linkedAt"`
		} `json:"instance"`
	}
	if err := json.Unmarshal(polled.Body.Bytes(), &answer); err != nil {
		t.Fatalf("decode the poll: %v", err)
	}
	if answer.Status != "linked" || answer.Instance.InstanceName != "desk" ||
		!slices.Equal(answer.Instance.Scopes, []string{"asset:receive"}) || answer.Instance.LinkedAt == "" {
		t.Fatalf("poll on the old path = %s, want status linked and the old field names", polled.Body.String())
	}
}

func TestAFileAddressSignedBeforeTheRenameStillServesTheSend(t *testing.T) {
	t.Parallel()
	r, session, works, _ := harness.NewExtensionRouter(t)
	workID := publishedSpindleExtension(t, r, session, works)
	app := apitest.ConnectApp(t, r, session, "Lumiverse", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareCapabilities(t, r, app.AccessToken, []string{apitest.LumiverseInstalls}, []string{extension.SpindleID})
	apitest.SendToApp(t, r, session, workID, app.ConnectedApp.ID)
	sent := apitest.DecodeResponse[apitest.CollectedSends](t, apitest.Collect(t, r, app.AccessToken, nil)).Sends[0]
	if !strings.Contains(sent.Files[0].URL, "/send/"+sent.ID+"/export") {
		t.Fatalf("file address = %q, want it under /send/", sent.Files[0].URL)
	}

	fetched := apitest.FetchSigned(t, r, works.SignedURL("/delivery/"+sent.ID+"/export"))

	if fetched.Code != http.StatusOK {
		t.Fatalf("old file address = %d: %s", fetched.Code, fetched.Body.String())
	}
	if fetched.Header().Get("X-Illarin-Export-Target") != extension.SpindleID ||
		fetched.Header().Get("X-Illarin-Format") != extension.SpindleID {
		t.Fatalf("headers = %v, want the format under both names", fetched.Header())
	}
}
