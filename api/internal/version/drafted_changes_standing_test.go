package version_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

type draftedChangesStanding struct {
	UnpublishedChanges bool `json:"unpublishedChanges"`
}

func readDraftedChangesStanding(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	workID string,
) draftedChangesStanding {
	t.Helper()
	response := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/works/"+workID+"?draftedChanges=true", nil), session))
	if response.Code != http.StatusOK {
		t.Fatalf("read the drafted changes = %d: %s", response.Code, response.Body.String())
	}
	var standing draftedChangesStanding
	if err := json.Unmarshal(response.Body.Bytes(), &standing); err != nil {
		t.Fatalf("decode the drafted changes: %v", err)
	}
	return standing
}

func TestADraftedChangesSaysWhetherReadersHaveSeenItYet(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)
	apitest.WriteCharacterFloor(t, r, session, started)
	if got := apitest.PublishWork(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200: %s", got.Code, got.Body.String())
	}
	if readDraftedChangesStanding(t, r, session, started.ID).UnpublishedChanges {
		t.Error("a freshly published work reads as having changes readers cannot see")
	}

	coreBlock := apitest.BlockNamed(t, started.Blocks, "character_core")
	core := apitest.EditableBlock(coreBlock)
	core.Elements[0].Content = json.RawMessage(`{"text":"She has moved to the east shelf."}`)
	if got := apitest.SaveBlock(t, r, session, started.ID, coreBlock.ID, core); got.Code != http.StatusOK {
		t.Fatalf("save the description status = %d, want 200: %s", got.Code, got.Body.String())
	}
	if !readDraftedChangesStanding(t, r, session, started.ID).UnpublishedChanges {
		t.Error("a private edit reads as though readers already have it")
	}

	update := apitest.PublishWorkVersion(t, r, session, started.ID,
		`{"summary":"Moved her to the east shelf"}`)
	if update.Code != http.StatusOK {
		t.Fatalf("publish an update status = %d, want 200: %s", update.Code, update.Body.String())
	}
	if readDraftedChangesStanding(t, r, session, started.ID).UnpublishedChanges {
		t.Error("a published update left the drafted changes reading as unpublished")
	}
}
