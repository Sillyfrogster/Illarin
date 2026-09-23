package format

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
)

type writerModule struct {
	declaration Declaration
}

func (m writerModule) ID() string               { return m.declaration.ID }
func (m writerModule) Declaration() Declaration { return m.declaration }
func (writerModule) Write(context.Context, ExportWork) (MainFile, error) {
	return MainFile{}, nil
}

func writerDeclaration(id string, grades map[block.Role]SupportGrade) Declaration {
	roles := make(map[block.Role]DirectionalRoleSupport, len(grades))
	for role, grade := range grades {
		roles[role] = DirectionalRoleSupport{
			Read:  RoleSupport{Grade: SupportNone},
			Write: RoleSupport{Grade: grade},
		}
	}
	return Declaration{
		ID: id, Label: id, Type: "character", Direction: Direction{Write: true},
		Roles:                 roles,
		Limits:                ContentLimits{PayloadBytes: 1024, CollectionItems: 100, ItemBytes: 100},
		ConsumedKeys:          []string{"payload"},
		Preservation:          PreservationDeclaration{Body: "card"},
		TestedOriginalFormats: []string{id, OriginalFormatIllarin},
	}
}

func registryOf(t *testing.T, declarations ...Declaration) *Registry {
	t.Helper()
	registry := NewRegistry()
	for _, declaration := range declarations {
		if err := registry.Register(writerModule{declaration: declaration}); err != nil {
			t.Fatalf("register %s: %v", declaration.ID, err)
		}
	}
	return registry
}

func fullCharacterGrades() map[block.Role]SupportGrade {
	grades := make(map[block.Role]SupportGrade)
	for _, role := range block.Roles() {
		grades[role] = SupportFull
	}
	return grades
}

func described(role block.Role, body string) block.Element {
	return block.Element{
		ID: uuid.New(), Type: block.TypeProse, Role: role, Content: block.Prose{Text: body},
	}
}

func writtenGreetings(texts ...string) block.Element {
	items := make([]block.TextItem, 0, len(texts))
	for _, body := range texts {
		items = append(items, block.TextItem{ID: uuid.New(), Text: body})
	}
	return block.Element{
		ID: uuid.New(), Type: block.TypeTextSet, Role: block.RoleGreetings,
		Content: block.TextSet{Texts: items},
	}
}

func filledCharacter(extra ...block.Element) []block.Element {
	return append([]block.Element{
		described(block.RoleDescription, "Keeps the archive."),
		writtenGreetings("Hello"),
	}, extra...)
}

func formatNamed(offered []Offered, id string) (Offered, bool) {
	for _, one := range offered {
		if one.Format == id {
			return one, true
		}
	}
	return Offered{}, false
}

func TestAnUntestedOriginOffersNoFormat(t *testing.T) {
	t.Parallel()
	registry := registryOf(t, writerDeclaration("preset_lumiverse", fullCharacterGrades()))
	offered := registry.OfferedFormats(CapabilitySubject{
		Type: "character", OriginalFormat: "chara_card_v2", Elements: filledCharacter(),
	})
	if len(offered) != 0 {
		t.Fatalf("offered = %+v, want none for an untested original format", offered)
	}
}

func TestAnWorkBuiltFromNothingIsOfferedEveryWriterTestedAgainstIllarin(t *testing.T) {
	t.Parallel()
	registry := registryOf(t,
		writerDeclaration("chara_card_v2", fullCharacterGrades()),
		writerDeclaration("chara_card_v3", fullCharacterGrades()),
	)
	offered := registry.OfferedFormats(CapabilitySubject{
		Type: "character", Elements: filledCharacter(),
	})
	if len(offered) != 2 {
		t.Fatalf("offered = %+v, want both writers", offered)
	}
}

func TestAFormatThatDropsARequiredRoleIsNotOffered(t *testing.T) {
	t.Parallel()
	grades := fullCharacterGrades()
	grades[block.RoleGreetings] = SupportNone
	registry := registryOf(t,
		writerDeclaration("chara_card_v2", grades),
		writerDeclaration("chara_card_v3", fullCharacterGrades()),
	)
	offered := registry.OfferedFormats(CapabilitySubject{
		Type: "character", Elements: filledCharacter(),
	})
	if _, found := formatNamed(offered, "chara_card_v2"); found {
		t.Error("a format that drops every greeting was offered")
	}
	if _, found := formatNamed(offered, "chara_card_v3"); !found {
		t.Error("the format that carries the greetings was not offered")
	}
}

func TestEmptyInAndEmptyOutIsNoLossAndBlocksNothing(t *testing.T) {
	t.Parallel()
	grades := fullCharacterGrades()
	grades[block.RoleGreetings] = SupportNone
	registry := registryOf(t, writerDeclaration("chara_card_v2", grades))
	offered := registry.OfferedFormats(CapabilitySubject{
		Type:     "character",
		Elements: []block.Element{described(block.RoleDescription, "Keeps the archive.")},
	})
	if len(offered) != 1 {
		t.Fatalf("offered = %+v, want the format offered", offered)
	}
	if losses := offered[0].Losses(); len(losses) != 0 {
		t.Errorf("losses = %+v, want none reported for a field nobody filled in", losses)
	}
}

func TestAFormatDroppingEveryOptionalRoleIsStillOffered(t *testing.T) {
	t.Parallel()
	grades := map[block.Role]SupportGrade{
		block.RoleDescription: SupportFull,
		block.RoleGreetings:   SupportFull,
	}
	optional := []block.Element{}
	for _, role := range block.Roles() {
		if role == block.RoleDescription || role == block.RoleGreetings {
			continue
		}
		grades[role] = SupportNone
		if role.Allows(block.TypeProse) {
			optional = append(optional, described(role, "Something."))
		}
	}
	registry := registryOf(t, writerDeclaration("chara_card_v2", grades))
	offered := registry.OfferedFormats(CapabilitySubject{
		Type: "character", Elements: filledCharacter(optional...),
	})
	if len(offered) != 1 {
		t.Fatalf("offered = %+v, want the format offered with its losses stated", offered)
	}
	if len(offered[0].Losses()) != len(optional) {
		t.Errorf("losses = %+v, want one per dropped optional role", offered[0].Losses())
	}
}

func TestAPartialGradeFiresOnlyWhereItsConditionHolds(t *testing.T) {
	t.Parallel()
	grades := fullCharacterGrades()
	declaration := writerDeclaration("chara_card_v2", grades)
	declaration.Roles[block.RoleGreetings] = DirectionalRoleSupport{
		Read: RoleSupport{Grade: SupportNone},
		Write: RoleSupport{
			Grade: SupportPartial,
			Condition: &ContentCondition{
				Description: "a name written on a greeting",
				Matches: func(content block.Content) bool {
					set, ok := content.(block.TextSet)
					if !ok {
						return false
					}
					return slices.ContainsFunc(set.Texts, func(item block.TextItem) bool {
						return item.Name != ""
					})
				},
			},
		},
	}
	registry := registryOf(t, declaration)

	plain := registry.OfferedFormats(CapabilitySubject{
		Type: "character", Elements: filledCharacter(),
	})
	if losses := plain[0].Losses(); len(losses) != 0 {
		t.Errorf("losses = %+v, want none where the condition does not hold", losses)
	}

	named := writtenGreetings("Hello")
	set := named.Content.(block.TextSet)
	set.Texts[0].Name = "First meeting"
	named.Content = set
	withName := registry.OfferedFormats(CapabilitySubject{
		Type: "character",
		Elements: []block.Element{
			described(block.RoleDescription, "Keeps the archive."), named,
		},
	})
	losses := withName[0].Losses()
	if len(losses) != 1 || losses[0].Verdict != Reduced ||
		!strings.Contains(losses[0].Reason, "a name written on a greeting") {
		t.Fatalf("losses = %+v, want one reduced verdict naming what went", losses)
	}
	if losses[0].Sample.Count != 1 || len(losses[0].Sample.Texts) != 1 {
		t.Errorf("sample = %+v, want a glance at what is at stake", losses[0].Sample)
	}
}

func TestADestinationNoteRidesOnACarriedVerdict(t *testing.T) {
	t.Parallel()
	declaration := writerDeclaration("chara_card_v3", fullCharacterGrades())
	declaration.Roles[block.RoleCreatorNotes] = DirectionalRoleSupport{
		Read: RoleSupport{Grade: SupportNone},
		Write: RoleSupport{
			Grade: SupportFull, Destination: "an extensions namespace only some clients read",
		},
	}
	registry := registryOf(t, declaration)
	offered := registry.OfferedFormats(CapabilitySubject{
		Type:     "character",
		Elements: filledCharacter(described(block.RoleCreatorNotes, "Built over a weekend.")),
	})
	var found RoleLoss
	for _, role := range offered[0].Roles {
		if role.Role == block.RoleCreatorNotes {
			found = role
		}
	}
	if found.Verdict != Carried || found.Destination == "" {
		t.Fatalf("creator notes verdict = %+v, want carried with a destination", found)
	}
	if len(offered[0].Losses()) != 0 {
		t.Errorf("losses = %+v, want a destination note counted as no loss", offered[0].Losses())
	}
}

func TestTheRecommendationPrefersReachOverCarryingTheMost(t *testing.T) {
	t.Parallel()
	wide := writerDeclaration("chara_card_v3", fullCharacterGrades())
	wide.Roles[block.RoleGallery] = DirectionalRoleSupport{
		Read: RoleSupport{Grade: SupportNone}, Write: RoleSupport{Grade: SupportNone},
	}
	narrow := writerDeclaration("byaf", fullCharacterGrades())
	registry := registryOf(t, wide, narrow)

	gallery := block.Element{
		ID: uuid.New(), Type: block.TypeImageSet, Role: block.RoleGallery,
		Content: block.ImageSet{Images: []block.ImageItem{{ID: uuid.New(), MediaID: uuid.New()}}},
	}
	offered := registry.OfferedFormats(CapabilitySubject{
		Type: "character", Elements: filledCharacter(gallery),
	})
	recommended, lossiest := "", ""
	for _, one := range offered {
		if one.Recommended {
			recommended = one.Format
		}
		if len(one.Losses()) == 0 {
			lossiest = one.Format
		}
	}
	if lossiest != "byaf" {
		t.Fatalf("the narrow format lost something; the rules do not disagree here")
	}
	if recommended != "chara_card_v3" {
		t.Fatalf("recommended = %q, want the format more apps can open", recommended)
	}
}

func TestPreservedDataTravelsByOriginMatchAlone(t *testing.T) {
	t.Parallel()
	card := writerDeclaration("chara_card_v3", fullCharacterGrades())
	card.Preservation = PreservationDeclaration{Body: "card", Container: []string{"extensions"}}
	sibling := writerDeclaration("charx", fullCharacterGrades())
	sibling.Preservation = card.Preservation
	stranger := writerDeclaration("theme_lumiverse", fullCharacterGrades())
	stranger.Preservation = PreservationDeclaration{Body: "bundle"}

	registry := registryOf(t, card, sibling, stranger)

	if !registry.TravelsWithOriginalFormat(card.ID, sibling) {
		t.Error("preserved data did not travel to its own family")
	}
	if registry.TravelsWithOriginalFormat(card.ID, stranger) {
		t.Error("preserved data reached another family")
	}
}

func TestPreservedDataFromARetiredOriginTravelsWhereTheFormatKeepsIt(t *testing.T) {
	t.Parallel()
	keeper := writerDeclaration("lorebook_lumiverse", fullCharacterGrades())
	keeper.Preservation = PreservationDeclaration{Body: "lorebook"}
	keeper.PreservesOriginalFormats = []string{"retired"}
	keeper.TestedOriginalFormats = append(keeper.TestedOriginalFormats, "retired")
	other := writerDeclaration("theme_lumiverse", fullCharacterGrades())
	other.Preservation = PreservationDeclaration{Body: "bundle"}
	registry := registryOf(t, keeper, other)

	if !registry.TravelsWithOriginalFormat("retired", keeper) {
		t.Error("preserved data from a retired original format did not reach the format that keeps it")
	}
	if registry.TravelsWithOriginalFormat("retired", other) {
		t.Error("preserved data from a retired original format reached a format that does not keep it")
	}
}

func TestTheCapabilityStampMovesWithADeclaration(t *testing.T) {
	t.Parallel()
	before := registryOf(t, writerDeclaration("chara_card_v2", fullCharacterGrades()))
	grades := fullCharacterGrades()
	grades[block.RoleGallery] = SupportNone
	after := registryOf(t, writerDeclaration("chara_card_v2", grades))

	if before.CapabilityStamp() == after.CapabilityStamp() {
		t.Fatal("the stamp did not move when a writer's role support changed")
	}
	if before.CapabilityStamp() != registryOf(t,
		writerDeclaration("chara_card_v2", fullCharacterGrades()),
	).CapabilityStamp() {
		t.Fatal("the stamp is not stable for one contract")
	}
}

func TestTheRecommendationPrefersTheFormatWhoseContentActuallyArrives(t *testing.T) {
	t.Parallel()
	noted := writerDeclaration("chara_card_v3", fullCharacterGrades())
	gallerySupport := noted.Roles[block.RoleGallery]
	gallerySupport.Write.Destination = "Only one app unpacks them."
	noted.Roles[block.RoleGallery] = gallerySupport
	plain := writerDeclaration("charx", fullCharacterGrades())
	registry := registryOf(t, noted, plain)

	gallery := block.Element{
		ID: uuid.New(), Type: block.TypeImageSet, Role: block.RoleGallery,
		Content: block.ImageSet{Images: []block.ImageItem{{ID: uuid.New(), MediaID: uuid.New()}}},
	}
	offered := registry.OfferedFormats(CapabilitySubject{
		Type: "character", Elements: filledCharacter(gallery),
	})
	for _, one := range offered {
		if one.Recommended && one.Format != "charx" {
			t.Fatalf("recommended %s, want the one that needs no note", one.Format)
		}
	}
}
