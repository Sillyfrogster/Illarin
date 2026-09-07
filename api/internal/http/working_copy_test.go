package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPrivateAssetEditsStayPrivateAcrossHTTPReads(t *testing.T) {
	_, router, session, _, pool := newVerifiedTestRoutersWithPool(t, 1<<20, DefaultDeadlines())
	id := publishedCharacter(t, router, session)
	before := fetchAssetPage(t, router, "/v1/assets/"+id)
	generation := contentGeneration(t, pool, id)
	if got := saveIdentity(t, router, session, id, `{"name":"Unpublished name","isNsfw":true}`); got.Code != http.StatusNoContent {
		t.Fatalf("save private header: %d %s", got.Code, got.Body.String())
	}
	owner := fetchStartedAsset(t, router, session, id)
	core := editableBlock(blockNamed(t, owner.Blocks, "character_core"))
	core.Elements[0].Content = json.RawMessage(`{"text":"Unpublished description"}`)
	if got := saveBlock(t, router, session, id, blockNamed(t, owner.Blocks, "character_core").ID, core); got.Code != http.StatusOK {
		t.Fatalf("save private block: %d %s", got.Code, got.Body.String())
	}
	public := send(t, router, httptest.NewRequest(http.MethodGet, "/v1/assets/"+id, nil))
	if public.Code != http.StatusOK || strings.Contains(public.Body.String(), "Unpublished") {
		t.Fatalf("public page leaked a private edit: %d", public.Code)
	}
	if got := fetchAssetPage(t, router, "/v1/assets/"+id); got.Name != before.Name || got.IsNSFW != before.IsNSFW {
		t.Fatal("private header changed public metadata")
	}
	working := send(t, router, authorized(httptest.NewRequest(http.MethodGet, "/v1/assets/"+id+"?workingCopy=true", nil), session))
	if working.Code != http.StatusOK || !strings.Contains(working.Body.String(), "Unpublished description") || !strings.Contains(working.Body.String(), "Unpublished name") {
		t.Fatalf("owner working copy: %d %s", working.Code, working.Body.String())
	}
	anonymous := send(t, router, httptest.NewRequest(http.MethodGet, "/v1/assets/"+id+"?workingCopy=true", nil))
	if anonymous.Code != http.StatusNotFound {
		t.Fatalf("anonymous working copy: %d", anonymous.Code)
	}
	if got := contentGeneration(t, pool, id); got != generation {
		t.Fatalf("private edits advanced generation from %d to %d", generation, got)
	}
}

func TestWorkingCopyMediaIsPrivateOnAPublishedAsset(t *testing.T) {
	router, session := newVerifiedTestRouter(t)
	started := startCharacter(t, router, session)
	writeCharacterFloor(t, router, session, started)
	if got := send(t, router, authorized(mediaUploadRequest(t, started.ID, "avatar", httpTestPNG(t, 40, 60)), session)); got.Code != http.StatusCreated {
		t.Fatalf("upload original cover: %d", got.Code)
	}
	if got := publishAsset(t, router, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish: %d", got.Code)
	}
	before := fetchAssetPage(t, router, "/v1/assets/"+started.ID)
	if got := send(t, router, authorized(mediaUploadRequest(t, started.ID, "avatar", httpTestPNG(t, 80, 120)), session)); got.Code != http.StatusCreated {
		t.Fatalf("upload private cover: %d", got.Code)
	}
	public := fetchAssetPage(t, router, "/v1/assets/"+started.ID)
	if len(public.Media) != 1 || public.Media[0].ID != before.Media[0].ID {
		t.Fatal("private cover changed the public media")
	}
	response := send(t, router, authorized(httptest.NewRequest(http.MethodGet, "/v1/assets/"+started.ID+"?workingCopy=true", nil), session))
	var working assetPageResponse
	if err := json.Unmarshal(response.Body.Bytes(), &working); err != nil || response.Code != http.StatusOK {
		t.Fatalf("working copy: %d, %v", response.Code, err)
	}
	if len(working.Media) != 2 || working.Media[0].ID == public.Media[0].ID {
		t.Fatal("working copy did not retain the new cover")
	}
	for _, signed := range []string{working.Media[0].DetailURL, working.Media[0].ThumbURL} {
		unsigned, _, _ := strings.Cut(signed, "?")
		for _, path := range []string{unsigned, signed} {
			if got := send(t, router, httptest.NewRequest(http.MethodGet, path, nil)); got.Code != http.StatusNotFound {
				t.Fatalf("anonymous private media access: %d", got.Code)
			}
		}
		served := send(t, router, authorized(httptest.NewRequest(http.MethodGet, signed, nil), session))
		if served.Code != http.StatusOK || served.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("owner private media access: %d", served.Code)
		}
	}
}

func TestPrivateProtectedTextDoesNotReachLinkedDelivery(t *testing.T) {
	router, session, assets, _ := newVerifiedIngestRouterWithPool(t, testRegistry(t))
	id := publishSealedPreset(t, router, session, "Recorded preset", "Recorded secret")
	owner := fetchStartedAsset(t, router, session, id)
	core := editableBlock(blockNamed(t, owner.Blocks, "preset_core"))
	core.Elements[0].Content = json.RawMessage(strings.ReplaceAll(string(core.Elements[0].Content), "Recorded secret", "Unpublished secret"))
	if got := saveBlock(t, router, session, id, blockNamed(t, owner.Blocks, "preset_core").ID, core); got.Code != http.StatusOK {
		t.Fatalf("save private protected text: %d %s", got.Code, got.Body.String())
	}
	exported, err := assets.OpenExportForLinkedInstance(t.Context(), uuid.MustParse(id), "preset_lumiverse")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(exported.Body), "Unpublished secret") || !strings.Contains(string(exported.Body), "Recorded secret") {
		t.Fatal("linked delivery did not use the recorded protected payload")
	}
}
