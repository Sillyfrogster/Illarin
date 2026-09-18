package version_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/Sillyfrogster/Illarin/api/internal/download"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestPrivateWorkEditsStayPrivateAcrossHTTPReads(t *testing.T) {
	t.Parallel()
	_, router, session, _, pool := harness.NewVerifiedRoutersWithPool(t, 1<<20, api.DefaultDeadlines())
	id := publishedCharacter(t, router, session)
	before := apitest.FetchWorkPage(t, router, "/v1/works/"+id)
	generation := apitest.ContentGeneration(t, pool, id)
	if got := apitest.SaveDetails(t, router, session, id, `{"name":"Unpublished name","blurb":"","isNsfw":true}`); got.Code != http.StatusNoContent {
		t.Fatalf("save private header: %d %s", got.Code, got.Body.String())
	}
	owner := apitest.FetchStartedWork(t, router, session, id)
	core := apitest.EditableBlock(apitest.BlockNamed(t, owner.Blocks, "character_core"))
	core.Elements[0].Content = json.RawMessage(`{"text":"Unpublished description"}`)
	if got := apitest.SaveBlock(t, router, session, id, apitest.BlockNamed(t, owner.Blocks, "character_core").ID, core); got.Code != http.StatusOK {
		t.Fatalf("save private block: %d %s", got.Code, got.Body.String())
	}
	public := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/works/"+id, nil))
	if public.Code != http.StatusOK || strings.Contains(public.Body.String(), "Unpublished") {
		t.Fatalf("public page leaked a private edit: %d", public.Code)
	}
	if got := apitest.FetchWorkPage(t, router, "/v1/works/"+id); got.Name != before.Name || got.IsNSFW != before.IsNSFW {
		t.Fatal("private header changed public metadata")
	}
	working := apitest.Send(t, router, apitest.Authorized(httptest.NewRequest(http.MethodGet, "/v1/works/"+id+"?workingCopy=true", nil), session))
	if working.Code != http.StatusOK || !strings.Contains(working.Body.String(), "Unpublished description") || !strings.Contains(working.Body.String(), "Unpublished name") {
		t.Fatalf("owner working copy: %d %s", working.Code, working.Body.String())
	}
	anonymous := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/works/"+id+"?workingCopy=true", nil))
	if anonymous.Code != http.StatusNotFound {
		t.Fatalf("anonymous working copy: %d", anonymous.Code)
	}
	if got := apitest.ContentGeneration(t, pool, id); got != generation {
		t.Fatalf("private edits advanced generation from %d to %d", generation, got)
	}
}

func TestWorkingCopyMediaIsPrivateOnAPublishedWork(t *testing.T) {
	t.Parallel()
	router, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, router, session)
	apitest.WriteCharacterFloor(t, router, session, started)
	if got := apitest.Send(t, router, apitest.Authorized(apitest.MediaUploadRequest(t, started.ID, "avatar", apitest.PNG(t, 40, 60)), session)); got.Code != http.StatusCreated {
		t.Fatalf("upload original cover: %d", got.Code)
	}
	if got := apitest.PublishWork(t, router, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish: %d", got.Code)
	}
	before := apitest.FetchWorkPage(t, router, "/v1/works/"+started.ID)
	if got := apitest.Send(t, router, apitest.Authorized(apitest.MediaUploadRequest(t, started.ID, "avatar", apitest.PNG(t, 80, 120)), session)); got.Code != http.StatusCreated {
		t.Fatalf("upload private cover: %d", got.Code)
	}
	public := apitest.FetchWorkPage(t, router, "/v1/works/"+started.ID)
	if len(public.Media) != 1 || public.Media[0].ID != before.Media[0].ID {
		t.Fatal("private cover changed the public media")
	}
	response := apitest.Send(t, router, apitest.Authorized(httptest.NewRequest(http.MethodGet, "/v1/works/"+started.ID+"?workingCopy=true", nil), session))
	var working apitest.WorkPageResponse
	if err := json.Unmarshal(response.Body.Bytes(), &working); err != nil || response.Code != http.StatusOK {
		t.Fatalf("working copy: %d, %v", response.Code, err)
	}
	if len(working.Media) != 1 || working.Media[0].ID == public.Media[0].ID {
		t.Fatalf("working copy did not take the new cover: %+v", working.Media)
	}
	for _, signed := range []string{working.Media[0].DetailURL, working.Media[0].ThumbURL} {
		unsigned, _, _ := strings.Cut(signed, "?")
		for _, path := range []string{unsigned, signed} {
			if got := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, path, nil)); got.Code != http.StatusNotFound {
				t.Fatalf("anonymous private media access: %d", got.Code)
			}
		}
		served := apitest.Send(t, router, apitest.Authorized(httptest.NewRequest(http.MethodGet, signed, nil), session))
		if served.Code != http.StatusOK || served.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("owner private media access: %d", served.Code)
		}
	}
}

func TestPrivateProtectedTextDoesNotReachLinkedDelivery(t *testing.T) {
	t.Parallel()
	router, session, works, pool := harness.NewVerifiedIngestRouterWithPool(t, apitest.Registry(t))
	id := apitest.PublishSealedPreset(t, router, session, "Recorded preset", "Recorded secret")
	owner := apitest.FetchStartedWork(t, router, session, id)
	core := apitest.EditableBlock(apitest.BlockNamed(t, owner.Blocks, "preset_core"))
	core.Elements[0].Content = json.RawMessage(strings.ReplaceAll(string(core.Elements[0].Content), "Recorded secret", "Unpublished secret"))
	if got := apitest.SaveBlock(t, router, session, id, apitest.BlockNamed(t, owner.Blocks, "preset_core").ID, core); got.Code != http.StatusOK {
		t.Fatalf("save private protected text: %d %s", got.Code, got.Body.String())
	}
	exported, err := download.NewService(pool, works).OpenExportForLinkedInstance(t.Context(), uuid.MustParse(id), "preset_lumiverse")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(exported.Body), "Unpublished secret") || !strings.Contains(string(exported.Body), "Recorded secret") {
		t.Fatal("linked delivery did not use the recorded protected payload")
	}
	core.Elements[0].Content = json.RawMessage(`{"groups":[],"fragments":[{"name":"Replacement","role":"system","text":"New private prompt","enabled":true}]}`)
	if got := apitest.SaveBlock(t, router, session, id, apitest.BlockNamed(t, owner.Blocks, "preset_core").ID, core); got.Code != http.StatusOK {
		t.Fatalf("replace protected working-copy prompt: %d %s", got.Code, got.Body.String())
	}
	for _, target := range []string{"preset_lumiverse", "preset_sillytavern"} {
		if _, err := download.NewService(pool, works).OpenExportForLinkedInstance(t.Context(), uuid.MustParse(id), target); !errors.Is(err, download.ErrLinkedInstallOnly) {
			t.Fatalf("delivery without a policy for %s: %v", target, err)
		}
	}
	if _, err := works.DeliverableWork(t.Context(), pool, uuid.MustParse(id)); !errors.Is(err, connect.ErrNotDeliverable) {
		t.Fatalf("advertised delivery without a policy: %v", err)
	}
}

func publishedCharacter(t *testing.T, router *gin.Engine, session *http.Cookie) string {
	t.Helper()
	started := apitest.StartCharacter(t, router, session)
	apitest.WriteCharacterFloor(t, router, session, started)
	if got := apitest.PublishWork(t, router, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200: %s", got.Code, got.Body.String())
	}
	return started.ID
}
