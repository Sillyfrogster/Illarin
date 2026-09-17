package http

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format/extension"
)

type extensionPage struct {
	Kind                  string   `json:"kind"`
	Name                  string   `json:"name"`
	Identifier            *string  `json:"identifier"`
	InstalledAppVersions  []string `json:"installedAppVersions"`
	ExtensionDependencies []struct {
		Name   string `json:"name"`
		Assets []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Creator string `json:"creator"`
		} `json:"assets"`
	} `json:"extensionDependencies"`
	Blocks []struct {
		ID         string `json:"id"`
		Definition string `json:"definition"`
		Elements   []struct {
			Role    string          `json:"role"`
			Pinned  bool            `json:"pinned"`
			Locked  bool            `json:"locked"`
			Content json.RawMessage `json:"content"`
		} `json:"elements"`
	} `json:"blocks"`
}

func TestASpindleExtensionIsListedDownloadedAndDeliveredAsTheUploadedArchive(t *testing.T) {
	t.Parallel()
	r, session, assets, pool := harness.NewExtensionRouter(t)
	upload := apitest.ExtensionZip(t, map[string]string{
		"spindle.json": apitest.ToolboxManifest, "dist/frontend.js": "export default {}",
	})
	assetID := apitest.UploadExtension(t, r, session, assets, upload)

	page := readExtensionPage(t, r, session, assetID)
	if page.Kind != "extension" || page.Identifier == nil || *page.Identifier != "quiet_toolbox" {
		t.Fatalf("page kind %q identifier %v, want an extension showing quiet_toolbox", page.Kind, page.Identifier)
	}
	if len(page.Blocks) != 2 {
		t.Fatalf("page blocks = %+v, want permissions and source", page.Blocks)
	}
	for _, holder := range page.Blocks {
		for _, element := range holder.Elements {
			if !element.Pinned || !element.Locked {
				t.Errorf("%s is pinned %t and locked %t, want both", element.Role, element.Pinned, element.Locked)
			}
		}
	}

	started := apitest.FetchStartedAsset(t, r, session, assetID)
	source := apitest.BlockNamed(t, started.Blocks, "extension_source")
	edited := apitest.EditableBlock(source)
	edited.Elements[0].Content = json.RawMessage(`{"fields":[{"name":"Version","value":"9.9.9"}]}`)
	refused := apitest.SaveBlock(t, r, session, assetID, source.ID, edited)
	if refused.Code != http.StatusBadRequest || !strings.Contains(refused.Body.String(), "archive") {
		t.Fatalf("edit a locked element = %d %s, want a refusal naming the archive", refused.Code, refused.Body.String())
	}

	if saved := apitest.SaveIdentity(t, r, session, assetID,
		`{"name":"A Renamed Toolbox","blurb":"","isNsfw":false}`); saved.Code != http.StatusNoContent {
		t.Fatalf("save identity = %d: %s", saved.Code, saved.Body.String())
	}
	if published := apitest.PublishAsset(t, r, session, assetID); published.Code != http.StatusOK {
		t.Fatalf("publish the extension = %d: %s", published.Code, published.Body.String())
	}

	menu := apitest.DownloadMenu(t, r, nil, assetID)
	if len(menu) != 1 || menu[0].Format != extension.SpindleID || len(menu[0].Roles) != 0 {
		t.Fatalf("download menu = %+v, want the archive with no loss report", menu)
	}
	download := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/download/"+assetID+"/"+extension.SpindleID, nil))
	if download.Code != http.StatusOK || download.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("download = %d %q: %s", download.Code, download.Header().Get("Content-Type"), download.Body.String())
	}
	assertSameBytes(t, "download", download.Body.Bytes(), upload)

	grant := apitest.LinkDeviceInstance(t, r, session, "Lumiverse", "desk", []string{apitest.ReceiveScope})
	declare(t, r, grant.AccessToken, []string{lumiverseInstalls}, []string{extension.SpindleID})
	if queued := sendToInstance(t, r, session, assetID, grant.Instance.ID); queued.Code != http.StatusAccepted {
		t.Fatalf("send to the instance = %d: %s", queued.Code, queued.Body.String())
	}
	work := apitest.DecodeResponse[deliveryWorkList](t, collect(t, r, grant.AccessToken, nil)).Deliveries[0]
	if work.Format != extension.SpindleID || work.Kind != "extension" {
		t.Fatalf("delivery = %+v, want the Spindle archive", work)
	}
	fetched := fetchSigned(t, r, work.Artifacts[0].URL)
	if fetched.Code != http.StatusOK {
		t.Fatalf("fetch the delivery = %d: %s", fetched.Code, fetched.Body.String())
	}
	assertSameBytes(t, "delivery", fetched.Body.Bytes(), upload)

	if lumiverse := browsedIDs(t, r, "/v1/assets?kind=extension&platform=lumiverse"); !strings.Contains(lumiverse, assetID) {
		t.Errorf("browsing Lumiverse extensions did not find the asset: %s", lumiverse)
	}
	if tavern := browsedIDs(t, r, "/v1/assets?kind=extension&platform=sillytavern"); strings.Contains(tavern, assetID) {
		t.Errorf("browsing SillyTavern extensions found a Spindle extension: %s", tavern)
	}
	if generation := apitest.ContentGeneration(t, pool, assetID); generation != 1 {
		t.Errorf("a listing edit moved the content generation to %d", generation)
	}
}

func TestASillyTavernExtensionListsItsDependenciesAndIsDownloadedAsTheUploadedArchive(t *testing.T) {
	t.Parallel()
	r, session, assets, _ := harness.NewExtensionRouter(t)
	library := apitest.ExtensionZip(t, map[string]string{
		"manifest.json": `{"display_name":"LALib","js":"index.js","author":"A developer",` +
			`"homePage":"https://github.com/example/SillyTavern-LALib"}`,
		"index.js": "",
	})
	libraryID := apitest.PublishExtension(t, r, session, assets, "LALib", library)

	upload := apitest.ExtensionZip(t, map[string]string{
		"manifest.json": `{"display_name":"Custom Sliders","js":"dist/index.js","author":"A developer",` +
			`"version":"1.0.0","homePage":"https://github.com/example/Extension-CustomSliders",` +
			`"dependencies":["third-party/SillyTavern-LALib","vectors"]}`,
		"dist/index.js": "export {}",
	})
	assetID := apitest.PublishExtension(t, r, session, assets, "Custom Sliders", upload)

	page := readExtensionPage(t, r, nil, assetID)
	if page.Identifier == nil || *page.Identifier != "Extension-CustomSliders" {
		t.Fatalf("identifier = %v, want the folder SillyTavern clones into", page.Identifier)
	}
	definitions := []string{}
	for _, holder := range page.Blocks {
		definitions = append(definitions, holder.Definition)
		for _, element := range holder.Elements {
			if !element.Locked {
				t.Errorf("%s is not locked", element.Role)
			}
		}
	}
	if strings.Join(definitions, " ") != "extension_dependencies extension_source" {
		t.Fatalf("blocks = %v, want dependencies and source with no permissions", definitions)
	}
	dependencies := page.ExtensionDependencies
	if len(dependencies) != 2 || dependencies[0].Name != "third-party/SillyTavern-LALib" || dependencies[1].Name != "vectors" {
		t.Fatalf("dependencies = %+v, want both names in manifest order", dependencies)
	}
	if len(dependencies[0].Assets) != 1 || dependencies[0].Assets[0].ID != libraryID || dependencies[0].Assets[0].Name != "LALib" {
		t.Errorf("third-party/SillyTavern-LALib links to %+v, want the LALib asset", dependencies[0].Assets)
	}
	if len(dependencies[1].Assets) != 0 {
		t.Errorf("the built-in vectors links to %+v, want the bare name", dependencies[1].Assets)
	}

	menu := apitest.DownloadMenu(t, r, nil, assetID)
	if len(menu) != 1 || menu[0].Format != extension.SillyTavernID || len(menu[0].Roles) != 0 {
		t.Fatalf("download menu = %+v, want the archive with no loss report", menu)
	}
	download := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/download/"+assetID+"/"+extension.SillyTavernID, nil))
	if download.Code != http.StatusOK {
		t.Fatalf("download = %d: %s", download.Code, download.Body.String())
	}
	assertSameBytes(t, "download", download.Body.Bytes(), upload)

	if tavern := browsedIDs(t, r, "/v1/assets?kind=extension&platform=sillytavern"); !strings.Contains(tavern, assetID) {
		t.Errorf("browsing SillyTavern extensions did not find the asset: %s", tavern)
	}
	if lumiverse := browsedIDs(t, r, "/v1/assets?kind=extension&platform=lumiverse"); strings.Contains(lumiverse, assetID) {
		t.Errorf("browsing Lumiverse extensions found a SillyTavern extension: %s", lumiverse)
	}
}

func TestARepositoryDownloadIsListedAndDownloadedUnchanged(t *testing.T) {
	t.Parallel()
	r, session, assets, _ := harness.NewExtensionRouter(t)
	upload := apitest.ExtensionZip(t, map[string]string{
		"quiet_toolbox-main/":                 "",
		"quiet_toolbox-main/spindle.json":     apitest.ToolboxManifest,
		"quiet_toolbox-main/dist/frontend.js": "export default {}",
	})
	assetID := apitest.PublishExtension(t, r, session, assets, "Quiet Toolbox", upload)

	if page := readExtensionPage(t, r, nil, assetID); page.Identifier == nil || *page.Identifier != "quiet_toolbox" {
		t.Fatalf("identifier = %v, want the manifest inside the folder read", page.Identifier)
	}
	download := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/download/"+assetID+"/"+extension.SpindleID, nil))
	if download.Code != http.StatusOK {
		t.Fatalf("download = %d: %s", download.Code, download.Body.String())
	}
	assertSameBytes(t, "download", download.Body.Bytes(), upload)
}

func TestAReplacementArchiveRefreshesTheLockedElementsAndTheGeneration(t *testing.T) {
	t.Parallel()
	r, session, assets, pool := harness.NewExtensionRouter(t)
	first := apitest.ExtensionZip(t, map[string]string{"spindle.json": apitest.ToolboxManifest, "dist/frontend.js": "one"})
	assetID := apitest.UploadExtension(t, r, session, assets, first)
	if saved := apitest.SaveIdentity(t, r, session, assetID,
		`{"name":"Quiet Toolbox","blurb":"","isNsfw":false}`); saved.Code != http.StatusNoContent {
		t.Fatalf("save identity = %d: %s", saved.Code, saved.Body.String())
	}
	if published := apitest.PublishAsset(t, r, session, assetID); published.Code != http.StatusOK {
		t.Fatalf("publish = %d: %s", published.Code, published.Body.String())
	}
	before := apitest.ContentGeneration(t, pool, assetID)

	manifest := strings.Replace(apitest.ToolboxManifest, `"ui_panels", "generation"`, `"ui_panels", "generation", "tools"`, 1)
	manifest = strings.Replace(manifest, `"version": "1.0.0"`, `"version": "1.1.0"`, 1)
	second := apitest.ExtensionZip(t, map[string]string{"spindle.json": manifest, "dist/frontend.js": "two"})
	revision := apitest.Send(t, r, apitest.Authorized(apitest.RevisionRequest(t, assetID, "toolbox.zip", second), session))
	if revision.Code != http.StatusAccepted {
		t.Fatalf("upload the replacement = %d: %s", revision.Code, revision.Body.String())
	}
	if _, err := apitest.Uploads(assets).ProcessNextIngest(context.Background()); err != nil {
		t.Fatalf("process the replacement: %v", err)
	}
	apitest.AcceptReplacementPreview(t, r, session, assetID, revision.Header().Get("Location"))
	if update := apitest.PublishAssetUpdate(t, r, session, assetID, `{"summary":"Adds tools"}`); update.Code != http.StatusOK {
		t.Fatalf("publish the update = %d: %s", update.Code, update.Body.String())
	}

	if after := apitest.ContentGeneration(t, pool, assetID); after <= before {
		t.Errorf("content generation stayed at %d after a new archive", after)
	}
	page := readExtensionPage(t, r, nil, assetID)
	contents := ""
	for _, holder := range page.Blocks {
		for _, element := range holder.Elements {
			contents += string(element.Content)
		}
	}
	if !strings.Contains(contents, `"tools"`) || !strings.Contains(contents, "1.1.0") {
		t.Errorf("locked elements after the replacement = %s, want the new permission and version", contents)
	}
	download := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/download/"+assetID+"/"+extension.SpindleID, nil))
	assertSameBytes(t, "download after the replacement", download.Body.Bytes(), second)
}

func TestAnExtensionPageListsWhatItsCodeAddsUntilANewArchiveSaysOtherwise(t *testing.T) {
	t.Parallel()
	r, session, assets, _ := harness.NewExtensionRouter(t)
	first := apitest.ExtensionZip(t, map[string]string{
		"spindle.json":     apitest.ToolboxManifest,
		"dist/backend.js":  `spindle.registerTool({ name: "tidy_reply" })`,
		"dist/frontend.js": `export function setup(ctx) { ctx.ui.registerDrawerTab({ title: "Quiet Toolbox" }) }`,
	})
	assetID := apitest.UploadExtension(t, r, session, assets, first)

	started := apitest.FetchStartedAsset(t, r, session, assetID)
	adds := apitest.BlockNamed(t, started.Blocks, "extension_additions")
	edited := apitest.EditableBlock(adds)
	edited.Elements[0].Content = json.RawMessage(`{"fields":[{"name":"Tools","value":"a_tool_it_lacks"}]}`)
	refused := apitest.SaveBlock(t, r, session, assetID, adds.ID, edited)
	if refused.Code != http.StatusBadRequest || !strings.Contains(refused.Body.String(), "archive") {
		t.Fatalf("edit what it adds = %d %s, want a refusal naming the archive", refused.Code, refused.Body.String())
	}
	if saved := apitest.SaveIdentity(t, r, session, assetID,
		`{"name":"Quiet Toolbox","blurb":"","isNsfw":false}`); saved.Code != http.StatusNoContent {
		t.Fatalf("save identity = %d: %s", saved.Code, saved.Body.String())
	}
	if published := apitest.PublishAsset(t, r, session, assetID); published.Code != http.StatusOK {
		t.Fatalf("publish = %d: %s", published.Code, published.Body.String())
	}
	if listed := additionsOnPage(t, r, assetID); listed != "Tools tidy_reply; UI surfaces Drawer tab: Quiet Toolbox" {
		t.Fatalf("what it adds = %q, want the tool and the drawer tab, locked", listed)
	}

	second := apitest.ExtensionZip(t, map[string]string{
		"spindle.json":     strings.Replace(apitest.ToolboxManifest, `"version": "1.0.0"`, `"version": "1.1.0"`, 1),
		"dist/backend.js":  `spindle.registerMacro({ name: "tidy" })`,
		"dist/frontend.js": `export function setup(ctx) { ctx.ui.registerDrawerTab({ title: "Quiet Toolbox" }) }`,
	})
	revision := apitest.Send(t, r, apitest.Authorized(apitest.RevisionRequest(t, assetID, "toolbox.zip", second), session))
	if revision.Code != http.StatusAccepted {
		t.Fatalf("upload the replacement = %d: %s", revision.Code, revision.Body.String())
	}
	if _, err := apitest.Uploads(assets).ProcessNextIngest(context.Background()); err != nil {
		t.Fatalf("process the replacement: %v", err)
	}
	apitest.AcceptReplacementPreview(t, r, session, assetID, revision.Header().Get("Location"))
	if update := apitest.PublishAssetUpdate(t, r, session, assetID, `{"summary":"Swaps the tool for a macro"}`); update.Code != http.StatusOK {
		t.Fatalf("publish the update = %d: %s", update.Code, update.Body.String())
	}
	if listed := additionsOnPage(t, r, assetID); listed != "Macros {{tidy}}; UI surfaces Drawer tab: Quiet Toolbox" {
		t.Fatalf("what it adds after the replacement = %q, want the macro in place of the tool", listed)
	}
}

// additionsOnPage reads the public page's locked list of what the extension adds, as group and name pairs.
func additionsOnPage(t *testing.T, r http.Handler, assetID string) string {
	t.Helper()
	page := readExtensionPage(t, r, nil, assetID)
	for _, holder := range page.Blocks {
		if holder.Definition != "extension_additions" {
			continue
		}
		element := holder.Elements[0]
		if element.Role != "extension_additions" || !element.Locked || !element.Pinned {
			t.Fatalf("what it adds is %+v, want it pinned and locked", element)
		}
		var list struct {
			Fields []struct{ Name, Value string } `json:"fields"`
		}
		if err := json.Unmarshal(element.Content, &list); err != nil {
			t.Fatalf("decode what it adds: %v", err)
		}
		pairs := make([]string, 0, len(list.Fields))
		for _, field := range list.Fields {
			pairs = append(pairs, field.Name+" "+field.Value)
		}
		return strings.Join(pairs, "; ")
	}
	t.Fatal("the page has no block for what the extension adds")
	return ""
}

func TestAnUnsafeOrInvalidExtensionArchiveIsRefused(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		files  map[string]string
		reason string
	}{
		"unsafe path": {
			files:  map[string]string{"spindle.json": apitest.ToolboxManifest, "dist/frontend.js": "", "../escape.js": ""},
			reason: "safety_violation",
		},
		"invalid identifier": {
			files: map[string]string{
				"spindle.json":     strings.Replace(apitest.ToolboxManifest, "quiet_toolbox", "Quiet-Toolbox", 1),
				"dist/frontend.js": "",
			},
			reason: "malformed_input",
		},
		"manifest beside other files in a folder": {
			files: map[string]string{
				"README.md": "", "packages/quiet/spindle.json": apitest.ToolboxManifest, "packages/quiet/dist/frontend.js": "",
			},
			reason: "malformed_input",
		},
		"both manifests": {
			files: map[string]string{
				"spindle.json": apitest.ToolboxManifest, "dist/frontend.js": "",
				"manifest.json": `{"display_name":"Quiet","js":"index.js","author":"A developer"}`, "index.js": "",
			},
			reason: "malformed_input",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r, session, assets, _ := harness.NewExtensionRouter(t)
			metadata := apitest.ExampleMetadata("Quiet Toolbox")
			metadata["filename"] = "toolbox.zip"
			finished := apitest.UploadAndFinish(t, r, session, assets, metadata, apitest.ExtensionZip(t, tc.files))
			var operation struct {
				Status  string `json:"status"`
				Failure *struct {
					Reason string `json:"reason"`
				} `json:"failure"`
			}
			if err := json.Unmarshal(finished.Body.Bytes(), &operation); err != nil {
				t.Fatalf("decode the ingest: %v", err)
			}
			if operation.Status != "failed" || operation.Failure == nil || operation.Failure.Reason != tc.reason {
				t.Fatalf("ingest = %s, want failed with %s", finished.Body.String(), tc.reason)
			}
		})
	}
}

func TestAnExtensionBuiltIntoManyFilesIsAccepted(t *testing.T) {
	t.Parallel()
	r, session, assets, _ := harness.NewExtensionRouter(t)
	files := map[string]string{"spindle.json": apitest.ToolboxManifest, "dist/frontend.js": ""}
	for index := range 600 {
		files[fmt.Sprintf("dist/chunks/%d.js", index)] = ""
	}
	apitest.UploadExtension(t, r, session, assets, apitest.ExtensionZip(t, files))
}

func TestAnExtensionCannotBeStartedWithoutAnArchive(t *testing.T) {
	t.Parallel()
	r, session, _, _ := harness.NewExtensionRouter(t)
	request := httptest.NewRequest(http.MethodPost, "/v1/assets", strings.NewReader(`{"kind":"extension"}`))
	request.Header.Set("Content-Type", "application/json")
	if response := apitest.Send(t, r, apitest.Authorized(request, session)); response.Code != http.StatusBadRequest {
		t.Fatalf("start an extension from nothing = %d: %s", response.Code, response.Body.String())
	}
}

func readExtensionPage(t *testing.T, r http.Handler, session *http.Cookie, assetID string) extensionPage {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/assets/"+assetID, nil)
	if session != nil {
		request = apitest.Authorized(request, session)
	}
	response := apitest.Send(t, r, request)
	if response.Code != http.StatusOK {
		t.Fatalf("read the extension = %d: %s", response.Code, response.Body.String())
	}
	var page extensionPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode the extension: %v", err)
	}
	return page
}

func browsedIDs(t *testing.T, r http.Handler, path string) string {
	t.Helper()
	response := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("browse %s = %d: %s", path, response.Code, response.Body.String())
	}
	return response.Body.String()
}

func assertSameBytes(t *testing.T, what string, got, want []byte) {
	t.Helper()
	if sha256.Sum256(got) != sha256.Sum256(want) {
		t.Fatalf("%s has %d bytes that differ from the %d uploaded", what, len(got), len(want))
	}
}
