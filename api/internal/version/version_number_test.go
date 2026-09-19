package version_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

// The version number is the one number a connected app compares, so it moves only when a version is published.
func TestTheVersionNumberMovesWhenAVersionIsPublishedAndNotBefore(t *testing.T) {
	t.Parallel()
	_, r, session, _, pool := harness.NewVerifiedRoutersWithPool(t, 1<<20, api.DefaultDeadlines())
	started := apitest.StartCharacter(t, r, session)
	if got := apitest.VersionNumber(t, pool, started.ID); got != 0 {
		t.Fatalf("a draft has version number %d, want none", got)
	}
	apitest.WriteCharacterFloor(t, r, session, started)
	if published := apitest.PublishWork(t, r, session, started.ID); published.Code != http.StatusOK {
		t.Fatalf("publish: %d %s", published.Code, published.Body.String())
	}
	if got := apitest.VersionNumber(t, pool, started.ID); got != 1 {
		t.Fatalf("version number after publishing = %d, want 1", got)
	}

	owner := apitest.FetchStartedWork(t, r, session, started.ID)
	coreBlock := apitest.BlockNamed(t, owner.Blocks, "character_core")
	core := apitest.EditableBlock(coreBlock)
	core.Elements[0].Content = json.RawMessage(`{"text":"She keeps the memories that books forget."}`)
	if saved := apitest.SaveBlock(t, r, session, started.ID, coreBlock.ID, core); saved.Code != http.StatusOK {
		t.Fatalf("draft a change: %d %s", saved.Code, saved.Body.String())
	}
	if got := apitest.VersionNumber(t, pool, started.ID); got != 1 {
		t.Fatalf("version number after a drafted change = %d, want 1 until it is published", got)
	}

	if published := apitest.PublishWorkVersion(t, r, session, started.ID, `{"summary":"A new description"}`); published.Code != http.StatusOK {
		t.Fatalf("publish the version: %d %s", published.Code, published.Body.String())
	}
	if got := apitest.VersionNumber(t, pool, started.ID); got != 2 {
		t.Fatalf("version number after publishing a version = %d, want 2", got)
	}
}

func TestTheOldWordsForVersionsAndDraftedChangesStillAnswer(t *testing.T) {
	t.Parallel()
	_, r, session, _, _ := harness.NewVerifiedRoutersWithPool(t, 1<<20, api.DefaultDeadlines())
	id := publishedCharacter(t, r, session)

	page := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/works/"+id+"?workingCopy=true", nil), session))
	if page.Code != http.StatusOK {
		t.Fatalf("read the drafted changes under the old query name: %d %s", page.Code, page.Body.String())
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(page.Body.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["workingCopyVersion"]) != string(fields["draftedChangesVersion"]) ||
		string(fields["latestUpdate"]) != string(fields["latestVersion"]) {
		t.Fatalf("the page does not repeat draftedChangesVersion and latestVersion under their old names: %s", page.Body.String())
	}

	request := apitest.AuthorizedJSONRequest(t, http.MethodPost, "/v1/works/"+id+"/updates",
		`{"summary":"Nothing changed"}`, session)
	request.Header.Set("X-Working-Copy-Version", string(fields["draftedChangesVersion"]))
	answered := apitest.Send(t, r, request)
	if answered.Code != http.StatusConflict {
		t.Fatalf("publishing under the old path and header = %d %s, want the no-changes refusal", answered.Code, answered.Body.String())
	}
}
