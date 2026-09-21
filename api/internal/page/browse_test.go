package page_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/download"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/character"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type browseModule struct{}

func (browseModule) ID() string { return "browse_card" }
func (browseModule) Declaration() format.Declaration {
	return apitest.ReaderDeclaration("browse_card", "character")
}
func (browseModule) Match(file format.Inspection) (format.Match, bool) {
	if len(file.Payloads) == 0 {
		return format.Match{}, false
	}
	return format.CompatibilityMatch(file.Payloads[0]), true
}
func (browseModule) Parse(_ context.Context, file format.Inspection, _ format.Match) (format.Parsed, error) {
	elements := []block.Element{
		{Type: block.TypeProse, Role: block.RoleDescription, Content: block.Prose{Text: "Test description"}},
		{Type: block.TypeTextSet, Role: block.RoleGreetings, Content: block.TextSet{Texts: []block.TextItem{{ID: block.NewItemID(), Text: "Hello"}}}},
	}
	if entries, ok := file.Payloads[0].String("lorebook"); ok {
		elements = append(elements, block.Element{
			Type: block.TypeEntryTable, Role: block.RoleLorebookEntries,
			Content: block.EntryTable{Entries: []block.Entry{{
				ID: block.NewItemID(), Name: entries, Keys: []string{entries},
				Text: entries, Enabled: true,
			}}},
		})
	}
	return format.Parsed{Type: "character", Format: "browse_card", Elements: elements}, nil
}

func TestBrowseReturnsOnlyCardContentAndTheReadersEffectiveCount(t *testing.T) {
	t.Parallel()
	router, session, works := harness.NewVerifiedUploadRouter(t, format.NewRegistry())
	metadata := apitest.ExampleMetadata("Velvet Night")
	metadata["filename"] = "velvet-night.lumitheme"
	metadata["blurb"] = "A quiet midnight theme."
	metadata["tags"] = []string{"Midnight"}
	metadata["isNsfw"] = true
	finished := apitest.UploadAndFinish(t, router, session, works, metadata, []byte("theme"))
	if finished.Code != http.StatusOK {
		t.Fatalf("finish upload status = %d, want 200: %s", finished.Code, finished.Body.String())
	}
	workID := apitest.WorkIDFromUpload(t, finished)
	gallery := apitest.Send(t, router, apitest.Authorized(apitest.MediaUploadRequest(
		t, workID, "gallery", apitest.PNG(t, 400, 300),
	), session))
	if gallery.Code != http.StatusCreated {
		t.Fatalf("add gallery status = %d, want 201: %s", gallery.Code, gallery.Body.String())
	}

	response := apitest.Send(t, router, httptest.NewRequest(
		http.MethodGet, "/v1/works?nsfw=blurred", nil,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("browse status = %d, want 200: %s", response.Code, response.Body.String())
	}

	var body struct {
		Items      []map[string]any `json:"items"`
		Total      int              `json:"total"`
		Suppressed int              `json:"suppressed"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode browse response: %v", err)
	}
	if body.Total != 1 || body.Suppressed != 0 || len(body.Items) != 1 {
		t.Fatalf("browse counts = total %d, suppressed %d, items %d; want 1, 0, 1",
			body.Total, body.Suppressed, len(body.Items))
	}
	wantKeys := map[string]bool{
		"id": true, "name": true, "creator": true, "type": true,
		"isNsfw": true, "cover": true, "kind": true, "apps": true,
	}
	for key := range body.Items[0] {
		if !wantKeys[key] {
			t.Errorf("browse card exposed %q; cards carry no catalog or format internals", key)
		}
	}
	if body.Items[0]["name"] != "Velvet Night" ||
		body.Items[0]["creator"] != "verified.creator" ||
		body.Items[0]["type"] != "character" || body.Items[0]["isNsfw"] != true {
		t.Errorf("browse card = %#v", body.Items[0])
	}
	if cover, present := body.Items[0]["cover"]; !present || cover != nil {
		t.Errorf("coverless card cover = %#v, present %v; want an explicit null", cover, present)
	}
}

func TestBrowseSearchUsesCatalogWordsAndItsTwoQualifiers(t *testing.T) {
	t.Parallel()
	router, session, works := harness.NewVerifiedUploadRouter(t, format.NewRegistry())
	entries := []struct {
		name        string
		blurb       string
		tags        []string
		sourceBytes string
	}{
		{
			name: "Moon Garden", blurb: "A place for patient stories.",
			tags: []string{"Botanical", "Original Character"}, sourceBytes: "buried-prompt-word",
		},
		{
			name: "Quiet Harbor", blurb: "Moonlit conversations by the water.",
			tags: []string{"Slow Burn", "Botanical"}, sourceBytes: "harbor",
		},
		{
			name: "Mood:gentle", blurb: "An unknown qualifier remains ordinary text.",
			tags: []string{"Comfort"}, sourceBytes: "gentle",
		},
	}
	for _, entry := range entries {
		metadata := apitest.ExampleMetadata(entry.name)
		metadata["filename"] = entry.name + ".lumitheme"
		metadata["blurb"] = entry.blurb
		metadata["tags"] = entry.tags
		apitest.UploadAndFinish(t, router, session, works, metadata, []byte(entry.sourceBytes))
	}

	cases := []struct {
		query string
		want  []string
	}{
		{query: "MOON", want: []string{"Quiet Harbor", "Moon Garden"}},
		{query: "verified.creator", want: []string{"Mood:gentle", "Quiet Harbor", "Moon Garden"}},
		{query: "botanical", want: nil},
		{query: "tag:botanical", want: []string{"Quiet Harbor", "Moon Garden"}},
		{query: `tag:"slow burn"`, want: []string{"Quiet Harbor"}},
		{query: `tag:botanical tag:"slow burn"`, want: []string{"Quiet Harbor"}},
		{query: "author:verified.creator patient", want: []string{"Moon Garden"}},
		{query: "author:verified.creator author:verified.creator patient", want: []string{"Moon Garden"}},
		{query: "mood:gentle", want: []string{"Mood:gentle"}},
		{query: "buried-prompt-word", want: nil},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			response := apitest.Send(t, router, httptest.NewRequest(
				http.MethodGet, "/v1/works?q="+url.QueryEscape(tc.query), nil,
			))
			if response.Code != http.StatusOK {
				t.Fatalf("browse status = %d, want 200: %s", response.Code, response.Body.String())
			}
			var body struct {
				Items []struct {
					Name string `json:"name"`
				} `json:"items"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode browse response: %v", err)
			}
			got := make([]string, len(body.Items))
			for i, item := range body.Items {
				got[i] = item.Name
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("search %q returned %v, want %v", tc.query, got, tc.want)
			}
		})
	}
}

type browseOption struct {
	Value    string `json:"value"`
	Label    string `json:"label"`
	Count    int    `json:"count"`
	Selected bool   `json:"selected"`
}

type browseFacetGroup struct {
	Key     string         `json:"key"`
	Label   string         `json:"label"`
	Options []browseOption `json:"options"`
}

type browseReading struct {
	Names  []string
	Apps   []browseOption
	Facets []browseFacetGroup
}

func readBrowse(t *testing.T, router http.Handler, path string) browseReading {
	t.Helper()
	response := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("browse %s status = %d: %s", path, response.Code, response.Body.String())
	}
	var body struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
		Apps   []browseOption     `json:"apps"`
		Facets []browseFacetGroup `json:"facets"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode browse response: %v", err)
	}
	reading := browseReading{Apps: body.Apps, Facets: body.Facets}
	for _, item := range body.Items {
		reading.Names = append(reading.Names, item.Name)
	}
	return reading
}

func facetGroup(t *testing.T, groups []browseFacetGroup, key string) browseFacetGroup {
	t.Helper()
	for _, group := range groups {
		if group.Key == key {
			return group
		}
	}
	t.Fatalf("no %s facet among %+v", key, groups)
	return browseFacetGroup{}
}

func TestFacetsAreTypeScopedAndFilterOnElementContent(t *testing.T) {
	t.Parallel()
	registry := format.NewRegistry()
	if err := registry.Register(browseModule{}); err != nil {
		t.Fatalf("register browse module: %v", err)
	}
	router, session, works := harness.NewVerifiedUploadRouter(t, registry)
	metadata := apitest.ExampleMetadata("Aster")
	metadata["filename"] = "aster.json"
	apitest.UploadAndFinish(t, router, session, works, metadata, []byte(`{"card":true,"lorebook":"Ash"}`))
	metadata = apitest.ExampleMetadata("Storm")
	metadata["filename"] = "storm.json"
	apitest.UploadAndFinish(t, router, session, works, metadata, []byte(`{"card":true}`))

	mixed := readBrowse(t, router, "/v1/works")
	if len(mixed.Facets) != 0 {
		t.Fatalf("the mixed catalog offered %+v, want no facets", mixed.Facets)
	}

	scoped := readBrowse(t, router, "/v1/works?kind=character")
	keys := make([]string, 0, len(scoped.Facets))
	for _, group := range scoped.Facets {
		keys = append(keys, group.Key)
	}
	want := []string{"lorebook", "alternate_greetings", "expressions", "gallery"}
	if !slices.Equal(keys, want) {
		t.Fatalf("character facets = %v, want %v", keys, want)
	}
	lorebook := facetGroup(t, scoped.Facets, "lorebook")
	if lorebook.Options[0].Count != 1 || lorebook.Options[1].Count != 1 {
		t.Errorf("lorebook counts = %+v, want one carrying and one not", lorebook.Options)
	}

	carried := readBrowse(t, router, "/v1/works?kind=character&facet=lorebook%3Dtrue")
	if !slices.Equal(carried.Names, []string{"Aster"}) {
		t.Fatalf("works with a lorebook = %v, want Aster", carried.Names)
	}
	none := readBrowse(t, router, "/v1/works?kind=character&facet=lorebook%3Dfalse")
	if !slices.Equal(none.Names, []string{"Storm"}) {
		t.Fatalf("works with no lorebook = %v, want Storm", none.Names)
	}
}

func TestArrangingThePageChangesNoFilterResult(t *testing.T) {
	t.Parallel()
	r, session, works, _ := harness.NewCharacterUploadRouterWithPool(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	apitest.GivePictures(t, r, session, workID, "gallery", "gallery")
	apitest.PublishCharacter(t, r, session, workID)

	before := readBrowse(t, r, "/v1/works?kind=character&facet=gallery%3Dtrue")
	if !slices.Equal(before.Names, []string{"Ana"}) {
		t.Fatalf("a gallery answered %v, want Ana", before.Names)
	}

	page := apitest.FetchStartedWork(t, r, session, workID)
	gallery := apitest.BlockNamed(t, page.Blocks, "gallery")
	messagesBlock := apitest.BlockNamed(t, page.Blocks, "messages")

	renamed := apitest.EditableBlock(gallery)
	title := "Concept art"
	renamed.Title = &title
	if response := apitest.SaveBlock(t, r, session, workID, gallery.ID, renamed); response.Code != http.StatusOK {
		t.Fatalf("rename the gallery: %d %s", response.Code, response.Body.String())
	}
	reordered := apitest.ArrangeBlocks(t, r, session, workID, []apitest.ArrangedBlock{
		{ID: gallery.ID, Width: gallery.Width},
		{ID: messagesBlock.ID, Width: messagesBlock.Width},
		{ID: apitest.BlockNamed(t, page.Blocks, "character_core").ID, Width: "full"},
	})
	if reordered.Code != http.StatusOK {
		t.Fatalf("reorder the page: %d %s", reordered.Code, reordered.Body.String())
	}

	messages := apitest.EditableBlock(messagesBlock)
	messages.Layout = "stack-3"
	messages.Elements[0].Slot = "top"
	messages.Elements[1].Slot = "middle"
	if response := apitest.SaveBlock(t, r, session, workID, messagesBlock.ID, messages); response.Code != http.StatusOK {
		t.Fatalf("make room in Messages: %d %s", response.Code, response.Body.String())
	}
	move := httptest.NewRequest(
		http.MethodPost,
		"/v1/works/"+workID+"/blocks/"+gallery.ID+"/move-and-remove",
		strings.NewReader(`{"destinationBlockId":"`+messagesBlock.ID+`"}`),
	)
	move.Header.Set("Content-Type", "application/json")
	if moved := apitest.Send(t, r, apitest.Authorized(move, session)); moved.Code != http.StatusOK {
		t.Fatalf("move the gallery into Messages: %d %s", moved.Code, moved.Body.String())
	}

	after := readBrowse(t, r, "/v1/works?kind=character&facet=gallery%3Dtrue")
	if !slices.Equal(after.Names, before.Names) {
		t.Fatalf("arranging the page changed the filter result to %v, want %v", after.Names, before.Names)
	}
}

func TestContentInsideACustomBlockAnswersNoFacet(t *testing.T) {
	t.Parallel()
	router, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, router, session)
	apitest.WriteCharacterFloor(t, router, session, started)
	custom := apitest.AddedBlock(t, apitest.AddBlock(t, router, session, started.ID, "custom_block", "text_set"))
	if response := apitest.PublishWork(t, router, session, started.ID); response.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", response.Code, response.Body.String())
	}

	body := apitest.EditableBlock(custom)
	body.Elements[0].Content = json.RawMessage(
		`{"texts":[{"text":"One"},{"text":"Two"},{"text":"Three"}]}`,
	)
	if response := apitest.SaveBlock(t, router, session, started.ID, custom.ID, body); response.Code != http.StatusOK {
		t.Fatalf("save the custom block status = %d: %s", response.Code, response.Body.String())
	}

	for _, bucket := range []string{"1", "2-4", "5-up"} {
		found := readBrowse(t, router, "/v1/works?kind=character&facet=alternate_greetings%3D"+bucket)
		if len(found.Names) != 0 {
			t.Fatalf("a heading the creator invented answered bucket %q: %v", bucket, found.Names)
		}
	}
}

func TestAnEmptyBlockNeverAnswersAsCarried(t *testing.T) {
	t.Parallel()
	r, session, works, _ := harness.NewCharacterUploadRouterWithPool(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	apitest.AddedBlock(t, apitest.AddBlock(t, r, session, workID, "expressions", "image_set"))
	apitest.PublishCharacter(t, r, session, workID)

	carried := readBrowse(t, r, "/v1/works?kind=character&facet=expressions%3Dtrue")
	if len(carried.Names) != 0 {
		t.Fatalf("an empty expression set answered as carried: %v", carried.Names)
	}
	none := readBrowse(t, r, "/v1/works?kind=character&facet=expressions%3Dfalse")
	if !slices.Equal(none.Names, []string{"Ana"}) {
		t.Fatalf("an empty expression set answered %v, want the none bucket", none.Names)
	}
}

func TestTheAppControlNamesAppsAndMatchesThroughOfferedFormats(t *testing.T) {
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
	apitest.UploadAndFinish(t, router, session, works, metadata, []byte(`{
		"spec":"chara_card_v3","spec_version":"3.0",
		"data":{"name":"Ana","description":"Keeps the archive.","first_mes":"Welcome back."}
	}`))

	all := readBrowse(t, router, "/v1/works")
	labels := make([]string, 0, len(all.Apps))
	for _, option := range all.Apps {
		labels = append(labels, option.Label)
	}
	if !slices.Equal(labels, []string{"SillyTavern", "RisuAI", "Lumiverse"}) {
		t.Fatalf("the app control offered %v, want the apps Illarin names", labels)
	}
	for _, option := range all.Apps {
		if option.Count != 1 {
			t.Errorf("%s count = %d, want the card every named app can open", option.Label, option.Count)
		}
	}

	named := readBrowse(t, router, "/v1/works?app=sillytavern")
	if !slices.Equal(named.Names, []string{"Ana"}) {
		t.Fatalf("SillyTavern returned %v, want Ana", named.Names)
	}
	unknown := readBrowse(t, router, "/v1/works?app=notepad")
	if !slices.Equal(unknown.Names, []string{"Ana"}) {
		t.Fatalf("an app Illarin does not name returned %v, want it ignored", unknown.Names)
	}
}

func facetComputedAt(t *testing.T, pool *pgxpool.Pool, workID string) time.Time {
	t.Helper()
	var computedAt time.Time
	if err := pool.QueryRow(context.Background(), `
		select facet_computed_at from work_summaries where work_id = $1
	`, workID).Scan(&computedAt); err != nil {
		t.Fatalf("read the facet summary: %v", err)
	}
	return computedAt
}

func TestPrivateArrangementKeepsPublishedFacetsAndExports(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewCharacterUploadRouterWithPool(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	apitest.GiveExpressions(t, r, session, workID)
	apitest.PublishCharacter(t, r, session, workID)

	shown := readBrowse(t, r, "/v1/works?kind=character&facet=expressions%3Dtrue")
	if !slices.Equal(shown.Names, []string{"Ana"}) {
		t.Fatalf("a shown expression set answered %v, want Ana", shown.Names)
	}

	exportedAt := apitest.SummaryComputedAt(t, pool, workID)
	measuredAt := facetComputedAt(t, pool, workID)
	number := apitest.VersionNumber(t, pool, workID)

	page := apitest.FetchStartedWork(t, r, session, workID)
	arrangement := make([]apitest.ArrangedBlock, 0, len(page.Blocks))
	for _, holder := range page.Blocks {
		arrangement = append(arrangement, apitest.ArrangedBlock{
			ID: holder.ID, Hidden: holder.Definition == "expressions", Width: holder.Width,
		})
	}
	if hidden := apitest.ArrangeBlocks(t, r, session, workID, arrangement); hidden.Code != http.StatusOK {
		t.Fatalf("hide the expressions: %d %s", hidden.Code, hidden.Body.String())
	}

	if after := apitest.SummaryComputedAt(t, pool, workID); !after.Equal(exportedAt) {
		t.Error("hiding a block moved the export half of the summary")
	}
	if after := facetComputedAt(t, pool, workID); !after.After(measuredAt) {
		t.Error("hiding a block left the facet half of the summary alone")
	}
	if after := apitest.VersionNumber(t, pool, workID); after != number {
		t.Errorf("version number = %d, want %d after a hide", after, number)
	}

	carried := readBrowse(t, r, "/v1/works?kind=character&facet=expressions%3Dtrue")
	if !slices.Equal(carried.Names, []string{"Ana"}) {
		t.Fatalf("private arrangement changed the published facet: %v", carried.Names)
	}
	none := readBrowse(t, r, "/v1/works?kind=character&facet=expressions%3Dfalse")
	if len(none.Names) != 0 {
		t.Fatalf("a hidden expression set answered %v, want the none bucket", none.Names)
	}

	export, err := download.NewService(works.Pool(), works).OpenExport(
		context.Background(), uuid.MustParse(workID), nil, "chara_card_v3", nil,
	)
	if err != nil {
		t.Fatalf("export a card with a hidden block: %v", err)
	}
	if !apitest.ContainsBytes(export.Body, []byte("happy")) {
		t.Fatal("a hidden block's content did not travel in the download")
	}
}

func TestSignedInBrowseUsesTheReadersSavedContentPreference(t *testing.T) {
	t.Parallel()
	router, session, works := harness.NewVerifiedUploadRouter(t, format.NewRegistry())
	metadata := apitest.ExampleMetadata("Veiled Garden")
	metadata["_keepDraft"] = true
	metadata["filename"] = "veiled-garden.lumitheme"
	metadata["isNsfw"] = true
	created := apitest.UploadAndFinish(t, router, session, works, metadata, []byte("garden"))
	workID := apitest.WorkIDFromUpload(t, created)
	added := apitest.Send(t, router, apitest.Authorized(apitest.MediaUploadRequest(
		t, workID, "avatar", apitest.PNG(t, 80, 120),
	), session))
	if added.Code != http.StatusCreated {
		t.Fatalf("add cover status = %d, want 201: %s", added.Code, added.Body.String())
	}

	if got := apitest.PublishWork(t, router, session, workID); got.Code != http.StatusOK {
		t.Fatalf("publish media: %d %s", got.Code, got.Body.String())
	}

	saved := apitest.Send(t, router, apitest.AuthorizedJSONRequest(
		t, http.MethodPut, "/v1/account/nsfw-preference", `{"preference":"hidden"}`, session,
	))
	if saved.Code != http.StatusNoContent {
		t.Fatalf("save visibility status = %d, want 204: %s", saved.Code, saved.Body.String())
	}

	type response struct {
		Items []struct {
			Cover *struct {
				URL string `json:"url"`
			} `json:"cover"`
		} `json:"items"`
		Total          int     `json:"total"`
		Suppressed     int     `json:"suppressed"`
		NSFWPreference string  `json:"nsfwPreference"`
		EmptyState     *string `json:"emptyState"`
	}
	read := func(request *http.Request) response {
		t.Helper()
		answer := apitest.Send(t, router, request)
		if answer.Code != http.StatusOK {
			t.Fatalf("browse status = %d, want 200: %s", answer.Code, answer.Body.String())
		}
		var body response
		if err := json.Unmarshal(answer.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode browse response: %v", err)
		}
		return body
	}

	signedIn := read(apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/works", nil), session,
	))
	if signedIn.Total != 0 || signedIn.Suppressed != 1 || signedIn.NSFWPreference != "hidden" ||
		signedIn.EmptyState == nil || *signedIn.EmptyState != "suppressed" {
		t.Fatalf("signed-in browse = %#v, want the saved hidden preference", signedIn)
	}
	signedOut := read(httptest.NewRequest(http.MethodGet, "/v1/works", nil))
	if signedOut.Total != 1 || signedOut.NSFWPreference != "blurred" || len(signedOut.Items) != 1 ||
		signedOut.Items[0].Cover == nil || !strings.Contains(signedOut.Items[0].Cover.URL, "/grid_blurred/") {
		t.Fatalf("signed-out browse = %#v, want one blurred card", signedOut)
	}
}
