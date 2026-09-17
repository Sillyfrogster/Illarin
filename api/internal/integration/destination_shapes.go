package integration

import (
	"time"

	"github.com/google/uuid"
)

type AddPublicationDestinationRequest struct {
	Address string              `json:"address"`
	Events  *[]PublicationEvent `json:"events,omitempty"`
	Name    string              `json:"name"`
}

type AddedPublicationDestination struct {
	Destination PublicationDestination `json:"destination"`
	Secret      string                 `json:"secret"`
}

type DestinationPolicyRequest struct {
	DefaultDestinationIds *[]uuid.UUID `json:"defaultDestinationIds,omitempty"`
	DestinationIds        *[]uuid.UUID `json:"destinationIds,omitempty"`
}

type PostDelivery struct {
	Attempts      int                        `json:"attempts"`
	Destination   string                     `json:"destination"`
	DueAt         time.Time                  `json:"dueAt"`
	EventId       uuid.UUID                  `json:"eventId"`
	EventType     string                     `json:"eventType"`
	Id            uuid.UUID                  `json:"id"`
	Kind          PostDeliveryKind           `json:"kind"`
	Last          *PostDeliveryAttempt       `json:"last,omitempty"`
	MessageId     string                     `json:"messageId"`
	OccurredAt    time.Time                  `json:"occurredAt"`
	PostId        uuid.UUID                  `json:"postId"`
	PostTitle     string                     `json:"postTitle"`
	Removed       bool                       `json:"removed"`
	RevisionId    uuid.UUID                  `json:"revisionId"`
	Run           int                        `json:"run"`
	SettledAt     *time.Time                 `json:"settledAt,omitempty"`
	SettledReason *PostDeliverySettledReason `json:"settledReason,omitempty"`
	State         PostDeliveryState          `json:"state"`
}

type PostDeliveryKind string

const (
	PostDeliveryKindDiscord PostDeliveryKind = "discord"
	PostDeliveryKindEmpty   PostDeliveryKind = ""
	PostDeliveryKindWebhook PostDeliveryKind = "webhook"
)

type PostDeliveryAttempt struct {
	AttemptedAt time.Time           `json:"attemptedAt"`
	Detail      string              `json:"detail"`
	Number      int                 `json:"number"`
	Outcome     PostDeliveryOutcome `json:"outcome"`
	Run         int                 `json:"run"`
	Status      *int                `json:"status,omitempty"`
	TookMs      int                 `json:"tookMs"`
}

type PostDeliveryAttemptList struct {
	Attempts []PostDeliveryAttempt `json:"attempts"`
}

type PostDeliveryList struct {
	Deliveries []PostDelivery `json:"deliveries"`
}

type PostDeliveryOutcome string

const (
	PostDeliveryOutcomeDelivered   PostDeliveryOutcome = "delivered"
	PostDeliveryOutcomeRefused     PostDeliveryOutcome = "refused"
	PostDeliveryOutcomeUnconfirmed PostDeliveryOutcome = "unconfirmed"
	PostDeliveryOutcomeUnreachable PostDeliveryOutcome = "unreachable"
)

type PostDeliverySettledReason string

const (
	PostDeliverySettledReasonArrived     PostDeliverySettledReason = "arrived"
	PostDeliverySettledReasonDisabled    PostDeliverySettledReason = "disabled"
	PostDeliverySettledReasonExhausted   PostDeliverySettledReason = "exhausted"
	PostDeliverySettledReasonGone        PostDeliverySettledReason = "gone"
	PostDeliverySettledReasonMoved       PostDeliverySettledReason = "moved"
	PostDeliverySettledReasonRefused     PostDeliverySettledReason = "refused"
	PostDeliverySettledReasonRemoved     PostDeliverySettledReason = "removed"
	PostDeliverySettledReasonUnconfirmed PostDeliverySettledReason = "unconfirmed"
)

type PostDeliveryState string

const (
	PostDeliveryStateDelivered   PostDeliveryState = "delivered"
	PostDeliveryStateFailed      PostDeliveryState = "failed"
	PostDeliveryStatePending     PostDeliveryState = "pending"
	PostDeliveryStateSending     PostDeliveryState = "sending"
	PostDeliveryStateUnconfirmed PostDeliveryState = "unconfirmed"
)

type PublicationChannel struct {
	ChannelId   string `json:"channelId"`
	GuildId     string `json:"guildId"`
	RoleId      string `json:"roleId"`
	RoleName    string `json:"roleName"`
	WebhookName string `json:"webhookName"`
}

type PublicationChannelRequest struct {
	Address  *string `json:"address,omitempty"`
	Name     string  `json:"name"`
	RoleId   *string `json:"roleId,omitempty"`
	RoleName *string `json:"roleName,omitempty"`
}

type PublicationDestination struct {
	Address             string                      `json:"address"`
	Channel             *PublicationChannel         `json:"channel,omitempty"`
	CreatedAt           time.Time                   `json:"createdAt"`
	DisabledAt          *time.Time                  `json:"disabledAt,omitempty"`
	Events              []PublicationEvent          `json:"events"`
	Host                string                      `json:"host"`
	Id                  uuid.UUID                   `json:"id"`
	Kind                PublicationDestinationKind  `json:"kind"`
	Name                string                      `json:"name"`
	PreviousSecretUntil *time.Time                  `json:"previousSecretUntil,omitempty"`
	SecretSetAt         time.Time                   `json:"secretSetAt"`
	State               PublicationDestinationState `json:"state"`
	VerifiedAt          *time.Time                  `json:"verifiedAt,omitempty"`
}

type PublicationDestinationChoice struct {
	ByDefault bool                        `json:"byDefault"`
	Events    []PublicationEvent          `json:"events"`
	Id        uuid.UUID                   `json:"id"`
	Kind      PublicationDestinationKind  `json:"kind"`
	Name      string                      `json:"name"`
	Role      string                      `json:"role"`
	State     PublicationDestinationState `json:"state"`
}

type PublicationDestinationChoiceList struct {
	Destinations []PublicationDestinationChoice `json:"destinations"`
	Inherited    bool                           `json:"inherited"`
}

type PublicationDestinationKind string

const (
	PublicationDestinationKindDiscord PublicationDestinationKind = "discord"
	PublicationDestinationKindWebhook PublicationDestinationKind = "webhook"
)

type PublicationDestinationList struct {
	Destinations []PublicationDestination `json:"destinations"`
}

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

type PublicationEventApp struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Url  string `json:"url"`
}

type PublicationPostEvent struct {
	Id         uuid.UUID              `json:"id"`
	Note       *string                `json:"note,omitempty"`
	OccurredAt time.Time              `json:"occurredAt"`
	Post       PublicationPostSummary `json:"post"`
	Type       PublicationEvent       `json:"type"`
}

type PublicationPostSummary struct {
	Byline struct {
		App    *PublicationEventApp `json:"app,omitempty"`
		Handle string               `json:"handle"`
		Name   string               `json:"name"`
		Url    string               `json:"url"`
	} `json:"byline"`
	Category struct {
		Label string `json:"label"`
		Slug  string `json:"slug"`
	} `json:"category"`
	Id          uuid.UUID `json:"id"`
	PublishedAt time.Time `json:"publishedAt"`
	Release     *struct {
		App     PublicationEventApp `json:"app"`
		Url     *string             `json:"url,omitempty"`
		Version string              `json:"version"`
	} `json:"release,omitempty"`
	RevisionId     uuid.UUID  `json:"revisionId"`
	SocialImageUrl *string    `json:"socialImageUrl,omitempty"`
	Summary        string     `json:"summary"`
	Title          string     `json:"title"`
	UpdatedAt      *time.Time `json:"updatedAt,omitempty"`
	Url            string     `json:"url"`
}

type RotatedPublicationSecret struct {
	Destination         PublicationDestination `json:"destination"`
	PreviousSecretUntil time.Time              `json:"previousSecretUntil"`
	Secret              string                 `json:"secret"`
}

type UpdatePublicationDestinationRequest struct {
	Address *string             `json:"address,omitempty"`
	Events  *[]PublicationEvent `json:"events,omitempty"`
	Name    *string             `json:"name,omitempty"`
}

type ListPublicationDeliveriesParams struct {
	State *PostDeliveryState `json:"state,omitempty"`
	Limit *int               `json:"limit,omitempty"`
}
