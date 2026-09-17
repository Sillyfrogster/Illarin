package upload

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/google/uuid"
)

const (
	openingTitle  = "About"
	restTitle     = "More from the README"
	shownSections = 3
)

// seedFromReadme turns a README into blocks and holds its pictures in the vault when a work is first made
func (s *Service) seedFromReadme(ctx context.Context, file format.Inspection, read preparedImport) (preparedImport, error) {
	source := read.Parsed.Readme
	if source == nil {
		return read, nil
	}
	held := archivedImages(file)
	page := readReadme(source.Text, inArchive(source.Root, source.Folder, func(entry string) bool {
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

func (s *Service) readmePictures(
	ctx context.Context, file format.Inspection, page readmePage, held map[string]uint32, targets []*uuid.UUID,
) ([]asset.PreparedMedia, []WaitingPicture, error) {
	var prepared []asset.PreparedMedia
	var vault []WaitingPicture
	if page.Cover != nil {
		cover, ok, err := s.seededPicture(ctx, file, held[page.Cover.Entry], asset.MediaAvatar)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			prepared = append(prepared, cover)
		}
	}
	stored := make(map[string]uuid.UUID)
	for index, section := range slices.Concat([]readmeSection{page.Opening}, page.Sections) {
		target := targets[index]
		for _, image := range section.Images {
			id, done := stored[image.Entry]
			if !done {
				picture, ok, err := s.seededPicture(ctx, file, held[image.Entry], asset.MediaGallery)
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
			vault = append(vault, WaitingPicture{
				ID: uuid.New(), MediaID: &media, Address: image.Entry, Name: image.Name,
				BlockID: target, Section: section.Title,
			})
		}
		for _, picture := range section.Remote {
			vault = append(vault, WaitingPicture{
				ID: uuid.New(), Address: picture.Address, Name: picture.Name, BlockID: target, Section: section.Title,
			})
		}
	}
	return prepared, vault, nil
}

func (s *Service) seededPicture(
	ctx context.Context, file format.Inspection, image uint32, role asset.MediaRole,
) (asset.PreparedMedia, bool, error) {
	prepared, err := s.assets.PrepareExtractedMedia(ctx, file, []format.Media{{Role: role, ImageID: image}})
	if errors.Is(err, mediaproc.ErrImageTooLarge) {
		return asset.PreparedMedia{}, false, nil
	}
	if err != nil || len(prepared) == 0 {
		return asset.PreparedMedia{}, false, err
	}
	prepared[0].Seeded = true
	return prepared[0], true, nil
}

func readmeBlocks(page readmePage) ([]block.Block, []block.Block, []*uuid.UUID) {
	var opening, sections []block.Block
	targets := make([]*uuid.UUID, 1+len(page.Sections))
	if seeded, ok := readmeBlock(openingTitle, prose(page.Opening.Text)); ok {
		opening = append(opening, seeded)
		targets[0] = &seeded.ID
	}
	placed, rest := page.Sections, []readmeSection(nil)
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

func passages(sections []readmeSection) *block.Element {
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
