package blog

import "github.com/google/uuid"

type PublicationDestinationChoice struct {
	ByDefault bool                        `json:"byDefault"`
	Events    []PublicationEvent          `json:"events"`
	Id        uuid.UUID                   `json:"id"`
	Kind      PublicationDestinationKind  `json:"kind"`
	Name      string                      `json:"name"`
	Role      string                      `json:"role"`
	State     PublicationDestinationState `json:"state"`
}

type PublicationDestinationKind string

const (
	PublicationDestinationKindDiscord PublicationDestinationKind = "discord"
	PublicationDestinationKindWebhook PublicationDestinationKind = "webhook"
)

type PublicationDestinationState string

const (
	PublicationDestinationStateActive     PublicationDestinationState = "active"
	PublicationDestinationStateDisabled   PublicationDestinationState = "disabled"
	PublicationDestinationStateUnverified PublicationDestinationState = "unverified"
)

type PublicationEvent string

const (
	PublicationEventPublicationPostPublishedV1 PublicationEvent = "publication.post.published.v1"
	PublicationEventPublicationPostUpdatedV1   PublicationEvent = "publication.post.updated.v1"
	PublicationEventPublicationPostWithdrawnV1 PublicationEvent = "publication.post.withdrawn.v1"
)

func ChoiceRows(held []Choice) []PublicationDestinationChoice {
	listed := make([]PublicationDestinationChoice, 0, len(held))
	for _, one := range held {
		events := make([]PublicationEvent, 0, len(one.Events))
		for _, name := range one.Events {
			events = append(events, PublicationEvent(name))
		}
		listed = append(listed, PublicationDestinationChoice{
			Id:        one.ID,
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
