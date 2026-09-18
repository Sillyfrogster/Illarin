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

const defaultDeliveryListing = 50

func (h *Handlers) ListPublicationDestinations(c *gin.Context) {
	if _, ok := h.access.Authority(c, "reading publication destinations"); !ok {
		return
	}
	configured, err := h.publications.Destinations(c.Request.Context())
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, PublicationDestinationList{
		Destinations: toAPIDestinations(configured),
	})
}

func (h *Handlers) AddPublicationDestination(c *gin.Context) {
	authority, ok := h.access.Authority(c, "configuring a publication destination")
	if !ok {
		return
	}
	var request AddPublicationDestinationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"Send the destination's name and address.", "address")
		return
	}
	added, err := h.publications.AddDestination(
		c.Request.Context(), authority.ID,
		blog.DestinationEdit{
			Name: request.Name, Address: request.Address, Events: readEvents(request.Events),
		},
	)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, AddedPublicationDestination{
		Destination: toAPIDestination(added.Destination),
		Secret:      added.Secret,
	})
}

func (h *Handlers) AddPublicationChannel(c *gin.Context) {
	authority, ok := h.access.Authority(c, "configuring a Discord destination")
	if !ok {
		return
	}
	edit, ok := readChannel(c)
	if !ok {
		return
	}
	added, err := h.publications.AddChannel(c.Request.Context(), authority.ID, edit)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toAPIDestination(added))
}

func (h *Handlers) UpdatePublicationChannel(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "changing a Discord destination")
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
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIDestination(updated))
}

func readChannel(c *gin.Context) (blog.ChannelEdit, bool) {
	var request PublicationChannelRequest
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

func (h *Handlers) UpdatePublicationDestination(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "changing a publication destination")
	if !ok {
		return
	}
	var request UpdatePublicationDestinationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"Send the change as JSON.", "address")
		return
	}
	updated, err := h.publications.UpdateDestination(
		c.Request.Context(), authority.ID, id,
		blog.DestinationUpdate{
			Name: request.Name, Address: request.Address, Events: readEvents(request.Events),
		},
	)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIDestination(updated))
}

func (h *Handlers) RemovePublicationDestination(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "removing a publication destination")
	if !ok {
		return
	}
	err := h.publications.RemoveDestination(c.Request.Context(), authority.ID, id)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) VerifyPublicationDestination(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "verifying a publication destination")
	if !ok {
		return
	}
	verified, err := h.publications.VerifyDestination(
		c.Request.Context(), authority.ID, id,
	)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIDestination(verified))
}

func (h *Handlers) DisablePublicationDestination(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "disabling a publication destination")
	if !ok {
		return
	}
	disabled, err := h.publications.DisableDestination(
		c.Request.Context(), authority.ID, id,
	)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIDestination(disabled))
}

func (h *Handlers) RotatePublicationDestinationSecret(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "rotating a destination's signing secret")
	if !ok {
		return
	}
	rotated, err := h.publications.RotateSecret(c.Request.Context(), authority.ID, id)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, RotatedPublicationSecret{
		Destination:         toAPIDestination(rotated.Destination),
		Secret:              rotated.Secret,
		PreviousSecretUntil: rotated.OldUntil,
	})
}

func (h *Handlers) ListPublicationDeliveries(c *gin.Context) {
	q := api.ReadQuery(c)
	params := ListPublicationDeliveriesParams{
		State: api.QueryText[PostDeliveryState](q, "state"),
		Limit: api.QueryNumber(q, "limit"),
	}
	if q.Refused(c) {
		return
	}
	if _, ok := h.access.Authority(c, "reading publication deliveries"); !ok {
		return
	}
	state := ""
	if params.State != nil {
		state = string(*params.State)
	}
	limit := defaultDeliveryListing
	if params.Limit != nil {
		limit = *params.Limit
	}
	sent, err := h.publications.Deliveries(c.Request.Context(), state, limit)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, PostDeliveryList{Deliveries: toAPIDeliveries(sent)})
}

func (h *Handlers) ListPublicationDeliveryAttempts(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	if _, ok := h.access.Authority(c, "reading what a delivery tried"); !ok {
		return
	}
	made, err := h.publications.DeliveryHistory(c.Request.Context(), id)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, PostDeliveryAttemptList{Attempts: toAPIAttempts(made)})
}

func (h *Handlers) ReplayPublicationDelivery(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "replaying a publication delivery")
	if !ok {
		return
	}
	queued, err := h.publications.ReplayDelivery(c.Request.Context(), authority.ID, id)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIDelivery(queued))
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
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handlers) SetPublicationGrantDestinations(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	authority, ok := h.access.Authority(c, "setting a contributor's destinations")
	if !ok {
		return
	}
	policy, ok := readDestinationPolicy(c)
	if !ok {
		return
	}
	err := h.publications.SetGrantDestinations(
		c.Request.Context(), authority.ID, id, policy,
	)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	result, err := h.access.Grant(c, id)
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handlers) ListPostDestinations(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.access.Editor(c, "reading a post's destinations")
	if !ok {
		return
	}
	found, err := h.publications.PostDestinations(c.Request.Context(), editor, id)
	if err != nil {
		h.access.PostError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIChoices(found, false))
}

func (h *Handlers) ListPostDeliveries(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.access.Editor(c, "reading what a post sent")
	if !ok {
		return
	}
	sent, err := h.publications.PostDeliveries(c.Request.Context(), editor, id)
	if err != nil {
		h.access.PostError(c, err)
		return
	}
	c.JSON(http.StatusOK, PostDeliveryList{Deliveries: toAPIDeliveries(sent)})
}

func readEvents(named *[]PublicationEvent) *[]string {
	if named == nil {
		return nil
	}
	events := make([]string, 0, len(*named))
	for _, one := range *named {
		events = append(events, string(one))
	}
	return &events
}

func readDestinationPolicy(c *gin.Context) (blog.DestinationPolicy, bool) {
	var request DestinationPolicyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"Send the destinations as JSON.", "destinationIds")
		return blog.DestinationPolicy{}, false
	}
	policy := blog.DestinationPolicy{Defaults: readIDs(request.DefaultDestinationIds)}
	if request.DestinationIds != nil {
		allowed := readIDs(request.DestinationIds)
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

func (h *Handlers) destinationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, blog.ErrDestinationNotFound):
		refusePublication(c, http.StatusNotFound, PublicationErrorCodeNotFound, "No such destination.")
	case errors.Is(err, blog.ErrDestinationRefused):
		refusePublication(c, http.StatusForbidden, PublicationErrorCodeForbidden,
			"This post may not send to that destination.")
	case errors.Is(err, blog.ErrRoleRefused):
		refusePublication(c, http.StatusForbidden, PublicationErrorCodeForbidden,
			"This post may not mention that destination's role.")
	case errors.Is(err, blog.ErrNotDiscord):
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"That destination is a generic webhook.")
	case errors.Is(err, blog.ErrNotWebhook):
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"That destination is a Discord channel.")
	case errors.Is(err, blog.ErrDeliveryNotFound):
		refusePublication(c, http.StatusNotFound, PublicationErrorCodeNotFound, "No such delivery.")
	case errors.Is(err, blog.ErrDeliveryUnsettled):
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"This delivery is still trying on its own.")
	case errors.Is(err, blog.ErrDeliveryUnsendable):
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"There is nowhere left to send this delivery.")
	default:
		h.access.PublicationError(c, err)
	}
}

func toAPIDestinations(configured []blog.Destination) []PublicationDestination {
	listed := make([]PublicationDestination, 0, len(configured))
	for _, one := range configured {
		listed = append(listed, toAPIDestination(one))
	}
	return listed
}

func toAPIDestination(found blog.Destination) PublicationDestination {
	events := make([]PublicationEvent, 0, len(found.Events))
	for _, one := range found.Events {
		events = append(events, PublicationEvent(one))
	}
	shown := PublicationDestination{
		Id:                  found.ID,
		Type:                PublicationDestinationType(found.Type),
		Name:                found.Name,
		Host:                found.Host,
		Address:             found.Address,
		State:               PublicationDestinationState(found.State),
		Events:              events,
		SecretSetAt:         found.SecretSetAt,
		PreviousSecretUntil: found.OldUntil,
		VerifiedAt:          found.VerifiedAt,
		DisabledAt:          found.DisabledAt,
		CreatedAt:           found.CreatedAt,
	}
	if found.Channel != nil {
		shown.Channel = &PublicationChannel{
			GuildId:     found.Channel.GuildID,
			ChannelId:   found.Channel.ChannelID,
			WebhookName: found.Channel.Webhook,
			RoleId:      found.Channel.RoleID,
			RoleName:    found.Channel.RoleName,
		}
	}
	return shown
}

func toAPIChoices(held []blog.Choice, inherited bool) PublicationDestinationChoiceList {
	return PublicationDestinationChoiceList{
		Destinations: announcements.ChoiceRows(held),
		Inherited:    inherited,
	}
}

func toAPIDeliveries(sent []blog.Delivery) []PostDelivery {
	listed := make([]PostDelivery, 0, len(sent))
	for _, one := range sent {
		listed = append(listed, toAPIDelivery(one))
	}
	return listed
}

func toAPIDelivery(one blog.Delivery) PostDelivery {
	shown := PostDelivery{
		Id:          one.ID,
		EventId:     one.EventID,
		EventType:   one.EventType,
		PostId:      one.PostID,
		PostTitle:   one.PostTitle,
		RevisionId:  one.RevisionID,
		Destination: one.Destination,
		Type:        PostDeliveryType(one.Type),
		MessageId:   one.MessageID,
		Removed:     one.Removed,
		State:       PostDeliveryState(one.State),
		Run:         one.Run,
		Attempts:    one.Attempts,
		OccurredAt:  one.OccurredAt,
		DueAt:       one.DueAt,
		SettledAt:   one.SettledAt,
	}
	if one.SettledReason != "" {
		reason := PostDeliverySettledReason(one.SettledReason)
		shown.SettledReason = &reason
	}
	if one.Last != nil {
		made := toAPIAttempt(*one.Last)
		shown.Last = &made
	}
	return shown
}

func toAPIAttempts(made []blog.DeliveryAttempt) []PostDeliveryAttempt {
	listed := make([]PostDeliveryAttempt, 0, len(made))
	for _, one := range made {
		listed = append(listed, toAPIAttempt(one))
	}
	return listed
}

func toAPIAttempt(one blog.DeliveryAttempt) PostDeliveryAttempt {
	return PostDeliveryAttempt{
		Run:         one.Run,
		Number:      one.Number,
		Outcome:     PostDeliveryOutcome(one.Outcome),
		Status:      one.Status,
		Detail:      one.Detail,
		TookMs:      int(one.Took.Milliseconds()),
		AttemptedAt: one.Attempted,
	}
}

const (
	PublicationDestinationTypeDiscord = announcements.PublicationDestinationTypeDiscord
	PublicationDestinationTypeWebhook = announcements.PublicationDestinationTypeWebhook
)

const (
	PublicationDestinationStateActive     = announcements.PublicationDestinationStateActive
	PublicationDestinationStateDisabled   = announcements.PublicationDestinationStateDisabled
	PublicationDestinationStateUnverified = announcements.PublicationDestinationStateUnverified
)

const (
	PublicationEventPublicationPostPublishedV1 = announcements.PublicationEventPublicationPostPublishedV1
	PublicationEventPublicationPostUpdatedV1   = announcements.PublicationEventPublicationPostUpdatedV1
	PublicationEventPublicationPostWithdrawnV1 = announcements.PublicationEventPublicationPostWithdrawnV1
)
