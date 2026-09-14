package http

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
		Type    string          `json:"type"`
		Role    string          `json:"role"`
		Pinned  bool            `json:"pinned"`
		Locked  bool            `json:"locked"`
		Content json.RawMessage `json:"content"`
	} `json:"elements"`
}

func TestAReadmeSeedsTheNewPageWithBlocksTheCreatorOwns(t *testing.T) {
	r, session, assets, _ := newExtensionRouter(t)
	assetID := uploadExtension(t, r, session, assets, extensionZip(t, map[string]string{
		"spindle.json": toolboxManifest, "dist/frontend.js": "export default {}", "README.md": seededReadme,
		"art/banner.png": pictureFile(t, 20), "art/settings.png": pictureFile(t, 200),
	}))

	page := readSeededPage(t, r, session, assetID+"?workingCopy=true")
	if got, want := arrangement(page), []string{
		"custom_block About", "extension_permissions Permissions", "extension_source Version and source",
		"custom_block Install", "custom_block Usage",
	}; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("blocks = %q, want %q", got, want)
	}
	for _, holder := range page.Blocks {
		for _, element := range holder.Elements {
			if seeded := holder.Definition == "custom_block"; seeded == (element.Pinned || element.Locked) {
				t.Errorf("%s %s element is pinned %t and locked %t", holder.Title, element.Type, element.Pinned, element.Locked)
			}
		}
	}
	if text := proseText(t, page.Blocks[0]); text != "Small tools for a calmer chat." {
		t.Errorf("about = %q", text)
	}
	install := page.Blocks[3]
	if proseText(t, install) != "Clone it." || len(imageIDs(t, install)) != 0 {
		t.Fatalf("install = %+v, want its text, with its picture waiting in the vault", install)
	}
	if len(page.Media) != 2 || !page.Media[0].IsCover {
		t.Errorf("media = %+v, want the banner as the cover beside the settings picture", page.Media)
	}
	if usage := proseText(t, page.Blocks[4]); usage != "Type `/quiet`." {
		t.Errorf("usage = %q, want the stylesheet left out", usage)
	}
	vault := readVault(t, r, session, assetID)
	if len(vault) != 2 || vault[0].Name != "Settings" || vault[0].BlockID != install.ID || vault[0].Media == nil ||
		vault[0].Media.ID != page.Media[1].ID || vault[0].Media.ThumbURL == "" {
		t.Fatalf("vault = %+v, want the settings picture waiting for the install block", vault)
	}
	if remote := vault[1]; remote.Media != nil || remote.Address != "https://example.com/wide.png" ||
		remote.Name != "Wide" || remote.BlockID != page.Blocks[4].ID {
		t.Errorf("vault[1] = %+v, want the remote picture listed by address for the usage block", remote)
	}
}

func TestACreatorPlacesOrDiscardsEachWaitingPicture(t *testing.T) {
	r, session, assets, pool := newExtensionRouter(t)
	assetID := uploadExtension(t, r, session, assets, extensionZip(t, map[string]string{
		"spindle.json": toolboxManifest, "dist/frontend.js": "export default {}", "README.md": seededReadme,
		"art/banner.png": pictureFile(t, 20), "art/settings.png": pictureFile(t, 200),
	}))
	vault := readVault(t, r, session, assetID)
	settings, wide := vault[0], vault[1]

	placed := send(t, r, authorized(httptest.NewRequest(
		http.MethodPost, "/v1/assets/"+assetID+"/vault/"+settings.ID+"/place", nil), session))
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

	refused := send(t, r, authorized(httptest.NewRequest(
		http.MethodPost, "/v1/assets/"+assetID+"/vault/"+wide.ID+"/place", nil), session))
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("place a remote picture without a copy = %d: %s", refused.Code, refused.Body.String())
	}
	uploaded := send(t, r, authorized(mediaUploadRequest(t, assetID, "gallery", []byte(pictureFile(t, 90))), session))
	if uploaded.Code != http.StatusCreated {
		t.Fatalf("upload a copy = %d: %s", uploaded.Code, uploaded.Body.String())
	}
	var copyMedia struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(uploaded.Body.Bytes(), &copyMedia); err != nil {
		t.Fatalf("decode the copy: %v", err)
	}
	withCopy := httptest.NewRequest(http.MethodPost, "/v1/assets/"+assetID+"/vault/"+wide.ID+"/place",
		strings.NewReader(fmt.Sprintf(`{"mediaId":%q}`, copyMedia.ID)))
	withCopy.Header.Set("Content-Type", "application/json")
	if placed := send(t, r, authorized(withCopy, session)); placed.Code != http.StatusOK {
		t.Fatalf("place the copy = %d: %s", placed.Code, placed.Body.String())
	}
	page := readSeededPage(t, r, session, assetID+"?workingCopy=true")
	if ids := imageIDs(t, blockTitledIn(t, page.Blocks, "Usage")); len(ids) != 1 || ids[0] != copyMedia.ID {
		t.Errorf("usage after placing the copy = %v, want the uploaded picture", ids)
	}
	if left := readVault(t, r, session, assetID); len(left) != 0 {
		t.Errorf("vault after placing both = %+v, want it empty", left)
	}

	second := uploadExtension(t, r, session, assets, extensionZip(t, map[string]string{
		"spindle.json": toolboxManifest, "dist/frontend.js": "export default {}",
		"README.md":   "# Quiet Toolbox\n\n## Screenshots\n\n![One](art/one.png)\n\n![Two](art/two.png)\n\n## Install\n\nClone it.\n",
		"art/one.png": pictureFile(t, 20), "art/two.png": pictureFile(t, 200),
	}))
	shots := readVault(t, r, session, second)
	if len(shots) != 2 || shots[0].BlockID != "" {
		t.Fatalf("vault = %+v, want two pictures with no block, since their section held nothing else", shots)
	}
	if placed := send(t, r, authorized(httptest.NewRequest(
		http.MethodPost, "/v1/assets/"+second+"/vault/"+shots[0].ID+"/place", nil), session)); placed.Code != http.StatusOK {
		t.Fatalf("place a picture with no block = %d: %s", placed.Code, placed.Body.String())
	}
	if placed := send(t, r, authorized(httptest.NewRequest(
		http.MethodPost, "/v1/assets/"+second+"/vault/"+shots[1].ID+"/place", nil), session)); placed.Code != http.StatusOK {
		t.Fatalf("place the second picture = %d: %s", placed.Code, placed.Body.String())
	}
	shotsPage := readSeededPage(t, r, session, second+"?workingCopy=true")
	if got := arrangement(shotsPage); strings.Join(got, "|") != "extension_permissions Permissions|extension_source Version and source|custom_block Install|custom_block Screenshots" {
		t.Fatalf("blocks after placing = %q, want one new block named after the section", got)
	}
	if ids := imageIDs(t, blockTitledIn(t, shotsPage.Blocks, "Screenshots")); len(ids) != 2 {
		t.Errorf("screenshots = %v, want both pictures in the one block", ids)
	}

	third := uploadExtension(t, r, session, assets, extensionZip(t, map[string]string{
		"spindle.json": toolboxManifest, "dist/frontend.js": "export default {}", "README.md": seededReadme,
		"art/banner.png": pictureFile(t, 20), "art/settings.png": pictureFile(t, 200),
	}))
	waiting := readVault(t, r, session, third)[0]
	discarded := send(t, r, authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/assets/"+third+"/vault/"+waiting.ID, nil), session))
	if discarded.Code != http.StatusNoContent {
		t.Fatalf("discard = %d: %s", discarded.Code, discarded.Body.String())
	}
	var kept int
	if err := pool.QueryRow(context.Background(),
		`select count(*) from asset_media where id = $1`, waiting.Media.ID).Scan(&kept); err != nil || kept != 0 {
		t.Errorf("discarded media rows = %d, %v; want the copy gone", kept, err)
	}
	if page := readSeededPage(t, r, session, third+"?workingCopy=true"); len(page.Media) != 1 {
		t.Errorf("media after discarding = %+v, want only the cover", page.Media)
	}
}

type vaultPicture struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Name    string `json:"name"`
	BlockID string `json:"blockId"`
	Media   *struct {
		ID       string `json:"id"`
		ThumbURL string `json:"thumbUrl"`
	} `json:"media"`
}

func readVault(t *testing.T, r http.Handler, session *http.Cookie, assetID string) []vaultPicture {
	t.Helper()
	response := send(t, r, authorized(httptest.NewRequest(http.MethodGet, "/v1/assets/"+assetID+"/vault", nil), session))
	if response.Code != http.StatusOK {
		t.Fatalf("read the vault = %d: %s", response.Code, response.Body.String())
	}
	var listed struct {
		Pictures []vaultPicture `json:"pictures"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode the vault: %v", err)
	}
	return listed.Pictures
}

func TestAReplacementArchiveLeavesTheSeededBlocksToTheCreator(t *testing.T) {
	r, session, assets, _ := newExtensionRouter(t)
	assetID := publishExtension(t, r, session, assets, "Quiet Toolbox", extensionZip(t, map[string]string{
		"spindle.json": toolboxManifest, "dist/frontend.js": "one", "README.md": seededReadme,
		"art/banner.png": pictureFile(t, 20), "art/settings.png": pictureFile(t, 200),
	}))
	started := fetchStartedAsset(t, r, session, assetID)
	install := blockTitled(t, started.Blocks, "Install")
	edited := editableBlock(install)
	edited.Title = &install.Title
	edited.Elements[0].Content = json.RawMessage(`{"text":"Clone it, then restart."}`)
	if saved := saveBlock(t, r, session, assetID, install.ID, edited); saved.Code >= http.StatusMultipleChoices {
		t.Fatalf("edit the seeded install text = %d: %s", saved.Code, saved.Body.String())
	}
	if update := publishAssetUpdate(t, r, session, assetID, `{"summary":"Clearer install"}`); update.Code != http.StatusOK {
		t.Fatalf("publish the edit = %d: %s", update.Code, update.Body.String())
	}
	before := readSeededPage(t, r, nil, assetID)

	manifest := strings.Replace(toolboxManifest, `"version": "1.0.0"`, `"version": "1.1.0"`, 1)
	second := extensionZip(t, map[string]string{
		"spindle.json": manifest, "dist/frontend.js": "two",
		"README.md": "# Quiet Toolbox\n\n## Changelog\n\nNew in 1.1.", "art/settings.png": pictureFile(t, 90),
	})
	revision := send(t, r, authorized(revisionRequest(t, assetID, "toolbox.zip", second), session))
	if revision.Code != http.StatusAccepted {
		t.Fatalf("upload the replacement = %d: %s", revision.Code, revision.Body.String())
	}
	if _, err := assets.ProcessNextIngest(context.Background()); err != nil {
		t.Fatalf("process the replacement: %v", err)
	}
	acceptReplacementPreview(t, r, session, assetID, revision.Header().Get("Location"))
	if update := publishAssetUpdate(t, r, session, assetID, `{"summary":"Version 1.1"}`); update.Code != http.StatusOK {
		t.Fatalf("publish the update = %d: %s", update.Code, update.Body.String())
	}

	after := readSeededPage(t, r, nil, assetID)
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
	request := httptest.NewRequest(http.MethodGet, "/v1/assets/"+address, nil)
	if session != nil {
		request = authorized(request, session)
	}
	response := send(t, r, request)
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

func blockTitled(t *testing.T, blocks []startedBlock, title string) startedBlock {
	t.Helper()
	found := []string{}
	for _, holder := range blocks {
		if holder.Title == title {
			return holder
		}
		found = append(found, holder.Definition+" "+holder.Title)
	}
	t.Fatalf("no block titled %s among %q", title, found)
	return startedBlock{}
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
