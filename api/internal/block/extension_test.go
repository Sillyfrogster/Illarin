package block

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func extensionElements() []Element {
	return []Element{
		{Type: TypeTextSet, Role: RoleExtensionGrantedPermissions, Content: TextSet{Texts: []TextItem{
			{ID: NewItemID(), Name: "ui_panels", Text: "Open floating widgets and docked panels."},
		}}},
		{Type: TypeTextSet, Role: RoleExtensionApprovedPermissions, Content: TextSet{Texts: []TextItem{}}},
		{Type: TypeFieldList, Role: RoleExtensionDetails, Content: FieldList{Fields: []FieldItem{
			{ID: NewItemID(), Name: "Version", Value: "1.0.0"},
		}}},
		{Type: TypeLinkList, Role: RoleExtensionLinks, Content: LinkList{Links: []LinkItem{
			{ID: NewItemID(), Label: "Repository", URL: "https://github.com/example/tool"},
		}}},
	}
}

func placedExtension(t *testing.T) []Block {
	t.Helper()
	blocks, err := Place("extension", extensionElements())
	if err != nil {
		t.Fatalf("place an extension: %v", err)
	}
	return blocks
}

func TestAnExtensionPlacesItsArchiveReadingsInLockedBlocks(t *testing.T) {
	blocks := placedExtension(t)
	if len(blocks) != 2 || blocks[0].Definition != ExtensionPermissions || blocks[1].Definition != ExtensionSource {
		t.Fatalf("placed blocks = %+v, want permissions then source", blocks)
	}
	for _, holder := range blocks {
		definition, _ := holder.Definition.Definition("extension")
		if !definition.Required || !definition.Hideable {
			t.Errorf("%s is required %t and hideable %t, want both", holder.Definition, definition.Required, definition.Hideable)
		}
		for _, element := range holder.Elements {
			if !holder.Pinned(element.Role, "extension") || !holder.Locked(element.Role, "extension") {
				t.Errorf("%s in %s is not pinned and locked", element.Role, holder.Definition)
			}
		}
	}
	if checks := ContentFloor("extension", blocks); len(checks) != 1 || !checks[0].Met {
		t.Errorf("extension floor = %+v, want the archive's details met", checks)
	}
	if checks := ContentFloor("extension", nil); len(checks) != 1 || checks[0].Met {
		t.Errorf("an extension with no archive meets its floor: %+v", checks)
	}
}

func TestAnExtensionPlacesOnlyTheBlocksItsArchiveSupplies(t *testing.T) {
	blocks, err := Place("extension", []Element{
		{Type: TypeTextSet, Role: RoleExtensionDependencies, Content: TextSet{Texts: []TextItem{
			{ID: NewItemID(), Text: "third-party/SillyTavern-LALib"},
		}}},
		{Type: TypeFieldList, Role: RoleExtensionDetails, Content: FieldList{Fields: []FieldItem{
			{ID: NewItemID(), Name: "Script", Value: "index.js"},
		}}},
	})
	if err != nil {
		t.Fatalf("place a SillyTavern extension: %v", err)
	}
	if len(blocks) != 2 || blocks[0].Definition != ExtensionDependencies || blocks[1].Definition != ExtensionSource {
		t.Fatalf("placed blocks = %+v, want dependencies then source and no permissions", blocks)
	}
	if len(blocks[1].Elements) != 1 {
		t.Errorf("source holds %+v, want only the details the archive supplied", blocks[1].Elements)
	}
	if !blocks[0].Locked(RoleExtensionDependencies, "extension") {
		t.Error("dependencies are not locked")
	}
	if offers, _ := Offers("extension"); len(offers) != len(shared) {
		t.Errorf("the extension tray offers %d blocks, want only the shared ones", len(offers))
	}
}

func TestWhatAnExtensionAddsSitsInALockedBlockAfterItsPermissions(t *testing.T) {
	blocks, err := Place("extension", append(extensionElements(), Element{
		Type: TypeFieldList, Role: RoleExtensionAdditions, Content: FieldList{Fields: []FieldItem{
			{ID: NewItemID(), Name: "Tools", Value: "search_notes"},
			{ID: NewItemID(), Name: "Generation hooks", Value: "Interceptor"},
		}},
	}))
	if err != nil {
		t.Fatalf("place an extension: %v", err)
	}
	definitions := []DefinitionID{}
	for _, holder := range blocks {
		definitions = append(definitions, holder.Definition)
	}
	if len(blocks) != 3 || blocks[1].Definition != ExtensionAdditions {
		t.Fatalf("placed blocks = %v, want permissions, what it adds, then source", definitions)
	}
	adds := blocks[1]
	definition, _ := adds.Definition.Definition("extension")
	if definition.Title != "What it adds" || !definition.Required || !definition.Hideable {
		t.Errorf("definition = %+v, want a required, hideable block titled What it adds", definition)
	}
	if !adds.Pinned(RoleExtensionAdditions, "extension") || !adds.Locked(RoleExtensionAdditions, "extension") {
		t.Error("what it adds is not pinned and locked")
	}
	if facts := adds.Elements[0].Facts(); len(facts) != 1 || facts[0] != "2 additions" {
		t.Errorf("facts = %v, want 2 additions", facts)
	}
}

func TestALockedElementMovesAndHidesButKeepsItsContent(t *testing.T) {
	before := placedExtension(t)
	after := cloneBlocks(before)
	after[0].Hidden = true
	after[0].Position, after[1].Position = 1, 0
	after[0], after[1] = after[1], after[0]
	if err := ValidateBuilderConstraints("extension", before, after); err != nil {
		t.Fatalf("moving and hiding locked blocks was refused: %v", err)
	}

	edited := cloneBlocks(before)
	edited[1].Elements[0].Content = FieldList{Fields: []FieldItem{
		{ID: NewItemID(), Name: "Version", Value: "9.9.9"},
	}}
	err := ValidateBuilderConstraints("extension", before, edited)
	if err == nil || !strings.Contains(err.Error(), "archive") {
		t.Fatalf("editing a locked element = %v, want a refusal that names the archive", err)
	}
}

func TestALockedElementCannotArriveThroughAnEdit(t *testing.T) {
	before := placedExtension(t)
	after := cloneBlocks(before)
	after[1].Elements[0].ID = uuid.New()
	if err := ValidateBuilderConstraints("extension", before, after); err == nil {
		t.Fatal("a new locked element was accepted from an edit")
	}
}

func cloneBlocks(blocks []Block) []Block {
	cloned := make([]Block, len(blocks))
	for i, holder := range blocks {
		cloned[i] = holder
		cloned[i].Elements = append([]Element(nil), holder.Elements...)
	}
	return cloned
}
