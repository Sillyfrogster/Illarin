package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

const defaultDeliveryListing = 50

func (h *Handlers) ListPublicationDestinations(c *gin.Context) {
	if _, ok := h.publicationAuthority(c, "reading publication destinations"); !ok {
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
	authority, ok := h.publicationAuthority(c, "configuring a publication destination")
	if !ok {
		return
	}
	var request AddPublicationDestinationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, CodeInvalid,
			"Send the destination's name and address.", "address")
		return
	}
	added, err := h.publications.AddDestination(
		c.Request.Context(), authority.ID,
		publication.DestinationEdit{
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
	authority, ok := h.publicationAuthority(c, "configuring a Discord destination")
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

func (h *Handlers) UpdatePublicationChannel(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "changing a Discord destination")
	if !ok {
		return
	}
	edit, ok := readChannel(c)
	if !ok {
		return
	}
	updated, err := h.publications.UpdateChannel(
		c.Request.Context(), authority.ID, uuid.UUID(id), edit,
	)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIDestination(updated))
}

func readChannel(c *gin.Context) (publication.ChannelEdit, bool) {
	var request PublicationChannelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, CodeInvalid,
			"Send the channel's name and address.", "address")
		return publication.ChannelEdit{}, false
	}
	edit := publication.ChannelEdit{Name: request.Name}
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

func (h *Handlers) UpdatePublicationDestination(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "changing a publication destination")
	if !ok {
		return
	}
	var request UpdatePublicationDestinationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, CodeInvalid,
			"Send the change as JSON.", "address")
		return
	}
	updated, err := h.publications.UpdateDestination(
		c.Request.Context(), authority.ID, uuid.UUID(id),
		publication.DestinationUpdate{
			Name: request.Name, Address: request.Address, Events: readEvents(request.Events),
		},
	)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIDestination(updated))
}

func (h *Handlers) RemovePublicationDestination(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "removing a publication destination")
	if !ok {
		return
	}
	err := h.publications.RemoveDestination(c.Request.Context(), authority.ID, uuid.UUID(id))
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) VerifyPublicationDestination(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "verifying a publication destination")
	if !ok {
		return
	}
	verified, err := h.publications.VerifyDestination(
		c.Request.Context(), authority.ID, uuid.UUID(id),
	)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIDestination(verified))
}

func (h *Handlers) DisablePublicationDestination(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "disabling a publication destination")
	if !ok {
		return
	}
	disabled, err := h.publications.DisableDestination(
		c.Request.Context(), authority.ID, uuid.UUID(id),
	)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIDestination(disabled))
}

func (h *Handlers) RotatePublicationDestinationSecret(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "rotating a destination's signing secret")
	if !ok {
		return
	}
	rotated, err := h.publications.RotateSecret(c.Request.Context(), authority.ID, uuid.UUID(id))
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

func (h *Handlers) ListPublicationDeliveries(
	c *gin.Context,
	params ListPublicationDeliveriesParams,
) {
	if _, ok := h.publicationAuthority(c, "reading publication deliveries"); !ok {
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

func (h *Handlers) ListPublicationDeliveryAttempts(c *gin.Context, id types.UUID) {
	if _, ok := h.publicationAuthority(c, "reading what a delivery tried"); !ok {
		return
	}
	made, err := h.publications.DeliveryHistory(c.Request.Context(), uuid.UUID(id))
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, PostDeliveryAttemptList{Attempts: toAPIAttempts(made)})
}

func (h *Handlers) ReplayPublicationDelivery(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "replaying a publication delivery")
	if !ok {
		return
	}
	queued, err := h.publications.ReplayDelivery(c.Request.Context(), authority.ID, uuid.UUID(id))
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIDelivery(queued))
}

func (h *Handlers) RepairDiscordAnnouncement(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "repairing a Discord announcement")
	if !ok {
		return
	}
	var body publication.DiscordRepair
	if err := c.ShouldBindJSON(&body); err != nil {
		refuseField(c, http.StatusBadRequest, CodeInvalid, "Send the repair as JSON.", "repair")
		return
	}
	result, err := h.publications.RepairDiscord(c.Request.Context(), authority.ID, uuid.UUID(id), body)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handlers) SetPublicationGrantDestinations(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "setting a contributor's destinations")
	if !ok {
		return
	}
	policy, ok := readDestinationPolicy(c)
	if !ok {
		return
	}
	err := h.publications.SetGrantDestinations(
		c.Request.Context(), authority.ID, uuid.UUID(id), policy,
	)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	grant, err := h.publications.Grant(c.Request.Context(), uuid.UUID(id))
	if err != nil {
		h.destinationError(c, err)
		return
	}
	listed, err := h.withHolders(c, []publication.Grant{grant})
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, listed[0])
}

func (h *Handlers) ListPostDestinations(c *gin.Context, id types.UUID) {
	editor, ok := h.postEditor(c, "reading a post's destinations")
	if !ok {
		return
	}
	found, err := h.publications.PostDestinations(c.Request.Context(), editor, uuid.UUID(id))
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIChoices(found, false))
}

func (h *Handlers) ListPostDeliveries(c *gin.Context, id types.UUID) {
	editor, ok := h.postEditor(c, "reading what a post sent")
	if !ok {
		return
	}
	sent, err := h.publications.PostDeliveries(c.Request.Context(), editor, uuid.UUID(id))
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, PostDeliveryList{Deliveries: toAPIDeliveries(sent)})
}

func announcementOf(
	destinations *[]types.UUID,
	roles *[]types.UUID,
	note *string,
) publication.Announcement {
	made := publication.Announcement{Ping: readIDs(roles)}
	if destinations != nil {
		chosen := readIDs(destinations)
		made.Destinations = &chosen
	}
	if note != nil {
		made.Note = *note
	}
	return made
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

func readDestinationPolicy(c *gin.Context) (publication.DestinationPolicy, bool) {
	var request DestinationPolicyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, CodeInvalid,
			"Send the destinations as JSON.", "destinationIds")
		return publication.DestinationPolicy{}, false
	}
	policy := publication.DestinationPolicy{Defaults: readIDs(request.DefaultDestinationIds)}
	if request.DestinationIds != nil {
		allowed := readIDs(request.DestinationIds)
		policy.Allowed = &allowed
	}
	return policy, true
}

func readIDs(listed *[]types.UUID) []uuid.UUID {
	if listed == nil {
		return nil
	}
	held := make([]uuid.UUID, 0, len(*listed))
	for _, one := range *listed {
		held = append(held, uuid.UUID(one))
	}
	return held
}

func (h *Handlers) destinationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, publication.ErrDestinationNotFound):
		refusePublication(c, http.StatusNotFound, CodeNotFound, "No such destination.")
	case errors.Is(err, publication.ErrDestinationRefused):
		refusePublication(c, http.StatusForbidden, CodeForbidden,
			"This post may not send to that destination.")
	case errors.Is(err, publication.ErrRoleRefused):
		refusePublication(c, http.StatusForbidden, CodeForbidden,
			"This post may not mention that destination's role.")
	case errors.Is(err, publication.ErrNotDiscord):
		refusePublication(c, http.StatusBadRequest, CodeInvalid,
			"That destination is a generic webhook.")
	case errors.Is(err, publication.ErrNotWebhook):
		refusePublication(c, http.StatusBadRequest, CodeInvalid,
			"That destination is a Discord channel.")
	case errors.Is(err, publication.ErrDeliveryNotFound):
		refusePublication(c, http.StatusNotFound, CodeNotFound, "No such delivery.")
	case errors.Is(err, publication.ErrDeliveryUnsettled):
		refusePublication(c, http.StatusBadRequest, CodeInvalid,
			"This delivery is still trying on its own.")
	case errors.Is(err, publication.ErrDeliveryUnsendable):
		refusePublication(c, http.StatusBadRequest, CodeInvalid,
			"There is nowhere left to send this delivery.")
	default:
		h.publicationError(c, err)
	}
}

func toAPIDestinations(configured []publication.Destination) []PublicationDestination {
	listed := make([]PublicationDestination, 0, len(configured))
	for _, one := range configured {
		listed = append(listed, toAPIDestination(one))
	}
	return listed
}

func toAPIDestination(found publication.Destination) PublicationDestination {
	events := make([]PublicationEvent, 0, len(found.Events))
	for _, one := range found.Events {
		events = append(events, PublicationEvent(one))
	}
	shown := PublicationDestination{
		Id:                  types.UUID(found.ID),
		Kind:                PublicationDestinationKind(found.Kind),
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

func toAPIChoices(held []publication.Choice, inherited bool) PublicationDestinationChoiceList {
	return PublicationDestinationChoiceList{
		Destinations: toAPIChoiceRows(held),
		Inherited:    inherited,
	}
}

func toAPIChoiceRows(held []publication.Choice) []PublicationDestinationChoice {
	listed := make([]PublicationDestinationChoice, 0, len(held))
	for _, one := range held {
		events := make([]PublicationEvent, 0, len(one.Events))
		for _, name := range one.Events {
			events = append(events, PublicationEvent(name))
		}
		listed = append(listed, PublicationDestinationChoice{
			Id:        types.UUID(one.ID),
			Name:      one.Name,
			Kind:      PublicationDestinationKind(one.Kind),
			State:     PublicationDestinationState(one.State),
			Events:    events,
			Role:      one.Role,
			ByDefault: one.ByDefault,
		})
	}
	return listed
}

func toAPIDeliveries(sent []publication.Delivery) []PostDelivery {
	listed := make([]PostDelivery, 0, len(sent))
	for _, one := range sent {
		listed = append(listed, toAPIDelivery(one))
	}
	return listed
}

func toAPIDelivery(one publication.Delivery) PostDelivery {
	shown := PostDelivery{
		Id:          types.UUID(one.ID),
		EventId:     types.UUID(one.EventID),
		EventType:   one.EventType,
		PostId:      types.UUID(one.PostID),
		PostTitle:   one.PostTitle,
		RevisionId:  types.UUID(one.RevisionID),
		Destination: one.Destination,
		Kind:        PostDeliveryKind(one.Kind),
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

func toAPIAttempts(made []publication.DeliveryAttempt) []PostDeliveryAttempt {
	listed := make([]PostDeliveryAttempt, 0, len(made))
	for _, one := range made {
		listed = append(listed, toAPIAttempt(one))
	}
	return listed
}

func toAPIAttempt(one publication.DeliveryAttempt) PostDeliveryAttempt {
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
