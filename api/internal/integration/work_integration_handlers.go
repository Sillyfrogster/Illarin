package integration

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) ListWorkIntegrations(c *gin.Context) {
	owner, ok := api.SignedIn(c, "reading your update integrations")
	if !ok {
		return
	}
	found, err := h.integrations.List(c.Request.Context(), owner.ID)
	if err != nil {
		h.workIntegrationError(c, err)
		return
	}
	listed := make([]WorkIntegration, 0, len(found))
	for _, one := range found {
		listed = append(listed, toWorkIntegration(one))
	}
	c.JSON(http.StatusOK, WorkIntegrationList{Integrations: listed})
}

func (h *Handlers) GetWorkIntegration(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.SignedIn(c, "reading your update integration")
	if !ok {
		return
	}
	found, err := h.integrations.Get(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.workIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toWorkIntegration(found))
}

func (h *Handlers) AddWorkIntegration(c *gin.Context) {
	owner, ok := api.Verified(c, "configuring update integrations")
	if !ok {
		return
	}
	var request AddWorkIntegrationRequest
	if !api.ReadBoundedJSON(c, &request, 4096, "The integration configuration is too large.") {
		return
	}
	if request.Address == nil {
		refuseField(c, http.StatusBadRequest, BlogErrorCodeInvalid, "Enter the integration address.", "address")
		return
	}
	added, err := h.integrations.Add(c.Request.Context(), owner.ID, string(request.Type), request.Name, *request.Address)
	if err != nil {
		h.workIntegrationError(c, err)
		return
	}
	answer := AddedWorkIntegration{Integration: toWorkIntegration(added.Integration)}
	if added.Secret != "" {
		answer.Secret = &added.Secret
	}
	c.JSON(http.StatusCreated, answer)
}

func (h *Handlers) workIntegrationError(c *gin.Context, err error) {
	var field FieldError
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, work.ErrNotFound):
		c.Status(http.StatusNotFound)
	case errors.Is(err, ErrChanged):
		api.Refuse(c, http.StatusConflict, "The integration changed. Check its configuration and try again.")
	case errors.Is(err, work.ErrWorkFrozen):
		api.Refuse(c, http.StatusConflict, "This work is frozen while it is taken down.")
	case errors.Is(err, version.ErrIntegrationIneligible):
		refuseField(c, http.StatusBadRequest, BlogErrorCodeInvalid, "Choose only your own verified, active integrations.", "integrationIds")
	case errors.As(err, &field):
		refuseField(c, http.StatusBadRequest, BlogErrorCodeInvalid, field.Message, field.Field)
	default:
		api.Refuse(c, http.StatusInternalServerError, "Could not update your integration. Try again.")
	}
}

func (h *Handlers) VerifyWorkIntegration(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "verifying update integrations")
	if !ok {
		return
	}
	found, err := h.integrations.Verify(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.workIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toWorkIntegration(found))
}

func (h *Handlers) UpdateWorkIntegration(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "changing update integrations")
	if !ok {
		return
	}
	var request UpdateWorkIntegrationRequest
	if !api.ReadBoundedJSON(c, &request, 4096, "The integration configuration is too large.") {
		return
	}
	found, err := h.integrations.Update(c.Request.Context(), owner.ID, id, request.Name, request.Address)
	if err != nil {
		h.workIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toWorkIntegration(found))
}

func (h *Handlers) DisableWorkIntegration(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "disabling update integrations")
	if !ok {
		return
	}
	found, err := h.integrations.Disable(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.workIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toWorkIntegration(found))
}

func (h *Handlers) RemoveWorkIntegration(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "removing update integrations")
	if !ok {
		return
	}
	if err := h.integrations.Remove(c.Request.Context(), owner.ID, id); err != nil {
		h.workIntegrationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) RotateWorkIntegrationSecret(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "rotating an update integration's secret")
	if !ok {
		return
	}
	added, err := h.integrations.RotateSecret(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.workIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusOK, AddedWorkIntegration{Integration: toWorkIntegration(added.Integration), Secret: &added.Secret})
}

func (h *Handlers) ListWorkIntegrationChoices(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.SignedIn(c, "reading a work's update integrations")
	if !ok {
		return
	}
	found, err := h.integrations.Integrations(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.workIntegrationError(c, err)
		return
	}
	choices := make([]WorkIntegrationChoice, 0, len(found))
	for _, one := range found {
		choices = append(choices, WorkIntegrationChoice{Id: one.ID, Name: one.Name, Type: WorkIntegrationType(one.Type), ByDefault: one.ByDefault})
	}
	c.JSON(http.StatusOK, WorkIntegrationChoices{Integrations: choices})
}

func (h *Handlers) SetWorkIntegrationDefaults(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "choosing a work's update integrations")
	if !ok {
		return
	}
	var request WorkIntegrationDefaultsRequest
	if !api.ReadBoundedJSON(c, &request, 4096, "The integration selection is too large.") {
		return
	}
	if request.IntegrationIds == nil {
		refuseField(c, http.StatusBadRequest, BlogErrorCodeInvalid, "Send integration IDs, or an empty list to disable default announcements.", "integrationIds")
		return
	}
	if err := h.integrations.SetIntegrations(c.Request.Context(), owner.ID, id, readIDs(&request.IntegrationIds)); err != nil {
		h.workIntegrationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func toWorkIntegration(d Integration) WorkIntegration {
	answer := WorkIntegration{
		Id: d.ID, Name: d.Name, Type: WorkIntegrationType(d.Type), Host: d.Host,
		Address: d.Address, State: WorkIntegrationState(d.State), CreatedAt: d.CreatedAt,
		SecretSetAt: d.SecretSetAt, PreviousSecretUntil: d.PreviousSecretUntil,
		VerifiedAt: d.VerifiedAt, DisabledAt: d.DisabledAt,
	}
	if d.GuildID != nil && d.ChannelID != nil {
		answer.Channel = &WorkIntegrationChannel{GuildId: *d.GuildID, ChannelId: *d.ChannelID}
	}
	return answer
}

func (h *Handlers) ListWorkAnnouncementAttempts(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.SignedIn(c, "reading what a work announced")
	if !ok {
		return
	}
	sent, err := h.integrations.Attempts(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.workIntegrationError(c, err)
		return
	}
	listed := make([]WorkAnnouncementAttempt, 0, len(sent))
	for _, one := range sent {
		listed = append(listed, toWorkAnnouncementAttempt(one))
	}
	c.JSON(http.StatusOK, WorkAnnouncementAttemptList{Attempts: listed})
}

func toWorkAnnouncementAttempt(one Attempt) WorkAnnouncementAttempt {
	shown := WorkAnnouncementAttempt{
		Id: one.ID, AnnouncementId: one.AnnouncementID, VersionId: one.VersionID, VersionNumber: one.VersionNumber,
		Integration: one.Integration, Type: WorkIntegrationType(one.Type),
		Removed: one.Removed, State: WorkAnnouncementAttemptState(one.State),
		MessageId: one.MessageID, Run: one.Run, Tries: one.Tries,
		OccurredAt: one.OccurredAt, DueAt: one.DueAt, SettledAt: one.SettledAt,
	}
	if one.SettledReason != "" {
		reason := WorkAnnouncementAttemptSettledReason(one.SettledReason)
		shown.SettledReason = &reason
	}
	if one.Last != nil {
		shown.Last = &WorkAnnouncementTry{
			Run: one.Last.Run, Number: one.Last.Number,
			Outcome: WorkAnnouncementTryOutcome(one.Last.Outcome),
			Status:  one.Last.Status, Detail: one.Last.Detail,
			TookMs: int(one.Last.Took.Milliseconds()), AttemptedAt: one.Last.Attempted,
		}
	}
	return shown
}
