package integration

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) ListWorkUpdateDestinations(c *gin.Context) {
	owner, ok := api.SignedIn(c, "reading your update destinations")
	if !ok {
		return
	}
	found, err := h.updateDestinations.List(c.Request.Context(), owner.ID)
	if err != nil {
		h.workDestinationError(c, err)
		return
	}
	listed := make([]WorkUpdateDestination, 0, len(found))
	for _, one := range found {
		listed = append(listed, toWorkUpdateDestination(one))
	}
	c.JSON(http.StatusOK, WorkUpdateDestinationList{Destinations: listed})
}

func (h *Handlers) GetWorkUpdateDestination(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.SignedIn(c, "reading your update destination")
	if !ok {
		return
	}
	found, err := h.updateDestinations.Get(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.workDestinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toWorkUpdateDestination(found))
}

func (h *Handlers) AddWorkUpdateDestination(c *gin.Context) {
	owner, ok := api.Verified(c, "configuring update destinations")
	if !ok {
		return
	}
	var request AddWorkUpdateDestinationRequest
	if !api.ReadBoundedJSON(c, &request, 4096, "The destination configuration is too large.") {
		return
	}
	if request.Address == nil {
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid, "Enter the destination address.", "address")
		return
	}
	added, err := h.updateDestinations.Add(c.Request.Context(), owner.ID, string(request.Type), request.Name, *request.Address)
	if err != nil {
		h.workDestinationError(c, err)
		return
	}
	answer := AddedWorkUpdateDestination{Destination: toWorkUpdateDestination(added.Destination)}
	if added.Secret != "" {
		answer.Secret = &added.Secret
	}
	c.JSON(http.StatusCreated, answer)
}

func (h *Handlers) workDestinationError(c *gin.Context, err error) {
	var field FieldError
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, work.ErrNotFound):
		c.Status(http.StatusNotFound)
	case errors.Is(err, ErrChanged):
		api.Refuse(c, http.StatusConflict, "The destination changed. Check its configuration and try again.")
	case errors.Is(err, work.ErrWorkFrozen):
		api.Refuse(c, http.StatusConflict, "This work is frozen while it is withheld.")
	case errors.Is(err, version.ErrUpdateDestinationIneligible):
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid, "Choose only your own verified, active destinations.", "destinationIds")
	case errors.As(err, &field):
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid, field.Message, field.Field)
	default:
		api.Refuse(c, http.StatusInternalServerError, "Could not update your destination. Try again.")
	}
}

func (h *Handlers) VerifyWorkUpdateDestination(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "verifying update destinations")
	if !ok {
		return
	}
	found, err := h.updateDestinations.Verify(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.workDestinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toWorkUpdateDestination(found))
}

func (h *Handlers) UpdateWorkUpdateDestination(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "changing update destinations")
	if !ok {
		return
	}
	var request UpdateWorkUpdateDestinationRequest
	if !api.ReadBoundedJSON(c, &request, 4096, "The destination configuration is too large.") {
		return
	}
	found, err := h.updateDestinations.Update(c.Request.Context(), owner.ID, id, request.Name, request.Address)
	if err != nil {
		h.workDestinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toWorkUpdateDestination(found))
}

func (h *Handlers) DisableWorkUpdateDestination(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "disabling update destinations")
	if !ok {
		return
	}
	found, err := h.updateDestinations.Disable(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.workDestinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toWorkUpdateDestination(found))
}

func (h *Handlers) RemoveWorkUpdateDestination(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "removing update destinations")
	if !ok {
		return
	}
	if err := h.updateDestinations.Remove(c.Request.Context(), owner.ID, id); err != nil {
		h.workDestinationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) RotateWorkUpdateDestinationSecret(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "rotating an update destination's secret")
	if !ok {
		return
	}
	added, err := h.updateDestinations.RotateSecret(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.workDestinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, AddedWorkUpdateDestination{Destination: toWorkUpdateDestination(added.Destination), Secret: &added.Secret})
}

func (h *Handlers) ListWorkUpdateDestinationChoices(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.SignedIn(c, "reading a work's update destinations")
	if !ok {
		return
	}
	found, err := h.updateDestinations.UpdateDestinations(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.workDestinationError(c, err)
		return
	}
	choices := make([]WorkUpdateDestinationChoice, 0, len(found))
	for _, one := range found {
		choices = append(choices, WorkUpdateDestinationChoice{Id: one.ID, Name: one.Name, Type: WorkUpdateDestinationType(one.Type), ByDefault: one.ByDefault})
	}
	c.JSON(http.StatusOK, WorkUpdateDestinationChoices{Destinations: choices})
}

func (h *Handlers) SetWorkUpdateDestinationDefaults(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "choosing a work's update destinations")
	if !ok {
		return
	}
	var request WorkUpdateDestinationDefaultsRequest
	if !api.ReadBoundedJSON(c, &request, 4096, "The destination selection is too large.") {
		return
	}
	if request.DestinationIds == nil {
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid, "Send destination IDs, or an empty list to disable default announcements.", "destinationIds")
		return
	}
	if err := h.updateDestinations.SetUpdateDestinations(c.Request.Context(), owner.ID, id, readIDs(&request.DestinationIds)); err != nil {
		h.workDestinationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func toWorkUpdateDestination(d Destination) WorkUpdateDestination {
	answer := WorkUpdateDestination{
		Id: d.ID, Name: d.Name, Type: WorkUpdateDestinationType(d.Type), Host: d.Host,
		Address: d.Address, State: WorkUpdateDestinationState(d.State), CreatedAt: d.CreatedAt,
		SecretSetAt: d.SecretSetAt, PreviousSecretUntil: d.PreviousSecretUntil,
		VerifiedAt: d.VerifiedAt, DisabledAt: d.DisabledAt,
	}
	if d.GuildID != nil && d.ChannelID != nil {
		answer.Channel = &WorkUpdateChannel{GuildId: *d.GuildID, ChannelId: *d.ChannelID}
	}
	return answer
}

func (h *Handlers) ListWorkUpdateAnnouncements(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.SignedIn(c, "reading what a work announced")
	if !ok {
		return
	}
	sent, err := h.updateDestinations.Announcements(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.workDestinationError(c, err)
		return
	}
	listed := make([]WorkUpdateAnnouncement, 0, len(sent))
	for _, one := range sent {
		listed = append(listed, toWorkUpdateAnnouncement(one))
	}
	c.JSON(http.StatusOK, WorkUpdateAnnouncementList{Announcements: listed})
}

func toWorkUpdateAnnouncement(one Announcement) WorkUpdateAnnouncement {
	shown := WorkUpdateAnnouncement{
		Id: one.ID, EventId: one.EventID, UpdateId: one.UpdateID, UpdateNumber: one.UpdateNumber,
		Destination: one.Destination, Type: WorkUpdateDestinationType(one.Type),
		Removed: one.Removed, State: WorkUpdateAnnouncementState(one.State),
		MessageId: one.MessageID, Run: one.Run, Attempts: one.Attempts,
		OccurredAt: one.OccurredAt, DueAt: one.DueAt, SettledAt: one.SettledAt,
	}
	if one.SettledReason != "" {
		reason := WorkUpdateAnnouncementSettledReason(one.SettledReason)
		shown.SettledReason = &reason
	}
	if one.Last != nil {
		shown.Last = &WorkUpdateAnnouncementAttempt{
			Run: one.Last.Run, Number: one.Last.Number,
			Outcome: WorkUpdateAnnouncementAttemptOutcome(one.Last.Outcome),
			Status:  one.Last.Status, Detail: one.Last.Detail,
			TookMs: int(one.Last.Took.Milliseconds()), AttemptedAt: one.Last.Attempted,
		}
	}
	return shown
}
