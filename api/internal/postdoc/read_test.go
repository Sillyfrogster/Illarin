package postdoc_test

import (
	"fmt"
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

func TestATableIsBoundedInBothDirections(t *testing.T) {
	cells := func(count int) string {
		one := `{"type":"tableCell","content":[{"type":"paragraph","content":[]}]}`
		return `{"type":"tableRow","content":[` + strings.Repeat(one+",", count-1) + one + `]}`
	}
	wide := `{"version":2,"content":[{"type":"table","content":[` + cells(11) + `]}]}`
	if _, err := postdoc.Read([]byte(wide)); err == nil {
		t.Error("a table eleven columns wide was accepted")
	}
	tall := `{"version":2,"content":[{"type":"table","content":[` +
		strings.Repeat(cells(1)+",", 60) + cells(1) + `]}]}`
	if _, err := postdoc.Read([]byte(tall)); err == nil {
		t.Error("a table sixty-one rows deep was accepted")
	}
}

func TestCodeCannotCarryAControlCharacter(t *testing.T) {
	_, err := postdoc.Read([]byte(
		`{"version":2,"content":[{"type":"codeBlock","language":"go","source":"one\u0007two"}]}`,
	))
	if err == nil {
		t.Fatal("code carrying a bell character was accepted")
	}
	if !strings.Contains(err.Error(), "control character") {
		t.Errorf("refusal = %v, want one about a control character", err)
	}
}

func TestAChosenHeadingAddressCannotTakeAWrittenOne(t *testing.T) {
	_, err := postdoc.Read([]byte(`{"version":2,"content":[
		{"type":"heading","level":2,"content":[{"type":"text","text":"Notes"}]},
		{"type":"heading","level":3,"anchor":"notes","content":[{"type":"text","text":"Later"}]}
	]}`))
	if err == nil {
		t.Fatal("a second heading took an address already in use")
	}
	if !strings.Contains(err.Error(), "already answers") {
		t.Errorf("refusal = %v, want one about the address being taken", err)
	}
}

func TestEveryVocabularySetIsClosed(t *testing.T) {
	for _, name := range postdoc.Languages {
		body := fmt.Sprintf(
			`{"version":2,"content":[{"type":"codeBlock","language":%q,"source":"x"}]}`, name)
		if _, err := postdoc.Read([]byte(body)); err != nil {
			t.Errorf("language %q is listed but refused: %v", name, err)
		}
	}
	for _, kind := range postdoc.CalloutKinds {
		body := fmt.Sprintf(
			`{"version":2,"content":[{"type":"callout","kind":%q,"content":[`+
				`{"type":"paragraph","content":[{"type":"text","text":"x"}]}]}]}`, kind)
		if _, err := postdoc.Read([]byte(body)); err != nil {
			t.Errorf("callout kind %q is listed but refused: %v", kind, err)
		}
	}
}
