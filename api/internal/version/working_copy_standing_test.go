package version_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

type workingCopyStanding struct {
	UnpublishedChanges bool `json:"unpublishedChanges"`
}

func readWorkingCopyStanding(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	assetID string,
) workingCopyStanding {
	t.Helper()
	response := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/assets/"+assetID+"?workingCopy=true", nil), session))
	if response.Code != http.StatusOK {
		t.Fatalf("read the working copy = %d: %s", response.Code, response.Body.String())
	}
	var standing workingCopyStanding
	if err := json.Unmarshal(response.Body.Bytes(), &standing); err != nil {
		t.Fatalf("decode the working copy: %v", err)
	}
	return standing
}

func TestAWorkingCopySaysWhetherReadersHaveSeenItYet(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)
	apitest.WriteCharacterFloor(t, r, session, started)
	if got := apitest.PublishAsset(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200: %s", got.Code, got.Body.String())
	}
	if readWorkingCopyStanding(t, r, session, started.ID).UnpublishedChanges {
		t.Error("a freshly published asset reads as having changes readers cannot see")
	}

	coreBlock := apitest.BlockNamed(t, started.Blocks, "character_core")
	core := apitest.EditableBlock(coreBlock)
	core.Elements[0].Content = json.RawMessage(`{"text":"She has moved to the east shelf."}`)
	if got := apitest.SaveBlock(t, r, session, started.ID, coreBlock.ID, core); got.Code != http.StatusOK {
		t.Fatalf("save the description status = %d, want 200: %s", got.Code, got.Body.String())
	}
	if !readWorkingCopyStanding(t, r, session, started.ID).UnpublishedChanges {
		t.Error("a private edit reads as though readers already have it")
	}

	update := apitest.PublishAssetUpdate(t, r, session, started.ID,
		`{"summary":"Moved her to the east shelf"}`)
	if update.Code != http.StatusOK {
		t.Fatalf("publish an update status = %d, want 200: %s", update.Code, update.Body.String())
	}
	if readWorkingCopyStanding(t, r, session, started.ID).UnpublishedChanges {
		t.Error("a published update left the working copy reading as unpublished")
	}
}
