package version_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
)

func TestRestoringARecordedVersionStagesItWithoutReplacingNewerWork(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)
	apitest.WriteCharacterFloor(t, r, session, started)
	if got := apitest.PublishWork(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", got.Code, got.Body.String())
	}

	page := apitest.FetchStartedWork(t, r, session, started.ID)
	coreBlock := apitest.BlockNamed(t, page.Blocks, "character_core")
	core := apitest.EditableBlock(coreBlock)
	core.Elements[0].Content = json.RawMessage(`{"text":"The second public description."}`)
	if got := apitest.SaveBlock(t, r, session, started.ID, coreBlock.ID, core); got.Code != http.StatusOK {
		t.Fatalf("save second description = %d: %s", got.Code, got.Body.String())
	}
	if got := apitest.PublishWorkUpdate(t, r, session, started.ID, `{"summary":"Second version"}`); got.Code != http.StatusOK {
		t.Fatalf("publish second version = %d: %s", got.Code, got.Body.String())
	}

	stale := apitest.Authorized(httptest.NewRequest(http.MethodPost,
		"/v1/works/"+started.ID+"/updates/1/restore", nil), session)
	apitest.WithReviewedVersion(t, r, stale)
	identity := apitest.AuthorizedJSONRequest(t, http.MethodPut, "/v1/works/"+started.ID+"/details",
		`{"name":"Unsaved newer work","blurb":"Kept in the working copy.","isNsfw":false}`, session)
	if got := apitest.Send(t, r, identity); got.Code != http.StatusNoContent {
		t.Fatalf("save newer work = %d: %s", got.Code, got.Body.String())
	}
	if got := apitest.Send(t, r, stale); got.Code != http.StatusConflict {
		t.Fatalf("stale restore = %d, want 409: %s", got.Code, got.Body.String())
	}
	if got := apitest.FetchStartedWork(t, r, session, started.ID); got.Name != "Unsaved newer work" {
		t.Fatalf("stale restore replaced %q", got.Name)
	}

	restore := apitest.Authorized(httptest.NewRequest(http.MethodPost,
		"/v1/works/"+started.ID+"/updates/1/restore", nil), session)
	if got := apitest.Send(t, r, restore); got.Code != http.StatusNoContent {
		t.Fatalf("restore = %d, want 204: %s", got.Code, got.Body.String())
	}
	working := apitest.FetchStartedWork(t, r, session, started.ID)
	if working.Name != "Ilse of the west shelf" || blockText(t, working.Blocks, "character_core") != "She keeps the books that forget themselves." {
		t.Fatalf("restored working copy = %q / %q", working.Name, blockText(t, working.Blocks, "character_core"))
	}
	public := apitest.FetchWork(t, r, nil, started.ID)
	if got := blockText(t, public.Blocks, "character_core"); got != "The second public description." {
		t.Fatalf("restore changed public description to %q", got)
	}
	if got := apitest.PublishWorkUpdate(t, r, session, started.ID, `{"summary":""}`); got.Code != http.StatusBadRequest {
		t.Fatalf("restoration published without fresh notes = %d: %s", got.Code, got.Body.String())
	}
	published := apitest.PublishWorkUpdate(t, r, session, started.ID, `{"summary":"Restored the original description"}`)
	if published.Code != http.StatusOK || !strings.Contains(published.Body.String(), `"number":3`) {
		t.Fatalf("restoration did not become a new update: %d %s", published.Code, published.Body.String())
	}
}

func TestRestoringARecordedVersionRestoresItsPictures(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)
	apitest.WriteCharacterFloor(t, r, session, started)
	first := apitest.Send(t, r, apitest.Authorized(apitest.MediaUploadRequest(
		t, started.ID, "gallery", apitest.PNG(t, 16, 16),
	), session))
	if first.Code != http.StatusCreated {
		t.Fatalf("add first picture = %d: %s", first.Code, first.Body.String())
	}
	var firstImage struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &firstImage); err != nil {
		t.Fatal(err)
	}
	if got := apitest.PublishWork(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", got.Code, got.Body.String())
	}

	second := apitest.Send(t, r, apitest.Authorized(apitest.MediaUploadRequest(
		t, started.ID, "gallery", apitest.PNG(t, 24, 24),
	), session))
	if second.Code != http.StatusCreated {
		t.Fatalf("add second picture = %d: %s", second.Code, second.Body.String())
	}
	var secondImage struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &secondImage); err != nil {
		t.Fatal(err)
	}
	if got := apitest.SaveDetails(t, r, session, started.ID,
		`{"name":"Ilse of the west shelf","blurb":"Now with another picture.","isNsfw":false}`); got.Code != http.StatusNoContent {
		t.Fatalf("save second version details = %d: %s", got.Code, got.Body.String())
	}
	if got := apitest.PublishWorkUpdate(t, r, session, started.ID, `{"summary":"Added another picture"}`); got.Code != http.StatusOK {
		t.Fatalf("publish second version = %d: %s", got.Code, got.Body.String())
	}
	restore := apitest.Authorized(httptest.NewRequest(http.MethodPost,
		"/v1/works/"+started.ID+"/updates/1/restore", nil), session)
	if got := apitest.Send(t, r, restore); got.Code != http.StatusNoContent {
		t.Fatalf("restore = %d: %s", got.Code, got.Body.String())
	}
	working := apitest.FetchStartedWork(t, r, session, started.ID)
	if len(working.Media) != 1 || working.Media[0].ID != firstImage.ID || working.Media[0].ID == secondImage.ID {
		t.Fatalf("restored pictures = %+v", working.Media)
	}
}

func TestRestorationKeepsCurrentPromptProtectionAndAllowedApps(t *testing.T) {
	t.Parallel()
	r, session, _, _ := harness.NewVerifiedIngestRouterWithPool(t, apitest.LumiverseRegistry(t))
	publicID, sealedID := uuid.New(), uuid.New()
	started := apitest.PublishTwoPromptPreset(t, r, session, publicID, sealedID,
		"First public prompt.", "First protected prompt.")
	working := apitest.FetchStartedWork(t, r, session, started.ID)
	coreBlock := apitest.BlockNamed(t, working.Blocks, "preset_core")
	core := apitest.EditableBlock(coreBlock)
	core.Elements[0].Content = apitest.SealedPresetPrompts(uuid.New(), uuid.New(),
		"Second public prompt.", "Second protected prompt.")
	core.AllowedApps = &[]string{"lumiverse"}
	if got := apitest.SaveBlock(t, r, session, started.ID, coreBlock.ID, core); got.Code != http.StatusOK {
		t.Fatalf("save second prompts = %d: %s", got.Code, got.Body.String())
	}
	if got := apitest.PublishWorkUpdate(t, r, session, started.ID, `{"summary":"Changed both prompts"}`); got.Code != http.StatusOK {
		t.Fatalf("publish second prompts = %d: %s", got.Code, got.Body.String())
	}
	restore := apitest.Authorized(httptest.NewRequest(http.MethodPost,
		"/v1/works/"+started.ID+"/updates/1/restore", nil), session)
	if got := apitest.Send(t, r, restore); got.Code != http.StatusNoContent {
		t.Fatalf("restore protected version = %d: %s", got.Code, got.Body.String())
	}
	restored := apitest.FetchStartedWork(t, r, session, started.ID)
	encoded, err := json.Marshal(restored)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, want := range []string{"First public prompt.", "First protected prompt.", `"allowedApps":["lumiverse"]`, `"linkedInstallOnly":true`} {
		if !strings.Contains(text, want) {
			t.Fatalf("restored working copy omitted %q: %s", want, text)
		}
	}
	if got := apitest.PublishWorkUpdate(t, r, session, started.ID, `{"summary":"Restored earlier prompts"}`); got.Code != http.StatusOK {
		t.Fatalf("publish restored prompts = %d: %s", got.Code, got.Body.String())
	}
	public := apitest.FetchWork(t, r, nil, started.ID)
	publicJSON, _ := json.Marshal(public)
	if strings.Contains(string(publicJSON), "First public prompt.") || strings.Contains(string(publicJSON), "First protected prompt.") || strings.Contains(string(publicJSON), "Second protected prompt.") {
		t.Fatal("restoration exposed a protected prompt")
	}
}

func TestRestoredOldContentMustPassCurrentPublicationValidation(t *testing.T) {
	t.Parallel()
	_, r, session, _, pool := harness.NewVerifiedRoutersWithPool(t, 1<<20, api.DefaultDeadlines())
	started := apitest.StartCharacter(t, r, session)
	apitest.WriteCharacterFloor(t, r, session, started)
	if got := apitest.PublishWork(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", got.Code, got.Body.String())
	}

	var payload, protected []byte
	var generation int
	if err := pool.QueryRow(t.Context(), `
		select payload, protected_payloads, content_generation
		  from work_snapshots where work_id = $1 and number = 1
	`, started.ID).Scan(&payload, &protected, &generation); err != nil {
		t.Fatal(err)
	}
	var recorded map[string]any
	if err := json.Unmarshal(payload, &recorded); err != nil {
		t.Fatal(err)
	}
	for _, value := range recorded["blocks"].([]any) {
		holder := value.(map[string]any)
		if holder["definition"] != "messages" {
			continue
		}
		elements := holder["elements"].([]any)
		content := elements[0].(map[string]any)["content"].(map[string]any)
		content["texts"] = []any{}
	}
	invalid, err := json.Marshal(recorded)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `
		insert into work_snapshots
			(work_id, number, content_generation, payload, protected_payloads, summary)
		values ($1, 2, $2, $3, $4, 'Recorded under an older rule')
	`, started.ID, generation, invalid, protected); err != nil {
		t.Fatal(err)
	}

	restore := apitest.Authorized(httptest.NewRequest(http.MethodPost,
		"/v1/works/"+started.ID+"/updates/2/restore", nil), session)
	if got := apitest.Send(t, r, restore); got.Code != http.StatusNoContent {
		t.Fatalf("restore older content = %d: %s", got.Code, got.Body.String())
	}
	publish := apitest.PublishWorkUpdate(t, r, session, started.ID, `{"summary":"Restore an older version"}`)
	if publish.Code != http.StatusConflict || !strings.Contains(publish.Body.String(), `"code":"not_ready"`) {
		t.Fatalf("invalid restored content published = %d: %s", publish.Code, publish.Body.String())
	}
}

func TestCorrectingNotesMarksTheEditWithoutPublishingContent(t *testing.T) {
	t.Parallel()
	_, r, session, _, pool := harness.NewVerifiedRoutersWithPool(t, 1<<20, api.DefaultDeadlines())
	started := apitest.StartCharacter(t, r, session)
	apitest.WriteCharacterFloor(t, r, session, started)
	if got := apitest.PublishWork(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", got.Code, got.Body.String())
	}
	before := apitest.ContentGeneration(t, pool, started.ID)

	req := apitest.AuthorizedJSONRequest(t, http.MethodPatch,
		"/v1/works/"+started.ID+"/updates/1/notes",
		`{"summary":"Corrected summary","notes":"Corrected context."}`, session)
	if got := apitest.Send(t, r, req); got.Code != http.StatusNoContent {
		t.Fatalf("correct notes = %d, want 204: %s", got.Code, got.Body.String())
	}
	if after := apitest.ContentGeneration(t, pool, started.ID); after != before {
		t.Fatalf("note correction moved generation from %d to %d", before, after)
	}

	response := readUpdateHistory(t, r, started.ID, nil)
	var history struct {
		Items []struct {
			Summary       string  `json:"summary"`
			Notes         string  `json:"notes"`
			NotesEditedAt *string `json:"notesEditedAt"`
		} `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &history); err != nil {
		t.Fatal(err)
	}
	if len(history.Items) != 1 || history.Items[0].Summary != "Corrected summary" ||
		history.Items[0].Notes != "Corrected context." || history.Items[0].NotesEditedAt == nil {
		t.Fatalf("corrected history = %+v", history.Items)
	}
}

func TestAnOlderVersionCanBeWithdrawnWithoutExposingItsSnapshot(t *testing.T) {
	t.Parallel()
	r, session, _, _ := harness.NewVerifiedIngestRouterWithPool(t, format.NewRegistry())
	started := apitest.StartCharacter(t, r, session)
	apitest.WriteCharacterFloor(t, r, session, started)
	if got := apitest.PublishWork(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", got.Code, got.Body.String())
	}
	withdrawnMedia := apitest.UploadedImageID(t, r, session, started.ID, "gallery", apitest.PNG(t, 600, 800))
	for number, description := range []string{"Second secret history", "Current public history"} {
		page := apitest.FetchStartedWork(t, r, session, started.ID)
		coreBlock := apitest.BlockNamed(t, page.Blocks, "character_core")
		core := apitest.EditableBlock(coreBlock)
		core.Elements[0].Content = json.RawMessage(`{"text":"` + description + `"}`)
		if got := apitest.SaveBlock(t, r, session, started.ID, coreBlock.ID, core); got.Code != http.StatusOK {
			t.Fatalf("save version %d = %d: %s", number+2, got.Code, got.Body.String())
		}
		notes := "Current notes"
		if number == 0 {
			notes = "Private after withdrawal"
		}
		if got := apitest.PublishWorkUpdate(t, r, session, started.ID,
			`{"summary":"Version `+description+`","notes":"`+notes+`"}`); got.Code != http.StatusOK {
			t.Fatalf("publish version %d = %d: %s", number+2, got.Code, got.Body.String())
		}
		if number == 0 {
			restore := apitest.Authorized(httptest.NewRequest(http.MethodPost,
				"/v1/works/"+started.ID+"/updates/1/restore", nil), session)
			if got := apitest.Send(t, r, restore); got.Code != http.StatusNoContent {
				t.Fatalf("restore without withdrawn picture = %d: %s", got.Code, got.Body.String())
			}
		}
	}

	withdraw := apitest.AuthorizedJSONRequest(t, http.MethodPost,
		"/v1/works/"+started.ID+"/updates/2/withdraw",
		`{"explanation":"This version gave incorrect guidance."}`, session)
	if got := apitest.Send(t, r, withdraw); got.Code != http.StatusNoContent {
		t.Fatalf("withdraw = %d, want 204: %s", got.Code, got.Body.String())
	}

	public := readUpdateHistory(t, r, started.ID, nil)
	if strings.Contains(public.Body.String(), "Second secret history") || strings.Contains(public.Body.String(), "Private after withdrawal") {
		t.Fatalf("public history exposed withdrawn snapshot: %s", public.Body.String())
	}
	if !strings.Contains(public.Body.String(), "This version gave incorrect guidance.") {
		t.Fatalf("public history omitted withdrawal explanation: %s", public.Body.String())
	}
	var publicHistory struct {
		Items []struct {
			ID           uuid.UUID `json:"id"`
			Initial      bool      `json:"initial"`
			VersionLabel string    `json:"versionLabel"`
			WithdrawnAt  *string   `json:"withdrawnAt"`
		} `json:"items"`
	}
	if err := json.Unmarshal(public.Body.Bytes(), &publicHistory); err != nil {
		t.Fatal(err)
	}
	withdrawn := publicHistory.Items[1]
	if withdrawn.ID != uuid.Nil || withdrawn.Initial || withdrawn.VersionLabel != "" || withdrawn.WithdrawnAt != nil {
		t.Fatalf("public withdrawal exposed private version metadata: %+v", withdrawn)
	}
	owner := readUpdateHistory(t, r, started.ID, session)
	if !strings.Contains(owner.Body.String(), "Second secret history") {
		t.Fatalf("owner lost withdrawn snapshot: %s", owner.Body.String())
	}

	download := apitest.Send(t, r, httptest.NewRequest(http.MethodGet,
		"/v1/works/"+started.ID+"/updates/2/downloads", nil))
	if download.Code != http.StatusNotFound {
		t.Fatalf("withdrawn download = %d, want 404: %s", download.Code, download.Body.String())
	}
	directMedia := apitest.Send(t, r, httptest.NewRequest(http.MethodGet,
		"/media/"+withdrawnMedia+"/thumb/2", nil))
	if directMedia.Code != http.StatusNotFound {
		t.Fatalf("withdrawn media = %d, want 404", directMedia.Code)
	}
	comparison := apitest.Send(t, r, httptest.NewRequest(http.MethodGet,
		"/v1/works/"+started.ID+"/updates/comparison?from=1&to=2", nil))
	if comparison.Code != http.StatusOK || !strings.Contains(comparison.Body.String(), "This version was withdrawn.") || strings.Contains(comparison.Body.String(), "Second secret history") {
		t.Fatalf("withdrawn comparison was not blocked: %d %s", comparison.Code, comparison.Body.String())
	}

	current := apitest.AuthorizedJSONRequest(t, http.MethodPost,
		"/v1/works/"+started.ID+"/updates/3/withdraw", `{"explanation":"Not alone."}`, session)
	if got := apitest.Send(t, r, current); got.Code != http.StatusConflict {
		t.Fatalf("current withdrawal = %d, want 409: %s", got.Code, got.Body.String())
	}
}

func blockText(t *testing.T, blocks []apitest.StartedBlock, definition string) string {
	t.Helper()
	holder := apitest.BlockNamed(t, blocks, definition)
	var prose struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(holder.Elements[0].Content, &prose); err != nil {
		t.Fatal(err)
	}
	return prose.Text
}
