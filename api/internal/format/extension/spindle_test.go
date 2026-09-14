package extension

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/probe"
	"github.com/google/uuid"
)

const sampleManifest = `{
	"version": "2.0.0",
	"name": "Quiet Toolbox",
	"identifier": "quiet_toolbox",
	"author": "A developer",
	"github": "https://github.com/example/quiet_toolbox",
	"homepage": "https://example.com/quiet-toolbox",
	"description": "Small tools for a calmer chat.",
	"permissions": ["ui_panels", "generation", "chats", "a_future_permission"],
	"entry_frontend": "dist/frontend.js",
	"minimum_lumiverse_version": "0.1.0"
}`

func TestSpindleDeclaresAWriterThatKeepsTheUpload(t *testing.T) {
	declaration := Spindle{}.Declaration()
	if declaration.ID != SpindleID || declaration.Kind != Kind ||
		!declaration.Direction.Read || !declaration.Direction.Write || !declaration.KeepsUpload {
		t.Fatalf("declaration = %+v, want a reading writer that keeps the upload", declaration)
	}
	if err := format.ValidateDeclaration(declaration); err != nil {
		t.Fatalf("declaration is incomplete: %v", err)
	}
}

func TestSpindleReadsTheManifestIntoTheHeaderAndLockedElements(t *testing.T) {
	parsed := parseSpindle(t, spindleZip(t, map[string]string{
		"spindle.json":     sampleManifest,
		"dist/frontend.js": "export default {}",
	}))

	if parsed.Kind != Kind || parsed.Format != SpindleID {
		t.Fatalf("parsed kind and format = %q %q", parsed.Kind, parsed.Format)
	}
	want := format.Header{
		Name: "Quiet Toolbox", AssetVersion: "2.0.0", CreditedAuthor: "A developer",
		Blurb: "Small tools for a calmer chat.", Identifier: "quiet_toolbox",
	}
	if parsed.Header != want {
		t.Errorf("header = %+v, want %+v", parsed.Header, want)
	}
	if len(parsed.Remainder) != 0 || len(parsed.Media) != 0 {
		t.Errorf("remainder %v and media %v, want neither because the archive keeps everything",
			parsed.Remainder, parsed.Media)
	}

	granted := textsOf(t, parsed, block.RoleExtensionGrantedPermissions)
	if len(granted) != 2 || granted[0].Name != "ui_panels" || granted[1].Name != "a_future_permission" {
		t.Fatalf("granted permissions = %+v, want ui_panels and the unknown one", granted)
	}
	if !strings.Contains(granted[0].Text, "panels") {
		t.Errorf("ui_panels reads %q, want plain words about panels", granted[0].Text)
	}
	if granted[1].Text == "" {
		t.Error("an unknown permission has no words, so the element would read as empty")
	}
	approved := textsOf(t, parsed, block.RoleExtensionApprovedPermissions)
	if len(approved) != 2 || approved[0].Name != "generation" || approved[1].Name != "chats" {
		t.Fatalf("approved permissions = %+v, want generation and chats", approved)
	}

	details := elementOf(t, parsed, block.RoleExtensionDetails).Content.(block.FieldList)
	if len(details.Fields) != 2 ||
		details.Fields[0].Name != "Version" || details.Fields[0].Value != "2.0.0" ||
		details.Fields[1].Name != "Minimum Lumiverse version" || details.Fields[1].Value != "0.1.0" {
		t.Errorf("details = %+v", details.Fields)
	}
	links := elementOf(t, parsed, block.RoleExtensionLinks).Content.(block.LinkList)
	if len(links.Links) != 2 ||
		links.Links[0].Label != "Repository" || links.Links[0].URL != "https://github.com/example/quiet_toolbox" ||
		links.Links[1].Label != "Homepage" || links.Links[1].URL != "https://example.com/quiet-toolbox" {
		t.Errorf("links = %+v", links.Links)
	}
}

func TestSpindleListsARepositoryOnceWhenTheHomepageIsTheSame(t *testing.T) {
	manifest := strings.Replace(sampleManifest,
		"https://example.com/quiet-toolbox", "https://github.com/example/quiet_toolbox", 1)
	parsed := parseSpindle(t, spindleZip(t, map[string]string{
		"spindle.json": manifest, "dist/frontend.js": "",
	}))
	links := elementOf(t, parsed, block.RoleExtensionLinks).Content.(block.LinkList)
	if len(links.Links) != 1 || links.Links[0].Label != "Repository" {
		t.Errorf("links = %+v, want the repository once", links.Links)
	}
}

func TestSpindleAcceptsSourceThatLumiverseBuildsAndNoReadme(t *testing.T) {
	manifest := strings.Replace(sampleManifest, `"entry_frontend": "dist/frontend.js",`, "", 1)
	parsed := parseSpindle(t, spindleZip(t, map[string]string{
		"spindle.json": manifest, "src/backend.ts": "export {}",
	}))
	if parsed.Header.Identifier != "quiet_toolbox" {
		t.Fatalf("parsed = %+v", parsed.Header)
	}
}

func TestSpindleRefusesAnArchiveLumiverseWouldNotInstall(t *testing.T) {
	cases := []struct {
		name    string
		files   map[string]string
		mention string
	}{
		{
			name:    "identifier outside the pattern",
			files:   withManifest(`"identifier": "quiet_toolbox"`, `"identifier": "Quiet-Toolbox"`),
			mention: "identifier",
		},
		{
			name:    "missing author",
			files:   withManifest(`"author": "A developer",`, ""),
			mention: "author",
		},
		{
			name:    "missing github",
			files:   withManifest(`"github": "https://github.com/example/quiet_toolbox",`, ""),
			mention: "github",
		},
		{
			name:    "missing homepage",
			files:   withManifest(`"homepage": "https://example.com/quiet-toolbox",`, ""),
			mention: "homepage",
		},
		{
			name: "permissions that are not a list",
			files: withManifest(`"permissions": ["ui_panels", "generation", "chats", "a_future_permission"]`,
				`"permissions": "generation"`),
			mention: "permissions",
		},
		{
			name:    "missing permissions",
			files:   withManifest(`"permissions": ["ui_panels", "generation", "chats", "a_future_permission"],`, ""),
			mention: "permissions",
		},
		{
			name:    "declared entry file absent with nothing to build it from",
			files:   map[string]string{"spindle.json": sampleManifest, "README.md": "# Quiet"},
			mention: "dist/frontend.js",
		},
		{
			name: "no code at all",
			files: map[string]string{
				"spindle.json": strings.Replace(sampleManifest, `"entry_frontend": "dist/frontend.js",`, "", 1),
			},
			mention: "src/backend.ts",
		},
		{
			name: "entry outside the archive",
			files: map[string]string{
				"spindle.json":   strings.Replace(sampleManifest, "dist/frontend.js", "../frontend.js", 1),
				"src/backend.ts": "",
			},
			mention: "entry_frontend",
		},
		{
			name: "both manifests",
			files: map[string]string{
				"spindle.json": sampleManifest, "dist/frontend.js": "",
				"manifest.json": `{"display_name":"Quiet","js":"index.js","author":"A developer"}`,
			},
			mention: "manifest.json",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tryParseSpindle(t, spindleZip(t, tc.files))
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

func TestSpindleRefusesAnArchiveOverThirtyTwoMegabytesWhole(t *testing.T) {
	var file bytes.Buffer
	archive := zip.NewWriter(&file)
	put(t, archive, "spindle.json", []byte(sampleManifest), zip.Deflate)
	put(t, archive, "dist/frontend.js", bytes.Repeat([]byte{'a'}, MaxArchiveBytes), zip.Store)
	if err := archive.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	_, err := tryParseSpindle(t, file.Bytes())
	if reason, ok := format.FailureOf(err); !ok || reason != format.FailureLimitExceeded {
		t.Fatalf("error = %v, want the archive refused over its limit", err)
	}
	if !strings.Contains(err.Error(), "an extension archive may be up to 32 MB") {
		t.Errorf("error = %q, want the limit in megabytes", err)
	}
}

func TestSpindleAcceptsAnArchiveOfManyFiles(t *testing.T) {
	files := map[string]string{"spindle.json": sampleManifest, "dist/frontend.js": ""}
	for index := range 600 {
		files[fmt.Sprintf("dist/chunks/%d.js", index)] = ""
	}
	parsed := parseSpindle(t, spindleZip(t, files))
	if parsed.Header.Identifier == "" {
		t.Fatalf("parsed = %+v, want the archive read", parsed.Header)
	}
}

func TestSpindleReadsAnArchiveWrappedInOneFolder(t *testing.T) {
	parsed := parseSpindle(t, spindleZip(t, map[string]string{
		"quiet_toolbox-main/":                        "",
		"quiet_toolbox-main/spindle.json":            sampleManifest,
		"quiet_toolbox-main/dist/frontend.js":        "",
		"__MACOSX/quiet_toolbox-main/._spindle.json": "",
	}))
	if parsed.Header.Identifier != "quiet_toolbox" {
		t.Fatalf("header = %+v, want the manifest inside the folder read", parsed.Header)
	}
}

func TestSpindleIgnoresAnotherAppsManifestDeeperInTheArchive(t *testing.T) {
	parsed := parseSpindle(t, spindleZip(t, map[string]string{
		"spindle.json": sampleManifest, "dist/frontend.js": "",
		"node_modules/some-package/manifest.json": `{"name":"not an extension"}`,
	}))
	if parsed.Format != SpindleID {
		t.Fatalf("format = %q, want the Spindle manifest at the top to decide", parsed.Format)
	}
}

func TestSpindleSaysWhereAMisplacedManifestIs(t *testing.T) {
	_, err := tryParseSpindle(t, spindleZip(t, map[string]string{
		"README.md": "# Quiet", "packages/quiet/spindle.json": sampleManifest, "packages/quiet/dist/frontend.js": "",
	}))
	if reason, ok := format.FailureOf(err); !ok || reason != format.FailureMalformedInput ||
		!strings.Contains(err.Error(), "packages/quiet/spindle.json") {
		t.Fatalf("error = %v, want a refusal that names where the manifest is", err)
	}
}

func TestSpindleWritesTheUploadedArchiveUnchanged(t *testing.T) {
	upload := spindleZip(t, map[string]string{"spindle.json": sampleManifest, "dist/frontend.js": ""})
	written, err := Spindle{}.Write(context.Background(), format.ExportAsset{
		Kind: Kind, Header: format.Header{Name: "A renamed listing"}, Upload: upload,
	})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if !bytes.Equal(written.Body, upload) || written.Extension != ".zip" ||
		written.MediaType != "application/zip" {
		t.Fatalf("written = %d bytes %q %q, want the upload unchanged as a zip",
			len(written.Body), written.MediaType, written.Extension)
	}
	if _, err := (Spindle{}).Write(context.Background(), format.ExportAsset{Kind: Kind}); err == nil {
		t.Fatal("writing with no upload succeeded")
	}
}

func withManifest(old, replacement string) map[string]string {
	return map[string]string{
		"spindle.json":     strings.Replace(sampleManifest, old, replacement, 1),
		"dist/frontend.js": "",
	}
}

func textsOf(t *testing.T, parsed format.Parsed, role block.Role) []block.TextItem {
	t.Helper()
	return elementOf(t, parsed, role).Content.(block.TextSet).Texts
}

func elementOf(t *testing.T, parsed format.Parsed, role block.Role) block.Element {
	t.Helper()
	for _, element := range parsed.Elements {
		if element.Role == role {
			return element
		}
	}
	t.Fatalf("no %s element in %+v", role, parsed.Elements)
	return block.Element{}
}

func parseSpindle(t *testing.T, data []byte) format.Parsed {
	t.Helper()
	parsed, err := tryParseSpindle(t, data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return parsed
}

func tryParseSpindle(t *testing.T, data []byte) (format.Parsed, error) {
	t.Helper()
	file := inspectZip(t, data)
	claim, ok := Spindle{}.Claim(file)
	if !ok {
		t.Fatal("the archive was not claimed")
	}
	return Spindle{}.Parse(context.Background(), file, claim)
}

func inspectZip(t *testing.T, data []byte) probe.Inspection {
	t.Helper()
	file, err := probe.Inspect(
		context.Background(), memoryStore{data: data}, uuid.New(), int64(len(data)), "extension.zip",
	)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	return file
}

func spindleZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var file bytes.Buffer
	archive := zip.NewWriter(&file)
	for name, content := range files {
		put(t, archive, name, []byte(content), zip.Deflate)
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	return file.Bytes()
}

func put(t *testing.T, archive *zip.Writer, name string, content []byte, method uint16) {
	t.Helper()
	entry, err := archive.CreateHeader(&zip.FileHeader{Name: name, Method: method})
	if err != nil {
		t.Fatalf("create %q: %v", name, err)
	}
	if _, err := entry.Write(content); err != nil {
		t.Fatalf("write %q: %v", name, err)
	}
}

type memoryStore struct{ data []byte }

func (store memoryStore) ReadRange(_ context.Context, _ uuid.UUID, offset, length int64) (io.ReadCloser, error) {
	if offset < 0 || offset+length > int64(len(store.data)) {
		return nil, errors.New("range outside the archive")
	}
	return io.NopCloser(bytes.NewReader(store.data[offset : offset+length])), nil
}
