package upload_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

const pastedMarkdown = "# Wren\n\nA lighthouse keeper who talks to gulls.\n\n![Wren](https://example.com/wren.png)\n\n" +
	"## Backstory\n\nWren grew up on the rocks.\n\n### Early years\n\nShe learned to swim first.\n\n" +
	"## Relationships\n\n- The gulls\n- Her brother\n\n![Brother](https://example.com/brother.png)\n"

func addMarkdown(t *testing.T, r http.Handler, session *http.Cookie, workID, markdown string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"markdown": markdown})
	if err != nil {
		t.Fatalf("encode the markdown: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/works/"+workID+"/shelf", strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	return apitest.Send(t, r, apitest.Authorized(request, session))
}

func placePiece(t *testing.T, r http.Handler, session *http.Cookie, workID, pieceID, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/v1/works/"+workID+"/shelf/pieces/"+pieceID+"/place", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return apitest.Send(t, r, apitest.Authorized(request, session))
}

func undoPlacement(t *testing.T, r http.Handler, session *http.Cookie, workID, pieceID string) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodPost, "/v1/works/"+workID+"/shelf/pieces/"+pieceID+"/undo", nil), session))
}

func sectionNamed(t *testing.T, pieces []shelfPiece, section string) shelfPiece {
	t.Helper()
	for _, piece := range pieces {
		if piece.Kind == "section" && piece.Section == section {
			return piece
		}
	}
	t.Fatalf("no section %q among %+v", section, pieces)
	return shelfPiece{}
}

func draftedVersion(t *testing.T, r http.Handler, session *http.Cookie, workID string) int64 {
	t.Helper()
	response := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/works/"+workID+"?draftedChanges=true", nil), session))
	var page struct {
		DraftedChangesVersion int64 `json:"draftedChangesVersion"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode the drafted page: %v", err)
	}
	return page.DraftedChangesVersion
}

func TestPastedMarkdownWaitsOnTheShelfAndLeavesThePageAlone(t *testing.T) {
	t.Parallel()
	r, session, _, _ := harness.NewExtensionRouter(t)
	workID := apitest.StartCharacter(t, r, session).ID
	before := readSeededPage(t, r, session, workID+"?draftedChanges=true")
	version := draftedVersion(t, r, session, workID)

	if added := addMarkdown(t, r, session, workID, pastedMarkdown); added.Code != http.StatusCreated {
		t.Fatalf("paste = %d: %s", added.Code, added.Body.String())
	}
	shelf := readShelf(t, r, session, workID)
	if len(shelf) != 1 || shelf[0].Source != "pasted" || shelf[0].Title != "Wren" {
		t.Fatalf("shelf = %+v, want one pasted import titled after the Markdown", shelf)
	}
	var got []string
	for _, piece := range shelf[0].Pieces {
		got = append(got, fmt.Sprintf("%s %q %q %s", piece.Kind, piece.Section, piece.Text, piece.Address))
	}
	want := []string{
		`section "" "A lighthouse keeper who talks to gulls." `,
		`picture "" "" https://example.com/wren.png`,
		`section "Backstory" "Wren grew up on the rocks." `,
		`section "Early years" "She learned to swim first." `,
		`section "Relationships" "- The gulls\n- Her brother" `,
		`picture "Relationships" "" https://example.com/brother.png`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("pieces =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	after := readSeededPage(t, r, session, workID+"?draftedChanges=true")
	if strings.Join(arrangement(after), "|") != strings.Join(arrangement(before), "|") || draftedVersion(t, r, session, workID) != version {
		t.Errorf("page after the paste = %q, want it unchanged: %q", arrangement(after), arrangement(before))
	}

	if refused := addMarkdown(t, r, session, workID, strings.Repeat("a", 1<<20+1)); refused.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("paste over 1 MB = %d: %s", refused.Code, refused.Body.String())
	}
	if refused := addMarkdown(t, r, session, workID, "  \n\n"); refused.Code != http.StatusBadRequest {
		t.Errorf("paste nothing = %d: %s", refused.Code, refused.Body.String())
	}
	if added := addMarkdown(t, r, session, workID, "Just a note."); added.Code != http.StatusCreated {
		t.Fatalf("second paste = %d: %s", added.Code, added.Body.String())
	}
	shelf = readShelf(t, r, session, workID)
	letGo := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/works/"+workID+"/shelf/imports/"+shelf[1].ID, nil), session))
	if letGo.Code != http.StatusNoContent {
		t.Fatalf("let go of the first import = %d: %s", letGo.Code, letGo.Body.String())
	}
	if left := readShelf(t, r, session, workID); len(left) != 1 || left[0].Pieces[0].Text != "Just a note." {
		t.Errorf("shelf after letting go = %+v, want only the second paste", left)
	}
}

func TestASectionPlacedOnAPublishedWorkWaitsInDraftedChangesUntilUndone(t *testing.T) {
	t.Parallel()
	r, session, _, _ := harness.NewExtensionRouter(t)
	workID := apitest.PublishedCharacter(t, r, session)
	if added := addMarkdown(t, r, session, workID, pastedMarkdown); added.Code != http.StatusCreated {
		t.Fatalf("paste = %d: %s", added.Code, added.Body.String())
	}
	pieces := shelfPieces(t, r, session, workID)
	backstory, relationships := sectionNamed(t, pieces, "Backstory"), sectionNamed(t, pieces, "Relationships")
	published := readSeededPage(t, r, nil, workID)

	if placed := placePiece(t, r, session, workID, backstory.ID, `{"position":1}`); placed.Code != http.StatusOK {
		t.Fatalf("place the backstory = %d: %s", placed.Code, placed.Body.String())
	}
	description := readSeededPage(t, r, session, workID+"?draftedChanges=true").Blocks[0]
	if description.Definition != "character_core" {
		t.Fatalf("first block = %s, want the character's description", description.Definition)
	}
	body := fmt.Sprintf(`{"elementId":%q}`, description.Elements[0].ID)
	if placed := placePiece(t, r, session, workID, relationships.ID, body); placed.Code != http.StatusOK {
		t.Fatalf("add the relationships to the description = %d: %s", placed.Code, placed.Body.String())
	}

	drafted := readSeededPage(t, r, session, workID+"?draftedChanges=true")
	if drafted.Blocks[1].Title != "Backstory" || proseText(t, drafted.Blocks[1]) != backstory.Text {
		t.Fatalf("drafted blocks = %q, want the backstory as the second block", arrangement(drafted))
	}
	if text := proseText(t, drafted.Blocks[0]); text != "She keeps the books that forget themselves.\n\n- The gulls\n- Her brother" {
		t.Errorf("drafted description = %q, want the section added at the end", text)
	}
	public := readSeededPage(t, r, nil, workID)
	if strings.Join(arrangement(public), "|") != strings.Join(arrangement(published), "|") ||
		proseText(t, public.Blocks[0]) != proseText(t, published.Blocks[0]) {
		t.Errorf("published page = %q, want it unchanged until the next version", arrangement(public))
	}
	if left := shelfPieces(t, r, session, workID); len(left) != 4 {
		t.Errorf("shelf after placing = %+v, want the two sections gone", left)
	}

	if undone := undoPlacement(t, r, session, workID, backstory.ID); undone.Code != http.StatusOK {
		t.Fatalf("undo the backstory = %d: %s", undone.Code, undone.Body.String())
	}
	if titles := arrangement(readSeededPage(t, r, session, workID+"?draftedChanges=true")); strings.Contains(strings.Join(titles, "|"), "Backstory") {
		t.Errorf("drafted blocks after undo = %q, want the backstory block gone", titles)
	}
	sectionNamed(t, shelfPieces(t, r, session, workID), "Backstory")

	if placed := placePiece(t, r, session, workID, backstory.ID, `{"position":1}`); placed.Code != http.StatusOK {
		t.Fatalf("place the backstory again = %d: %s", placed.Code, placed.Body.String())
	}
	started := apitest.FetchStartedWork(t, r, session, workID)
	placedBlock := blockTitled(t, started.Blocks, "Backstory")
	edited := apitest.EditableBlock(placedBlock)
	edited.Title = &placedBlock.Title
	edited.Elements[0].Content = json.RawMessage(`{"text":"Rewritten."}`)
	if saved := apitest.SaveBlock(t, r, session, workID, placedBlock.ID, edited); saved.Code != http.StatusOK {
		t.Fatalf("edit the placed block = %d: %s", saved.Code, saved.Body.String())
	}
	if refused := undoPlacement(t, r, session, workID, backstory.ID); refused.Code != http.StatusConflict {
		t.Fatalf("undo after an edit = %d: %s", refused.Code, refused.Body.String())
	}
	if proseText(t, blockTitledIn(t, readSeededPage(t, r, session, workID+"?draftedChanges=true").Blocks, "Backstory")) != "Rewritten." {
		t.Error("the refused undo changed the edited block")
	}
}

func TestALongReadmeShelvesEverySectionAfterTheThird(t *testing.T) {
	t.Parallel()
	r, session, works, _ := harness.NewExtensionRouter(t)
	readme := "# Quiet Toolbox\n\nSmall tools.\n\n## One\n\nFirst.\n\n## Two\n\nSecond.\n\n## Three\n\nThird.\n\n" +
		"## Four\n\nFourth.\n\n![Shot](art/shot.png)\n\n## Five\n\nFifth.\n"
	workID := apitest.UploadExtension(t, r, session, works, apitest.ExtensionZip(t, map[string]string{
		"spindle.json": apitest.ToolboxManifest, "dist/frontend.js": "export default {}", "README.md": readme,
		"art/shot.png": pictureFile(t, 90),
	}))

	page := readSeededPage(t, r, session, workID+"?draftedChanges=true")
	if got := strings.Join(arrangement(page), "|"); got != "custom_block About|extension_permissions Permissions|"+
		"extension_source Version and source|custom_block One|custom_block Two|custom_block Three" {
		t.Fatalf("blocks = %s, want the opening and three sections", got)
	}
	shelf := readShelf(t, r, session, workID)
	if len(shelf) != 1 || shelf[0].Source != "readme" {
		t.Fatalf("shelf = %+v, want one README import", shelf)
	}
	var got []string
	for _, piece := range shelf[0].Pieces {
		got = append(got, piece.Kind+" "+piece.Section+" "+piece.Text+piece.BlockID)
	}
	if strings.Join(got, "|") != "section Four Fourth.|picture Four |section Five Fifth." {
		t.Fatalf("pieces = %q, want the later sections and the picture with no block yet", got)
	}

	four := sectionNamed(t, shelf[0].Pieces, "Four")
	if placed := placePiece(t, r, session, workID, four.ID, `{"position":6}`); placed.Code != http.StatusOK {
		t.Fatalf("place section four = %d: %s", placed.Code, placed.Body.String())
	}
	shot := shelfPieces(t, r, session, workID)[0]
	if placed := placePiece(t, r, session, workID, shot.ID, ""); placed.Code != http.StatusOK {
		t.Fatalf("place the picture = %d: %s", placed.Code, placed.Body.String())
	}
	placedPage := readSeededPage(t, r, session, workID+"?draftedChanges=true")
	if len(placedPage.Blocks) != 7 || len(imageIDs(t, blockTitledIn(t, placedPage.Blocks, "Four"))) != 1 {
		t.Errorf("blocks = %q, want the picture in the block section four became", arrangement(placedPage))
	}
}

func TestAnotherAccountCannotReachTheShelf(t *testing.T) {
	t.Parallel()
	r, session, _, pool := harness.NewExtensionRouter(t)
	workID := apitest.StartCharacter(t, r, session).ID
	if added := addMarkdown(t, r, session, workID, pastedMarkdown); added.Code != http.StatusCreated {
		t.Fatalf("paste = %d: %s", added.Code, added.Body.String())
	}
	piece := shelfPieces(t, r, session, workID)[0]
	stranger := apitest.AddVerifiedUser(t, r, pool, "stranger@example.com", "stranger")

	read := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(http.MethodGet, "/v1/works/"+workID+"/shelf", nil), stranger))
	if read.Code != http.StatusNotFound {
		t.Errorf("stranger reads the shelf = %d: %s", read.Code, read.Body.String())
	}
	version := fmt.Sprint(draftedVersion(t, r, session, workID))
	place := httptest.NewRequest(http.MethodPost, "/v1/works/"+workID+"/shelf/pieces/"+piece.ID+"/place",
		strings.NewReader(`{"position":0}`))
	place.Header.Set("Content-Type", "application/json")
	place.Header.Set("X-Drafted-Changes-Version", version)
	if placed := apitest.Send(t, r, apitest.Authorized(place, stranger)); placed.Code != http.StatusNotFound {
		t.Errorf("stranger places a piece = %d: %s", placed.Code, placed.Body.String())
	}
	if added := addMarkdown(t, r, stranger, workID, "Not yours."); added.Code != http.StatusNotFound {
		t.Errorf("stranger pastes = %d: %s", added.Code, added.Body.String())
	}
	letGoRequest := httptest.NewRequest(http.MethodDelete, "/v1/works/"+workID+"/shelf/pieces/"+piece.ID, nil)
	letGoRequest.Header.Set("X-Drafted-Changes-Version", version)
	letGo := apitest.Send(t, r, apitest.Authorized(letGoRequest, stranger))
	if letGo.Code != http.StatusNotFound {
		t.Errorf("stranger lets a piece go = %d: %s", letGo.Code, letGo.Body.String())
	}
	if left := shelfPieces(t, r, session, workID); len(left) != 6 {
		t.Errorf("shelf after the stranger = %d pieces, want all 6", len(left))
	}
}

func TestPlacingAllOfAnImportMakesABlockPerSectionAndOneUndoTakesThemBack(t *testing.T) {
	t.Parallel()
	r, session, _, _ := harness.NewExtensionRouter(t)
	workID := apitest.PublishedCharacter(t, r, session)
	if added := addMarkdown(t, r, session, workID, pastedMarkdown); added.Code != http.StatusCreated {
		t.Fatalf("paste = %d: %s", added.Code, added.Body.String())
	}
	held := readShelf(t, r, session, workID)[0]
	before := readSeededPage(t, r, session, workID+"?draftedChanges=true")
	published := readSeededPage(t, r, nil, workID)

	request := httptest.NewRequest(http.MethodPost, "/v1/works/"+workID+"/shelf/imports/"+held.ID+"/place", nil)
	placed := apitest.Send(t, r, apitest.Authorized(request, session))
	if placed.Code != http.StatusOK {
		t.Fatalf("place all = %d: %s", placed.Code, placed.Body.String())
	}
	var answer struct {
		PieceIds []string `json:"pieceIds"`
	}
	if err := json.Unmarshal(placed.Body.Bytes(), &answer); err != nil || len(answer.PieceIds) != 4 {
		t.Fatalf("placed = %s, want the four sections named", placed.Body.String())
	}
	drafted := arrangement(readSeededPage(t, r, session, workID+"?draftedChanges=true"))
	want := append(arrangement(before), "custom_block Wren", "custom_block Backstory", "custom_block Early years", "custom_block Relationships")
	if strings.Join(drafted, "|") != strings.Join(want, "|") {
		t.Fatalf("drafted blocks = %q, want a block per section at the end: %q", drafted, want)
	}
	if left := shelfPieces(t, r, session, workID); len(left) != 2 || left[0].Kind != "picture" || left[1].Kind != "picture" {
		t.Errorf("shelf after placing all = %+v, want only the pictures held by address", left)
	}
	if strings.Join(arrangement(readSeededPage(t, r, nil, workID)), "|") != strings.Join(arrangement(published), "|") {
		t.Error("placing all changed the published page")
	}

	body, _ := json.Marshal(map[string][]string{"pieceIds": answer.PieceIds})
	undo := httptest.NewRequest(http.MethodPost, "/v1/works/"+workID+"/shelf/undo", strings.NewReader(string(body)))
	undo.Header.Set("Content-Type", "application/json")
	if undone := apitest.Send(t, r, apitest.Authorized(undo, session)); undone.Code != http.StatusOK {
		t.Fatalf("undo all = %d: %s", undone.Code, undone.Body.String())
	}
	if got := arrangement(readSeededPage(t, r, session, workID+"?draftedChanges=true")); strings.Join(got, "|") != strings.Join(arrangement(before), "|") {
		t.Errorf("drafted blocks after undo = %q, want %q", got, arrangement(before))
	}
	if left := shelfPieces(t, r, session, workID); len(left) != 6 {
		t.Errorf("shelf after undo = %d pieces, want all 6 back", len(left))
	}
}
