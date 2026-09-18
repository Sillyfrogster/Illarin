package connect_test

import (
	"encoding/json"
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
}
