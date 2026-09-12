package asset

import (
	"context"
	"errors"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
)

func recordedBlock(id uuid.UUID, definition block.DefinitionID, elements ...block.Element) block.Block {
	return block.Block{
		ID: id, Definition: definition, Position: 0,
		Layout: block.Layout("single"), Width: block.Width("full"), Elements: elements,
	}
}

func recordedElement(role block.Role, kind block.Type, content block.Content) block.Element {
	return block.Element{ID: uuid.New(), Type: kind, Role: role, Content: content}
}

func recordedVersionOf(kind string, blocks ...block.Block) recordedVersion {
	return recordedVersion{kind: kind, metadata: versionMetadata{Name: "Recorded"}, blocks: blocks}
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

func changeKinds(changes []Change) []ChangeKind {
	kinds := make([]ChangeKind, 0, len(changes))
	for _, change := range changes {
		kinds = append(kinds, change.Kind)
	}
	return kinds
}

func subjectsOf(groups []ChangeGroup) []string {
	subjects := make([]string, 0, len(groups))
	for _, group := range groups {
		subjects = append(subjects, group.Subject)
	}
	return subjects
}

func number(value float64) *block.Value { return &block.Value{Number: &value} }

func TestComparisonGroupsEachKindsContentByWhatItMeans(t *testing.T) {
	page := uuid.New()
	greeting := uuid.New()
	entry := uuid.New()
	setting := uuid.New()
	colour := uuid.New()
	record := uuid.New()
	for _, test := range []struct {
		name    string
		kind    string
		earlier block.Block
		later   block.Block
		subject string
		want    []ChangeKind
	}{
		{
			name: "a rewritten greeting", kind: "character", subject: "greetings",
			earlier: recordedBlock(page, block.CharacterCore, recordedElement(block.RoleGreetings, block.TypeTextSet,
				block.TextSet{Texts: []block.TextItem{{ID: greeting, Text: "Hello there"}}})),
			later: recordedBlock(page, block.CharacterCore, recordedElement(block.RoleGreetings, block.TypeTextSet,
				block.TextSet{Texts: []block.TextItem{{ID: greeting, Text: "Well met"}}})),
			want: []ChangeKind{ChangeEdited},
		},
		{
			name: "a new lorebook entry", kind: "lorebook", subject: "lorebook_entries",
			earlier: recordedBlock(page, block.LorebookCore, recordedElement(block.RoleLorebookEntries, block.TypeEntryTable,
				block.EntryTable{Entries: []block.Entry{{ID: entry, Name: "Harbour", Text: "Ships dock here."}}})),
			later: recordedBlock(page, block.LorebookCore, recordedElement(block.RoleLorebookEntries, block.TypeEntryTable,
				block.EntryTable{Entries: []block.Entry{
					{ID: entry, Name: "Harbour", Text: "Ships dock here."},
					{ID: uuid.New(), Name: "Market", Text: "Traders meet here."},
				}})),
			want: []ChangeKind{ChangeAdded},
		},
		{
			name: "a retuned sampler", kind: "preset", subject: "sampler_settings",
			earlier: recordedBlock(page, block.PresetCore, recordedElement(block.RoleSamplerSettings, block.TypeSettingGroup,
				block.SettingGroup{Settings: []block.Setting{{ID: setting, Name: "temperature", Value: number(0.8)}}})),
			later: recordedBlock(page, block.PresetCore, recordedElement(block.RoleSamplerSettings, block.TypeSettingGroup,
				block.SettingGroup{Settings: []block.Setting{{ID: setting, Name: "temperature", Value: number(1.1)}}})),
			want: []ChangeKind{ChangeEdited},
		},
		{
			name: "a dropped colour", kind: "theme", subject: "theme_tokens",
			earlier: recordedBlock(page, block.ThemeCore, recordedElement(block.RoleThemeTokens, block.TypeColorSet,
				block.ColorSet{Modes: []block.ColorMode{{Name: "dark", Colors: []block.Color{
					{ID: colour, Name: "Background", Value: "#101010"},
				}}}})),
			later: recordedBlock(page, block.ThemeCore, recordedElement(block.RoleThemeTokens, block.TypeColorSet,
				block.ColorSet{Modes: []block.ColorMode{{Name: "dark"}}})),
			want: []ChangeKind{ChangeRemoved},
		},
		{
			name: "a renamed pack item", kind: "pack", subject: "pack_items",
			earlier: recordedBlock(page, block.PackCore, recordedElement(block.RolePackItems, block.TypeRecordList,
				block.RecordList{Schema: block.LumiaRecordSchema, Records: []block.LumiaRecord{
					{ID: record, LumiaName: "Wren", LumiaDefinition: "A quiet scout."},
				}})),
			later: recordedBlock(page, block.PackCore, recordedElement(block.RolePackItems, block.TypeRecordList,
				block.RecordList{Schema: block.LumiaRecordSchema, Records: []block.LumiaRecord{
					{ID: record, LumiaName: "Wrenna", LumiaDefinition: "A quiet scout."},
				}})),
			want: []ChangeKind{ChangeEdited},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			groups := compareVersions(
				recordedVersionOf(test.kind, test.earlier),
				recordedVersionOf(test.kind, test.later),
			)
			changes := changesUnder(t, groups, test.subject)
			if got := changeKinds(changes); len(got) != len(test.want) || got[0] != test.want[0] {
				t.Fatalf("changes = %v, want %v", got, test.want)
			}
		})
	}
}

func TestReorderedAndReimportedItemsAreNotChanges(t *testing.T) {
	page := uuid.New()
	first, second := uuid.New(), uuid.New()
	greetings := func(texts ...block.TextItem) recordedVersion {
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
	for _, later := range []recordedVersion{reordered, reimported} {
		if groups := compareVersions(earlier, later); len(groups) != 0 {
			t.Fatalf("groups = %+v, want none", groups)
		}
	}
}

func TestRenamesReadAsEditsOnlyWithIdentity(t *testing.T) {
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
	if len(edited) != 1 || edited[0].Kind != ChangeEdited ||
		edited[0].PreviousName != "Height" || edited[0].Name != "Stature" {
		t.Fatalf("renamed with identity = %+v", edited)
	}
	guessed := changesUnder(t, compareVersions(earlier, replaced), string(block.TypeFieldList))
	if got := changeKinds(guessed); len(got) != 2 || got[0] != ChangeAdded || got[1] != ChangeRemoved {
		t.Fatalf("renamed without identity = %+v", guessed)
	}
}

func TestRemovedTextKeepsTheWordsItTookAway(t *testing.T) {
	page := uuid.New()
	earlier := recordedVersionOf("character",
		recordedBlock(page, block.CharacterCore, recordedElement(block.RoleScenario, block.TypeProse,
			block.Prose{Text: "They meet at the harbour."})))
	later := recordedVersionOf("character",
		recordedBlock(page, block.CharacterCore, recordedElement(block.RoleScenario, block.TypeProse,
			block.Prose{Text: ""})))
	changes := changesUnder(t, compareVersions(earlier, later), string(block.RoleScenario))
	if len(changes) != 1 || changes[0].Kind != ChangeRemoved ||
		changes[0].Before != "They meet at the harbour." || changes[0].After != "" {
		t.Fatalf("changes = %+v", changes)
	}
}

func TestChangedPicturesNameTheOldAndNewMedia(t *testing.T) {
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
	owner := uuid.New()
	earlier := recordedVersionOf("character")
	earlier.preserved = []versionPreserved{
		{Owner: "asset", OwnerID: owner, Namespace: "test", Payload: `{"secret":"before"}`},
	}
	later := recordedVersionOf("character")
	later.preserved = []versionPreserved{
		{Owner: "asset", OwnerID: owner, Namespace: "test", Payload: `{"secret":"after"}`},
	}
	changes := changesUnder(t, compareVersions(earlier, later), preservedSubject)
	if len(changes) != 1 || changes[0].Kind != ChangeEdited || changes[0].Name != "test" {
		t.Fatalf("changes = %+v", changes)
	}
	if changes[0].Before != "" || changes[0].After != "" {
		t.Fatal("the comparison carried preserved payloads")
	}
}

func TestPresentationOnlyUpdateExplainsThePage(t *testing.T) {
	page := uuid.New()
	core := recordedBlock(page, block.CharacterCore, recordedElement(block.RoleDescription, block.TypeProse,
		block.Prose{Text: "A quiet scout."}))
	moved := recordedBlock(uuid.New(), block.Lorebook)
	earlier := recordedVersionOf("character", core, moved)

	restyled := core
	restyled.Width = block.Width("half")
	later := recordedVersionOf("character", moved, restyled)

	groups := compareVersions(earlier, later)
	if got := subjectsOf(groups); len(got) != 1 || got[0] != presentationSubject {
		t.Fatalf("subjects = %v, want only the page", got)
	}
	changes := changesUnder(t, groups, presentationSubject)
	if len(changes) != 2 || changes[1].Name != "Page order" {
		t.Fatalf("changes = %+v", changes)
	}
	if changes[0].Name != "The character width" || changes[0].After != "half" {
		t.Fatalf("width change = %+v", changes[0])
	}
}

func TestComparisonDefaultsToTheVersionBeforeThePublishedOne(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()
	owner, id := publishedAsset(t, svc, pool, "compare.owner")
	open := func(Version) string { return "" }

	if _, err := svc.Compare(ctx, ComparisonRequest{AssetID: id, Access: open}); !errors.Is(err, ErrNoEarlierVersion) {
		t.Fatalf("first version compared against nothing: %v", err)
	}
	saveDescription(t, svc, owner, id, pool, "Second description")
	publishUpdate(t, svc, owner, id, "Rewrote the description")
	saveDescription(t, svc, owner, id, pool, "Third description")
	adult := false
	if err := svc.SetIdentity(ctx, Identity{
		OwnerID: owner, AssetID: id, Name: "Renamed", Blurb: "A changed pitch", IsNSFW: &adult,
	}, currentCandidate(t, svc, id)); err != nil {
		t.Fatal(err)
	}
	publishUpdate(t, svc, owner, id, "Rewrote it again")

	latest, err := svc.Compare(ctx, ComparisonRequest{AssetID: id, Access: open})
	if err != nil {
		t.Fatal(err)
	}
	if latest.From.Number != 2 || latest.To.Number != 3 {
		t.Fatalf("default comparison ran %d against %d", latest.From.Number, latest.To.Number)
	}
	changes := changesUnder(t, latest.Groups, string(block.RoleDescription))
	if len(changes) != 1 || changes[0].Before != "Second description" ||
		changes[0].After != "Third description" {
		t.Fatalf("changes = %+v", changes)
	}
	renamed := changesUnder(t, latest.Groups, metadataSubject)
	if len(renamed) != 2 || renamed[0].Name != "Name" ||
		renamed[0].Before != "Published name" || renamed[0].After != "Renamed" ||
		renamed[1].Name != "Blurb" || renamed[1].Before != "" || renamed[1].After != "A changed pitch" {
		t.Fatalf("metadata changes = %+v", renamed)
	}
	chosen, err := svc.Compare(ctx, ComparisonRequest{AssetID: id, From: 1, To: 3, Access: open})
	if err != nil {
		t.Fatal(err)
	}
	changes = changesUnder(t, chosen.Groups, string(block.RoleDescription))
	if len(changes) != 1 || changes[0].Before != "Published description" {
		t.Fatalf("chosen comparison = %+v", changes)
	}
}

func TestComparisonNeedsAccessRulesAndExplainsAVersionItCannotOpen(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()
	owner, id := publishedAsset(t, svc, pool, "gated.owner")
	saveDescription(t, svc, owner, id, pool, "Second description")
	publishUpdate(t, svc, owner, id, "Rewrote the description")

	if _, err := svc.Compare(ctx, ComparisonRequest{AssetID: id}); !errors.Is(err, ErrAccessRequired) {
		t.Fatalf("comparison ran without access rules: %v", err)
	}
	withheld, err := svc.Compare(ctx, ComparisonRequest{AssetID: id, Access: func(version Version) string {
		if version.Number == 1 {
			return "That version was withdrawn."
		}
		return ""
	}})
	if err != nil {
		t.Fatal(err)
	}
	if withheld.Unavailable != "That version was withdrawn." || len(withheld.Groups) != 0 {
		t.Fatalf("withheld comparison = %+v", withheld)
	}
}

func publishUpdate(t *testing.T, svc *Service, owner, id uuid.UUID, summary string) {
	t.Helper()
	if _, _, err := svc.PublishUpdate(context.Background(), UpdateRequest{
		OwnerID: owner, AssetID: id, Summary: summary,
	}, currentCandidate(t, svc, id)); err != nil {
		t.Fatalf("publish the update: %v", err)
	}
}
