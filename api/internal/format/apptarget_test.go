package format

import (
	"slices"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
)

func TestEveryAppIsOfferedTheFormatThatLandsMostOfTheWorkInIt(t *testing.T) {
	t.Parallel()
	registry := registryOf(t, noteDeclaration("chara_card_v3"), plainDeclaration("charx"))
	targets := registry.OfferedTargets(galleryCharacter())
	byFormat := make(map[string]Target, len(targets))
	for _, target := range targets {
		byFormat[target.Format] = target
	}

	offered := AppTargets(targets, registry)
	if len(offered) != len(Apps()) {
		t.Fatalf("offered %d apps, want all %d", len(offered), len(Apps()))
	}
	for _, app := range offered {
		if lost := byFormat[app.Format].LossesFor(app.ID); lost != 0 {
			t.Errorf("%s was offered %q, which leaves out %d", app.ID, app.Format, lost)
		}
	}
}

func TestNoAppIsSentAFormatOthersWillNotShowWhenOneReadsEverywhere(t *testing.T) {
	t.Parallel()
	registry := registryOf(t, noteDeclaration("chara_card_v3"), plainDeclaration("charx"))
	targets := registry.OfferedTargets(galleryCharacter())

	for _, app := range AppTargets(targets, registry) {
		if app.Format != "charx" {
			t.Errorf("%s = %q, want the format whose gallery every app shows", app.ID, app.Format)
		}
	}
}

func TestAnAppThatUnpacksTheNoteIsOfferedTheFormatCarryingIt(t *testing.T) {
	t.Parallel()
	registry := registryOf(t, noteDeclaration("chara_card_v3"))
	targets := registry.OfferedTargets(galleryCharacter())

	offered := AppTargets(targets, registry)
	if len(offered) == 0 {
		t.Fatal("no app was offered anything")
	}
	for _, app := range offered {
		if app.Format != "chara_card_v3" {
			t.Errorf("%s = %q, want the only format on offer", app.ID, app.Format)
		}
	}
}

func TestAnAppThatReadsNoOfferedFormatIsOfferedNothing(t *testing.T) {
	t.Parallel()
	registry := registryOf(t, plainDeclaration("byaf"))
	targets := registry.OfferedTargets(galleryCharacter())

	if offered := AppTargets(targets, registry); len(offered) != 0 {
		t.Fatalf("offered = %v, want nothing for a format no app reads", offered)
	}
}

func TestAFormatOnlySomeAppsUnpackCountsAsALossToTheRest(t *testing.T) {
	t.Parallel()
	registry := registryOf(t, noteDeclaration("chara_card_v3"))
	target := registry.OfferedTargets(galleryCharacter())[0]

	if lost := target.LossesFor("risu"); lost != 0 {
		t.Errorf("RisuAI loses %d, want nothing from a format it unpacks", lost)
	}
	if lost := target.LossesFor("sillytavern"); lost != 1 {
		t.Errorf("SillyTavern loses %d, want the gallery it never shows", lost)
	}
}

func TestARoleCarriesTheAppsThatShowIt(t *testing.T) {
	t.Parallel()
	registry := registryOf(t, noteDeclaration("chara_card_v3"))
	target := registry.OfferedTargets(galleryCharacter())[0]

	for _, role := range target.Roles {
		if role.Role != block.RoleGallery {
			continue
		}
		if !slices.Equal(role.ShownBy, []string{"risu"}) {
			t.Fatalf("shownBy = %v, want the one app that unpacks it", role.ShownBy)
		}
		return
	}
	t.Fatal("the report named no gallery")
}

func noteDeclaration(id string) Declaration {
	declaration := plainDeclaration(id)
	gallery := declaration.Roles[block.RoleGallery]
	gallery.Write.Destination = "Written into the card itself."
	gallery.Write.ShownBy = []string{"risu"}
	declaration.Roles[block.RoleGallery] = gallery
	return declaration
}

func plainDeclaration(id string) Declaration {
	return writerDeclaration(id, fullCharacterGrades())
}

func galleryCharacter() CapabilitySubject {
	return CapabilitySubject{Type: "character", Elements: []block.Element{
		described(block.RoleDescription, "Keeps the archive."),
		{
			ID: uuid.New(), Type: block.TypeImageSet, Role: block.RoleGallery,
			Content: block.ImageSet{Images: []block.ImageItem{
				{ID: uuid.New(), MediaID: uuid.New(), Name: "At the door"},
			}},
		},
	}}
}
