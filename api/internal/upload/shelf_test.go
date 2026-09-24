package upload_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

const seededReadme = "# Quiet Toolbox\n\n![Banner](art/banner.png)\n\nSmall tools for a calmer chat.\n\n" +
	"## Install\n\nClone it.\n\n![Settings](art/settings.png)\n\n" +
	"## <script>alert(1)</script>Usage\n\n<style>body{display:none}</style>\n\nType `/quiet`.\n\n![Wide](https://example.com/wide.png)\n"

type seededPage struct {
	Media []struct {
		ID      string `json:"id"`
		IsCover bool   `json:"isCover"`
	} `json:"media"`
	Blocks []seededBlock `json:"blocks"`
}

type seededBlock struct {
	ID         string `json:"id"`
	Definition string `json:"definition"`
	Title      string `json:"title"`
	Elements   []struct {
		ID       string          `json:"id"`
		Type     string          `json:"type"`
		Role     string          `json:"role"`
		Pinned   bool            `json:"pinned"`
		FromFile bool            `json:"fromFile"`
		Content  json.RawMessage `json:"content"`
	} `json:"elements"`
}

func TestAReadmeSeedsTheNewPageWithBlocksTheCreatorOwns(t *testing.T) {
	t.Parallel()
	r, session, works, _ := harness.NewExtensionRouter(t)
	workID := apitest.UploadExtension(t, r, session, works, apitest.ExtensionZip(t, map[string]string{
		"spindle.json": apitest.ToolboxManifest, "dist/frontend.js": "export default {}", "README.md": seededReadme,
		"art/banner.png": pictureFile(t, 20), "art/settings.png": pictureFile(t, 200),
	}))

	page := readSeededPage(t, r, session, workID+"?draftedChanges=true")
	if got, want := arrangement(page), []string{
		"custom_block About", "extension_permissions Permissions", "extension_source Version and source",
		"custom_block Install", "custom_block Usage",
	}; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("blocks = %q, want %q", got, want)
	}
	for _, holder := range page.Blocks {
		for _, element := range holder.Elements {
			if seeded := holder.Definition == "custom_block"; seeded == (element.Pinned || element.FromFile) {
				t.Errorf("%s %s element is pinned %t and from the file %t", holder.Title, element.Type, element.Pinned, element.FromFile)
			}
		}
	}
	if text := proseText(t, page.Blocks[0]); text != "Small tools for a calmer chat." {
		t.Errorf("about = %q", text)
	}
	install := page.Blocks[3]
	if proseText(t, install) != "Clone it." || len(imageIDs(t, install)) != 0 {
		t.Fatalf("install = %+v, want its text, with its picture waiting on the shelf", install)
	}
	if len(page.Media) != 2 || !page.Media[0].IsCover {
		t.Errorf("media = %+v, want the banner as the cover beside the settings picture", page.Media)
	}
	if usage := proseText(t, page.Blocks[4]); usage != "Type `/quiet`." {
		t.Errorf("usage = %q, want the stylesheet left out", usage)
	}
	waiting := shelfPieces(t, r, session, workID)
	if len(waiting) != 2 || waiting[0].Name != "Settings" || waiting[0].BlockID != install.ID || waiting[0].Media == nil ||
		waiting[0].Media.ID != page.Media[1].ID || waiting[0].Media.ThumbURL == "" {
		t.Fatalf("shelf = %+v, want the settings picture waiting for the install block", waiting)
	}
	if remote := waiting[1]; remote.Media != nil || remote.Address != "https://example.com/wide.png" ||
		remote.Name != "Wide" || remote.BlockID != page.Blocks[4].ID {
		t.Errorf("shelf[1] = %+v, want the remote picture listed by address for the usage block", remote)
	}
}

func TestACreatorPlacesOrDiscardsEachWaitingPicture(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewExtensionRouter(t)
	workID := apitest.UploadExtension(t, r, session, works, apitest.ExtensionZip(t, map[string]string{
		"spindle.json": apitest.ToolboxManifest, "dist/frontend.js": "export default {}", "README.md": seededReadme,
		"art/banner.png": pictureFile(t, 20), "art/settings.png": pictureFile(t, 200),
	}))
	pieces := shelfPieces(t, r, session, workID)
	settings, wide := pieces[0], pieces[1]

	placed := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodPost, "/v1/works/"+workID+"/shelf/pieces/"+settings.ID+"/place", nil), session))
	if placed.Code != http.StatusOK {
		t.Fatalf("place the settings picture = %d: %s", placed.Code, placed.Body.String())
	}
	var blocks []seededBlock
	if err := json.Unmarshal(placed.Body.Bytes(), &blocks); err != nil {
		t.Fatalf("decode the placed page: %v", err)
	}
	install := blockTitledIn(t, blocks, "Install")
	if ids := imageIDs(t, install); len(ids) != 1 || ids[0] != settings.Media.ID || len(install.Elements) != 2 {
		t.Fatalf("install after placing = %+v, want the picture beside the text", install)
	}

	refused := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodPost, "/v1/works/"+workID+"/shelf/pieces/"+wide.ID+"/place", nil), session))
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("place a remote picture without a copy = %d: %s", refused.Code, refused.Body.String())
	}
	uploaded := apitest.Send(t, r, apitest.Authorized(apitest.MediaUploadRequest(t, workID, "gallery", []byte(pictureFile(t, 90))), session))
	if uploaded.Code != http.StatusCreated {
		t.Fatalf("upload a copy = %d: %s", uploaded.Code, uploaded.Body.String())
	}
	var copyMedia struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(uploaded.Body.Bytes(), &copyMedia); err != nil {
		t.Fatalf("decode the copy: %v", err)
	}
	withCopy := httptest.NewRequest(http.MethodPost, "/v1/works/"+workID+"/shelf/pieces/"+wide.ID+"/place",
		strings.NewReader(fmt.Sprintf(`{"mediaId":%q}`, copyMedia.ID)))
	withCopy.Header.Set("Content-Type", "application/json")
	if placed := apitest.Send(t, r, apitest.Authorized(withCopy, session)); placed.Code != http.StatusOK {
		t.Fatalf("place the copy = %d: %s", placed.Code, placed.Body.String())
	}
	page := readSeededPage(t, r, session, workID+"?draftedChanges=true")
	if ids := imageIDs(t, blockTitledIn(t, page.Blocks, "Usage")); len(ids) != 1 || ids[0] != copyMedia.ID {
		t.Errorf("usage after placing the copy = %v, want the uploaded picture", ids)
	}
	if left := shelfPieces(t, r, session, workID); len(left) != 0 {
		t.Errorf("shelf after placing both = %+v, want it empty", left)
	}

	second := apitest.UploadExtension(t, r, session, works, apitest.ExtensionZip(t, map[string]string{
		"spindle.json": apitest.ToolboxManifest, "dist/frontend.js": "export default {}",
		"README.md":   "# Quiet Toolbox\n\n## Screenshots\n\n![One](art/one.png)\n\n![Two](art/two.png)\n\n## Install\n\nClone it.\n",
		"art/one.png": pictureFile(t, 20), "art/two.png": pictureFile(t, 200),
	}))
	shots := shelfPieces(t, r, session, second)
	if len(shots) != 2 || shots[0].BlockID != "" {
		t.Fatalf("shelf = %+v, want two pictures with no block, since their section held nothing else", shots)
	}
	if placed := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodPost, "/v1/works/"+second+"/shelf/pieces/"+shots[0].ID+"/place", nil), session)); placed.Code != http.StatusOK {
		t.Fatalf("place a picture with no block = %d: %s", placed.Code, placed.Body.String())
	}
	if placed := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodPost, "/v1/works/"+second+"/shelf/pieces/"+shots[1].ID+"/place", nil), session)); placed.Code != http.StatusOK {
		t.Fatalf("place the second picture = %d: %s", placed.Code, placed.Body.String())
	}
	shotsPage := readSeededPage(t, r, session, second+"?draftedChanges=true")
	if got := arrangement(shotsPage); strings.Join(got, "|") != "extension_permissions Permissions|extension_source Version and source|custom_block Install|custom_block Screenshots" {
		t.Fatalf("blocks after placing = %q, want one new block named after the section", got)
	}
	if ids := imageIDs(t, blockTitledIn(t, shotsPage.Blocks, "Screenshots")); len(ids) != 2 {
		t.Errorf("screenshots = %v, want both pictures in the one block", ids)
	}

	third := apitest.UploadExtension(t, r, session, works, apitest.ExtensionZip(t, map[string]string{
		"spindle.json": apitest.ToolboxManifest, "dist/frontend.js": "export default {}", "README.md": seededReadme,
		"art/banner.png": pictureFile(t, 20), "art/settings.png": pictureFile(t, 200),
	}))
	waiting := shelfPieces(t, r, session, third)[0]
	discarded := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/works/"+third+"/shelf/pieces/"+waiting.ID, nil), session))
	if discarded.Code != http.StatusNoContent {
		t.Fatalf("discard = %d: %s", discarded.Code, discarded.Body.String())
	}
	var kept int
	if err := pool.QueryRow(context.Background(),
		`select count(*) from work_media where id = $1`, waiting.Media.ID).Scan(&kept); err != nil || kept != 0 {
		t.Errorf("discarded media rows = %d, %v; want the copy gone", kept, err)
	}
	if page := readSeededPage(t, r, session, third+"?draftedChanges=true"); len(page.Media) != 1 {
		t.Errorf("media after discarding = %+v, want only the cover", page.Media)
	}
}

type shelfPiece struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Section string `json:"section"`
	Text    string `json:"text"`
	Address string `json:"address"`
	Name    string `json:"name"`
	BlockID string `json:"blockId"`
	Media   *struct {
		ID       string `json:"id"`
		ThumbURL string `json:"thumbUrl"`
	} `json:"media"`
}

type shelfImport struct {
	ID     string       `json:"id"`
	Source string       `json:"source"`
	Title  string       `json:"title"`
	Pieces []shelfPiece `json:"pieces"`
}

func readShelf(t *testing.T, r http.Handler, session *http.Cookie, workID string) []shelfImport {
	t.Helper()
	response := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(http.MethodGet, "/v1/works/"+workID+"/shelf", nil), session))
	if response.Code != http.StatusOK {
		t.Fatalf("read the shelf = %d: %s", response.Code, response.Body.String())
	}
	var shelf struct {
		Imports []shelfImport `json:"imports"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &shelf); err != nil {
		t.Fatalf("decode the shelf: %v", err)
	}
	return shelf.Imports
}

func shelfPieces(t *testing.T, r http.Handler, session *http.Cookie, workID string) []shelfPiece {
	t.Helper()
	pieces := []shelfPiece{}
	for _, held := range readShelf(t, r, session, workID) {
		pieces = append(pieces, held.Pieces...)
	}
	return pieces
}

func TestAReplacementArchiveLeavesTheSeededBlocksToTheCreator(t *testing.T) {
	t.Parallel()
	r, session, works, _ := harness.NewExtensionRouter(t)
	workID := apitest.PublishExtension(t, r, session, works, "Quiet Toolbox", apitest.ExtensionZip(t, map[string]string{
		"spindle.json": apitest.ToolboxManifest, "dist/frontend.js": "one", "README.md": seededReadme,
		"art/banner.png": pictureFile(t, 20), "art/settings.png": pictureFile(t, 200),
	}))
	started := apitest.FetchStartedWork(t, r, session, workID)
	install := blockTitled(t, started.Blocks, "Install")
	edited := apitest.EditableBlock(install)
	edited.Title = &install.Title
	edited.Elements[0].Content = json.RawMessage(`{"text":"Clone it, then restart."}`)
	if saved := apitest.SaveBlock(t, r, session, workID, install.ID, edited); saved.Code >= http.StatusMultipleChoices {
		t.Fatalf("edit the seeded install text = %d: %s", saved.Code, saved.Body.String())
	}
	if update := apitest.PublishWorkVersion(t, r, session, workID, `{"summary":"Clearer install"}`); update.Code != http.StatusOK {
		t.Fatalf("publish the edit = %d: %s", update.Code, update.Body.String())
	}
	before := readSeededPage(t, r, nil, workID)

	manifest := strings.Replace(apitest.ToolboxManifest, `"version": "1.0.0"`, `"version": "1.1.0"`, 1)
	second := apitest.ExtensionZip(t, map[string]string{
		"spindle.json": manifest, "dist/frontend.js": "two",
		"README.md": "# Quiet Toolbox\n\n## Changelog\n\nNew in 1.1.", "art/settings.png": pictureFile(t, 90),
	})
	uploaded := apitest.Send(t, r, apitest.Authorized(apitest.OriginalFileRequest(t, workID, "toolbox.zip", second), session))
	if uploaded.Code != http.StatusAccepted {
		t.Fatalf("upload the replacement = %d: %s", uploaded.Code, uploaded.Body.String())
	}
	if _, err := apitest.Uploads(works).ProcessNextUpload(context.Background()); err != nil {
		t.Fatalf("process the replacement: %v", err)
	}
	apitest.AcceptReplacementPreview(t, r, session, workID, uploaded.Header().Get("Location"))
	if update := apitest.PublishWorkVersion(t, r, session, workID, `{"summary":"Version 1.1"}`); update.Code != http.StatusOK {
		t.Fatalf("publish the update = %d: %s", update.Code, update.Body.String())
	}

	after := readSeededPage(t, r, nil, workID)
	if got, want := arrangement(after), arrangement(before); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("blocks after the replacement = %q, want the page as the creator left it: %q", got, want)
	}
	if text := proseText(t, blockTitledIn(t, after.Blocks, "Install")); text != "Clone it, then restart." {
		t.Errorf("install after the replacement = %q, want the creator's edit", text)
	}
	if details := string(blockTitledIn(t, after.Blocks, "Version and source").Elements[0].Content); !strings.Contains(details, "1.1.0") {
		t.Errorf("details after the replacement = %s, want the new version", details)
	}
	if before.Media[0].ID != after.Media[0].ID || !after.Media[0].IsCover || len(after.Media) != len(before.Media) {
		t.Errorf("media after the replacement = %+v, want the seeded pictures kept: %+v", after.Media, before.Media)
	}
}

func readSeededPage(t *testing.T, r http.Handler, session *http.Cookie, address string) seededPage {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/works/"+address, nil)
	if session != nil {
		request = apitest.Authorized(request, session)
	}
	response := apitest.Send(t, r, request)
	if response.Code != http.StatusOK {
		t.Fatalf("read the extension = %d: %s", response.Code, response.Body.String())
	}
	var page seededPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode the extension: %v", err)
	}
	return page
}

func arrangement(page seededPage) []string {
	found := make([]string, 0, len(page.Blocks))
	for _, holder := range page.Blocks {
		found = append(found, holder.Definition+" "+holder.Title)
	}
	return found
}

func blockTitled(t *testing.T, blocks []apitest.StartedBlock, title string) apitest.StartedBlock {
	t.Helper()
	found := []string{}
	for _, holder := range blocks {
		if holder.Title == title {
			return holder
		}
		found = append(found, holder.Definition+" "+holder.Title)
	}
	t.Fatalf("no block titled %s among %q", title, found)
	return apitest.StartedBlock{}
}

func blockTitledIn(t *testing.T, blocks []seededBlock, title string) seededBlock {
	t.Helper()
	for _, holder := range blocks {
		if holder.Title == title {
			return holder
		}
	}
	t.Fatalf("no block titled %s among %q", title, arrangement(seededPage{Blocks: blocks}))
	return seededBlock{}
}

func proseText(t *testing.T, holder seededBlock) string {
	t.Helper()
	for _, element := range holder.Elements {
		if element.Type != "prose" {
			continue
		}
		var prose struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(element.Content, &prose); err != nil {
			t.Fatalf("decode %s prose: %v", holder.Title, err)
		}
		return prose.Text
	}
	return ""
}

func imageIDs(t *testing.T, holder seededBlock) []string {
	t.Helper()
	var ids []string
	for _, element := range holder.Elements {
		if element.Type != "image_set" {
			continue
		}
		var set struct {
			Images []struct {
				MediaID string `json:"mediaId"`
			} `json:"images"`
		}
		if err := json.Unmarshal(element.Content, &set); err != nil {
			t.Fatalf("decode %s images: %v", holder.Title, err)
		}
		for _, item := range set.Images {
			ids = append(ids, item.MediaID)
		}
	}
	return ids
}

func pictureFile(t *testing.T, shade uint8) string {
	t.Helper()
	canvas := image.NewRGBA(image.Rect(0, 0, 32, 24))
	for x := range 32 {
		for y := range 24 {
			canvas.Set(x, y, color.RGBA{R: shade, G: uint8(x * 8), B: uint8(y * 10), A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, canvas); err != nil {
		t.Fatalf("encode a picture: %v", err)
	}
	return encoded.String()
}
