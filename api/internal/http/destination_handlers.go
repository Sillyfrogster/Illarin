package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

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
		publication.DestinationEdit{Name: request.Name, Address: request.Address},
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
		publication.DestinationUpdate{Name: request.Name, Address: request.Address},
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

func (h *Handlers) SetPublicationAppDestinations(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "setting an app's destinations")
	if !ok {
		return
	}
	policy, ok := readDestinationPolicy(c)
	if !ok {
		return
	}
	err := h.publications.SetAppDestinations(
		c.Request.Context(), authority.ID, uuid.UUID(id), policy,
	)
	if err != nil {
		h.destinationError(c, err)
		return
	}
	allowed, err := h.publications.AppChoices(c.Request.Context(), uuid.UUID(id))
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIChoices(allowed, false))
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
	allowed, err := h.publications.GrantChoices(c.Request.Context(), uuid.UUID(id))
	if err != nil {
		h.destinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIChoices(allowed, policy.Allowed == nil))
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

// announcementOf reads a transition's delivery choice, keeping an absent list
// apart from an empty one because they mean opposite things.
func announcementOf(destinations *[]types.UUID, note *string) publication.Announcement {
	made := publication.Announcement{}
	if destinations != nil {
		chosen := readIDs(destinations)
		made.Destinations = &chosen
	}
	if note != nil {
		made.Note = *note
	}
	return made
}

// readDestinationPolicy reads an allowed and default set, keeping an absent
// list apart from an empty one because they mean opposite things.
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
	return PublicationDestination{
		Id:         types.UUID(found.ID),
		Kind:       PublicationDestinationKind(found.Kind),
		Name:       found.Name,
		Host:       found.Host,
		Address:    found.Address,
		State:      PublicationDestinationState(found.State),
		VerifiedAt: found.VerifiedAt,
		DisabledAt: found.DisabledAt,
		CreatedAt:  found.CreatedAt,
	}
}

func toAPIChoices(held []publication.Choice, inherited bool) PublicationDestinationChoiceList {
	listed := make([]PublicationDestinationChoice, 0, len(held))
	for _, one := range held {
		listed = append(listed, PublicationDestinationChoice{
			Id:        types.UUID(one.ID),
			Name:      one.Name,
			Kind:      PublicationDestinationChoiceKind(one.Kind),
			State:     PublicationDestinationState(one.State),
			ByDefault: one.ByDefault,
		})
	}
	return PublicationDestinationChoiceList{Destinations: listed, Inherited: inherited}
}

func toAPIDeliveries(sent []publication.Delivery) []PostDelivery {
	listed := make([]PostDelivery, 0, len(sent))
	for _, one := range sent {
		shown := PostDelivery{
			Id:          types.UUID(one.ID),
			EventId:     types.UUID(one.EventID),
			EventType:   one.EventType,
			PostId:      types.UUID(one.PostID),
			RevisionId:  types.UUID(one.RevisionID),
			Destination: one.Destination,
			State:       PostDeliveryState(one.State),
			Attempts:    one.Attempts,
			OccurredAt:  one.OccurredAt,
			SettledAt:   one.SettledAt,
		}
		if one.Last != nil {
			shown.Last = &PostDeliveryAttempt{
				Number:      one.Last.Number,
				Outcome:     PostDeliveryOutcome(one.Last.Outcome),
				Status:      one.Last.Status,
				Detail:      one.Last.Detail,
				TookMs:      int(one.Last.Took.Milliseconds()),
				AttemptedAt: one.Last.Attempted,
			}
		}
		listed = append(listed, shown)
	}
	return listed
}
