package lorebook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"slices"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/character"
	"github.com/google/uuid"
)

const twoEntries = `{
	"name": "Zenless lore",
	"description": "A compendium of events, factions and locations.",
	"scan_depth": 4,
	"token_budget": 500,
	"extensions": {"risuai": {"folders": []}},
	"entries": [
		{
			"name": "Story Timeline",
			"keys": ["Timeline", "2167"],
			"content": "It is widely believed the story takes place in 2167.",
			"enabled": true,
			"insertion_order": 100,
			"constant": true,
			"position": "after_char",
			"uid": "4d920bb9",
			"probability": 100
		},
		{
			"keys": ["New Eridu"],
			"content": "The last city.",
			"enabled": false,
			"insertion_order": 20,
			"case_sensitive": true
		}
	]
}`

func TestTheModuleReadsAndWritesTheLorebookType(t *testing.T) {
	t.Parallel()
	declaration := Module{}.Declaration()
	if declaration.Type != Type || declaration.ID != ID {
		t.Errorf("declaration identity = %q/%q, want %q/%q",
			declaration.ID, declaration.Type, ID, Type)
	}
	if !declaration.Direction.Read || !declaration.Direction.Write {
		t.Errorf("direction = %+v, want read and write", declaration.Direction)
	}
	if err := format.ValidateDeclaration(declaration); err != nil {
		t.Fatalf("declaration: %v", err)
	}
	if len(declaration.Recognition) != 1 ||
		declaration.Recognition[0].Type != format.RecognitionSignature {
		t.Errorf("recognition = %+v, want one structural signature", declaration.Recognition)
	}
	if len(declaration.Slots) != 0 || len(declaration.Boilerplate) != 0 {
		t.Errorf("declaration invented slots %v or boilerplate %v",
			declaration.Slots, declaration.Boilerplate)
	}
	if !slices.Contains(declaration.TestedOrigins, ID) ||
		!slices.Contains(declaration.TestedOrigins, format.OriginIllarin) {
		t.Errorf("tested origins = %v, want its own format and Illarin", declaration.TestedOrigins)
	}
	if slices.Contains(declaration.ConsumedKeys, "description") {
		t.Error("the module declared the book's description consumed")
	}
}

func TestTheSignatureDoesNotOverlapAnotherModules(t *testing.T) {
	t.Parallel()
	if err := testRegistry(t).ValidateDeclarations(); err != nil {
		t.Fatalf("declarations across every module: %v", err)
	}
}

func TestOnlyADocumentHoldingEntriesIsClaimed(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		body     string
		claimant string
	}{
		{name: "a book", body: `{"entries": []}`, claimant: ID},
		{name: "a book with fields around it", body: twoEntries, claimant: ID},
		{
			name:     "a card before any spec existed",
			body:     `{"name":"a","description":"b","personality":"c","scenario":"d","first_mes":"e"}`,
			claimant: "chara_card_v2",
		},
		{
			name:     "entries as an object",
			body:     `{"entries": {"0": {}}}`,
			claimant: SillyTavernID,
		},
		{name: "nothing recognisable", body: `{"colours": []}`, claimant: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolution, claimed, err := testRegistry(t).Resolve(document(t, test.body))
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			claimant := ""
			if claimed {
				claimant = resolution.Module.ID()
			}
			if claimant != test.claimant {
				t.Fatalf("claimed by %q, want %q", claimant, test.claimant)
			}
		})
	}
}

func TestReadingABookFillsTheEntryRoleAndKeepsTheRest(t *testing.T) {
	t.Parallel()
	parsed := parse(t, twoEntries)

	if parsed.Type != Type || parsed.Format != ID {
		t.Fatalf("parsed type %q format %q, want %q and %q", parsed.Type, parsed.Format, Type, ID)
	}
	if parsed.Header.Name != "Zenless lore" {
		t.Errorf("name = %q, want the book's own", parsed.Header.Name)
	}
	if parsed.Header.Blurb != "A compendium of events, factions and locations." {
		t.Errorf("blurb = %q, want the book's description", parsed.Header.Blurb)
	}
	table := onlyEntryTable(t, parsed.Elements)
	if len(table.Entries) != 2 {
		t.Fatalf("read %d entries, want 2", len(table.Entries))
	}
	first := table.Entries[0]
	if first.Name != "Story Timeline" || !first.Constant || !first.Enabled ||
		first.Order != 100 || first.Position != block.AfterCharacter ||
		!reflect.DeepEqual(first.Keys, []string{"Timeline", "2167"}) {
		t.Errorf("first entry = %+v, want the book's own values", first)
	}
	if second := table.Entries[1]; second.Enabled || !second.CaseSensitive {
		t.Errorf("second entry = %+v, want switched off and case sensitive", second)
	}

	body := preservedPayload(t, parsed.Remainder, format.OwnerWork, bookNamespace)
	for _, key := range []string{"description", "scan_depth", "token_budget"} {
		if _, held := body[key]; !held {
			t.Errorf("the book's %s was not preserved", key)
		}
	}
	if _, held := preserved(parsed.Remainder, format.OwnerWork, "risuai"); !held {
		t.Error("the extensions namespace was not preserved as its own namespace")
	}
	entry := preservedPayload(t, parsed.Remainder, format.OwnerItem, entryNamespace)
	if _, held := entry["uid"]; !held {
		t.Error("the entry's own identifier was not preserved beside it")
	}
}

func TestAMalformedFieldInOneEntryCostsThatFieldAlone(t *testing.T) {
	t.Parallel()
	parsed := parse(t, `{
		"entries": [
			{"keys": "Timeline", "content": "kept", "insertion_order": 100},
			{"keys": ["New Eridu"], "content": "whole", "insertion_order": 20},
			"not an entry at all"
		]
	}`)
	table := onlyEntryTable(t, parsed.Elements)
	if len(table.Entries) != 3 {
		t.Fatalf("read %d entries, want all 3 kept", len(table.Entries))
	}
	damaged := table.Entries[0]
	if len(damaged.Keys) != 0 {
		t.Errorf("keys = %v, want the unreadable value left out of the entry", damaged.Keys)
	}
	if damaged.Text != "kept" || damaged.Order != 100 {
		t.Errorf("entry = %+v, want the fields beside the bad one read", damaged)
	}
	whole := table.Entries[1]
	if whole.Text != "whole" || !reflect.DeepEqual(whole.Keys, []string{"New Eridu"}) {
		t.Errorf("the next entry = %+v, want it untouched", whole)
	}

	kept := make(map[uuid.UUID]map[string]json.RawMessage)
	for _, row := range parsed.Remainder {
		if row.Owner != format.OwnerItem {
			continue
		}
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(row.Payload, &fields)
		kept[row.OwnerID] = fields
	}
	if _, held := kept[damaged.ID]["keys"]; !held {
		t.Error("the unreadable keys value was dropped rather than preserved")
	}
	if _, held := kept[whole.ID]; held {
		t.Error("the whole entry was given preserved data it did not need")
	}
	if _, held := kept[table.Entries[2].ID]; !held {
		t.Error("the entry that was not an object was dropped rather than preserved")
	}
}

func TestEntriesThatAreNotAListRefuseTheImport(t *testing.T) {
	t.Parallel()
	file := document(t, `{"entries": {"0": {"content": "x"}}}`)
	_, err := Module{}.Parse(context.Background(), file, format.CompatibilityClaim(file.Payloads[0]))
	if reason, classified := format.FailureOf(err); !classified ||
		reason != format.FailureMalformedInput {
		t.Fatalf("parse error = %v, want a malformed input refusal", err)
	}
}

func TestABookWrittenBackCarriesItsContentAndEverythingPreserved(t *testing.T) {
	t.Parallel()
	parsed := parse(t, twoEntries)
	written, err := Module{}.Write(context.Background(), format.ExportWork{
		Type: Type, Header: parsed.Header, Elements: parsed.Elements,
		Preserved: parsed.Remainder,
	})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if written.MediaType != "application/json" || written.Extension != ".json" {
		t.Errorf("artifact = %s%s, want a JSON document", written.MediaType, written.Extension)
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(written.Body, &body); err != nil {
		t.Fatalf("read the written book: %v", err)
	}
	if string(body["name"]) != `"Zenless lore"` {
		t.Errorf("written name = %s, want the work's own", body["name"])
	}
	for _, key := range []string{"description", "scan_depth", "token_budget", "extensions"} {
		if _, held := body[key]; !held {
			t.Errorf("%s did not travel back into the file", key)
		}
	}
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(body["entries"], &entries); err != nil {
		t.Fatalf("read the written entries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("wrote %d entries, want 2", len(entries))
	}
	if string(entries[0]["uid"]) != `"4d920bb9"` ||
		string(entries[0]["probability"]) != "100" {
		t.Errorf("entry = %v, want its preserved keys back on it", entries[0])
	}
	if string(entries[0]["position"]) != `"after_char"` {
		t.Errorf("position = %s, want the wording a book uses", entries[0]["position"])
	}
	if string(entries[1]["enabled"]) != "false" {
		t.Errorf("second entry enabled = %s, want the creator's switch", entries[1]["enabled"])
	}

	again := parse(t, string(written.Body))
	before, after := onlyEntryTable(t, parsed.Elements), onlyEntryTable(t, again.Elements)
	for i := range before.Entries {
		before.Entries[i].ID = uuid.Nil
		after.Entries[i].ID = uuid.Nil
	}
	if !reflect.DeepEqual(before, after) {
		t.Errorf("round trip changed the book:\n%+v\n%+v", before, after)
	}
}

func TestTheLossReportNamesWhatALorebookFileCannotCarry(t *testing.T) {
	t.Parallel()
	targets := testRegistry(t).OfferedTargets(format.CapabilitySubject{
		Type: Type, Origin: ID,
		Elements: []block.Element{
			{
				ID: uuid.New(), Type: block.TypeEntryTable, Role: block.RoleLorebookEntries,
				Content: block.EntryTable{Entries: []block.Entry{{Text: "kept"}}},
			},
			{
				ID: uuid.New(), Type: block.TypeImageSet, Role: block.RoleGallery,
				Content: block.ImageSet{Images: []block.ImageItem{{MediaID: uuid.New()}}},
			},
		},
	})
	if len(targets) != 1 || targets[0].Format != ID || !targets[0].Recommended {
		t.Fatalf("offered %+v, want the lorebook format alone and recommended", targets)
	}
	verdicts := make(map[block.Role]format.Verdict)
	for _, loss := range targets[0].Roles {
		verdicts[loss.Role] = loss.Verdict
	}
	if verdicts[block.RoleLorebookEntries] != format.Carried {
		t.Errorf("entries verdict = %q, want carried", verdicts[block.RoleLorebookEntries])
	}
	if verdicts[block.RoleGallery] != format.Dropped {
		t.Errorf("gallery verdict = %q, want dropped", verdicts[block.RoleGallery])
	}
}

func TestNoCardWriterIsOfferedForABook(t *testing.T) {
	t.Parallel()
	targets := testRegistry(t).OfferedTargets(format.CapabilitySubject{
		Type: "character", Origin: ID,
	})
	if len(targets) != 0 {
		t.Fatalf("offered %+v for a lorebook origin under the character type, want none", targets)
	}
}

func TestAnImportedBookIsPlacedIntoTheLorebookCatalog(t *testing.T) {
	t.Parallel()
	parsed := parse(t, twoEntries)
	blocks, err := block.Place(parsed.Type, parsed.Elements)
	if err != nil {
		t.Fatalf("place: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("placed %d blocks, want the one block a lorebook has", len(blocks))
	}
	placed := blocks[0]
	if placed.Definition != block.LorebookCore || placed.Layout != block.Single ||
		placed.Width != block.Full {
		t.Fatalf("block = %+v, want a full width single-slot lorebook core", placed)
	}
	if len(placed.Elements) != 1 || placed.Elements[0].Role != block.RoleLorebookEntries {
		t.Fatalf("elements = %+v, want the entry table alone", placed.Elements)
	}
	if !placed.Pinned(block.RoleLorebookEntries, Type) {
		t.Error("the entries can be taken off the page a lorebook is")
	}
}

func testRegistry(t *testing.T) *format.Registry {
	t.Helper()
	registry := format.NewRegistry()
	for _, module := range slices.Concat(character.Modules(), Modules()) {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %q: %v", module.ID(), err)
		}
	}
	return registry
}

func parse(t *testing.T, body string) format.Parsed {
	t.Helper()
	file := document(t, body)
	resolution, claimed, err := testRegistry(t).Resolve(file)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !claimed {
		t.Fatal("no module claimed the book")
	}
	parsed, err := resolution.Module.Parse(context.Background(), file, resolution.Claim)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return parsed
}

func onlyEntryTable(t *testing.T, elements []block.Element) block.EntryTable {
	t.Helper()
	for _, element := range elements {
		if element.Role != block.RoleLorebookEntries {
			continue
		}
		table, ok := element.Content.(block.EntryTable)
		if !ok {
			t.Fatalf("the entries element holds %T", element.Content)
		}
		return table
	}
	t.Fatalf("no entries element in %+v", elements)
	return block.EntryTable{}
}

func preserved(
	rows []format.Remainder,
	owner format.Owner,
	namespace string,
) (format.Remainder, bool) {
	for _, row := range rows {
		if row.Owner == owner && row.Namespace == namespace {
			return row, true
		}
	}
	return format.Remainder{}, false
}

func preservedPayload(
	t *testing.T,
	rows []format.Remainder,
	owner format.Owner,
	namespace string,
) map[string]json.RawMessage {
	t.Helper()
	row, held := preserved(rows, owner, namespace)
	if !held {
		t.Fatalf("nothing preserved for %s %s", owner, namespace)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(row.Payload, &payload); err != nil {
		t.Fatalf("read the %s payload: %v", namespace, err)
	}
	return payload
}

func document(t *testing.T, body string) format.Inspection {
	t.Helper()
	data := []byte(body)
	file, err := format.Inspect(
		context.Background(), memoryStore{data: data}, uuid.New(), int64(len(data)), "book.json",
	)
	if err != nil {
		t.Fatalf("inspect the document: %v", err)
	}
	return file
}

type memoryStore struct{ data []byte }

func (s memoryStore) ReadRange(
	_ context.Context,
	_ uuid.UUID,
	offset, length int64,
) (io.ReadCloser, error) {
	if offset < 0 || offset+length > int64(len(s.data)) {
		return nil, errors.New("range outside the blob")
	}
	return io.NopCloser(bytes.NewReader(s.data[offset : offset+length])), nil
}
