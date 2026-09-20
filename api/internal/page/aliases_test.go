package page_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/character"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
)

func TestAWorkPageCarriesTheFieldNamesItHadBeforeTheRename(t *testing.T) {
	t.Parallel()
	router, session, works := harness.NewVerifiedUploadRouter(t, format.NewRegistry())
	workID := apitest.UploadVisibilityTestWork(t, router, session, works, work.VisibilityUnlisted)

	answer := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/assets/"+workID, nil))
	if answer.Code != http.StatusOK {
		t.Fatalf("work page status = %d, want 200: %s", answer.Code, answer.Body.String())
	}
	var page map[string]any
	if err := json.Unmarshal(answer.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode work page: %v", err)
	}
	if page["kind"] != page["type"] || page["kind"] == nil {
		t.Errorf("kind = %v, type = %v", page["kind"], page["type"])
	}
	if page["discovery"] != "unlisted" {
		t.Errorf("discovery = %v, want unlisted", page["discovery"])
	}
}

func TestVisibilityStillArrivesUnderItsOldFieldName(t *testing.T) {
	t.Parallel()
	router, session, works := harness.NewVerifiedUploadRouter(t, format.NewRegistry())
	workID := apitest.UploadVisibilityTestWork(t, router, session, works, work.VisibilityListed)

	changed := apitest.Send(t, router, apitest.AuthorizedJSONRequest(
		t, http.MethodPut, "/v1/assets/"+workID+"/discovery", `{"discovery":"unlisted"}`, session,
	))
	if changed.Code != http.StatusNoContent {
		t.Fatalf("change visibility status = %d, want 204: %s", changed.Code, changed.Body.String())
	}

	page := apitest.FetchWorkPage(t, router, "/v1/works/"+workID)
	if page.Visibility != "unlisted" {
		t.Fatalf("visibility = %q, want unlisted", page.Visibility)
	}
}

func TestBrowseStillFiltersOnTheOldNameForTheType(t *testing.T) {
	t.Parallel()
	router, session, works := harness.NewVerifiedUploadRouter(t, format.NewRegistry())
	metadata := apitest.ExampleMetadata("Quiet Shelf")
	metadata["filename"] = "quiet-shelf.lumitheme"
	metadata["blurb"] = "A theme for a quiet shelf."
	metadata["isNsfw"] = false
	workID := apitest.WorkIDFromUpload(
		t, apitest.UploadAndFinish(t, router, session, works, metadata, []byte("theme")))
	gallery := apitest.Send(t, router, apitest.Authorized(apitest.MediaUploadRequest(
		t, workID, "gallery", apitest.PNG(t, 400, 300),
	), session))
	if gallery.Code != http.StatusCreated {
		t.Fatalf("add gallery status = %d, want 201: %s", gallery.Code, gallery.Body.String())
	}

	answer := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/works?kind=character", nil))
	if answer.Code != http.StatusOK {
		t.Fatalf("browse status = %d, want 200: %s", answer.Code, answer.Body.String())
	}
	var listed struct {
		Items          []map[string]any `json:"items"`
		NSFWPreference string           `json:"visibility"`
	}
	if err := json.Unmarshal(answer.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode browse: %v", err)
	}
	if len(listed.Items) != 1 || listed.Items[0]["kind"] != "character" {
		t.Fatalf("browse by kind = %v, want one character", listed.Items)
	}
	if listed.NSFWPreference != "blurred" {
		t.Fatalf("browse visibility = %q, want the blurred preference under its old name", listed.NSFWPreference)
	}
}

func TestBrowseAndTheWorkPageStillAnswerToTheOldAppNames(t *testing.T) {
	t.Parallel()
	registry := format.NewRegistry()
	for _, module := range character.Modules() {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %s: %v", module.ID(), err)
		}
	}
	router, session, works := harness.NewVerifiedUploadRouter(t, registry)
	metadata := apitest.ExampleMetadata("Ana")
	metadata["filename"] = "ana.json"
	workID := apitest.WorkIDFromUpload(t, apitest.UploadAndFinish(t, router, session, works, metadata, []byte(`{
		"spec":"chara_card_v3","spec_version":"3.0",
		"data":{"name":"Ana","description":"Keeps the archive.","first_mes":"Welcome back."}
	}`)))

	var listed struct {
		Items     []map[string]any `json:"items"`
		Platforms []map[string]any `json:"platforms"`
		Apps      []map[string]any `json:"apps"`
	}
	answer := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/works?platform=sillytavern", nil))
	if err := json.Unmarshal(answer.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode browse: %v", err)
	}
	if len(listed.Items) != 1 || len(listed.Platforms) == 0 || len(listed.Platforms) != len(listed.Apps) {
		t.Fatalf("browse by platform = %s, want Ana and the app control under both names", answer.Body.String())
	}
	if none := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/works?platform=notepad", nil)); !json.Valid(none.Body.Bytes()) ||
		strings.Contains(none.Body.String(), `"name":"Ana"`) {
		t.Fatalf("an unknown platform answered %s, want nothing", none.Body.String())
	}

	var page struct {
		AppTargets []format.AppFormat `json:"appTargets"`
		AppFormats []format.AppFormat `json:"appFormats"`
	}
	opened := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/works/"+workID, nil))
	if err := json.Unmarshal(opened.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode work page: %v", err)
	}
	if len(page.AppFormats) == 0 || len(page.AppTargets) != len(page.AppFormats) {
		t.Fatalf("work page = %s, want appTargets repeating appFormats", opened.Body.String())
	}
}
