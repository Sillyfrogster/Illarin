package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

func (h *Handlers) ListDistinctions(c *gin.Context) {
	if _, ok := h.publicationAuthority(c, "reading distinctions"); !ok {
		return
	}
	defined, err := h.publications.Distinctions(c.Request.Context())
	if err != nil {
		h.distinctionError(c, err)
		return
	}
	c.JSON(http.StatusOK, DistinctionList{Definitions: toAPIDistinctions(defined)})
}

func (h *Handlers) DefineDistinction(c *gin.Context) {
	authority, ok := h.publicationAuthority(c, "defining a distinction")
	if !ok {
		return
	}
	var request DefineDistinctionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the definition as JSON."})
		return
	}
	defined, err := h.publications.Define(c.Request.Context(), authority.ID, publication.DistinctionEdit{
		Form:        publication.Form(request.Form),
		Name:        request.Name,
		Explanation: valueOrEmpty(request.Explanation),
	})
	if err != nil {
		h.distinctionError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toAPIDistinction(defined))
}

func (h *Handlers) UpdateDistinction(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "changing a distinction")
	if !ok {
		return
	}
	var request UpdateDistinctionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the change as JSON."})
		return
	}
	updated, err := h.publications.Update(
		c.Request.Context(), authority.ID, uuid.UUID(id), publication.DistinctionUpdate{
			Name:        request.Name,
			Explanation: request.Explanation,
			Retired:     request.Retired,
		},
	)
	if err != nil {
		h.distinctionError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIDistinction(updated))
}

func (h *Handlers) OrderDistinctions(c *gin.Context) {
	authority, ok := h.publicationAuthority(c, "ordering distinctions")
	if !ok {
		return
	}
	var request OrderDistinctionsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the order as JSON."})
		return
	}
	ordered, err := h.publications.Order(
		c.Request.Context(), authority.ID,
		publication.Form(request.Form), toUUIDs(request.DistinctionIds),
	)
	if err != nil {
		h.distinctionError(c, err)
		return
	}
	c.JSON(http.StatusOK, DistinctionList{Definitions: toAPIDistinctions(ordered)})
}

func (h *Handlers) SetDistinctionMark(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "changing a badge mark")
	if !ok {
		return
	}
	parts, err := c.Request.MultipartReader()
	if err != nil {
		h.refuseProfile(c, refusal{reason: "send the image as form data", cause: err})
		return
	}
	file, err := nextPart(parts, filePart)
	if err != nil {
		h.refuseProfile(c, err)
		return
	}
	limitedFile := http.MaxBytesReader(c.Writer, file, h.maxUploadBytes)
	defer limitedFile.Close()
	marked, err := h.publications.SetMark(c.Request.Context(), authority.ID, uuid.UUID(id), limitedFile)
	if errors.Is(err, publication.ErrNotFound) || errors.Is(err, publication.ErrDistinctionForm) {
		h.distinctionError(c, err)
		return
	}
	if err != nil {
		h.refuseProfile(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIDistinction(marked))
}

func (h *Handlers) ClearDistinctionMark(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "removing a badge mark")
	if !ok {
		return
	}
	cleared, err := h.publications.ClearMark(c.Request.Context(), authority.ID, uuid.UUID(id))
	if err != nil {
		h.distinctionError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIDistinction(cleared))
}

func (h *Handlers) ListAccountDistinctions(c *gin.Context, handle string) {
	if _, ok := h.publicationAuthority(c, "reading an account's distinctions"); !ok {
		return
	}
	held, err := h.publications.Assignments(c.Request.Context(), handle)
	if err != nil {
		h.distinctionError(c, err)
		return
	}
	c.JSON(http.StatusOK, DistinctionAssignmentList{
		Handle:      handle,
		Assignments: toAPIAssignments(held),
	})
}

func (h *Handlers) AssignDistinction(c *gin.Context, handle string) {
	authority, ok := h.publicationAuthority(c, "assigning a distinction")
	if !ok {
		return
	}
	var request AssignDistinctionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the distinction as JSON."})
		return
	}
	made, err := h.publications.Assign(
		c.Request.Context(), authority.ID, handle, uuid.UUID(request.DistinctionId),
	)
	if err != nil {
		h.distinctionError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toAPIAssignment(made))
}

func (h *Handlers) OrderAccountDistinctions(c *gin.Context, handle string) {
	authority, ok := h.publicationAuthority(c, "ordering an account's distinctions")
	if !ok {
		return
	}
	var request OrderAssignmentsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the order as JSON."})
		return
	}
	ordered, err := h.publications.OrderAssignments(
		c.Request.Context(), authority.ID, handle, toUUIDs(request.AssignmentIds),
	)
	if err != nil {
		h.distinctionError(c, err)
		return
	}
	c.JSON(http.StatusOK, DistinctionAssignmentList{
		Handle:      handle,
		Assignments: toAPIAssignments(ordered),
	})
}

func (h *Handlers) RemoveAccountDistinction(c *gin.Context, handle string, assignmentId types.UUID) {
	authority, ok := h.publicationAuthority(c, "removing a distinction")
	if !ok {
		return
	}
	err := h.publications.Withdraw(
		c.Request.Context(), authority.ID, handle, uuid.UUID(assignmentId),
	)
	if err != nil {
		h.distinctionError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// publicationAuthority answers the signed-in account only when it holds it
func (h *Handlers) publicationAuthority(c *gin.Context, action string) (accountIdentity, bool) {
	current, ok := h.verifiedAccount(c, action)
	if !ok {
		return accountIdentity{}, false
	}
	held, err := h.publications.HoldsAuthority(c.Request.Context(), current.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not check publication authority."})
		return accountIdentity{}, false
	}
	if !held {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only Illarin's publication authority can do that.",
		})
		return accountIdentity{}, false
	}
	return accountIdentity{ID: current.ID, Handle: current.Handle}, true
}

// accountIdentity is the little a distinction change needs about its actor.
type accountIdentity struct {
	ID     uuid.UUID
	Handle string
}

func (h *Handlers) distinctionError(c *gin.Context, err error) {
	var field publication.FieldError
	switch {
	case errors.As(err, &field):
		c.JSON(http.StatusBadRequest, gin.H{"error": field.Message, "field": field.Field})
	case errors.Is(err, publication.ErrAccountNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such account."})
	case errors.Is(err, publication.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such distinction."})
	case errors.Is(err, publication.ErrAlreadyAssigned):
		c.JSON(http.StatusConflict, gin.H{"error": "That account already holds this distinction."})
	case errors.Is(err, publication.ErrRetiredAssigning):
		c.JSON(http.StatusBadRequest, gin.H{"error": "A retired distinction cannot be assigned."})
	case errors.Is(err, publication.ErrDistinctionForm):
		c.JSON(http.StatusBadRequest, gin.H{"error": "A position carries no mark."})
	case errors.Is(err, publication.ErrIncompleteOrder):
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Name every one of them exactly once to set the order.",
		})
	case errors.Is(err, storage.ErrInsufficientSpace):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Uploads are temporarily unavailable because storage is low.",
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not change the distinction."})
	}
}

func toUUIDs(given []types.UUID) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(given))
	for _, id := range given {
		ids = append(ids, uuid.UUID(id))
	}
	return ids
}

func toAPIDistinctions(defined []publication.Distinction) []Distinction {
	listed := make([]Distinction, 0, len(defined))
	for _, one := range defined {
		listed = append(listed, toAPIDistinction(one))
	}
	return listed
}

func toAPIDistinction(found publication.Distinction) Distinction {
	return Distinction{
		Id:          types.UUID(found.ID),
		Form:        DistinctionForm(found.Form),
		Name:        found.Name,
		Explanation: found.Explanation,
		Mark:        toAPIMark(found.Mark),
		Position:    found.Position,
		Retired:     found.Retired,
	}
}

func toAPIProfileDistinctions(shown []publication.Distinction) []ProfileDistinction {
	listed := make([]ProfileDistinction, 0, len(shown))
	for _, one := range shown {
		listed = append(listed, ProfileDistinction{
			Id:          types.UUID(one.ID),
			Name:        one.Name,
			Explanation: one.Explanation,
			Mark:        toAPIMark(one.Mark),
		})
	}
	return listed
}

func toAPIMark(mark *publication.Mark) *DistinctionMark {
	if mark == nil {
		return nil
	}
	return &DistinctionMark{
		Url:    publication.MarkURL(mark.MediaID, mark.DerivativeVersion),
		Width:  mark.Width,
		Height: mark.Height,
	}
}

func toAPIAssignments(held []publication.Assignment) []DistinctionAssignment {
	listed := make([]DistinctionAssignment, 0, len(held))
	for _, one := range held {
		listed = append(listed, toAPIAssignment(one))
	}
	return listed
}

func toAPIAssignment(held publication.Assignment) DistinctionAssignment {
	assignment := DistinctionAssignment{
		Id:          types.UUID(held.ID),
		Distinction: toAPIDistinction(held.Distinction),
		Source:      held.Source,
		AssignedAt:  held.AssignedAt,
		Active:      held.Active,
		Position:    held.Position,
	}
	if held.IssuedBy != nil {
		issuer := held.IssuedBy.String()
		assignment.IssuedBy = &issuer
	}
	return assignment
}
