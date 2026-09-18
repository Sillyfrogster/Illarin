package version_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestEditingAnElementMovesTheCounterAndRearrangingThePageDoesNot(t *testing.T) {
	t.Parallel()
	_, r, session, _, pool := harness.NewVerifiedRoutersWithPool(t, 1<<20, api.DefaultDeadlines())
	started := apitest.StartCharacter(t, r, session)
	if got := apitest.ContentGeneration(t, pool, started.ID); got != 1 {
		t.Fatalf("a new draft is at content generation %d, want 1", got)
	}

	coreBlock := apitest.BlockNamed(t, started.Blocks, "character_core")
	core := apitest.EditableBlock(coreBlock)
	core.Elements[0].Content = []byte(`{"text":"She keeps the memories that books forget."}`)
	if response := apitest.SaveBlock(t, r, session, started.ID, coreBlock.ID, core); response.Code != http.StatusOK {
		t.Fatalf("save the description: %d %s", response.Code, response.Body.String())
	}
	edited := apitest.ContentGeneration(t, pool, started.ID)
	if edited != 2 {
		t.Fatalf("content generation = %d, want 2 after an edit", edited)
	}

	messages := apitest.BlockNamed(t, started.Blocks, "messages")
	response := apitest.ArrangeBlocks(t, r, session, started.ID, []apitest.ArrangedBlock{
		{ID: messages.ID, Width: "full"},
		{ID: coreBlock.ID, Width: "half"},
	})
	if response.Code != http.StatusOK {
		t.Fatalf("rearrange the page: %d %s", response.Code, response.Body.String())
	}
	if got := apitest.ContentGeneration(t, pool, started.ID); got != edited {
		t.Fatalf("content generation = %d, want %d after a reorder and a width change", got, edited)
	}

	request := httptest.NewRequest(http.MethodPut, "/v1/assets/"+started.ID+"/identity",
		strings.NewReader(`{"name":"","blurb":"","isNsfw":true}`))
	request.Header.Set("Content-Type", "application/json")
	if answered := apitest.Send(t, r, apitest.Authorized(request, session)); answered.Code != http.StatusNoContent {
		t.Fatalf("answer the adult content question: %d %s", answered.Code, answered.Body.String())
	}
	if got := apitest.ContentGeneration(t, pool, started.ID); got != edited {
		t.Fatalf("content generation = %d, want %d after the adult content answer", got, edited)
	}
}

func TestProtectedPromptGenerationFollowsCompleteArtifactBytes(t *testing.T) {
	t.Parallel()
	_, router, session, _, pool := harness.NewVerifiedRoutersWithPool(t, 1<<20, api.DefaultDeadlines())
	started := apitest.StartPreset(t, router, session, "lumiverse")
	coreBlock := apitest.BlockNamed(t, started.Blocks, "preset_core")
	core := apitest.EditableBlock(coreBlock)
	const original = "Keep every room quiet."
	core.Elements[0].Content = json.RawMessage(`{"groups":[],"fragments":[
		{"name":"House rule","role":"system","text":"` + original + `","enabled":true}
	]}`)
	if response := apitest.SaveBlock(t, router, session, started.ID, coreBlock.ID, core); response.Code != http.StatusOK {
		t.Fatalf("save public prompt: %d %s", response.Code, response.Body.String())
	}
	publicGeneration := apitest.ContentGeneration(t, pool, started.ID)

	owner := apitest.FetchStartedWork(t, router, session, started.ID)
	core = apitest.EditableBlock(apitest.BlockNamed(t, owner.Blocks, "preset_core"))
	core.Elements[0].Content = json.RawMessage(strings.Replace(
		string(core.Elements[0].Content), `"enabled":true`, `"protected":true,"enabled":true`, 1,
	))
	apps := []string{"lumiverse"}
	core.AllowedApps = &apps
	if response := apitest.SaveBlock(t, router, session, started.ID, coreBlock.ID, core); response.Code != http.StatusOK {
		t.Fatalf("seal unchanged prompt: %d %s", response.Code, response.Body.String())
	}
	if got := apitest.ContentGeneration(t, pool, started.ID); got != publicGeneration {
		t.Fatalf("content generation after sealing unchanged text = %d, want %d", got, publicGeneration)
	}
	owner = apitest.FetchStartedWork(t, router, session, started.ID)
	if !owner.LinkedInstallOnly || len(owner.AllowedApps) != 1 || owner.AllowedApps[0] != "lumiverse" {
		t.Fatalf("sealed prompt availability = linked install only %t, apps %v",
			owner.LinkedInstallOnly, owner.AllowedApps)
	}

	core = apitest.EditableBlock(apitest.BlockNamed(t, owner.Blocks, "preset_core"))
	core.Elements[0].Content = json.RawMessage(strings.Replace(
		string(core.Elements[0].Content), original, "Keep every room completely quiet.", 1,
	))
	core.AllowedApps = &apps
	if response := apitest.SaveBlock(t, router, session, started.ID, coreBlock.ID, core); response.Code != http.StatusOK {
		t.Fatalf("edit sealed prompt: %d %s", response.Code, response.Body.String())
	}
	editedGeneration := apitest.ContentGeneration(t, pool, started.ID)
	if editedGeneration != publicGeneration+1 {
		t.Fatalf("content generation after editing sealed text = %d, want %d", editedGeneration, publicGeneration+1)
	}

	owner = apitest.FetchStartedWork(t, router, session, started.ID)
	core = apitest.EditableBlock(apitest.BlockNamed(t, owner.Blocks, "preset_core"))
	core.Elements[0].Content = json.RawMessage(strings.Replace(
		string(core.Elements[0].Content), `,"protected":true`, "", 1,
	))
	core.AllowedApps = &[]string{}
	confirmed := true
	core.ExposeProtected = &confirmed
	if response := apitest.SaveBlock(t, router, session, started.ID, coreBlock.ID, core); response.Code != http.StatusOK {
		t.Fatalf("unseal unchanged prompt: %d %s", response.Code, response.Body.String())
	}
	if got := apitest.ContentGeneration(t, pool, started.ID); got != editedGeneration {
		t.Fatalf("content generation after unsealing unchanged text = %d, want %d", got, editedGeneration)
	}
	owner = apitest.FetchStartedWork(t, router, session, started.ID)
	if owner.LinkedInstallOnly || len(owner.AllowedApps) != 0 {
		t.Fatalf("unsealed prompt availability = linked install only %t, apps %v",
			owner.LinkedInstallOnly, owner.AllowedApps)
	}
}
