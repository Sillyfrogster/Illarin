package integration

import (
	"errors"
	"net/http"

	announcements "github.com/Sillyfrogster/Illarin/api/internal/integration/blog"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/blog"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const defaultAttemptListing = 50

func (h *Handlers) ListBlogIntegrations(c *gin.Context) {
	if _, ok := h.access.Authority(c, "reading publication integrations"); !ok {
		return
	}
	configured, err := h.publications.Integrations(c.Request.Context())
	if err != nil {
		h.blogIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusOK, BlogIntegrationList{
		Integrations: toAPIIntegrations(configured),
	})
}

func (h *Handlers) AddBlogIntegration(c *gin.Context) {
	authority, ok := h.access.Authority(c, "configuring a publication integration")
	if !ok {
		return
	}
	var request AddBlogIntegrationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"Send the integration's name and address.", "address")
		return
	}
	added, err := h.publications.AddIntegration(
		c.Request.Context(), authority.ID,
		blog.IntegrationEdit{
			Name: request.Name, Address: request.Address, Announcements: readAnnouncementTypes(request.Announcements),
		},
	)
	if err != nil {
		h.blogIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, AddedBlogIntegration{
		Integration: toAPIIntegration(added.Integration),
		Secret:      added.Secret,
	})
}

func (h *Handlers) AddBlogChannel(c *gin.Context) {
	authority, ok := h.access.Authority(c, "configuring a Discord integration")
	if !ok {
		return
	}
	edit, ok := readChannel(c)
	if !ok {
		return
	}
	added, err := h.publications.AddChannel(c.Request.Context(), authority.ID, edit)
	if err != nil {
		h.blogIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toAPIIntegration(added))
}

func (h *Handlers) UpdateBlogChannel(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "changing a Discord integration")
	if !ok {
		return
	}
	edit, ok := readChannel(c)
	if !ok {
		return
	}
	updated, err := h.publications.UpdateChannel(
		c.Request.Context(), authority.ID, id, edit,
	)
	if err != nil {
		h.blogIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIIntegration(updated))
}

func readChannel(c *gin.Context) (blog.ChannelEdit, bool) {
	var request BlogChannelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"Send the channel's name and address.", "address")
		return blog.ChannelEdit{}, false
	}
	edit := blog.ChannelEdit{Name: request.Name}
	if request.Address != nil {
		edit.Address = *request.Address
	}
	if request.RoleId != nil {
		edit.RoleID = *request.RoleId
	}
	if request.RoleName != nil {
		edit.RoleName = *request.RoleName
	}
	return edit, true
}

func (h *Handlers) UpdateBlogIntegration(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "changing a publication integration")
	if !ok {
		return
	}
	var request UpdateBlogIntegrationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"Send the change as JSON.", "address")
		return
	}
	updated, err := h.publications.UpdateIntegration(
		c.Request.Context(), authority.ID, id,
		blog.IntegrationUpdate{
			Name: request.Name, Address: request.Address, Announcements: readAnnouncementTypes(request.Announcements),
		},
	)
	if err != nil {
		h.blogIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIIntegration(updated))
}

func (h *Handlers) RemoveBlogIntegration(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "removing a publication integration")
	if !ok {
		return
	}
	err := h.publications.RemoveIntegration(c.Request.Context(), authority.ID, id)
	if err != nil {
		h.blogIntegrationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) VerifyBlogIntegration(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "verifying a publication integration")
	if !ok {
		return
	}
	verified, err := h.publications.VerifyIntegration(
		c.Request.Context(), authority.ID, id,
	)
	if err != nil {
		h.blogIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIIntegration(verified))
}

func (h *Handlers) DisableBlogIntegration(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "disabling a publication integration")
	if !ok {
		return
	}
	disabled, err := h.publications.DisableIntegration(
		c.Request.Context(), authority.ID, id,
	)
	if err != nil {
		h.blogIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIIntegration(disabled))
}

func (h *Handlers) RotateBlogIntegrationSecret(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "rotating a integration's signing secret")
	if !ok {
		return
	}
	rotated, err := h.publications.RotateSecret(c.Request.Context(), authority.ID, id)
	if err != nil {
		h.blogIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusOK, RotatedBlogSecret{
		Integration:         toAPIIntegration(rotated.Integration),
		Secret:              rotated.Secret,
		PreviousSecretUntil: rotated.OldUntil,
	})
}

func (h *Handlers) ListBlogAnnouncementAttempts(c *gin.Context) {
	q := api.ReadQuery(c)
	params := ListBlogAnnouncementAttemptsParams{
		State: api.QueryText[BlogAnnouncementAttemptState](q, "state"),
		Limit: api.QueryNumber(q, "limit"),
	}
	if q.Refused(c) {
		return
	}
	if _, ok := h.access.Authority(c, "reading publication attempts"); !ok {
		return
	}
	state := ""
	if params.State != nil {
		state = string(*params.State)
	}
	limit := defaultAttemptListing
	if params.Limit != nil {
		limit = *params.Limit
	}
	sent, err := h.publications.Attempts(c.Request.Context(), state, limit)
	if err != nil {
		h.blogIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusOK, BlogAnnouncementAttemptList{Attempts: toAPIAttempts(sent)})
}

func (h *Handlers) ListBlogAnnouncementTries(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	if _, ok := h.access.Authority(c, "reading what a attempt tried"); !ok {
		return
	}
	made, err := h.publications.Tries(c.Request.Context(), id)
	if err != nil {
		h.blogIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusOK, BlogAnnouncementTryList{Tries: toAPITries(made)})
}

func (h *Handlers) ReplayBlogAnnouncementAttempt(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "replaying a publication attempt")
	if !ok {
		return
	}
	queued, err := h.publications.ReplayAttempt(c.Request.Context(), authority.ID, id)
	if err != nil {
		h.blogIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIAttempt(queued))
}

func (h *Handlers) RepairDiscordAnnouncement(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "repairing a Discord announcement")
	if !ok {
		return
	}
	var body blog.DiscordRepair
	if err := c.ShouldBindJSON(&body); err != nil {
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid, "Send the repair as JSON.", "repair")
		return
	}
	result, err := h.publications.RepairDiscord(c.Request.Context(), authority.ID, id, body)
	if err != nil {
		h.blogIntegrationError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handlers) SetBlogGrantIntegrations(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "setting a contributor's integrations")
	if !ok {
		return
	}
	policy, ok := readIntegrationPolicy(c)
	if !ok {
		return
	}
	err := h.publications.SetGrantIntegrations(
		c.Request.Context(), authority.ID, id, policy,
	)
	if err != nil {
		h.blogIntegrationError(c, err)
		return
	}
	result, err := h.access.Grant(c, id)
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handlers) ListPostIntegrations(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.access.Editor(c, "reading a post's integrations")
	if !ok {
		return
	}
	found, err := h.publications.PostIntegrations(c.Request.Context(), editor, id)
	if err != nil {
		h.access.PostError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIChoices(found, false))
}

func (h *Handlers) ListPostAnnouncementAttempts(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.access.Editor(c, "reading what a post sent")
	if !ok {
		return
	}
	sent, err := h.publications.PostAttempts(c.Request.Context(), editor, id)
	if err != nil {
		h.access.PostError(c, err)
		return
	}
	c.JSON(http.StatusOK, BlogAnnouncementAttemptList{Attempts: toAPIAttempts(sent)})
}

func readAnnouncementTypes(named *[]BlogAnnouncementType) *[]string {
	if named == nil {
		return nil
	}
	announced := make([]string, 0, len(*named))
	for _, one := range *named {
		announced = append(announced, string(one))
	}
	return &announced
}

func readIntegrationPolicy(c *gin.Context) (blog.IntegrationPolicy, bool) {
	var request IntegrationPolicyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"Send the integrations as JSON.", "integrationIds")
		return blog.IntegrationPolicy{}, false
	}
	policy := blog.IntegrationPolicy{Defaults: readIDs(request.DefaultIntegrationIds)}
	if request.IntegrationIds != nil {
		allowed := readIDs(request.IntegrationIds)
		policy.Allowed = &allowed
	}
	return policy, true
}

func readIDs(listed *[]uuid.UUID) []uuid.UUID {
	if listed == nil {
		return nil
	}
	held := make([]uuid.UUID, 0, len(*listed))
	for _, one := range *listed {
		held = append(held, one)
	}
	return held
}

func (h *Handlers) blogIntegrationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, blog.ErrIntegrationNotFound):
		refusePublication(c, http.StatusNotFound, PublicationErrorCodeNotFound, "No such integration.")
	case errors.Is(err, blog.ErrIntegrationRefused):
		refusePublication(c, http.StatusForbidden, PublicationErrorCodeForbidden,
			"This post may not send to that integration.")
	case errors.Is(err, blog.ErrRoleRefused):
		refusePublication(c, http.StatusForbidden, PublicationErrorCodeForbidden,
			"This post may not mention that integration's role.")
	case errors.Is(err, blog.ErrNotDiscord):
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"That integration is a generic webhook.")
	case errors.Is(err, blog.ErrNotWebhook):
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"That integration is a Discord channel.")
	case errors.Is(err, blog.ErrAttemptNotFound):
		refusePublication(c, http.StatusNotFound, PublicationErrorCodeNotFound, "No such attempt.")
	case errors.Is(err, blog.ErrAttemptUnsettled):
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"This attempt is still trying on its own.")
	case errors.Is(err, blog.ErrAttemptUnsendable):
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"There is nowhere left to send this attempt.")
	default:
		h.access.PublicationError(c, err)
	}
}

func toAPIIntegrations(configured []blog.Integration) []BlogIntegration {
	listed := make([]BlogIntegration, 0, len(configured))
	for _, one := range configured {
		listed = append(listed, toAPIIntegration(one))
	}
	return listed
}

func toAPIIntegration(found blog.Integration) BlogIntegration {
	announced := make([]BlogAnnouncementType, 0, len(found.Announcements))
	for _, one := range found.Announcements {
		announced = append(announced, BlogAnnouncementType(one))
	}
	shown := BlogIntegration{
		Id:                  found.ID,
		Type:                BlogIntegrationType(found.Type),
		Name:                found.Name,
		Host:                found.Host,
		Address:             found.Address,
		State:               BlogIntegrationState(found.State),
		Announcements:       announced,
		SecretSetAt:         found.SecretSetAt,
		PreviousSecretUntil: found.OldUntil,
		VerifiedAt:          found.VerifiedAt,
		DisabledAt:          found.DisabledAt,
		CreatedAt:           found.CreatedAt,
	}
	if found.Channel != nil {
		shown.Channel = &BlogChannel{
			GuildId:     found.Channel.GuildID,
			ChannelId:   found.Channel.ChannelID,
			WebhookName: found.Channel.Webhook,
			RoleId:      found.Channel.RoleID,
			RoleName:    found.Channel.RoleName,
		}
	}
	return shown
}

func toAPIChoices(held []blog.Choice, inherited bool) BlogIntegrationChoiceList {
	return BlogIntegrationChoiceList{
		Integrations: announcements.ChoiceRows(held),
		Inherited:    inherited,
	}
}

func toAPIAttempts(sent []blog.Attempt) []BlogAnnouncementAttempt {
	listed := make([]BlogAnnouncementAttempt, 0, len(sent))
	for _, one := range sent {
		listed = append(listed, toAPIAttempt(one))
	}
	return listed
}

func toAPIAttempt(one blog.Attempt) BlogAnnouncementAttempt {
	shown := BlogAnnouncementAttempt{
		Id:               one.ID,
		AnnouncementId:   one.AnnouncementID,
		AnnouncementType: one.AnnouncementType,
		PostId:           one.PostID,
		PostTitle:        one.PostTitle,
		RevisionId:       one.RevisionID,
		Integration:      one.Integration,
		Type:             BlogAnnouncementAttemptType(one.Type),
		MessageId:        one.MessageID,
		Removed:          one.Removed,
		State:            BlogAnnouncementAttemptState(one.State),
		Run:              one.Run,
		Tries:            one.Tries,
		OccurredAt:       one.OccurredAt,
		DueAt:            one.DueAt,
		SettledAt:        one.SettledAt,
	}
	if one.SettledReason != "" {
		reason := BlogAnnouncementAttemptSettledReason(one.SettledReason)
		shown.SettledReason = &reason
	}
	if one.Last != nil {
		made := toAPITry(*one.Last)
		shown.Last = &made
	}
	return shown
}

func toAPITries(made []blog.Try) []BlogAnnouncementTry {
	listed := make([]BlogAnnouncementTry, 0, len(made))
	for _, one := range made {
		listed = append(listed, toAPITry(one))
	}
	return listed
}

func toAPITry(one blog.Try) BlogAnnouncementTry {
	return BlogAnnouncementTry{
		Run:         one.Run,
		Number:      one.Number,
		Outcome:     BlogAnnouncementTryOutcome(one.Outcome),
		Status:      one.Status,
		Detail:      one.Detail,
		TookMs:      int(one.Took.Milliseconds()),
		AttemptedAt: one.Attempted,
	}
}

const (
	BlogIntegrationTypeDiscord = announcements.BlogIntegrationTypeDiscord
	BlogIntegrationTypeWebhook = announcements.BlogIntegrationTypeWebhook
)

const (
	BlogIntegrationStateActive     = announcements.BlogIntegrationStateActive
	BlogIntegrationStateDisabled   = announcements.BlogIntegrationStateDisabled
	BlogIntegrationStateUnverified = announcements.BlogIntegrationStateUnverified
)

const (
	BlogAnnouncementTypePublicationPostPublishedV1 = announcements.BlogAnnouncementTypePublicationPostPublishedV1
	BlogAnnouncementTypePublicationPostUpdatedV1   = announcements.BlogAnnouncementTypePublicationPostUpdatedV1
	BlogAnnouncementTypePublicationPostWithdrawnV1 = announcements.BlogAnnouncementTypePublicationPostWithdrawnV1
)
