package extension

import (
	"strconv"
	"strings"
	"testing"
)

func TestTheReadmeBesideTheManifestIsReadForThePage(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		files  map[string]string
		text   string
		root   string
		folder string
	}{
		"at the top of the zip": {
			files: map[string]string{"spindle.json": sampleManifest, "dist/frontend.js": "", "README.md": "# Quiet"},
			text:  "# Quiet",
		},
		"named in any case": {
			files: map[string]string{"spindle.json": sampleManifest, "dist/frontend.js": "", "Readme.markdown": "# Any"},
			text:  "# Any",
		},
		"inside the folder a download wraps it in": {
			files: map[string]string{
				"quiet-main/spindle.json": sampleManifest, "quiet-main/dist/frontend.js": "",
				"quiet-main/README.md": "# Wrapped",
			},
			text: "# Wrapped", root: "quiet-main/",
		},
		"in .github, which GitHub shows ahead of the one at the top": {
			files: map[string]string{
				"quiet-main/spindle.json": sampleManifest, "quiet-main/dist/frontend.js": "",
				"quiet-main/README.md": "# Top", "quiet-main/.github/README.md": "# Shown",
			},
			text: "# Shown", root: "quiet-main/", folder: ".github/",
		},
		"in docs when there is none higher": {
			files: map[string]string{"spindle.json": sampleManifest, "dist/frontend.js": "", "docs/README.md": "# Docs"},
			text:  "# Docs", folder: "docs/",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			parsed := parseSpindle(t, spindleZip(t, tc.files))
			if parsed.Readme == nil || parsed.Readme.Text != tc.text ||
				parsed.Readme.Root != tc.root || parsed.Readme.Folder != tc.folder {
				t.Fatalf("readme = %+v, want %q in %q of %q", parsed.Readme, tc.text, tc.folder, tc.root)
			}
		})
	}
}

func TestNoReadmeIsReadWhereThereIsNoneToRead(t *testing.T) {
	t.Parallel()
	cases := map[string]map[string]string{
		"none":                 {},
		"deeper in a folder":   {"notes/README.md": "# Deeper"},
		"not text":             {"README.md": "\xff\xfe\x00"},
		"over a megabyte long": {"README.md": varied(maxReadmeBytes + 1)},
	}
	for name, extra := range cases {
		t.Run(name, func(t *testing.T) {
			files := map[string]string{"spindle.json": sampleManifest, "dist/frontend.js": ""}
			for entry, content := range extra {
				files[entry] = content
			}
			if parsed := parseSpindle(t, spindleZip(t, files)); parsed.Readme != nil {
				t.Fatalf("readme = %+v, want none", parsed.Readme)
			}
		})
	}
}

func TestSillyTavernReadsItsReadme(t *testing.T) {
	t.Parallel()
	parsed := parseTavern(t, spindleZip(t, map[string]string{
		"manifest.json": `{"display_name":"Dice","js":"index.js","author":"A developer"}`,
		"index.js":      "", "README.md": "# Dice\n\nRolls dice.",
	}))
	if parsed.Readme == nil || parsed.Readme.Text != "# Dice\n\nRolls dice." {
		t.Fatalf("readme = %+v", parsed.Readme)
	}
}

func varied(length int) string {
	var written strings.Builder
	for number := 1; written.Len() < length; number++ {
		written.WriteString(strconv.Itoa(number * 7919 % 100003))
		written.WriteByte(' ')
	}
	return written.String()[:length]
}
