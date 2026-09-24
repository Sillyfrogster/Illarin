package upload

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MaxPastedMarkdown is the most Markdown one paste can put on the shelf
const MaxPastedMarkdown = 1 << 20

// WaitingPiece is a section or a picture on the shelf, waiting for the creator to place it or let it go
type WaitingPiece struct {
	ID       uuid.UUID
	ImportID uuid.UUID
	Kind     ShelfPieceKind
	Section  string
	Text     string
	MediaID  *uuid.UUID
	Address  string
	Name     string
	BlockID  *uuid.UUID
	Width    int
	Height   int
	ThumbURL string
	// Heading names a placed section's block: its own heading, or the import's title for the opening
	Heading string
}

// WaitingImport is one paste or README and the pieces of it still on the shelf
type WaitingImport struct {
	ID        uuid.UUID
	Source    ShelfSource
	Title     string
	CreatedAt time.Time
	Pieces    []WaitingPiece
}

// Placement puts a section in a new block at Position or at the end of the text element ElementID
type Placement struct {
	MediaID   *uuid.UUID
	Position  *int
	ElementID *uuid.UUID
}

var (
	ErrShelfPieceNotFound  = errors.New("no such piece is on the shelf")
	ErrShelfImportNotFound = errors.New("no such import is on the shelf")
	// ErrShelfNeedsCopy means a picture from another site needs an uploaded copy before it can be placed
	ErrShelfNeedsCopy     = errors.New("the picture needs an uploaded copy")
	ErrShelfPasteTooLarge = errors.New("the Markdown is larger than a paste may be")
	ErrShelfPasteEmpty    = errors.New("the Markdown has no text or pictures")
	ErrShelfUndoStale     = errors.New("the placed block changed after the placement")
)

// PlacementRefusal says in the creator's words why a piece cannot go where they asked
type PlacementRefusal string

func (r PlacementRefusal) Error() string { return string(r) }

func insertShelf(
	ctx context.Context, tx pgx.Tx, workID uuid.UUID, source ShelfSource, title string, pieces []WaitingPiece,
) error {
	if len(pieces) == 0 {
		return nil
	}
	importID := uuid.New()
	if _, err := tx.Exec(ctx, `
		insert into work_shelf_imports (id, work_id, source, title) values ($1, $2, $3, $4)
	`, importID, workID, source, title); err != nil {
		return fmt.Errorf("record a shelf import: %w", err)
	}
	for position, piece := range pieces {
		if _, err := tx.Exec(ctx, `
			insert into work_shelf_pieces
				(id, work_id, import_id, kind, media_id, address, name, block_id, section, text, position)
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, piece.ID, workID, importID, piece.Kind, piece.MediaID, piece.Address, piece.Name, piece.BlockID,
			piece.Section, piece.Text, position); err != nil {
			return fmt.Errorf("put a piece on the shelf: %w", err)
		}
	}
	return nil
}

// ListShelf lists the imports with pieces still waiting, newest first, each in the order its Markdown had them
func (s *Service) ListShelf(ctx context.Context, ownerID, workID uuid.UUID) ([]WaitingImport, error) {
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
	imports, err := s.shelfImports(ctx, workID)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		select piece.id, piece.import_id, piece.kind, piece.section, piece.text, piece.media_id, piece.address,
		       piece.name, piece.block_id, coalesce(media.width, 0), coalesce(media.height, 0)
		  from work_shelf_pieces piece
		  left join work_media media on media.id = piece.media_id
		 where piece.work_id = $1 and piece.placed_at is null
		 order by piece.position, piece.created_at
	`, workID)
	if err != nil {
		return nil, fmt.Errorf("list the shelf: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var piece WaitingPiece
		if err := rows.Scan(
			&piece.ID, &piece.ImportID, &piece.Kind, &piece.Section, &piece.Text, &piece.MediaID, &piece.Address,
			&piece.Name, &piece.BlockID, &piece.Width, &piece.Height,
		); err != nil {
			return nil, fmt.Errorf("read a shelf piece: %w", err)
		}
		if piece.MediaID != nil {
			piece.ThumbURL = s.works.ImageAddress(*piece.MediaID, "grid", false, true)
		}
		at := slices.IndexFunc(imports, func(held WaitingImport) bool { return held.ID == piece.ImportID })
		if at >= 0 {
			imports[at].Pieces = append(imports[at].Pieces, piece)
		}
	}
	return imports, rows.Err()
}

func (s *Service) shelfImports(ctx context.Context, workID uuid.UUID) ([]WaitingImport, error) {
	rows, err := s.pool.Query(ctx, `
		select shelf_import.id, shelf_import.source, shelf_import.title, shelf_import.created_at
		  from work_shelf_imports shelf_import
		 where shelf_import.work_id = $1
		   and exists (select 1 from work_shelf_pieces piece
		                where piece.import_id = shelf_import.id and piece.placed_at is null)
		 order by shelf_import.created_at desc, shelf_import.id
	`, workID)
	if err != nil {
		return nil, fmt.Errorf("list the shelf imports: %w", err)
	}
	defer rows.Close()
	imports := []WaitingImport{}
	for rows.Next() {
		var held WaitingImport
		if err := rows.Scan(&held.ID, &held.Source, &held.Title, &held.CreatedAt); err != nil {
			return nil, fmt.Errorf("read a shelf import: %w", err)
		}
		imports = append(imports, held)
	}
	return imports, rows.Err()
}

// AddMarkdown splits pasted Markdown at its headings and puts every section and picture on the shelf
func (s *Service) AddMarkdown(ctx context.Context, ownerID, workID uuid.UUID, markdown string) ([]WaitingImport, error) {
	if len(markdown) > MaxPastedMarkdown {
		return nil, ErrShelfPasteTooLarge
	}
	page := readPaste(markdown)
	_, pieces, err := s.readmePieces(ctx, format.Inspection{}, page, nil, make([]*uuid.UUID, 1+len(page.Sections)))
	if err != nil {
		return nil, err
	}
	if len(pieces) == 0 {
		return nil, ErrShelfPasteEmpty
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := work.LockEditable(ctx, tx, ownerID, workID); err != nil {
		return nil, err
	}
	if err := sweepShelf(ctx, tx, workID); err != nil {
		return nil, err
	}
	if err := insertShelf(ctx, tx, workID, ShelfSourcePasted, page.Title, pieces); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.ListShelf(ctx, ownerID, workID)
}

// PlaceShelfPiece moves a waiting piece onto the page and remembers what it changed so the creator can undo it
func (s *Service) PlaceShelfPiece(
	ctx context.Context, ownerID, workID, pieceID uuid.UUID, placement Placement, candidate *work.Candidate,
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
	if err := sweepShelf(ctx, tx, workID); err != nil {
		return work.SavedBlocks{}, err
	}
	piece, err := lockShelfPiece(ctx, tx, workID, pieceID, false)
	if err != nil {
		return work.SavedBlocks{}, err
	}
	page, err := block.Read(ctx, tx, workID)
	if err != nil {
		return work.SavedBlocks{}, err
	}
	after, changed, err := s.placePiece(ctx, tx, workID, page, piece, placement)
	if err != nil {
		return work.SavedBlocks{}, err
	}
	if err := s.writePage(ctx, tx, workType, workID, after); err != nil {
		return work.SavedBlocks{}, err
	}
	if err := rememberPlacement(ctx, tx, piece, page, after, changed); err != nil {
		return work.SavedBlocks{}, err
	}
	if err := candidate.Commit(ctx, tx, workID); err != nil {
		return work.SavedBlocks{}, err
	}
	return work.SavedBlocks{Type: workType, Blocks: after}, nil
}

func (s *Service) placePiece(
	ctx context.Context, tx pgx.Tx, workID uuid.UUID, page []block.Block, piece WaitingPiece, placement Placement,
) ([]block.Block, uuid.UUID, error) {
	if piece.Kind == ShelfPieceKindSection {
		after, changed, err := placeSection(page, piece, placement)
		if err != nil {
			return nil, uuid.Nil, err
		}
		if placement.Position != nil {
			err = pointSectionPictures(ctx, tx, workID, piece, changed)
		}
		return after, changed, err
	}
	if placement.Position != nil || placement.ElementID != nil {
		return nil, uuid.Nil, PlacementRefusal("A picture goes into its section's block.")
	}
	if piece.MediaID == nil {
		if placement.MediaID == nil {
			return nil, uuid.Nil, ErrShelfNeedsCopy
		}
		if err := checkGalleryMedia(ctx, tx, workID, *placement.MediaID); err != nil {
			return nil, uuid.Nil, err
		}
		piece.MediaID = placement.MediaID
	}
	after, changed, made, err := placePicture(page, piece)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("%w: %v", work.ErrInvalidBlock, err)
	}
	if made {
		err = pointSectionPictures(ctx, tx, workID, piece, changed)
	}
	return after, changed, err
}

// pointSectionPictures sends the section's other waiting pictures to the block the section now has
func pointSectionPictures(ctx context.Context, tx pgx.Tx, workID uuid.UUID, piece WaitingPiece, blockID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `
		update work_shelf_pieces set block_id = $4
		 where work_id = $1 and import_id = $2 and section = $3 and kind = 'picture'
		   and block_id is null and placed_at is null
	`, workID, piece.ImportID, piece.Section, blockID); err != nil {
		return fmt.Errorf("point the section's pictures at its block: %w", err)
	}
	return nil
}

// UndoShelfPlacements puts placed pieces back on the shelf, newest first, when every block is exactly as its placement left it
func (s *Service) UndoShelfPlacements(
	ctx context.Context, ownerID, workID uuid.UUID, pieceIDs []uuid.UUID, candidate *work.Candidate,
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
	if err := sweepShelf(ctx, tx, workID); err != nil {
		return work.SavedBlocks{}, err
	}
	page, err := block.Read(ctx, tx, workID)
	if err != nil {
		return work.SavedBlocks{}, err
	}
	for index := len(pieceIDs) - 1; index >= 0; index-- {
		if page, err = undoPlacement(ctx, tx, workID, pieceIDs[index], page); err != nil {
			return work.SavedBlocks{}, err
		}
	}
	if err := s.writePage(ctx, tx, workType, workID, page); err != nil {
		return work.SavedBlocks{}, err
	}
	if err := candidate.Commit(ctx, tx, workID); err != nil {
		return work.SavedBlocks{}, err
	}
	return work.SavedBlocks{Type: workType, Blocks: page}, nil
}

func undoPlacement(ctx context.Context, tx pgx.Tx, workID, pieceID uuid.UUID, page []block.Block) ([]block.Block, error) {
	if _, err := lockShelfPiece(ctx, tx, workID, pieceID, true); err != nil {
		return nil, err
	}
	var blockID uuid.UUID
	var before, placed []byte
	if err := tx.QueryRow(ctx, `
		select placed_block_id, placed_before, placed_after from work_shelf_pieces where id = $1
	`, pieceID).Scan(&blockID, &before, &placed); err != nil {
		return nil, fmt.Errorf("read the placement: %w", err)
	}
	after, err := revertPlacement(page, blockID, before, placed)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		update work_shelf_pieces
		   set placed_at = null, placed_block_id = null, placed_before = null, placed_after = null
		 where id = $1
	`, pieceID); err != nil {
		return nil, fmt.Errorf("put the piece back on the shelf: %w", err)
	}
	return after, nil
}

// PlaceShelfImport makes every waiting section of an import its own block at the end of the page, in order,
// and puts each uploaded picture into its section's block
func (s *Service) PlaceShelfImport(
	ctx context.Context, ownerID, workID, importID uuid.UUID, candidate *work.Candidate,
) (work.SavedBlocks, []uuid.UUID, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return work.SavedBlocks{}, nil, err
	}
	defer tx.Rollback(ctx)

	workType, err := candidate.Lock(ctx, tx, ownerID, workID)
	if err != nil {
		return work.SavedBlocks{}, nil, err
	}
	if err := sweepShelf(ctx, tx, workID); err != nil {
		return work.SavedBlocks{}, nil, err
	}
	rows, err := tx.Query(ctx, `
		select id from work_shelf_pieces
		 where work_id = $1 and import_id = $2 and placed_at is null
		   and (kind = 'section' or media_id is not null)
		 order by position, created_at
	`, workID, importID)
	if err != nil {
		return work.SavedBlocks{}, nil, fmt.Errorf("list the import's pieces: %w", err)
	}
	waiting, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return work.SavedBlocks{}, nil, fmt.Errorf("read the import's pieces: %w", err)
	}
	if len(waiting) == 0 {
		return work.SavedBlocks{}, nil, ErrShelfImportNotFound
	}
	page, err := block.Read(ctx, tx, workID)
	if err != nil {
		return work.SavedBlocks{}, nil, err
	}
	for _, pieceID := range waiting {
		piece, err := lockShelfPiece(ctx, tx, workID, pieceID, false)
		if err != nil {
			return work.SavedBlocks{}, nil, err
		}
		end := len(page)
		after, changed, err := s.placePiece(ctx, tx, workID, page, piece, sectionAt(piece, &end))
		if err != nil {
			return work.SavedBlocks{}, nil, err
		}
		if err := rememberPlacement(ctx, tx, piece, page, after, changed); err != nil {
			return work.SavedBlocks{}, nil, err
		}
		page = after
	}
	if err := s.writePage(ctx, tx, workType, workID, page); err != nil {
		return work.SavedBlocks{}, nil, err
	}
	if err := candidate.Commit(ctx, tx, workID); err != nil {
		return work.SavedBlocks{}, nil, err
	}
	return work.SavedBlocks{Type: workType, Blocks: page}, waiting, nil
}

// sectionAt sends a section to the end of the page and leaves a picture to find its own block
func sectionAt(piece WaitingPiece, end *int) Placement {
	if piece.Kind == ShelfPieceKindSection {
		return Placement{Position: end}
	}
	return Placement{}
}

// LetGoOfShelfPiece deletes a waiting piece and the stored copy of its picture
func (s *Service) LetGoOfShelfPiece(
	ctx context.Context, ownerID, workID, pieceID uuid.UUID, candidate *work.Candidate,
) error {
	return s.letGo(ctx, ownerID, workID, candidate, func(tx pgx.Tx) (pgx.Rows, error) {
		if _, err := lockShelfPiece(ctx, tx, workID, pieceID, false); err != nil {
			return nil, err
		}
		return tx.Query(ctx, `delete from work_shelf_pieces where id = $1 returning media_id`, pieceID)
	})
}

// LetGoOfShelfImport deletes every piece of one import still waiting on the shelf
func (s *Service) LetGoOfShelfImport(
	ctx context.Context, ownerID, workID, importID uuid.UUID, candidate *work.Candidate,
) error {
	return s.letGo(ctx, ownerID, workID, candidate, func(tx pgx.Tx) (pgx.Rows, error) {
		var found uuid.UUID
		err := tx.QueryRow(ctx, `
			select id from work_shelf_imports where id = $1 and work_id = $2 for update
		`, importID, workID).Scan(&found)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrShelfImportNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("find the shelf import: %w", err)
		}
		return tx.Query(ctx, `
			delete from work_shelf_pieces where import_id = $1 and placed_at is null returning media_id
		`, importID)
	})
}

func (s *Service) letGo(
	ctx context.Context, ownerID, workID uuid.UUID, candidate *work.Candidate, remove func(pgx.Tx) (pgx.Rows, error),
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := candidate.Lock(ctx, tx, ownerID, workID); err != nil {
		return err
	}
	rows, err := remove(tx)
	if err != nil {
		return err
	}
	media, err := pgx.CollectRows(rows, pgx.RowTo[*uuid.UUID])
	if err != nil {
		return fmt.Errorf("take the pieces off the shelf: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		delete from work_media
		 where work_id = $1 and id = any($2)
		   and not exists (select 1 from work_shelf_pieces where media_id = work_media.id)
	`, workID, media); err != nil {
		return fmt.Errorf("let the pictures go: %w", err)
	}
	if err := sweepShelf(ctx, tx, workID); err != nil {
		return err
	}
	return candidate.Commit(ctx, tx, workID)
}

// sweepShelf forgets placements too old to undo and imports with nothing left in them
func sweepShelf(ctx context.Context, tx pgx.Tx, workID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `
		delete from work_shelf_pieces where work_id = $1 and placed_at < now() - interval '1 hour'
	`, workID); err != nil {
		return fmt.Errorf("forget old placements: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		delete from work_shelf_imports shelf_import
		 where work_id = $1
		   and not exists (select 1 from work_shelf_pieces piece where piece.import_id = shelf_import.id)
	`, workID); err != nil {
		return fmt.Errorf("forget empty imports: %w", err)
	}
	return nil
}

func lockShelfPiece(ctx context.Context, tx pgx.Tx, workID, pieceID uuid.UUID, placed bool) (WaitingPiece, error) {
	piece := WaitingPiece{ID: pieceID}
	err := tx.QueryRow(ctx, `
		select piece.import_id, piece.kind, piece.section, piece.text, piece.media_id, piece.address, piece.name,
		       piece.block_id, coalesce(nullif(piece.section, ''), shelf_import.title)
		  from work_shelf_pieces piece
		  join work_shelf_imports shelf_import on shelf_import.id = piece.import_id
		 where piece.id = $1 and piece.work_id = $2 and (piece.placed_at is not null) = $3
		 for update of piece
	`, pieceID, workID, placed).Scan(
		&piece.ImportID, &piece.Kind, &piece.Section, &piece.Text, &piece.MediaID, &piece.Address, &piece.Name,
		&piece.BlockID, &piece.Heading)
	if errors.Is(err, pgx.ErrNoRows) {
		return WaitingPiece{}, ErrShelfPieceNotFound
	}
	if err != nil {
		return WaitingPiece{}, fmt.Errorf("find the shelf piece: %w", err)
	}
	return piece, nil
}

func (s *Service) writePage(ctx context.Context, tx pgx.Tx, workType string, workID uuid.UUID, after []block.Block) error {
	if err := block.ValidateBuilderConstraints(workType, after, after); err != nil {
		return fmt.Errorf("%w: %v", work.ErrInvalidBlock, err)
	}
	if _, err := tx.Exec(ctx, `delete from work_blocks where work_id = $1`, workID); err != nil {
		return fmt.Errorf("replace the blocks: %w", err)
	}
	if err := block.Insert(ctx, tx, workID, after); err != nil {
		return err
	}
	return s.writeSummary(ctx, tx, workID)
}

func rememberPlacement(
	ctx context.Context, tx pgx.Tx, piece WaitingPiece, page, after []block.Block, changed uuid.UUID,
) error {
	var before []byte
	if at := blockIndex(page, changed); at >= 0 {
		var err error
		if before, err = blockJSON(page[at]); err != nil {
			return err
		}
	}
	placed, err := blockJSON(after[blockIndex(after, changed)])
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		update work_shelf_pieces
		   set placed_at = now(), placed_block_id = $2, placed_before = $3, placed_after = $4
		 where id = $1
	`, piece.ID, changed, before, placed); err != nil {
		return fmt.Errorf("take the piece off the shelf: %w", err)
	}
	return nil
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

// placeSection makes a section a new block at a position, or adds its text to the end of a text element
func placeSection(page []block.Block, piece WaitingPiece, placement Placement) ([]block.Block, uuid.UUID, error) {
	switch {
	case (placement.Position == nil) == (placement.ElementID == nil):
		return nil, uuid.Nil, PlacementRefusal("Choose a place on the page or a text element, not both.")
	case placement.Position != nil:
		at := *placement.Position
		if at < 0 || at > len(page) {
			return nil, uuid.Nil, PlacementRefusal("That place isn't on the page anymore. Reload and try again.")
		}
		made, _ := readmeBlock(piece.Heading, prose(piece.Text))
		after := slices.Insert(slices.Clone(page), at, made)
		for index := range after {
			after[index].Position = index
		}
		return after, made.ID, nil
	}
	for at, holder := range page {
		index := slices.IndexFunc(holder.Elements, func(element block.Element) bool {
			return element.ID == *placement.ElementID
		})
		if index < 0 {
			continue
		}
		written, ok := holder.Elements[index].Content.(block.Prose)
		if holder.Elements[index].Type != block.TypeProse || !ok {
			return nil, uuid.Nil, PlacementRefusal("A section only goes into a text element.")
		}
		if written.Text != "" {
			written.Text += "\n\n"
		}
		written.Text += piece.Text
		after := slices.Clone(page)
		after[at].Elements = slices.Clone(holder.Elements)
		after[at].Elements[index].Content = written
		return after, holder.ID, nil
	}
	return nil, uuid.Nil, PlacementRefusal("That element isn't on the page anymore. Reload and try again.")
}

// revertPlacement restores the placed block as it was before, or removes it when the placement made it
func revertPlacement(page []block.Block, blockID uuid.UUID, before, placed []byte) ([]block.Block, error) {
	at := blockIndex(page, blockID)
	if at < 0 {
		return nil, ErrShelfUndoStale
	}
	var stored block.Block
	if err := json.Unmarshal(placed, &stored); err != nil {
		return nil, fmt.Errorf("read the placed block: %w", err)
	}
	now, err := blockJSON(page[at])
	if err != nil {
		return nil, err
	}
	if left, err := blockJSON(stored); err != nil || string(left) != string(now) {
		return nil, ErrShelfUndoStale
	}
	if before == nil {
		after := slices.Delete(slices.Clone(page), at, at+1)
		for index := range after {
			after[index].Position = index
		}
		return after, nil
	}
	var restored block.Block
	if err := json.Unmarshal(before, &restored); err != nil {
		return nil, fmt.Errorf("read the block as it was: %w", err)
	}
	restored.Position = page[at].Position
	after := slices.Clone(page)
	after[at] = restored
	return after, nil
}

func blockIndex(page []block.Block, id uuid.UUID) int {
	return slices.IndexFunc(page, func(holder block.Block) bool { return holder.ID == id })
}

// blockJSON writes a block without its position, which moving other blocks changes
func blockJSON(holder block.Block) ([]byte, error) {
	holder.Position = 0
	encoded, err := json.Marshal(holder)
	if err != nil {
		return nil, fmt.Errorf("write the placed block: %w", err)
	}
	return encoded, nil
}

// placePicture adds the picture to its block and reports the block it changed and whether it had to make it
func placePicture(page []block.Block, piece WaitingPiece) ([]block.Block, uuid.UUID, bool, error) {
	item := block.ImageItem{ID: block.NewItemID(), MediaID: *piece.MediaID, Name: piece.Name}
	after := slices.Clone(page)
	at := -1
	if piece.BlockID != nil {
		at = slices.IndexFunc(after, func(holder block.Block) bool {
			return holder.ID == *piece.BlockID && holder.Definition == block.CustomBlock
		})
	}
	made := at < 0
	if made {
		title := piece.Heading
		if title == "" {
			title = "Pictures"
		}
		after = append(after, block.Block{
			ID: uuid.New(), Definition: block.CustomBlock, Title: &title,
			Layout: block.Single, Width: block.Full, Position: len(after),
		})
		at = len(after) - 1
	}
	holder := after[at]
	elements := slices.Clone(holder.Elements)
	placed := false
	for index, element := range elements {
		set, ok := element.Content.(block.ImageSet)
		if element.Type != block.TypeImageSet || !ok || element.Role != "" {
			continue
		}
		set.Images = append(slices.Clone(set.Images), item)
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
		return nil, uuid.Nil, false, err
	}
	after[at] = holder
	return after, holder.ID, made, nil
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
