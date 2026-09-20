package upload

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// WaitingPicture is a picture from a README that waits for the creator to place it or let it go
type WaitingPicture struct {
	ID       uuid.UUID
	MediaID  *uuid.UUID
	Address  string
	Name     string
	BlockID  *uuid.UUID
	Section  string
	Width    int
	Height   int
	ThumbURL string
}

var ErrFoundImageNotFound = errors.New("no such picture is waiting in found images")

// ErrFoundImageNeedsCopy means a picture from another site needs an uploaded copy before it can be placed
var ErrFoundImageNeedsCopy = errors.New("upload your own copy of this picture before placing it")

func insertFoundImages(ctx context.Context, tx pgx.Tx, workID uuid.UUID, pictures []WaitingPicture) error {
	for position, picture := range pictures {
		if _, err := tx.Exec(ctx, `
			insert into work_found_images (id, work_id, media_id, address, name, block_id, section, position)
			values ($1, $2, $3, $4, $5, $6, $7, $8)
		`, picture.ID, workID, picture.MediaID, picture.Address, picture.Name, picture.BlockID, picture.Section, position); err != nil {
			return fmt.Errorf("record a found image: %w", err)
		}
	}
	return nil
}

// ListFoundImages lists the waiting pictures in the order the README showed them
func (s *Service) ListFoundImages(ctx context.Context, ownerID, workID uuid.UUID) ([]WaitingPicture, error) {
	var found uuid.UUID
	err := s.pool.QueryRow(ctx, `
		select id from works where id = $1 and owner_id = $2 and deleted_at is null
	`, workID, ownerID).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, work.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find the work: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
		select picture.id, picture.media_id, picture.address, picture.name, picture.block_id, picture.section,
		       coalesce(media.width, 0), coalesce(media.height, 0)
		  from work_found_images picture
		  left join work_media media on media.id = picture.media_id
		 where picture.work_id = $1
		 order by picture.position, picture.created_at
	`, workID)
	if err != nil {
		return nil, fmt.Errorf("list found images: %w", err)
	}
	defer rows.Close()
	pictures := []WaitingPicture{}
	for rows.Next() {
		var picture WaitingPicture
		if err := rows.Scan(
			&picture.ID, &picture.MediaID, &picture.Address, &picture.Name, &picture.BlockID, &picture.Section,
			&picture.Width, &picture.Height,
		); err != nil {
			return nil, fmt.Errorf("read a found image: %w", err)
		}
		if picture.MediaID != nil {
			picture.ThumbURL = s.works.ImageAddress(*picture.MediaID, "grid", false, true)
		}
		pictures = append(pictures, picture)
	}
	return pictures, rows.Err()
}

// PlaceFoundImage puts a waiting picture into the block its section became
func (s *Service) PlaceFoundImage(
	ctx context.Context, ownerID, workID, pictureID uuid.UUID, mediaID *uuid.UUID, candidate *work.Candidate,
) (work.SavedBlocks, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return work.SavedBlocks{}, err
	}
	defer tx.Rollback(ctx)

	workType, err := candidate.Lock(ctx, tx, ownerID, workID)
	if err != nil {
		return work.SavedBlocks{}, err
	}
	picture, err := lockFoundImage(ctx, tx, workID, pictureID)
	if err != nil {
		return work.SavedBlocks{}, err
	}
	if picture.MediaID == nil {
		if mediaID == nil {
			return work.SavedBlocks{}, ErrFoundImageNeedsCopy
		}
		if err := checkGalleryMedia(ctx, tx, workID, *mediaID); err != nil {
			return work.SavedBlocks{}, err
		}
		picture.MediaID = mediaID
	}
	after, err := s.placeInPage(ctx, tx, workType, workID, picture)
	if err != nil {
		return work.SavedBlocks{}, err
	}
	if err := candidate.Commit(ctx, tx, workID); err != nil {
		return work.SavedBlocks{}, err
	}
	return work.SavedBlocks{Type: workType, Blocks: after}, nil
}

func (s *Service) placeInPage(
	ctx context.Context, tx pgx.Tx, workType string, workID uuid.UUID, picture WaitingPicture,
) ([]block.Block, error) {
	page, err := block.Read(ctx, tx, workID)
	if err != nil {
		return nil, err
	}
	after, made, err := placePicture(page, picture)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", work.ErrInvalidBlock, err)
	}
	if made != nil {
		if _, err := tx.Exec(ctx, `
			update work_found_images set block_id = $3
			 where work_id = $1 and block_id is null and section = $2
		`, workID, picture.Section, *made); err != nil {
			return nil, fmt.Errorf("point the section's other pictures at its new block: %w", err)
		}
	}
	if err := block.ValidateBuilderConstraints(workType, after, after); err != nil {
		return nil, fmt.Errorf("%w: %v", work.ErrInvalidBlock, err)
	}
	if _, err := tx.Exec(ctx, `delete from work_blocks where work_id = $1`, workID); err != nil {
		return nil, fmt.Errorf("replace the blocks: %w", err)
	}
	if err := block.Insert(ctx, tx, workID, after); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `delete from work_found_images where id = $1`, picture.ID); err != nil {
		return nil, fmt.Errorf("take the picture out of found images: %w", err)
	}
	if err := s.writeSummary(ctx, tx, workID); err != nil {
		return nil, err
	}
	return after, nil
}

// DiscardFoundImage lets a waiting picture go along with the copy the archive gave it
func (s *Service) DiscardFoundImage(
	ctx context.Context, ownerID, workID, pictureID uuid.UUID, candidate *work.Candidate,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := candidate.Lock(ctx, tx, ownerID, workID); err != nil {
		return err
	}
	picture, err := lockFoundImage(ctx, tx, workID, pictureID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `delete from work_found_images where id = $1`, pictureID); err != nil {
		return fmt.Errorf("take the picture out of found images: %w", err)
	}
	if picture.MediaID != nil {
		if _, err := tx.Exec(ctx, `
			delete from work_media
			 where id = $1 and work_id = $2
			   and not exists (select 1 from work_found_images where media_id = $1)
		`, *picture.MediaID, workID); err != nil {
			return fmt.Errorf("let the picture go: %w", err)
		}
	}
	return candidate.Commit(ctx, tx, workID)
}

func lockFoundImage(ctx context.Context, tx pgx.Tx, workID, pictureID uuid.UUID) (WaitingPicture, error) {
	picture := WaitingPicture{ID: pictureID}
	err := tx.QueryRow(ctx, `
		select media_id, address, name, block_id, section
		  from work_found_images
		 where id = $1 and work_id = $2
		 for update
	`, pictureID, workID).Scan(&picture.MediaID, &picture.Address, &picture.Name, &picture.BlockID, &picture.Section)
	if errors.Is(err, pgx.ErrNoRows) {
		return WaitingPicture{}, ErrFoundImageNotFound
	}
	if err != nil {
		return WaitingPicture{}, fmt.Errorf("find the found image: %w", err)
	}
	return picture, nil
}

func checkGalleryMedia(ctx context.Context, tx pgx.Tx, workID, mediaID uuid.UUID) error {
	var found bool
	err := tx.QueryRow(ctx, `
		select exists (
			select 1 from work_media
			 where id = $1 and work_id = $2 and role = 'gallery' and is_current and blob_id is not null)
	`, mediaID, workID).Scan(&found)
	if err != nil {
		return fmt.Errorf("check the uploaded picture: %w", err)
	}
	if !found {
		return work.ErrMediaNotFound
	}
	return nil
}

// placePicture adds the picture to its block and reports a block it had to make
func placePicture(page []block.Block, picture WaitingPicture) ([]block.Block, *uuid.UUID, error) {
	item := block.ImageItem{ID: block.NewItemID(), MediaID: *picture.MediaID, Name: picture.Name}
	after := make([]block.Block, len(page))
	copy(after, page)
	at := -1
	if picture.BlockID != nil {
		for index, holder := range after {
			if holder.ID == *picture.BlockID && holder.Definition == block.CustomBlock {
				at = index
			}
		}
	}
	var made *uuid.UUID
	if at < 0 {
		title := picture.Section
		if title == "" {
			title = "Pictures"
		}
		id := uuid.New()
		after = append(after, block.Block{
			ID: id, Definition: block.CustomBlock, Title: &title,
			Layout: block.Single, Width: block.Full, Position: len(after),
		})
		at, made = len(after)-1, &id
	}
	holder := after[at]
	elements := make([]block.Element, len(holder.Elements))
	copy(elements, holder.Elements)
	placed := false
	for index, element := range elements {
		set, ok := element.Content.(block.ImageSet)
		if element.Type != block.TypeImageSet || !ok || element.Role != "" {
			continue
		}
		set.Images = append(append([]block.ImageItem{}, set.Images...), item)
		elements[index].Content = set
		placed = true
		break
	}
	if !placed {
		elements = append(elements, block.Element{
			ID: uuid.New(), Type: block.TypeImageSet, Options: block.Options{ItemSize: block.ItemMedium},
			Content: block.ImageSet{Images: []block.ImageItem{item}},
		})
	}
	layout := holder.Layout
	if len(elements) > len(layout.Slots()) {
		layout = layoutHolding(len(elements))
	}
	for index := range elements {
		elements[index].Slot = layout.Slots()[index]
	}
	holder.Elements, holder.Layout = elements, layout
	if holder.Width.Columns() < layout.MinimumWidth().Columns() {
		holder.Width = layout.MinimumWidth()
	}
	if err := block.ValidateStructure(holder); err != nil {
		return nil, nil, err
	}
	after[at] = holder
	return after, made, nil
}

func layoutHolding(count int) block.Layout {
	switch count {
	case 1:
		return block.Single
	case 2:
		return block.MainAside
	default:
		return block.Trio
	}
}
