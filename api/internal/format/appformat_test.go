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
	formats := registry.OfferedFormats(galleryCharacter())
	byFormat := make(map[string]Offered, len(formats))
	for _, one := range formats {
		byFormat[one.Format] = one
	}

	offered := AppFormats(formats, registry)
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
	formats := registry.OfferedFormats(galleryCharacter())

	for _, app := range AppFormats(formats, registry) {
		if app.Format != "charx" {
			t.Errorf("%s = %q, want the format whose gallery every app shows", app.ID, app.Format)
		}
	}
}

func TestAnAppThatUnpacksTheNoteIsOfferedTheFormatCarryingIt(t *testing.T) {
	t.Parallel()
	registry := registryOf(t, noteDeclaration("chara_card_v3"))
	formats := registry.OfferedFormats(galleryCharacter())

	offered := AppFormats(formats, registry)
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
	formats := registry.OfferedFormats(galleryCharacter())

	if offered := AppFormats(formats, registry); len(offered) != 0 {
		t.Fatalf("offered = %v, want nothing for a format no app reads", offered)
	}
}

func TestAFormatOnlySomeAppsUnpackCountsAsALossToTheRest(t *testing.T) {
	t.Parallel()
	registry := registryOf(t, noteDeclaration("chara_card_v3"))
	one := registry.OfferedFormats(galleryCharacter())[0]

	if lost := one.LossesFor("risu"); lost != 0 {
		t.Errorf("RisuAI loses %d, want nothing from a format it unpacks", lost)
	}
	if lost := one.LossesFor("sillytavern"); lost != 1 {
		t.Errorf("SillyTavern loses %d, want the gallery it never shows", lost)
	}
}

func TestARoleCarriesTheAppsThatShowIt(t *testing.T) {
	t.Parallel()
	registry := registryOf(t, noteDeclaration("chara_card_v3"))
	one := registry.OfferedFormats(galleryCharacter())[0]

	for _, role := range one.Roles {
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
