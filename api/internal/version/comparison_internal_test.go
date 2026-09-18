package version

import (
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

func recordedBlock(id uuid.UUID, definition block.DefinitionID, elements ...block.Element) block.Block {
	return block.Block{
		ID: id, Definition: definition, Position: 0,
		Layout: block.Layout("single"), Width: block.Width("full"), Elements: elements,
	}
}

func recordedElement(role block.Role, workType block.Type, content block.Content) block.Element {
	return block.Element{ID: uuid.New(), Type: workType, Role: role, Content: content}
}

func recordedVersionOf(workType string, blocks ...block.Block) work.Snapshot {
	return work.Snapshot{Type: workType, Metadata: work.VersionMetadata{Name: "Recorded"}, Blocks: blocks}
}

func changesUnder(t *testing.T, groups []ChangeGroup, subject string) []Change {
	t.Helper()
	for _, group := range groups {
		if group.Subject == subject {
			return group.Changes
		}
	}
	t.Fatalf("no %s changes in %+v", subject, groups)
	return nil
}

func changeTypes(changes []Change) []ChangeType {
	types := make([]ChangeType, 0, len(changes))
	for _, change := range changes {
		types = append(types, change.Type)
	}
	return types
}

func subjectsOf(groups []ChangeGroup) []string {
	subjects := make([]string, 0, len(groups))
	for _, group := range groups {
		subjects = append(subjects, group.Subject)
	}
	return subjects
}

func number(value float64) *block.Value { return &block.Value{Number: &value} }

func TestComparisonGroupsEachTypesContentByWhatItMeans(t *testing.T) {
	t.Parallel()
	page := uuid.New()
	greeting := uuid.New()
	entry := uuid.New()
	setting := uuid.New()
	colour := uuid.New()
	record := uuid.New()
	for _, test := range []struct {
		name     string
		workType string
		earlier  block.Block
		later    block.Block
		subject  string
		want     []ChangeType
	}{
		{
			name: "a rewritten greeting", workType: "character", subject: "greetings",
			earlier: recordedBlock(page, block.CharacterCore, recordedElement(block.RoleGreetings, block.TypeTextSet,
				block.TextSet{Texts: []block.TextItem{{ID: greeting, Text: "Hello there"}}})),
			later: recordedBlock(page, block.CharacterCore, recordedElement(block.RoleGreetings, block.TypeTextSet,
				block.TextSet{Texts: []block.TextItem{{ID: greeting, Text: "Well met"}}})),
			want: []ChangeType{ChangeEdited},
		},
		{
			name: "a new lorebook entry", workType: "lorebook", subject: "lorebook_entries",
			earlier: recordedBlock(page, block.LorebookCore, recordedElement(block.RoleLorebookEntries, block.TypeEntryTable,
				block.EntryTable{Entries: []block.Entry{{ID: entry, Name: "Harbour", Text: "Ships dock here."}}})),
			later: recordedBlock(page, block.LorebookCore, recordedElement(block.RoleLorebookEntries, block.TypeEntryTable,
				block.EntryTable{Entries: []block.Entry{
					{ID: entry, Name: "Harbour", Text: "Ships dock here."},
					{ID: uuid.New(), Name: "Market", Text: "Traders meet here."},
				}})),
			want: []ChangeType{ChangeAdded},
		},
		{
			name: "a retuned sampler", workType: "preset", subject: "sampler_settings",
			earlier: recordedBlock(page, block.PresetCore, recordedElement(block.RoleSamplerSettings, block.TypeSettingGroup,
				block.SettingGroup{Settings: []block.Setting{{ID: setting, Name: "temperature", Value: number(0.8)}}})),
			later: recordedBlock(page, block.PresetCore, recordedElement(block.RoleSamplerSettings, block.TypeSettingGroup,
				block.SettingGroup{Settings: []block.Setting{{ID: setting, Name: "temperature", Value: number(1.1)}}})),
			want: []ChangeType{ChangeEdited},
		},
		{
			name: "a dropped colour", workType: "theme", subject: "theme_tokens",
			earlier: recordedBlock(page, block.ThemeCore, recordedElement(block.RoleThemeTokens, block.TypeColorSet,
				block.ColorSet{Modes: []block.ColorMode{{Name: "dark", Colors: []block.Color{
					{ID: colour, Name: "Background", Value: "#101010"},
				}}}})),
			later: recordedBlock(page, block.ThemeCore, recordedElement(block.RoleThemeTokens, block.TypeColorSet,
				block.ColorSet{Modes: []block.ColorMode{{Name: "dark"}}})),
			want: []ChangeType{ChangeRemoved},
		},
		{
			name: "a renamed pack item", workType: "pack", subject: "pack_items",
			earlier: recordedBlock(page, block.PackCore, recordedElement(block.RolePackItems, block.TypeRecordList,
				block.RecordList{Schema: block.LumiaRecordSchema, Records: []block.LumiaRecord{
					{ID: record, LumiaName: "Wren", LumiaDefinition: "A quiet scout."},
				}})),
			later: recordedBlock(page, block.PackCore, recordedElement(block.RolePackItems, block.TypeRecordList,
				block.RecordList{Schema: block.LumiaRecordSchema, Records: []block.LumiaRecord{
					{ID: record, LumiaName: "Wrenna", LumiaDefinition: "A quiet scout."},
				}})),
			want: []ChangeType{ChangeEdited},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			groups := compareVersions(
				recordedVersionOf(test.workType, test.earlier),
				recordedVersionOf(test.workType, test.later),
			)
			changes := changesUnder(t, groups, test.subject)
			if got := changeTypes(changes); len(got) != len(test.want) || got[0] != test.want[0] {
				t.Fatalf("changes = %v, want %v", got, test.want)
			}
		})
	}
}

func TestReorderedAndReimportedItemsAreNotChanges(t *testing.T) {
	t.Parallel()
	page := uuid.New()
	first, second := uuid.New(), uuid.New()
	greetings := func(texts ...block.TextItem) work.Snapshot {
		return recordedVersionOf("character",
			recordedBlock(page, block.CharacterCore, recordedElement(block.RoleGreetings, block.TypeTextSet,
				block.TextSet{Texts: texts})))
	}
	earlier := greetings(
		block.TextItem{ID: first, Text: "Hello there"},
		block.TextItem{ID: second, Text: "Well met"})
	reordered := greetings(
		block.TextItem{ID: second, Text: "Well met"},
		block.TextItem{ID: first, Text: "Hello there"})
	reimported := greetings(
		block.TextItem{ID: uuid.New(), Text: "Hello there"},
		block.TextItem{ID: uuid.New(), Text: "Well met"})
	for _, later := range []work.Snapshot{reordered, reimported} {
		if groups := compareVersions(earlier, later); len(groups) != 0 {
			t.Fatalf("groups = %+v, want none", groups)
		}
	}
}

func TestRenamesReadAsEditsOnlyWithIdentity(t *testing.T) {
	t.Parallel()
	page, kept := uuid.New(), uuid.New()
	earlier := recordedVersionOf("character",
		recordedBlock(page, block.CharacterCore, recordedElement("", block.TypeFieldList,
			block.FieldList{Fields: []block.FieldItem{{ID: kept, Name: "Height", Value: "Tall"}}})))
	renamed := recordedVersionOf("character",
		recordedBlock(page, block.CharacterCore, recordedElement("", block.TypeFieldList,
			block.FieldList{Fields: []block.FieldItem{{ID: kept, Name: "Stature", Value: "Tall"}}})))
	replaced := recordedVersionOf("character",
		recordedBlock(page, block.CharacterCore, recordedElement("", block.TypeFieldList,
			block.FieldList{Fields: []block.FieldItem{{ID: uuid.New(), Name: "Stature", Value: "Towering"}}})))

	edited := changesUnder(t, compareVersions(earlier, renamed), string(block.TypeFieldList))
	if len(edited) != 1 || edited[0].Type != ChangeEdited ||
		edited[0].PreviousName != "Height" || edited[0].Name != "Stature" {
		t.Fatalf("renamed with identity = %+v", edited)
	}
	guessed := changesUnder(t, compareVersions(earlier, replaced), string(block.TypeFieldList))
	if got := changeTypes(guessed); len(got) != 2 || got[0] != ChangeAdded || got[1] != ChangeRemoved {
		t.Fatalf("renamed without identity = %+v", guessed)
	}
}

func TestRemovedTextKeepsTheWordsItTookAway(t *testing.T) {
	t.Parallel()
	page := uuid.New()
	earlier := recordedVersionOf("character",
		recordedBlock(page, block.CharacterCore, recordedElement(block.RoleScenario, block.TypeProse,
			block.Prose{Text: "They meet at the harbour."})))
	later := recordedVersionOf("character",
		recordedBlock(page, block.CharacterCore, recordedElement(block.RoleScenario, block.TypeProse,
			block.Prose{Text: ""})))
	changes := changesUnder(t, compareVersions(earlier, later), string(block.RoleScenario))
	if len(changes) != 1 || changes[0].Type != ChangeRemoved ||
		changes[0].Before != "They meet at the harbour." || changes[0].After != "" {
		t.Fatalf("changes = %+v", changes)
	}
}

func TestChangedPicturesNameTheOldAndNewMedia(t *testing.T) {
	t.Parallel()
	page, image := uuid.New(), uuid.New()
	was, now := uuid.New(), uuid.New()
	earlier := recordedVersionOf("character",
		recordedBlock(page, block.CharacterCore, recordedElement(block.RoleGallery, block.TypeImageSet,
			block.ImageSet{Images: []block.ImageItem{{ID: image, MediaID: was}}})))
	later := recordedVersionOf("character",
		recordedBlock(page, block.CharacterCore, recordedElement(block.RoleGallery, block.TypeImageSet,
			block.ImageSet{Images: []block.ImageItem{{ID: image, MediaID: now}}})))
	changes := changesUnder(t, compareVersions(earlier, later), string(block.RoleGallery))
	if len(changes) != 1 || changes[0].BeforeMedia == nil || *changes[0].BeforeMedia != was ||
		changes[0].AfterMedia == nil || *changes[0].AfterMedia != now {
		t.Fatalf("changes = %+v", changes)
	}
}

func TestPreservedDataReportsItsNamespaceAndNothingElse(t *testing.T) {
	t.Parallel()
	owner := uuid.New()
	earlier := recordedVersionOf("character")
	earlier.Preserved = []work.VersionPreserved{
		{Owner: "asset", OwnerID: owner, Namespace: "test", Payload: `{"secret":"before"}`},
	}
	later := recordedVersionOf("character")
	later.Preserved = []work.VersionPreserved{
		{Owner: "asset", OwnerID: owner, Namespace: "test", Payload: `{"secret":"after"}`},
	}
	changes := changesUnder(t, compareVersions(earlier, later), PreservedSubject)
	if len(changes) != 1 || changes[0].Type != ChangeEdited || changes[0].Name != "test" {
		t.Fatalf("changes = %+v", changes)
	}
	if changes[0].Before != "" || changes[0].After != "" {
		t.Fatal("the comparison carried preserved payloads")
	}
}

func TestPresentationOnlyUpdateExplainsThePage(t *testing.T) {
	t.Parallel()
	page := uuid.New()
	core := recordedBlock(page, block.CharacterCore, recordedElement(block.RoleDescription, block.TypeProse,
		block.Prose{Text: "A quiet scout."}))
	moved := recordedBlock(uuid.New(), block.Lorebook)
	earlier := recordedVersionOf("character", core, moved)

	restyled := core
	restyled.Width = block.Width("half")
	later := recordedVersionOf("character", moved, restyled)

	groups := compareVersions(earlier, later)
	if got := subjectsOf(groups); len(got) != 1 || got[0] != PresentationSubject {
		t.Fatalf("subjects = %v, want only the page", got)
	}
	changes := changesUnder(t, groups, PresentationSubject)
	if len(changes) != 2 || changes[1].Name != "Page order" {
		t.Fatalf("changes = %+v", changes)
	}
	if changes[0].Name != "The character width" || changes[0].After != "half" {
		t.Fatalf("width change = %+v", changes[0])
	}
}
