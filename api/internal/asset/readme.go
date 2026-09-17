package asset

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/Sillyfrogster/Illarin/api/internal/readme"
	"github.com/google/uuid"
)

const (
	// openingTitle names the block holding what a README says before its first section.
	openingTitle = "About"
	// restTitle names the block that lists every section after the first few.
	restTitle = "More from the README"
	// shownSections is how many sections become blocks of their own before the rest are listed.
	shownSections = 3
)

// seedFromReadme adds the blocks a README seeds and puts its pictures in the vault, once, when its asset is made.
func (s *Service) seedFromReadme(ctx context.Context, file format.Inspection, read preparedImport) (preparedImport, error) {
	source := read.Parsed.Readme
	if source == nil {
		return read, nil
	}
	held := archivedImages(file)
	page := readme.Read(source.Text, readme.InArchive(source.Root, source.Folder, func(entry string) bool {
		_, ok := held[entry]
		return ok
	}))
	opening, sections, targets := readmeBlocks(page)
	blocks := slices.Concat(opening, read.Blocks, sections)
	for index := range blocks {
		blocks[index].Position = index
	}
	prepared, vault, err := s.readmePictures(ctx, file, page, held, targets)
	if err != nil {
		return preparedImport{}, err
	}
	read.Blocks = blocks
	read.Media = append(read.Media, prepared...)
	read.Vault = vault
	return read, nil
}

func archivedImages(file format.Inspection) map[string]uint32 {
	held := make(map[string]uint32, len(file.Images))
	for _, image := range file.Images {
		if image.Locator.Container == format.ZIP {
			held[image.Locator.Name] = image.ID
		}
	}
	return held
}

// readmePictures stores the cover, and puts every other picture in the vault for the block its section became.
func (s *Service) readmePictures(
	ctx context.Context, file format.Inspection, page readme.Page, held map[string]uint32, targets []*uuid.UUID,
) ([]preparedMedia, []VaultPicture, error) {
	var prepared []preparedMedia
	var vault []VaultPicture
	if page.Cover != nil {
		cover, ok, err := s.seededPicture(ctx, file, held[page.Cover.Entry], MediaAvatar)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			prepared = append(prepared, cover)
		}
	}
	stored := make(map[string]uuid.UUID)
	for index, section := range slices.Concat([]readme.Section{page.Opening}, page.Sections) {
		target := targets[index]
		for _, image := range section.Images {
			id, done := stored[image.Entry]
			if !done {
				picture, ok, err := s.seededPicture(ctx, file, held[image.Entry], MediaGallery)
				if err != nil {
					return nil, nil, err
				}
				if !ok {
					continue
				}
				prepared = append(prepared, picture)
				id, stored[image.Entry] = picture.ID, picture.ID
			}
			media := id
			vault = append(vault, VaultPicture{
				ID: uuid.New(), MediaID: &media, Address: image.Entry, Name: image.Name,
				BlockID: target, Section: section.Title,
			})
		}
		for _, picture := range section.Remote {
			vault = append(vault, VaultPicture{
				ID: uuid.New(), Address: picture.Address, Name: picture.Name, BlockID: target, Section: section.Title,
			})
		}
	}
	return prepared, vault, nil
}

// seededPicture prepares one README picture, leaving out one the media processor will not take.
func (s *Service) seededPicture(
	ctx context.Context, file format.Inspection, image uint32, role MediaRole,
) (preparedMedia, bool, error) {
	prepared, err := s.prepareExtractedMedia(ctx, file, []format.Media{{Role: role, ImageID: image}})
	if errors.Is(err, mediaproc.ErrImageTooLarge) {
		return preparedMedia{}, false, nil
	}
	if err != nil || len(prepared) == 0 {
		return preparedMedia{}, false, err
	}
	prepared[0].Seeded = true
	return prepared[0], true, nil
}

// readmeBlocks makes the opening and section blocks, and says which block each section's pictures wait for.
func readmeBlocks(page readme.Page) ([]block.Block, []block.Block, []*uuid.UUID) {
	var opening, sections []block.Block
	targets := make([]*uuid.UUID, 1+len(page.Sections))
	if seeded, ok := readmeBlock(openingTitle, prose(page.Opening.Text)); ok {
		opening = append(opening, seeded)
		targets[0] = &seeded.ID
	}
	placed, rest := page.Sections, []readme.Section(nil)
	if len(placed) > shownSections+1 {
		placed, rest = placed[:shownSections], placed[shownSections:]
	}
	for index, section := range placed {
		if seeded, ok := readmeBlock(section.Title, prose(section.Text)); ok {
			sections = append(sections, seeded)
			targets[1+index] = &seeded.ID
		}
	}
	if seeded, ok := readmeBlock(restTitle, passages(rest)); ok {
		sections = append(sections, seeded)
		for index := range rest {
			targets[1+len(placed)+index] = &seeded.ID
		}
	}
	return opening, sections, targets
}

func prose(text string) *block.Element {
	if text == "" {
		return nil
	}
	return &block.Element{
		ID: uuid.New(), Type: block.TypeProse, Options: block.Options{Display: block.DisplayRich},
		Content: block.Prose{Text: text},
	}
}

// passages lists each section after the first few under its own title, so a reader can browse them.
func passages(sections []readme.Section) *block.Element {
	texts := []block.TextItem{}
	for index, section := range sections {
		if section.Text == "" {
			continue
		}
		name := section.Title
		if name == "" {
			name = fmt.Sprintf("Section %d", shownSections+index+1)
		}
		texts = append(texts, block.TextItem{ID: block.NewItemID(), Name: name, Text: section.Text})
	}
	if len(texts) == 0 {
		return nil
	}
	return &block.Element{
		ID: uuid.New(), Type: block.TypeTextSet, Options: block.Options{Display: block.DisplayRich},
		Content: block.TextSet{Texts: texts},
	}
}

// readmeBlock makes one ordinary block of a README's writing, the creator's to edit from then on.
func readmeBlock(title string, writing *block.Element) (block.Block, bool) {
	if writing == nil {
		return block.Block{}, false
	}
	writing.Slot = block.Single.Slots()[0]
	var named *string
	if title != "" {
		named = &title
	}
	return block.Block{
		ID: uuid.New(), Definition: block.CustomBlock, Title: named,
		Layout: block.Single, Width: block.Full, Elements: []block.Element{*writing},
	}, true
}
