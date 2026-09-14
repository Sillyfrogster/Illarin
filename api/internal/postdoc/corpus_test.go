package postdoc_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/postdoc"
)

type corpusCase struct {
	Note      string          `json:"note"`
	Path      string          `json:"path"`
	Document  json.RawMessage `json:"document"`
	Canonical json.RawMessage `json:"canonical"`
}

func TestEveryValidCorpusDocumentReadsBackUnchanged(t *testing.T) {
	t.Parallel()
	for name, one := range corpus(t, "valid") {
		t.Run(name, func(t *testing.T) {
			read, err := postdoc.Read(one.Document)
			if err != nil {
				t.Fatalf("read %s: %v", one.Note, err)
			}
			written, err := json.Marshal(read)
			if err != nil {
				t.Fatalf("write %s: %v", one.Note, err)
			}
			want := one.Canonical
			if want == nil {
				want = one.Document
			}
			if compact(t, written) != compact(t, want) {
				t.Errorf("canonical form = %s, want %s", written, compact(t, want))
			}
			again, err := postdoc.Read(written)
			if err != nil {
				t.Fatalf("read the canonical form back: %v", err)
			}
			round, err := json.Marshal(again)
			if err != nil {
				t.Fatalf("write the canonical form again: %v", err)
			}
			if string(round) != string(written) {
				t.Errorf("second pass = %s, want %s", round, written)
			}
		})
	}
}

func TestEveryInvalidCorpusDocumentIsRefusedWhereItSaysSo(t *testing.T) {
	t.Parallel()
	for name, one := range corpus(t, "invalid") {
		t.Run(name, func(t *testing.T) {
			_, err := postdoc.Read(one.Document)
			if err == nil {
				t.Fatalf("%s was accepted", one.Note)
			}
			problem, ok := err.(postdoc.Problem)
			if !ok {
				t.Fatalf("refusal is %T, want a postdoc.Problem", err)
			}
			if problem.Path != one.Path {
				t.Errorf("refused at %q, want %q (%s)", problem.Path, one.Path, problem.Message)
			}
			if problem.Message == "" {
				t.Error("the refusal says nothing about what to fix")
			}
		})
	}
}

func TestAnEmptyBodyIsReadableButNotPublishable(t *testing.T) {
	t.Parallel()
	empty, err := postdoc.Read([]byte(`{"version":1,"content":[]}`))
	if err != nil {
		t.Fatalf("read an empty body: %v", err)
	}
	if !empty.Empty() {
		t.Error("an empty body does not report itself empty")
	}
	written, err := postdoc.Read([]byte(
		`{"version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Words."}]}]}`,
	))
	if err != nil {
		t.Fatalf("read a written body: %v", err)
	}
	if written.Empty() {
		t.Error("a body with words reports itself empty")
	}
}

func corpus(t *testing.T, group string) map[string]corpusCase {
	t.Helper()
	root := filepath.Join("testdata", "corpus", group)
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read the %s corpus: %v", group, err)
	}
	found := make(map[string]corpusCase, len(entries))
	for _, entry := range entries {
		raw, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		var one corpusCase
		if err := json.Unmarshal(raw, &one); err != nil {
			t.Fatalf("decode %s: %v", entry.Name(), err)
		}
		found[entry.Name()] = one
	}
	if len(found) == 0 {
		t.Fatalf("the %s corpus is empty", group)
	}
	return found
}

func compact(t *testing.T, raw []byte) string {
	t.Helper()
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("compact: %v", err)
	}
	out, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("compact: %v", err)
	}
	return string(out)
}
