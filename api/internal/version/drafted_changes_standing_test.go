package version_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

type draftedChangesStanding struct {
	UnpublishedChanges bool  `json:"unpublishedChanges"`
	Version            int64 `json:"draftedChangesVersion"`
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

func TestArrangementIsLiveWhileContentWaitsForPublish(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	id := publishedCharacter(t, r, session)
	owner := apitest.FetchStartedWork(t, r, session, id)
	core := apitest.BlockNamed(t, owner.Blocks, "character_core")
	messages := apitest.BlockNamed(t, owner.Blocks, "messages")
	if got := apitest.ArrangeBlocks(t, r, session, id, []apitest.ArrangedBlock{
		{ID: messages.ID, Width: "half"}, {ID: core.ID, Width: "half", Hidden: true},
	}); got.Code != http.StatusOK {
		t.Fatalf("arrange: %d %s", got.Code, got.Body.String())
	}
	if readDraftedChangesStanding(t, r, session, id).UnpublishedChanges {
		t.Fatal("arranging blocks created drafted changes")
	}
	readPublic := func() apitest.StartedWork {
		t.Helper()
		response := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/v1/works/"+id, nil))
		var page apitest.StartedWork
		if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &page) != nil {
			t.Fatalf("read public page: %d", response.Code)
		}
		return page
	}
	public := readPublic()
	if public.Blocks[0].ID != messages.ID || public.Blocks[0].Width != "half" {
		t.Fatal("public page did not take the arrangement")
	}
	owner = apitest.FetchStartedWork(t, r, session, id)
	core = apitest.BlockNamed(t, owner.Blocks, "character_core")
	update := apitest.EditableBlock(core)
	title := "A live title"
	update.Title = &title
	if got := apitest.SaveBlock(t, r, session, id, core.ID, update); got.Code != http.StatusOK {
		t.Fatalf("rename: %d %s", got.Code, got.Body.String())
	}
	if readDraftedChangesStanding(t, r, session, id).UnpublishedChanges {
		t.Fatal("renaming a block created drafted changes")
	}
	if got := apitest.PublishWorkVersion(t, r, session, id, `{"summary":"Only arranged"}`); got.Code != http.StatusConflict {
		t.Fatalf("publish arrangement: %d %s", got.Code, got.Body.String())
	}
	update.Elements[0].Content = json.RawMessage(`{"text":"A private description"}`)
	update.Width = "full"
	if got := apitest.SaveBlock(t, r, session, id, core.ID, update); got.Code != http.StatusOK {
		t.Fatalf("edit: %d %s", got.Code, got.Body.String())
	}
	if !readDraftedChangesStanding(t, r, session, id).UnpublishedChanges {
		t.Fatal("content did not create drafted changes")
	}
	comparisonPath := "/v1/works/" + id + "/versions/drafted-changes"
	request := apitest.Authorized(httptest.NewRequest(http.MethodGet, comparisonPath, nil), session)
	request.Header.Set("X-Drafted-Changes-Version", strconv.FormatInt(readDraftedChangesStanding(t, r, session, id).Version, 10))
	comparison := apitest.Send(t, r, request)
	if comparison.Code != http.StatusOK || !strings.Contains(comparison.Body.String(), "A private description") || strings.Contains(comparison.Body.String(), `"subject":"presentation"`) {
		t.Fatalf("drafted comparison: %d %s", comparison.Code, comparison.Body.String())
	}
	if anonymous := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, comparisonPath, nil)); anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous comparison: %d", anonymous.Code)
	}
	publicCore := apitest.BlockNamed(t, readPublic().Blocks, "character_core")
	if publicCore.Width != "full" || publicCore.Title != title || !publicCore.Hidden {
		t.Fatalf("public arrangement: %+v", publicCore)
	}
	if string(publicCore.Elements[0].Content) == string(update.Elements[0].Content) {
		t.Fatal("arranging leaked private content")
	}
	if got := apitest.PublishWorkVersion(t, r, session, id, `{"summary":"Changed description"}`); got.Code != http.StatusOK {
		t.Fatalf("publish content: %d %s", got.Code, got.Body.String())
	}
	if readDraftedChangesStanding(t, r, session, id).UnpublishedChanges {
		t.Fatal("published content remained drafted")
	}
	publicCore = apitest.BlockNamed(t, readPublic().Blocks, "character_core")
	var content struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(publicCore.Elements[0].Content, &content); err != nil || content.Text != "A private description" {
		t.Fatal("publishing did not release the content")
	}
	update.Layout = "trio"
	for i, slot := range []string{"left", "middle", "right"} {
		update.Elements[i].Slot = slot
	}
	if got := apitest.SaveBlock(t, r, session, id, core.ID, update); got.Code != http.StatusOK {
		t.Fatalf("save columns: %d %s", got.Code, got.Body.String())
	}
	if got := apitest.PublishWorkVersion(t, r, session, id, `{"summary":"Changed layout"}`); got.Code != http.StatusOK {
		t.Fatalf("publish columns: %d %s", got.Code, got.Body.String())
	}
	update.Layout, update.Width = "stack-3", "third"
	for i, slot := range []string{"top", "middle", "bottom"} {
		update.Elements[i].Slot = slot
	}
	if got := apitest.SaveBlock(t, r, session, id, core.ID, update); got.Code != http.StatusBadRequest {
		t.Fatalf("narrow unpublished layout: %d %s", got.Code, got.Body.String())
	}
}
