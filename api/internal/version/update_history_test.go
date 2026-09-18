package version_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

type latestUpdateBody struct {
	LatestUpdate *struct {
		Number  int    `json:"number"`
		Initial bool   `json:"initial"`
		Summary string `json:"summary"`
	} `json:"latestUpdate"`
}

type recordedVersionListBody struct {
	Items []struct {
		Number  int    `json:"number"`
		Initial bool   `json:"initial"`
		Summary string `json:"summary"`
	} `json:"items"`
}

func readUpdateHistory(
	t *testing.T,
	r http.Handler,
	workID string,
	session *http.Cookie,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/works/"+workID+"/updates", nil)
	if session != nil {
		request = apitest.Authorized(request, session)
	}
	return apitest.Send(t, r, request)
}

func TestTheWorkPageCarriesTheVersionReadersHave(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)
	apitest.WriteCharacterFloor(t, r, session, started)
	if got := apitest.PublishWork(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200: %s", got.Code, got.Body.String())
	}

	first := readLatestUpdate(t, r, started.ID)
	if first.Number != 1 || first.Initial || first.Summary != "" {
		t.Fatalf("first publication reads as %+v", first)
	}

	coreBlock := apitest.BlockNamed(t, started.Blocks, "character_core")
	core := apitest.EditableBlock(coreBlock)
	core.Elements[0].Content = json.RawMessage(`{"text":"She has moved to the east shelf."}`)
	if got := apitest.SaveBlock(t, r, session, started.ID, coreBlock.ID, core); got.Code != http.StatusOK {
		t.Fatalf("save the description status = %d, want 200: %s", got.Code, got.Body.String())
	}
	update := apitest.PublishWorkUpdate(t, r, session, started.ID,
		`{"summary":"Moved her to the east shelf"}`)
	if update.Code != http.StatusOK {
		t.Fatalf("publish an update status = %d, want 200: %s", update.Code, update.Body.String())
	}

	second := readLatestUpdate(t, r, started.ID)
	if second.Number != 2 || second.Summary != "Moved her to the east shelf" {
		t.Fatalf("published update reads as %+v", second)
	}

	history := readUpdateHistory(t, r, started.ID, nil)
	if history.Code != http.StatusOK {
		t.Fatalf("history status = %d, want 200: %s", history.Code, history.Body.String())
	}
	var recorded recordedVersionListBody
	if err := json.Unmarshal(history.Body.Bytes(), &recorded); err != nil {
		t.Fatalf("decode the history: %v", err)
	}
	if len(recorded.Items) != 2 || recorded.Items[0].Number != 2 || recorded.Items[1].Number != 1 {
		t.Fatalf("history = %+v, want both versions newest first", recorded.Items)
	}
	if recorded.Items[0].Initial || recorded.Items[1].Initial {
		t.Error("a published version reads as an initial recording")
	}
}

func TestUpdateHistoryFollowsTheWorksCurrentAccess(t *testing.T) {
	t.Parallel()
	_, r, session, _, pool := harness.NewVerifiedRoutersWithPool(t, 1<<20, api.DefaultDeadlines())
	started := apitest.StartCharacter(t, r, session)
	apitest.WriteCharacterFloor(t, r, session, started)
	if got := apitest.PublishWork(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200: %s", got.Code, got.Body.String())
	}
	unlisted := apitest.Send(t, r, apitest.AuthorizedJSONRequest(t, http.MethodPut,
		"/v1/works/"+started.ID+"/visibility", `{"visibility":"unlisted"}`, session))
	if unlisted.Code != http.StatusNoContent {
		t.Fatalf("unlist status = %d, want 204: %s", unlisted.Code, unlisted.Body.String())
	}
	if open := readUpdateHistory(t, r, started.ID, nil); open.Code != http.StatusOK {
		t.Fatalf("an unlisted work's history status = %d, want 200: %s", open.Code, open.Body.String())
	}

	staff := "11111111-1111-1111-1111-111111111111"
	if _, err := pool.Exec(context.Background(),
		`insert into users (id, username) values ($1, 'staff')`, staff); err != nil {
		t.Fatalf("seed staff account: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		update works set withheld_at = now(), withheld_by = $2, withheld_reason = 'testing'
		 where id = $1
	`, started.ID, staff); err != nil {
		t.Fatalf("withhold work: %v", err)
	}

	if stranger := readUpdateHistory(t, r, started.ID, nil); stranger.Code != http.StatusNotFound {
		t.Fatalf("a withheld work's history status = %d, want 404: %s",
			stranger.Code, stranger.Body.String())
	}
	if owner := readUpdateHistory(t, r, started.ID, session); owner.Code != http.StatusOK {
		t.Fatalf("the owner's withheld history status = %d, want 200: %s",
			owner.Code, owner.Body.String())
	}
}

func readLatestUpdate(t *testing.T, r http.Handler, workID string) struct {
	Number  int    `json:"number"`
	Initial bool   `json:"initial"`
	Summary string `json:"summary"`
} {
	t.Helper()
	response := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/v1/works/"+workID, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("read the work page = %d: %s", response.Code, response.Body.String())
	}
	var page latestUpdateBody
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode the work page: %v", err)
	}
	if page.LatestUpdate == nil {
		t.Fatal("a published work's page carries no recorded version")
	}
	return *page.LatestUpdate
}
