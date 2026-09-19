package extension

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
)

const tavernManifest = `{
	"display_name": "Custom Sliders",
	"loading_order": 1,
	"requires": [],
	"optional": [],
	"dependencies": ["third-party/SillyTavern-LALib", "vectors", "third-party/SillyTavern-LALib"],
	"js": "dist/index.js",
	"css": "",
	"author": "A developer",
	"version": "1.0.0",
	"description": "Sliders for any request parameter.",
	"minimum_client_version": "1.12.0",
	"homePage": "https://github.com/example/Extension-CustomSliders.git"
}`

func TestSillyTavernDeclaresAWriterThatKeepsTheUpload(t *testing.T) {
	t.Parallel()
	declaration := SillyTavern{}.Declaration()
	if declaration.ID != SillyTavernID || declaration.Type != Type || !declaration.KeepsUpload {
		t.Fatalf("declaration = %+v, want an extension writer that keeps the upload", declaration)
	}
	if err := format.ValidateDeclaration(declaration); err != nil {
		t.Fatalf("declaration is incomplete: %v", err)
	}
}

func TestSillyTavernReadsTheManifestIntoTheHeaderAndElementsFromTheFile(t *testing.T) {
	t.Parallel()
	parsed := parseTavern(t, spindleZip(t, map[string]string{
		"manifest.json": tavernManifest, "dist/index.js": "export {}",
	}))

	if parsed.Type != Type || parsed.Format != SillyTavernID {
		t.Fatalf("parsed type and format = %q %q", parsed.Type, parsed.Format)
	}
	want := format.Header{
		Name: "Custom Sliders", WorkVersion: "1.0.0", CreditedAuthor: "A developer",
		Blurb: "Sliders for any request parameter.", Identifier: "Extension-CustomSliders",
	}
	if parsed.Header != want {
		t.Errorf("header = %+v, want %+v", parsed.Header, want)
	}

	dependencies := textsOf(t, parsed, block.RoleExtensionDependencies)
	if len(dependencies) != 2 ||
		dependencies[0].Text != "third-party/SillyTavern-LALib" || dependencies[1].Text != "vectors" {
		t.Fatalf("dependencies = %+v, want each name once in manifest order", dependencies)
	}
	if dependencies[0].Name != "SillyTavern-LALib" || dependencies[1].Name != "" {
		t.Errorf("dependencies refer to %q and %q, want the third-party folder and nothing for the built-in",
			dependencies[0].Name, dependencies[1].Name)
	}
	details := elementOf(t, parsed, block.RoleExtensionDetails).Content.(block.FieldList)
	if len(details.Fields) != 2 ||
		details.Fields[0].Name != "Version" || details.Fields[0].Value != "1.0.0" ||
		details.Fields[1].Name != "Minimum SillyTavern version" || details.Fields[1].Value != "1.12.0" {
		t.Errorf("details = %+v", details.Fields)
	}
	links := elementOf(t, parsed, block.RoleExtensionLinks).Content.(block.LinkList)
	if len(links.Links) != 1 || links.Links[0].Label != "Repository" ||
		links.Links[0].URL != "https://github.com/example/Extension-CustomSliders.git" {
		t.Errorf("links = %+v", links.Links)
	}
	for _, element := range parsed.Elements {
		if element.Role == block.RoleExtensionGrantedPermissions || element.Role == block.RoleExtensionApprovedPermissions {
			t.Errorf("a SillyTavern extension has a %s element, and SillyTavern asks for no permissions", element.Role)
		}
	}
}

func TestSillyTavernLeavesOutWhatTheManifestDoesNotSay(t *testing.T) {
	t.Parallel()
	parsed := parseTavern(t, spindleZip(t, map[string]string{
		"manifest.json": `{"display_name":"Bare","js":"index.js","author":"A developer"}`,
		"index.js":      "",
	}))
	if parsed.Header.Identifier != "" || parsed.Header.WorkVersion != "" {
		t.Errorf("header = %+v, want no identifier or version", parsed.Header)
	}
	roles := map[block.Role]bool{}
	for _, element := range parsed.Elements {
		roles[element.Role] = true
	}
	if roles[block.RoleExtensionDependencies] || roles[block.RoleExtensionLinks] {
		t.Errorf("elements = %+v, want no dependencies or links", parsed.Elements)
	}
	details := elementOf(t, parsed, block.RoleExtensionDetails).Content.(block.FieldList)
	if details.Empty() {
		t.Error("the details are empty, so a bare extension could never be published")
	}
}

func TestSillyTavernNamesTheFolderAfterTheRepository(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		`"homePage": "https://github.com/example/SillyTavern-LALib"`:  "SillyTavern-LALib",
		`"homepage": "https://github.com/example/SillyTavern-LALib/"`: "SillyTavern-LALib",
		`"homePage": "not a link"`:                                    "",
		`"homePage": "https://example.com"`:                           "",
	}
	for field, want := range cases {
		manifest := `{"display_name":"Lib","js":"index.js","author":"A developer",` + field + `}`
		parsed := parseTavern(t, spindleZip(t, map[string]string{"manifest.json": manifest, "index.js": ""}))
		if parsed.Header.Identifier != want {
			t.Errorf("%s gives identifier %q, want %q", field, parsed.Header.Identifier, want)
		}
	}
}

func TestSillyTavernRefusesAnArchiveSillyTavernWouldNotLoad(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		files   map[string]string
		mention string
	}{
		{
			name:    "missing display name",
			files:   withTavernManifest(`"display_name": "Custom Sliders",`, ""),
			mention: "display_name",
		},
		{
			name:    "missing author",
			files:   withTavernManifest(`"author": "A developer",`, ""),
			mention: "author",
		},
		{
			name:    "missing js",
			files:   withTavernManifest(`"js": "dist/index.js",`, ""),
			mention: "js",
		},
		{
			name:    "script not in the archive",
			files:   map[string]string{"manifest.json": tavernManifest, "index.js": ""},
			mention: "dist/index.js",
		},
		{
			name: "script outside the archive",
			files: map[string]string{
				"manifest.json": strings.Replace(tavernManifest, "dist/index.js", "../index.js", 1),
				"dist/index.js": "",
			},
			mention: "js",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tryParseTavern(t, spindleZip(t, tc.files))
			reason, ok := format.FailureOf(err)
			if !ok || reason != format.FailureMalformedInput {
				t.Fatalf("error = %v, want a malformed input refusal", err)
			}
			if !strings.Contains(err.Error(), tc.mention) {
				t.Errorf("error %q does not name %q", err, tc.mention)
			}
		})
	}
}

func TestAnArchiveWithBothManifestsIsRefusedAsAmbiguous(t *testing.T) {
	t.Parallel()
	registry := format.NewRegistry()
	for _, module := range Modules() {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %s: %v", module.ID(), err)
		}
	}
	file := inspectZip(t, spindleZip(t, map[string]string{
		"spindle.json": sampleManifest, "dist/frontend.js": "",
		"manifest.json": tavernManifest, "dist/index.js": "",
	}))
	resolution, claimed, err := registry.Resolve(file)
	if err != nil || !claimed {
		t.Fatalf("resolve = %v %t, want one module to take the archive and refuse it", err, claimed)
	}
	_, err = resolution.Module.Parse(context.Background(), file, resolution.Claim)
	if reason, ok := format.FailureOf(err); !ok || reason != format.FailureMalformedInput ||
		!strings.Contains(err.Error(), "spindle.json") || !strings.Contains(err.Error(), "manifest.json") {
		t.Fatalf("parse = %v, want a refusal naming both manifests", err)
	}
}

func TestSillyTavernReadsAnArchiveWrappedInOneFolder(t *testing.T) {
	t.Parallel()
	parsed := parseTavern(t, spindleZip(t, map[string]string{
		"Extension-CustomSliders-main/manifest.json": tavernManifest,
		"Extension-CustomSliders-main/dist/index.js": "",
	}))
	if parsed.Header.Name != "Custom Sliders" {
		t.Fatalf("header = %+v, want the manifest inside the folder read", parsed.Header)
	}
}

func TestSillyTavernSaysWhereAMisplacedManifestIs(t *testing.T) {
	t.Parallel()
	_, err := tryParseTavern(t, spindleZip(t, map[string]string{
		"README.md": "# Sliders", "extension/manifest.json": tavernManifest, "extension/dist/index.js": "",
	}))
	if reason, ok := format.FailureOf(err); !ok || reason != format.FailureMalformedInput ||
		!strings.Contains(err.Error(), "extension/manifest.json") {
		t.Fatalf("error = %v, want a refusal that names where the manifest is", err)
	}
}

func TestSillyTavernWritesTheUploadedArchiveUnchanged(t *testing.T) {
	t.Parallel()
	upload := spindleZip(t, map[string]string{"manifest.json": tavernManifest, "dist/index.js": ""})
	written, err := SillyTavern{}.Write(context.Background(), format.ExportWork{Type: Type, Upload: upload})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if !bytes.Equal(written.Body, upload) || written.Extension != ".zip" {
		t.Fatalf("written = %d bytes %q, want the upload unchanged as a zip", len(written.Body), written.Extension)
	}
}

func withTavernManifest(old, replacement string) map[string]string {
	return map[string]string{
		"manifest.json": strings.Replace(tavernManifest, old, replacement, 1),
		"dist/index.js": "",
	}
}

func parseTavern(t *testing.T, data []byte) format.Parsed {
	t.Helper()
	parsed, err := tryParseTavern(t, data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return parsed
}

func tryParseTavern(t *testing.T, data []byte) (format.Parsed, error) {
	t.Helper()
	file := inspectZip(t, data)
	claim, ok := SillyTavern{}.Claim(file)
	if !ok {
		t.Fatal("the archive was not claimed")
	}
	return SillyTavern{}.Parse(context.Background(), file, claim)
}
