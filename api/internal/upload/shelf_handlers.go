package upload

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// maxMarkdownBody leaves room for JSON escaping around the largest paste
const maxMarkdownBody = 4 * MaxPastedMarkdown

const tooMuchMarkdown = "Paste at most 1 MB of Markdown at a time."

const (
	maxUndoPieces = 500
	maxUndoBody   = 64 << 10
)

func (h *Handlers) ListShelf(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.SignedIn(c, "reading the shelf")
	if !ok {
		return
	}
	imports, err := h.uploads.ListShelf(c.Request.Context(), owner.ID, id)
	switch {
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such work.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Illarin could not read the shelf. Try again.")
	default:
		c.JSON(http.StatusOK, toAPIShelf(imports))
	}
}

func (h *Handlers) AddMarkdown(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "adding Markdown to the shelf")
	if !ok {
		return
	}
	var request AddMarkdownRequest
	if !api.ReadBoundedJSON(c, &request, maxMarkdownBody, tooMuchMarkdown) {
		return
	}
	imports, err := h.uploads.AddMarkdown(c.Request.Context(), owner.ID, id, request.Markdown)
	switch {
	case errors.Is(err, ErrShelfPasteTooLarge):
		api.Refuse(c, http.StatusRequestEntityTooLarge, tooMuchMarkdown)
	case errors.Is(err, ErrShelfPasteEmpty):
		api.Refuse(c, http.StatusBadRequest, "This Markdown has no text or pictures to add.")
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such work.")
	case errors.Is(err, work.ErrWorkFrozen):
		api.Refuse(c, http.StatusConflict, "This work was taken down, so it can't change.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Illarin could not add the Markdown. Try again.")
	default:
		c.JSON(http.StatusCreated, toAPIShelf(imports))
	}
}

func (h *Handlers) PlaceShelfPiece(c *gin.Context) {
	id, pieceID, candidate, owner, ok := shelfWrite(c, "placing a piece from the shelf", "pieceId")
	if !ok {
		return
	}
	var request PlaceShelfPieceRequest
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || (len(body) > 0 && api.DecodeOneJSON(bytes.NewReader(body), &request) != nil) {
		api.Refuse(c, http.StatusBadRequest, "Send where the piece goes, or nothing.")
		return
	}
	saved, err := h.uploads.PlaceShelfPiece(c.Request.Context(), owner, id, pieceID, Placement{
		MediaID: request.MediaId, Position: request.Position, ElementID: request.ElementId,
	}, candidate)
	if page.CandidateResult(c, candidate, err) {
		return
	}
	var refusal PlacementRefusal
	switch {
	case errors.Is(err, work.ErrNotFound), errors.Is(err, ErrShelfPieceNotFound):
		api.Refuse(c, http.StatusNotFound, "That piece isn't on the shelf anymore.")
	case errors.Is(err, ErrShelfNeedsCopy):
		api.Refuse(c, http.StatusBadRequest, "Upload your own copy of this picture before placing it.")
	case errors.As(err, &refusal):
		api.Refuse(c, http.StatusBadRequest, refusal.Error())
	case errors.Is(err, work.ErrInvalidBlock), errors.Is(err, work.ErrMediaNotFound):
		api.Refuse(c, http.StatusBadRequest, err.Error())
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Illarin could not place the piece. Try again.")
	default:
		answerBlocks(c, saved)
	}
}

func (h *Handlers) UndoShelfPlacement(c *gin.Context) {
	id, pieceID, candidate, owner, ok := shelfWrite(c, "undoing a placement", "pieceId")
	if !ok {
		return
	}
	saved, err := h.uploads.UndoShelfPlacements(c.Request.Context(), owner, id, []uuid.UUID{pieceID}, candidate)
	h.answerUndo(c, candidate, saved, err)
}

// UndoShelfPlacements takes back a batch of placements, such as a whole import placed at once
func (h *Handlers) UndoShelfPlacements(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	version, ok := api.DraftedChangesVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "undoing placements")
	if !ok {
		return
	}
	var request UndoShelfPlacementsRequest
	if !api.ReadBoundedJSON(c, &request, maxUndoBody, "Send at most 500 pieces to undo.") {
		return
	}
	if len(request.PieceIds) == 0 || len(request.PieceIds) > maxUndoPieces {
		api.Refuse(c, http.StatusBadRequest, "Send between 1 and 500 pieces to undo.")
		return
	}
	candidate := &work.Candidate{Version: version}
	saved, err := h.uploads.UndoShelfPlacements(c.Request.Context(), owner.ID, id, request.PieceIds, candidate)
	h.answerUndo(c, candidate, saved, err)
}

func (h *Handlers) answerUndo(c *gin.Context, candidate *work.Candidate, saved work.SavedBlocks, err error) {
	if page.CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, work.ErrNotFound), errors.Is(err, ErrShelfPieceNotFound):
		api.Refuse(c, http.StatusNotFound, "It's too late to undo this.")
	case errors.Is(err, ErrShelfUndoStale):
		api.Refuse(c, http.StatusConflict, "This block changed after you placed the piece, so it can't be undone.")
	case errors.Is(err, work.ErrInvalidBlock):
		api.Refuse(c, http.StatusBadRequest, err.Error())
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Illarin could not undo that. Try again.")
	default:
		answerBlocks(c, saved)
	}
}

// PlaceShelfImport makes every waiting section of an import its own block, in order
func (h *Handlers) PlaceShelfImport(c *gin.Context) {
	id, importID, candidate, owner, ok := shelfWrite(c, "placing an import", "importId")
	if !ok {
		return
	}
	saved, placed, err := h.uploads.PlaceShelfImport(c.Request.Context(), owner, id, importID, candidate)
	if page.CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, work.ErrNotFound), errors.Is(err, ErrShelfImportNotFound):
		api.Refuse(c, http.StatusNotFound, "Nothing from that import can be placed.")
	case errors.Is(err, work.ErrInvalidBlock):
		api.Refuse(c, http.StatusBadRequest, err.Error())
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Illarin could not place the import. Try again.")
	default:
		blocks, err := block.ToBlocks(saved.Type, saved.Blocks)
		if err != nil {
			api.Refuse(c, http.StatusInternalServerError, "Illarin could not read the page back. Reload it.")
			return
		}
		c.JSON(http.StatusOK, PlacedImport{Blocks: blocks, PieceIds: placed})
	}
}

func (h *Handlers) LetGoOfShelfPiece(c *gin.Context) {
	id, pieceID, candidate, owner, ok := shelfWrite(c, "letting go of a piece", "pieceId")
	if !ok {
		return
	}
	err := h.uploads.LetGoOfShelfPiece(c.Request.Context(), owner, id, pieceID, candidate)
	h.answerLetGo(c, candidate, err, ErrShelfPieceNotFound)
}

func (h *Handlers) LetGoOfShelfImport(c *gin.Context) {
	id, importID, candidate, owner, ok := shelfWrite(c, "letting go of an import", "importId")
	if !ok {
		return
	}
	err := h.uploads.LetGoOfShelfImport(c.Request.Context(), owner, id, importID, candidate)
	h.answerLetGo(c, candidate, err, ErrShelfImportNotFound)
}

func (h *Handlers) answerLetGo(c *gin.Context, candidate *work.Candidate, err, missing error) {
	if page.CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, work.ErrNotFound), errors.Is(err, missing):
		api.Refuse(c, http.StatusNotFound, "That isn't on the shelf anymore.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Illarin could not let that go. Try again.")
	default:
		c.Status(http.StatusNoContent)
	}
}

// shelfWrite reads the work, the piece or import, the drafted-changes version and the verified owner of a shelf write
func shelfWrite(c *gin.Context, action, param string) (uuid.UUID, uuid.UUID, *work.Candidate, uuid.UUID, bool) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return uuid.Nil, uuid.Nil, nil, uuid.Nil, false
	}
	target, ok := api.PathID(c, param)
	if !ok {
		return uuid.Nil, uuid.Nil, nil, uuid.Nil, false
	}
	version, ok := api.DraftedChangesVersion(c)
	if !ok {
		return uuid.Nil, uuid.Nil, nil, uuid.Nil, false
	}
	owner, ok := api.Verified(c, action)
	if !ok {
		return uuid.Nil, uuid.Nil, nil, uuid.Nil, false
	}
	return id, target, &work.Candidate{Version: version}, owner.ID, true
}

func answerBlocks(c *gin.Context, saved work.SavedBlocks) {
	blocks, err := block.ToBlocks(saved.Type, saved.Blocks)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Illarin could not read the page back. Reload it.")
		return
	}
	c.JSON(http.StatusOK, blocks)
}

func toAPIShelf(imports []WaitingImport) Shelf {
	shelf := Shelf{Imports: make([]ShelfImport, 0, len(imports))}
	for _, held := range imports {
		listed := ShelfImport{
			Id: held.ID, Source: held.Source, Title: held.Title, CreatedAt: held.CreatedAt,
			Pieces: make([]ShelfPiece, 0, len(held.Pieces)),
		}
		for _, piece := range held.Pieces {
			listed.Pieces = append(listed.Pieces, toAPIShelfPiece(piece))
		}
		shelf.Imports = append(shelf.Imports, listed)
	}
	return shelf
}

func toAPIShelfPiece(piece WaitingPiece) ShelfPiece {
	listed := ShelfPiece{
		Id: piece.ID, Kind: piece.Kind, Section: piece.Section, Text: piece.Text,
		Address: piece.Address, Name: piece.Name, BlockId: piece.BlockID,
	}
	if piece.MediaID != nil {
		listed.Media = &struct {
			Height   int       `json:"height"`
			Id       uuid.UUID `json:"id"`
			ThumbUrl string    `json:"thumbUrl"`
			Width    int       `json:"width"`
		}{Height: piece.Height, Id: *piece.MediaID, ThumbUrl: piece.ThumbURL, Width: piece.Width}
	}
	return listed
}
