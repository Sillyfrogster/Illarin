package upload

import (
	"context"
	"errors"
	"slices"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

const (
	openingTitle  = "About"
	shownSections = 3
)

// seedFromReadme turns a README into blocks and puts its later sections and its pictures on the shelf when a work is first made
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
	prepared, pieces, err := s.readmePieces(ctx, file, page, held, targets)
	if err != nil {
		return preparedImport{}, err
	}
	read.Blocks = blocks
	read.Media = append(read.Media, prepared...)
	read.Shelf = pieces
	return read, nil
}

func archivedImages(file format.Inspection) map[string]uint32 {
	held := make(map[string]uint32, len(file.Images))
	for _, image := range file.Images {
		if image.Location.Container == format.ZIP {
			held[image.Location.Name] = image.ID
		}
	}
	return held
}

// readmePieces shelves each section that has no block and each picture, remembering the block its section became
func (s *Service) readmePieces(
	ctx context.Context, file format.Inspection, page readmePage, held map[string]uint32, targets []*uuid.UUID,
) ([]work.PreparedMedia, []WaitingPiece, error) {
	var prepared []work.PreparedMedia
	var pieces []WaitingPiece
	if page.Cover != nil {
		cover, ok, err := s.seededPicture(ctx, file, held[page.Cover.Entry], work.MediaAvatar)
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
		if target == nil && section.Text != "" {
			pieces = append(pieces, WaitingPiece{
				ID: uuid.New(), Kind: ShelfPieceKindSection, Section: section.Title, Text: section.Text,
			})
		}
		for _, image := range section.Images {
			id, done := stored[image.Entry]
			if !done {
				picture, ok, err := s.seededPicture(ctx, file, held[image.Entry], work.MediaGallery)
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
			pieces = append(pieces, WaitingPiece{
				ID: uuid.New(), Kind: ShelfPieceKindPicture, MediaID: &media, Address: image.Entry, Name: image.Name,
				BlockID: target, Section: section.Title,
			})
		}
		for _, picture := range section.Remote {
			pieces = append(pieces, WaitingPiece{
				ID: uuid.New(), Kind: ShelfPieceKindPicture, Address: picture.Address, Name: picture.Name,
				BlockID: target, Section: section.Title,
			})
		}
	}
	return prepared, pieces, nil
}

func (s *Service) seededPicture(
	ctx context.Context, file format.Inspection, image uint32, role work.MediaRole,
) (work.PreparedMedia, bool, error) {
	prepared, err := s.works.PrepareExtractedMedia(ctx, file, []format.Media{{Role: role, ImageID: image}})
	if errors.Is(err, mediaproc.ErrImageTooLarge) {
		return work.PreparedMedia{}, false, nil
	}
	if err != nil || len(prepared) == 0 {
		return work.PreparedMedia{}, false, err
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
	for index, section := range page.Sections[:min(shownSections, len(page.Sections))] {
		if seeded, ok := readmeBlock(section.Title, prose(section.Text)); ok {
			sections = append(sections, seeded)
			targets[1+index] = &seeded.ID
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
