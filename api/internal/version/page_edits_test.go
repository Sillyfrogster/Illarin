package version_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format/modules"
	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
)

func TestThePublishedViewsKnowWhichFieldsEachFileCarries(t *testing.T) {
	t.Parallel()
	registry, err := modules.Registry()
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.Connect(t)
	for _, workType := range block.Types() {
		var stored []string
		if err := pool.QueryRow(context.Background(), `select work_file_fields($1)`, workType).Scan(&stored); err != nil {
			t.Fatal(err)
		}
		if written := registry.FileFields(workType); !slices.Equal(stored, written) {
			t.Errorf("work_file_fields(%q) = %v, but its formats write %v", workType, stored, written)
		}
	}
}

func TestAPageEditOnAPublishedPresetIsLiveWithoutAVersion(t *testing.T) {
	t.Parallel()
	_, router, session, _, pool := harness.NewVerifiedRoutersWithPool(t, 1<<20, api.DefaultDeadlines())
	started := apitest.StartPreset(t, router, session, "lumiverse")
	core := apitest.EditableBlock(apitest.BlockNamed(t, started.Blocks, "preset_core"))
	core.Elements[0].Content = json.RawMessage(`{"groups":[],"fragments":[{"name":"House rule","role":"system","text":"Stay in the scene.","enabled":true}]}`)
	if got := apitest.SaveBlock(t, router, session, started.ID, started.Blocks[0].ID, core); got.Code != http.StatusOK {
		t.Fatalf("save the prompt: %d %s", got.Code, got.Body.String())
	}
	if got := apitest.SaveDetails(t, router, session, started.ID, `{"name":"Night shift","blurb":"Slow noir.","isNsfw":false}`); got.Code != http.StatusNoContent {
		t.Fatalf("save the details: %d %s", got.Code, got.Body.String())
	}
	if got := apitest.PublishWork(t, router, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish: %d %s", got.Code, got.Body.String())
	}
	number := apitest.VersionNumber(t, pool, started.ID)
	before := downloadPreset(t, router, started.ID)

	if got := apitest.SaveDetails(t, router, session, started.ID, `{"name":"Night shift","blurb":"Slow noir.","tags":["noir"],"isNsfw":false}`); got.Code != http.StatusNoContent {
		t.Fatalf("save the tags: %d %s", got.Code, got.Body.String())
	}
	if got := apitest.Send(t, router, apitest.Authorized(apitest.MediaUploadRequest(t, started.ID, "avatar", apitest.PNG(t, 30, 40)), session)); got.Code != http.StatusCreated {
		t.Fatalf("upload the cover: %d %s", got.Code, got.Body.String())
	}

	public := apitest.FetchWorkPage(t, router, "/v1/works/"+started.ID)
	if len(public.Tags) != 1 || public.Tags[0].Label != "noir" {
		t.Fatalf("readers do not see the new tags: %+v", public.Tags)
	}
	if len(public.Media) != 1 || !public.Media[0].IsCover {
		t.Fatalf("readers do not see the new cover: %+v", public.Media)
	}
	if got := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, public.Media[0].DetailURL, nil)); got.Code == http.StatusNotFound {
		t.Fatal("readers cannot load the new cover")
	}
	if got := apitest.VersionNumber(t, pool, started.ID); got != number {
		t.Fatalf("a page edit moved the version from %d to %d", number, got)
	}
	if !bytes.Equal(downloadPreset(t, router, started.ID), before) {
		t.Fatal("a page edit changed the downloaded bytes")
	}
	if ownerSeesDraftedChanges(t, router, session, started.ID) {
		t.Fatal("a page edit left drafted changes to publish")
	}

	if got := apitest.SaveDetails(t, router, session, started.ID, `{"name":"Day shift","blurb":"Slow noir.","tags":["noir"],"isNsfw":false}`); got.Code != http.StatusNoContent {
		t.Fatalf("save the name: %d %s", got.Code, got.Body.String())
	}
	if got := apitest.FetchWorkPage(t, router, "/v1/works/"+started.ID); got.Name != "Night shift" {
		t.Fatalf("the preset's name is in its file but went live as %q", got.Name)
	}
	if !ownerSeesDraftedChanges(t, router, session, started.ID) {
		t.Fatal("a name the file carries did not wait for Publish")
	}
}

func downloadPreset(t *testing.T, router http.Handler, workID string) []byte {
	t.Helper()
	response := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/download/"+workID+"/preset_lumiverse", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("download: %d %s", response.Code, response.Body.String())
	}
	return response.Body.Bytes()
}

func ownerSeesDraftedChanges(t *testing.T, router http.Handler, session *http.Cookie, workID string) bool {
	t.Helper()
	response := apitest.Send(t, router, apitest.Authorized(httptest.NewRequest(http.MethodGet, "/v1/works/"+workID+"?draftedChanges=true", nil), session))
	var page struct {
		UnpublishedChanges bool `json:"unpublishedChanges"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &page) != nil {
		t.Fatalf("read the drafted changes: %d %s", response.Code, response.Body.String())
	}
	return page.UnpublishedChanges
}
