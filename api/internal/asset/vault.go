package asset

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// VaultPicture is a picture a README showed, waiting for the creator to put it on the page or let it go.
type VaultPicture struct {
	ID      uuid.UUID
	MediaID *uuid.UUID
	Address string
	Name    string
	BlockID *uuid.UUID
	// Section titles the README section the picture came from, naming its block when none stands.
	Section string
	Width   int
	Height  int
	// ThumbURL shows an archived picture to its creator, and a picture from another site has none.
	ThumbURL string
}

var ErrVaultPictureNotFound = errors.New("no such picture is waiting in the vault")

// ErrVaultPictureNeedsMedia says a picture shown from another site cannot be placed until the creator uploads a copy.
var ErrVaultPictureNeedsMedia = errors.New("upload your own copy of this picture before placing it")

func insertVaultPictures(ctx context.Context, tx pgx.Tx, assetID uuid.UUID, pictures []VaultPicture) error {
	for position, picture := range pictures {
		if _, err := tx.Exec(ctx, `
			insert into asset_vault_pictures (id, asset_id, media_id, address, name, block_id, section, position)
			values ($1, $2, $3, $4, $5, $6, $7, $8)
		`, picture.ID, assetID, picture.MediaID, picture.Address, picture.Name, picture.BlockID, picture.Section, position); err != nil {
			return fmt.Errorf("record a vault picture: %w", err)
		}
	}
	return nil
}

// ListVault lists the pictures waiting for the creator, in the order the README showed them.
func (s *Service) ListVault(ctx context.Context, ownerID, assetID uuid.UUID) ([]VaultPicture, error) {
	var found uuid.UUID
	err := s.pool.QueryRow(ctx, `
		select id from assets where id = $1 and owner_id = $2 and deleted_at is null
	`, assetID, ownerID).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find the asset: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
		select picture.id, picture.media_id, picture.address, picture.name, picture.block_id, picture.section,
		       coalesce(media.width, 0), coalesce(media.height, 0)
		  from asset_vault_pictures picture
		  left join asset_media media on media.id = picture.media_id
		 where picture.asset_id = $1
		 order by picture.position, picture.created_at
	`, assetID)
	if err != nil {
		return nil, fmt.Errorf("list the vault: %w", err)
	}
	defer rows.Close()
	pictures := []VaultPicture{}
	for rows.Next() {
		var picture VaultPicture
		if err := rows.Scan(
			&picture.ID, &picture.MediaID, &picture.Address, &picture.Name, &picture.BlockID, &picture.Section,
			&picture.Width, &picture.Height,
		); err != nil {
			return nil, fmt.Errorf("read a vault picture: %w", err)
		}
		if picture.MediaID != nil {
			picture.ThumbURL = s.variantURL(*picture.MediaID, "grid", false, true)
		}
		pictures = append(pictures, picture)
	}
	return pictures, rows.Err()
}

// PlaceVaultPicture puts a waiting picture into the block its section became, and takes it out of the vault.
func (s *Service) PlaceVaultPicture(
	ctx context.Context, ownerID, assetID, pictureID uuid.UUID, mediaID *uuid.UUID, candidate *Candidate,
) (SavedBlocks, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SavedBlocks{}, err
	}
	defer tx.Rollback(ctx)

	kind, err := candidate.Lock(ctx, tx, ownerID, assetID)
	if err != nil {
		return SavedBlocks{}, err
	}
	picture, err := lockVaultPicture(ctx, tx, assetID, pictureID)
	if err != nil {
		return SavedBlocks{}, err
	}
	if picture.MediaID == nil {
		if mediaID == nil {
			return SavedBlocks{}, ErrVaultPictureNeedsMedia
		}
		if err := checkGalleryMedia(ctx, tx, assetID, *mediaID); err != nil {
			return SavedBlocks{}, err
		}
		picture.MediaID = mediaID
	}
	fingerprint, err := s.contentFingerprint(ctx, tx, assetID)
	if err != nil {
		return SavedBlocks{}, err
	}
	page, err := readBlocks(ctx, tx, assetID)
	if err != nil {
		return SavedBlocks{}, err
	}
	after, made, err := placePicture(page, picture)
	if err != nil {
		return SavedBlocks{}, fmt.Errorf("%w: %v", ErrInvalidBlock, err)
	}
	if made != nil {
		if _, err := tx.Exec(ctx, `
			update asset_vault_pictures set block_id = $3
			 where asset_id = $1 and block_id is null and section = $2
		`, assetID, picture.Section, *made); err != nil {
			return SavedBlocks{}, fmt.Errorf("point the section's other pictures at its new block: %w", err)
		}
	}
	if err := block.ValidateBuilderConstraints(kind, after, after); err != nil {
		return SavedBlocks{}, fmt.Errorf("%w: %v", ErrInvalidBlock, err)
	}
	if _, err := tx.Exec(ctx, `delete from asset_blocks where asset_id = $1`, assetID); err != nil {
		return SavedBlocks{}, fmt.Errorf("replace the blocks: %w", err)
	}
	if err := insertBlocks(ctx, tx, assetID, after); err != nil {
		return SavedBlocks{}, err
	}
	if _, err := tx.Exec(ctx, `delete from asset_vault_pictures where id = $1`, pictureID); err != nil {
		return SavedBlocks{}, fmt.Errorf("take the picture out of the vault: %w", err)
	}
	if err := s.writeProjections(ctx, tx, assetID); err != nil {
		return SavedBlocks{}, err
	}
	if err := s.moveContentGeneration(ctx, tx, assetID, fingerprint); err != nil {
		return SavedBlocks{}, err
	}
	if err := candidate.commit(ctx, tx, assetID); err != nil {
		return SavedBlocks{}, err
	}
	return SavedBlocks{Kind: kind, Blocks: after}, nil
}

// DiscardVaultPicture lets a waiting picture go, along with the copy the archive gave it.
func (s *Service) DiscardVaultPicture(
	ctx context.Context, ownerID, assetID, pictureID uuid.UUID, candidate *Candidate,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := candidate.Lock(ctx, tx, ownerID, assetID); err != nil {
		return err
	}
	picture, err := lockVaultPicture(ctx, tx, assetID, pictureID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `delete from asset_vault_pictures where id = $1`, pictureID); err != nil {
		return fmt.Errorf("take the picture out of the vault: %w", err)
	}
	if picture.MediaID != nil {
		if _, err := tx.Exec(ctx, `
			delete from asset_media
			 where id = $1 and asset_id = $2
			   and not exists (select 1 from asset_vault_pictures where media_id = $1)
		`, *picture.MediaID, assetID); err != nil {
			return fmt.Errorf("let the picture go: %w", err)
		}
	}
	return candidate.commit(ctx, tx, assetID)
}

func lockVaultPicture(ctx context.Context, tx pgx.Tx, assetID, pictureID uuid.UUID) (VaultPicture, error) {
	picture := VaultPicture{ID: pictureID}
	err := tx.QueryRow(ctx, `
		select media_id, address, name, block_id, section
		  from asset_vault_pictures
		 where id = $1 and asset_id = $2
		 for update
	`, pictureID, assetID).Scan(&picture.MediaID, &picture.Address, &picture.Name, &picture.BlockID, &picture.Section)
	if errors.Is(err, pgx.ErrNoRows) {
		return VaultPicture{}, ErrVaultPictureNotFound
	}
	if err != nil {
		return VaultPicture{}, fmt.Errorf("find the vault picture: %w", err)
	}
	return picture, nil
}

func checkGalleryMedia(ctx context.Context, tx pgx.Tx, assetID, mediaID uuid.UUID) error {
	var found bool
	err := tx.QueryRow(ctx, `
		select exists (
			select 1 from asset_media
			 where id = $1 and asset_id = $2 and role = 'gallery' and is_current and blob_id is not null)
	`, mediaID, assetID).Scan(&found)
	if err != nil {
		return fmt.Errorf("check the uploaded picture: %w", err)
	}
	if !found {
		return ErrMediaNotFound
	}
	return nil
}

// placePicture adds the picture to its block's image set, giving the block one when it has none, and says when it made the block.
func placePicture(page []block.Block, picture VaultPicture) ([]block.Block, *uuid.UUID, error) {
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

// layoutHolding picks the custom block layout that sets writing beside its pictures.
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
