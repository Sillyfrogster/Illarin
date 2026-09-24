package upload_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestACharacterBuiltFromNothingLandsOnItsTwoRequiredBlocks(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)

	started := apitest.StartCharacter(t, r, session)

	if started.Type != "character" || started.Name != "" {
		t.Fatalf("started work = %+v, want an unnamed character", started)
	}
	if len(started.Blocks) != 2 {
		t.Fatalf("blocks = %d, want the two the type requires", len(started.Blocks))
	}
	if started.Blocks[0].Definition != "character_core" || started.Blocks[0].Position != 0 {
		t.Errorf("first block = %+v, want the character core", started.Blocks[0])
	}
	if started.Blocks[1].Definition != "messages" || started.Blocks[1].Position != 1 {
		t.Errorf("second block = %+v, want messages", started.Blocks[1])
	}

	core := apitest.BlockNamed(t, started.Blocks, "character_core")
	if !core.Required || !core.Hideable || !core.IsEmpty {
		t.Errorf("character core = %+v, want required, hideable and empty", core)
	}
	if core.Title != "The character" || !core.TitleIsDefault {
		t.Errorf("character core title = %q, want the definition's default", core.Title)
	}
	if core.Layout != "stack-3" || core.Width != "two_thirds" {
		t.Errorf("character core = %q at %q, want stack-3 at two thirds", core.Layout, core.Width)
	}
	if strings.Join(core.AllowedLayouts, ",") != "stack-3,trio" {
		t.Errorf("character core layouts = %v, want stack-3 and trio", core.AllowedLayouts)
	}
	wantRoles := []string{"description", "personality", "scenario"}
	if len(core.Elements) != 3 {
		t.Fatalf("character core elements = %d, want three", len(core.Elements))
	}
	for i, element := range core.Elements {
		if element.Role != wantRoles[i] {
			t.Errorf("element %d role = %q, want %q", i, element.Role, wantRoles[i])
		}
		if element.Type != "prose" || element.Display != "rich" {
			t.Errorf("%s is a %q element displayed %q, want rich prose",
				element.Role, element.Type, element.Display)
		}
		if !element.Pinned || !element.IsEmpty || element.Label == "" {
			t.Errorf("%s = %+v, want a pinned, empty, labelled element", element.Role, element)
		}
	}

	messages := apitest.BlockNamed(t, started.Blocks, "messages")
	if !messages.Required || messages.Hideable {
		t.Errorf("messages = %+v, want required and never hidden", messages)
	}
	if messages.Layout != "stack-2" {
		t.Errorf("messages layout = %q, want stack-2 with no group-only greetings",
			messages.Layout)
	}
	if len(messages.Elements) != 2 {
		t.Fatalf("messages elements = %d, want greetings and example dialogue",
			len(messages.Elements))
	}
}

func TestAnWorkBuiltFromNothingStartsAsAnUnansweredDraft(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)

	started := apitest.StartCharacter(t, r, session)

	if started.Lifecycle != "draft" {
		t.Errorf("lifecycle = %q, want draft", started.Lifecycle)
	}
	if started.IsNSFW != nil {
		t.Errorf("the adult content question was answered for the creator: %v", *started.IsNSFW)
	}
	if !started.IsOwner {
		t.Errorf("the creator is not read as the owner of the work they just made")
	}
}

func TestADraftResolvesForItsOwnerAndReturnsTheUniform404ForEveryoneElse(t *testing.T) {
	t.Parallel()
	setup, r, session, _ := harness.NewVerifiedRoutersWithService(t, 1<<20, api.DefaultDeadlines())
	started := apitest.StartCharacter(t, r, session)

	owner := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/works/"+started.ID, nil), session))
	if owner.Code != http.StatusOK {
		t.Fatalf("the owner got %d for their own draft", owner.Code)
	}

	stranger := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/v1/works/"+started.ID, nil))
	if stranger.Code != http.StatusNotFound {
		t.Errorf("a signed-out reader got %d for a draft, want 404", stranger.Code)
	}

	other := apitest.SignUp(t, setup, "other@example.com", "other.creator")
	signedIn := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/works/"+started.ID, nil), other))
	if signedIn.Code != http.StatusNotFound {
		t.Errorf("another account got %d for someone else's draft, want 404", signedIn.Code)
	}
}

func TestADraftIsInNoBrowseOrSearchResult(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)

	for _, path := range []string{
		"/v1/works",
		"/v1/works?kind=character",
		"/v1/works?q=character",
	} {
		listing := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(http.MethodGet, path, nil), session))
		if listing.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200: %s", path, listing.Code, listing.Body.String())
		}
		if strings.Contains(listing.Body.String(), started.ID) {
			t.Errorf("a draft appeared in %s", path)
		}
	}
}

func TestATypeIllarinCannotBuildIsRefusedRatherThanStarted(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)

	for _, body := range []string{`{"type":"nonsense"}`, `{"type":""}`, `{}`} {
		request := httptest.NewRequest(http.MethodPost, "/v1/works", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := apitest.Send(t, r, apitest.Authorized(request, session))
		if response.Code != http.StatusBadRequest {
			t.Errorf("POST %s status = %d, want 400: %s", body, response.Code, response.Body.String())
		}
	}
}

func TestStartingAnWorkNeedsAVerifiedAccount(t *testing.T) {
	t.Parallel()
	r := harness.NewRouter(t)

	request := httptest.NewRequest(http.MethodPost, "/v1/works",
		strings.NewReader(`{"type":"character"}`))
	request.Header.Set("Content-Type", "application/json")
	response := apitest.Send(t, r, request)

	if response.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 with no account signed in", response.Code)
	}
}

func TestBuildChoicesNameEachBuildableTypeAndTheAppsItAsksFor(t *testing.T) {
	t.Parallel()
	r, _, _, _ := harness.NewExtensionRouter(t)

	response := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/v1/build-choices", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	choices := apitest.DecodeResponse[struct {
		Types []struct {
			Type   string   `json:"type"`
			Blocks []string `json:"blocks"`
			Apps   []struct {
				ID    string `json:"id"`
				Label string `json:"label"`
			} `json:"apps"`
		} `json:"types"`
	}](t, response)
	asked := map[string]int{}
	for _, choice := range choices.Types {
		asked[choice.Type] = len(choice.Apps)
		if len(choice.Blocks) == 0 {
			t.Errorf("%s names no block its empty draft opens with", choice.Type)
		}
		for _, app := range choice.Apps {
			if app.ID == "" || app.Label == "" {
				t.Errorf("%s offers an app with no id or label: %+v", choice.Type, app)
			}
		}
	}
	if asked["character"] != 0 || asked["preset"] == 0 || asked["theme"] == 0 {
		t.Fatalf("choices = %s, want a character asking for no app and a preset and theme asking for one", response.Body.String())
	}
	if _, listed := asked["extension"]; listed {
		t.Fatalf("choices = %s, want no extension, which only arrives as an upload", response.Body.String())
	}
}
