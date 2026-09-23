package upload

import (
	"fmt"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
)

func sectionsNamed(count int) []readmeSection {
	sections := make([]readmeSection, 0, count)
	for index := range count {
		sections = append(sections, readmeSection{
			Title: fmt.Sprintf("Part %d", index+1), Text: fmt.Sprintf("Text %d", index+1),
		})
	}
	return sections
}

func titlesOf(blocks []block.Block) []string {
	titles := make([]string, 0, len(blocks))
	for _, holder := range blocks {
		title := ""
		if holder.Title != nil {
			title = *holder.Title
		}
		titles = append(titles, title)
	}
	return titles
}

func TestAReadmeOfFourSectionsKeepsEachAsABlock(t *testing.T) {
	t.Parallel()
	_, sections, _ := readmeBlocks(readmePage{Sections: sectionsNamed(4)})
	if got := fmt.Sprint(titlesOf(sections)); got != "[Part 1 Part 2 Part 3 Part 4]" {
		t.Fatalf("blocks = %s, want one block for each of the four sections", got)
	}
}

func TestALongReadmeGathersEverySectionAfterTheThirdIntoOneList(t *testing.T) {
	t.Parallel()
	page := readmePage{Opening: readmeSection{Text: "Intro"}, Sections: sectionsNamed(6)}
	opening, sections, targets := readmeBlocks(page)

	if got := fmt.Sprint(titlesOf(opening), titlesOf(sections)); got != "[About] [Part 1 Part 2 Part 3 More from the README]" {
		t.Fatalf("blocks = %s, want the first three sections and one list of the rest", got)
	}
	rest := sections[3]
	list, ok := rest.Elements[0].Content.(block.TextSet)
	if !ok || rest.Elements[0].Options.Display != block.DisplayRich {
		t.Fatalf("the rest opens with %+v, want a list of rich passages", rest.Elements[0])
	}
	var got []string
	for _, item := range list.Texts {
		got = append(got, item.Name+": "+item.Text)
	}
	if fmt.Sprint(got) != "[Part 4: Text 4 Part 5: Text 5 Part 6: Text 6]" {
		t.Errorf("the rest lists %q, want each later section by its title", got)
	}
	if *targets[0] != opening[0].ID || *targets[2] != sections[1].ID || *targets[5] != rest.ID || *targets[6] != rest.ID {
		t.Errorf("targets = %v, want the opening, its own block for the first three sections, and the list for the rest", targets)
	}
}

func TestAWaitingPictureJoinsItsBlockBesideTheWriting(t *testing.T) {
	t.Parallel()
	title := "Install"
	page := []block.Block{{
		ID: uuid.New(), Definition: block.CustomBlock, Title: &title, Layout: block.Single, Width: block.Full,
		Elements: []block.Element{{
			ID: uuid.New(), Type: block.TypeProse, Slot: "main", Options: block.Options{Display: block.DisplayRich},
			Content: block.Prose{Text: "Clone it."},
		}},
	}}
	media := uuid.New()
	after, _, err := placePicture(page, WaitingPicture{MediaID: &media, Name: "Settings", BlockID: &page[0].ID})
	if err != nil {
		t.Fatalf("place: %v", err)
	}
	holder := after[0]
	if holder.Layout != block.MainAside || len(holder.Elements) != 2 || holder.Elements[1].Slot != "aside" {
		t.Fatalf("block = %+v, want the writing beside a new image set", holder)
	}
	set, ok := holder.Elements[1].Content.(block.ImageSet)
	if !ok || len(set.Images) != 1 || set.Images[0].MediaID != media || set.Images[0].Name != "Settings" {
		t.Errorf("image set = %+v, want the one placed picture", holder.Elements[1].Content)
	}
	if len(page[0].Elements) != 1 {
		t.Error("placing a picture changed the page it was given")
	}

	second := uuid.New()
	again, _, err := placePicture(after, WaitingPicture{MediaID: &second, Name: "More", BlockID: &page[0].ID})
	if err != nil {
		t.Fatalf("place again: %v", err)
	}
	if set := again[0].Elements[1].Content.(block.ImageSet); len(set.Images) != 2 || len(again[0].Elements) != 2 {
		t.Errorf("block after a second picture = %+v, want both in the one image set", again[0])
	}
}

func TestAWaitingPictureWhoseBlockIsGoneGetsABlockOfItsOwn(t *testing.T) {
	t.Parallel()
	media, gone := uuid.New(), uuid.New()
	after, made, err := placePicture(nil, WaitingPicture{MediaID: &media, Name: "Shot", BlockID: &gone, Section: "Screenshots"})
	if err != nil {
		t.Fatalf("place: %v", err)
	}
	if len(after) != 1 || *after[0].Title != "Screenshots" || after[0].Elements[0].Type != block.TypeImageSet {
		t.Fatalf("page = %+v, want one new block named after the section", after)
	}
	if made == nil || *made != after[0].ID {
		t.Errorf("made = %v, want the new block's id %s", made, after[0].ID)
	}
}
