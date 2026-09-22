package connect_test

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func documentedRequest(t *testing.T, path string) map[string]any {
	t.Helper()
	source, err := os.ReadFile("../../../web/src/content/docs/app-integration.md")
	if err != nil {
		t.Fatal(err)
	}
	_, block, found := strings.Cut(string(source), "```http\nPOST "+path+"\n")
	if !found {
		t.Fatalf("no POST example for %s", path)
	}
	_, body, found := strings.Cut(block, "\n\n")
	if !found {
		t.Fatalf("no body for %s", path)
	}
	body, _, found = strings.Cut(body, "\n```")
	if !found {
		t.Fatalf("no end of example for %s", path)
	}
	var request map[string]any
	if err := json.Unmarshal([]byte(body), &request); err != nil {
		t.Fatalf("invalid JSON example for %s: %v", path, err)
	}
	return request
}

func TestPublishedProtocolExamples(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	start := documentedRequest(t, "/api/v1/connect/requests")
	started, _ := apitest.StartConnection(t, router, start)
	_, pending := apitest.ReviewConnectionRequest(t, router, session, started.UserCode)
	approved := apitest.Send(t, router, apitest.BrowserRequest(
		t, http.MethodPost, "/v1/connect/requests/"+started.UserCode+"/approve",
		map[string]any{"approvalToken": pending.ApprovalToken, "permissions": []string{"work:receive"}}, session,
	))
	if approved.Code != http.StatusOK {
		t.Fatalf("documented connection approval = %d: %s", approved.Code, approved.Body.String())
	}
	connected := apitest.DecodeResponse[apitest.PolledConnection](t, apitest.Poll(t, router, started.DeviceCode))
	if connected.AccessToken == nil || connected.ConnectedApp == nil {
		t.Fatal("documented connection returned no credential or installation")
	}
	apitest.DeclareFormats(t, router, *connected.AccessToken, []string{"test_opaque"})

	collect := documentedRequest(t, "/api/v1/sends/collect")
	collect["acknowledge"] = []string{}
	withoutGrant := apitest.ConnectApp(t, router, session, "Paper Lantern", "no grant", []string{})
	denied := apitest.Send(t, router, apitest.AsApp(t, http.MethodPost, "/v1/sends/collect", withoutGrant.AccessToken, collect))
	if denied.Code != http.StatusForbidden {
		t.Fatalf("documented collect without permission = %d, want 403", denied.Code)
	}

	workID := apitest.PublishedCharacter(t, router, session)
	if sent := apitest.SendToApp(t, router, session, workID, connected.ConnectedApp.ID); sent.Code != http.StatusAccepted {
		t.Fatalf("send before documented collect = %d: %s", sent.Code, sent.Body.String())
	}
	received := apitest.Send(t, router, apitest.AsApp(t, http.MethodPost, "/v1/sends/collect", *connected.AccessToken, collect))
	if received.Code != http.StatusOK || len(apitest.DecodeResponse[apitest.CollectedSends](t, received).Sends) != 1 {
		t.Fatalf("documented collect = %d: %s", received.Code, received.Body.String())
	}

	library := documentedRequest(t, "/api/v1/library/sync")
	library["entries"] = []map[string]any{{"workId": workID, "versionNumber": 1}}
	library["removed"] = []string{}
	sharing := apitest.ConnectApp(t, router, session, "Paper Lantern", "library grant", []string{"library:sync"})
	synced := apitest.Send(t, router, apitest.AsApp(t, http.MethodPost, "/v1/library/sync", sharing.AccessToken, library))
	if synced.Code != http.StatusOK {
		t.Fatalf("documented library report = %d: %s", synced.Code, synced.Body.String())
	}
}
