package postdoc_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/postdoc"
)

type importCase struct {
	Note     string          `json:"note"`
	Markdown string          `json:"markdown"`
	Document json.RawMessage `json:"document"`
	Warnings []expectedNote  `json:"warnings"`
	Refusals []expectedNote  `json:"refusals"`
}

type expectedNote struct {
	Line int    `json:"line"`
	Says string `json:"says"`
}

func TestEveryCarriedImportBecomesItsCanonicalDocument(t *testing.T) {
	t.Parallel()
	for name, one := range imports(t, "carried") {
		t.Run(name, func(t *testing.T) {
			document, warnings, err := postdoc.FromMarkdown(one.Markdown)
			if err != nil {
				t.Fatalf("convert %s: %v", one.Note, err)
			}
			if compact(t, document) != compact(t, one.Document) {
				t.Errorf("converted to %s, want %s", document, compact(t, one.Document))
			}
			checkNotes(t, warnings, one.Warnings)
			if _, err := postdoc.Read(document); err != nil {
				t.Errorf("the converted document does not validate: %v", err)
			}
		})
	}
}

func TestEveryRefusedImportNamesWhatStoppedIt(t *testing.T) {
	t.Parallel()
	for name, one := range imports(t, "refused") {
		t.Run(name, func(t *testing.T) {
			document, _, err := postdoc.FromMarkdown(one.Markdown)
			if err == nil {
				t.Fatalf("%s was accepted as %s", one.Note, document)
			}
			refused, ok := err.(postdoc.Refused)
			if !ok {
				t.Fatalf("refusal is %T, want a postdoc.Refused", err)
			}
			checkNotes(t, refused.Notes, one.Refusals)
			if document != nil {
				t.Error("a refused import still answered with a document")
			}
		})
	}
}

func TestAnImportKeepsNoMarkdownBesideTheDocument(t *testing.T) {
	t.Parallel()
	source := "## Release notes\n\nIllarin ships **bold** prose.\n"
	document, _, err := postdoc.FromMarkdown(source)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	for _, mark := range []string{"##", "**", "Markdown", "markdown"} {
		if strings.Contains(string(document), mark) {
			t.Errorf("the document still carries %q: %s", mark, document)
		}
	}
}

func TestAnImportRefusesMoreMarkdownThanAPostCanHold(t *testing.T) {
	t.Parallel()
	_, _, err := postdoc.FromMarkdown(strings.Repeat("word ", 200000))
	if err == nil {
		t.Fatal("an oversized import was accepted")
	}
}

func checkNotes(t *testing.T, said []postdoc.Note, want []expectedNote) {
	t.Helper()
	if len(said) != len(want) {
		t.Fatalf("said %v, want %d of them", said, len(want))
	}
	for at, one := range want {
		if said[at].Line != one.Line {
			t.Errorf("note %d is about line %d, want line %d (%s)",
				at, said[at].Line, one.Line, said[at].Message)
		}
		if !strings.Contains(said[at].Message, one.Says) {
			t.Errorf("note %d says %q, want it to mention %q", at, said[at].Message, one.Says)
		}
	}
}

func imports(t *testing.T, group string) map[string]importCase {
	t.Helper()
	root := filepath.Join("testdata", "markdown", group)
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read the %s imports: %v", group, err)
	}
	found := make(map[string]importCase, len(entries))
	for _, entry := range entries {
		raw, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		var one importCase
		if err := json.Unmarshal(raw, &one); err != nil {
			t.Fatalf("decode %s: %v", entry.Name(), err)
		}
		found[entry.Name()] = one
	}
	if len(found) == 0 {
		t.Fatalf("the %s imports are empty", group)
	}
	return found
}
