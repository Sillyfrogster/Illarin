package private_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestAPrivatePromptSavedUnderTheOldNamesStaysPrivate(t *testing.T) {
	t.Parallel()
	router, session := harness.NewVerifiedRouter(t)
	started := apitest.StartPreset(t, router, session, "lumiverse")
	core := apitest.EditableBlock(apitest.BlockNamed(t, started.Blocks, "preset_core"))
	core.Elements[0].Content = json.RawMessage(`{"groups":[],"fragments":[{"name":"Kept back","role":"system","text":"Old words.","protected":true,"enabled":true}]}`)
	apps := []string{"lumiverse"}
	core.AllowedApps = &apps
	if got := apitest.SaveBlock(t, router, session, started.ID, started.Blocks[0].ID, core); got.Code != http.StatusOK {
		t.Fatalf("save under the old field = %d: %s", got.Code, got.Body.String())
	}
	saved := apitest.BlockNamed(t, apitest.FetchStartedWork(t, router, session, started.ID).Blocks, "preset_core")
	if !strings.Contains(string(saved.Elements[0].Content), `"private":true`) {
		t.Fatalf("the prompt saved under the old field is not private: %s", saved.Elements[0].Content)
	}

	public := apitest.EditableBlock(apitest.BlockNamed(t, apitest.FetchStartedWork(t, router, session, started.ID).Blocks, "preset_core"))
	public.Elements[0].Content = json.RawMessage(`{"groups":[],"fragments":[{"name":"Kept back","role":"system","text":"Old words.","enabled":true}]}`)
	body, err := json.Marshal(public)
	if err != nil {
		t.Fatalf("encode the save: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatalf("decode the save: %v", err)
	}
	fields["makePromptsPublic"] = true
	confirmed, err := json.Marshal(fields)
	if err != nil {
		t.Fatalf("encode the confirmed save: %v", err)
	}
	made := apitest.Send(t, router, apitest.AuthorizedJSONRequest(t, http.MethodPut,
		"/v1/works/"+started.ID+"/blocks/"+started.Blocks[0].ID, string(confirmed), session))
	if made.Code != http.StatusOK {
		t.Fatalf("confirm under the old field = %d, want 200: %s", made.Code, made.Body.String())
	}
}

func TestTheWorkPageStillCarriesTheOldNamesForPrivateAndPreservedPrompts(t *testing.T) {
	t.Parallel()
	stack := newPreservedStack(t)
	stack.preserve(t, "1.0.0", "jailbreak", "The withheld one.")

	answer := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/works/"+stack.workID, nil,
	), stack.session))
	var page map[string]any
	if err := json.Unmarshal(answer.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode the owner's page: %v", err)
	}
	if page["sealedBlocks"] != page["preservedPrompts"] || page["sealedBlocks"] != float64(1) {
		t.Errorf("sealedBlocks = %v, preservedPrompts = %v; want 1", page["sealedBlocks"], page["preservedPrompts"])
	}
	if page["linkedInstallOnly"] != page["hasPrivatePrompts"] || page["linkedInstallOnly"] == nil {
		t.Errorf("linkedInstallOnly = %v, hasPrivatePrompts = %v", page["linkedInstallOnly"], page["hasPrivatePrompts"])
	}

	exported := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/works/"+stack.workID+"/sealed", nil,
	), stack.session))
	if exported.Code != http.StatusOK {
		t.Fatalf("export under the old path = %d: %s", exported.Code, exported.Body.String())
	}
}
