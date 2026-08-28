package postdoc_test

import (
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/postdoc"
)

func TestNestingStopsBeforeItCanBeUsedToExhaustAReader(t *testing.T) {
	body := `{"type":"paragraph","content":[{"type":"text","text":"Deep"}]}`
	for range 12 {
		body = `{"type":"bulletList","content":[{"type":"listItem","content":[` + body + `]}]}`
	}
	_, err := postdoc.Read([]byte(`{"version":1,"content":[` + body + `]}`))
	if err == nil {
		t.Fatal("a document nested twelve lists deep was accepted")
	}
	if !strings.Contains(err.Error(), "deeply") {
		t.Errorf("refusal = %v, want one about depth", err)
	}
}

func TestADocumentWithTooManyBlocksIsRefused(t *testing.T) {
	paragraph := `{"type":"paragraph","content":[{"type":"text","text":"x"}]}`
	body := strings.Repeat(paragraph+",", 2000) + paragraph
	_, err := postdoc.Read([]byte(`{"version":1,"content":[` + body + `]}`))
	if err == nil {
		t.Fatal("a document of four thousand nodes was accepted")
	}
	if !strings.Contains(err.Error(), "too many blocks") {
		t.Errorf("refusal = %v, want one about size", err)
	}
}

func TestWhitespaceAloneDoesNotCountAsWriting(t *testing.T) {
	blank, err := postdoc.Read([]byte(
		`{"version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"   "}]}]}`,
	))
	if err != nil {
		t.Fatalf("read a blank paragraph: %v", err)
	}
	if !blank.Empty() {
		t.Error("a paragraph of spaces counts as a written body")
	}
}
