package integration

import (
	"time"

	"github.com/google/uuid"
)

type AddWorkUpdateDestinationRequest struct {
	Address *string                   `json:"address,omitempty"`
	Type    WorkUpdateDestinationType `json:"kind"`
	Name    string                    `json:"name"`
}

type AddedWorkUpdateDestination struct {
	Destination WorkUpdateDestination `json:"destination"`
	Secret      *string               `json:"secret,omitempty"`
}

type WorkUpdateAnnouncement struct {
	Attempts      int                                  `json:"attempts"`
	Destination   string                               `json:"destination"`
	DueAt         time.Time                            `json:"dueAt"`
	EventId       uuid.UUID                            `json:"eventId"`
	Id            uuid.UUID                            `json:"id"`
	Type          WorkUpdateDestinationType            `json:"kind"`
	Last          *WorkUpdateAnnouncementAttempt       `json:"last,omitempty"`
	MessageId     string                               `json:"messageId"`
	OccurredAt    time.Time                            `json:"occurredAt"`
	Removed       bool                                 `json:"removed"`
	Run           int                                  `json:"run"`
	SettledAt     *time.Time                           `json:"settledAt,omitempty"`
	SettledReason *WorkUpdateAnnouncementSettledReason `json:"settledReason,omitempty"`
	State         WorkUpdateAnnouncementState          `json:"state"`
	UpdateId      uuid.UUID                            `json:"updateId"`
	UpdateNumber  int                                  `json:"updateNumber"`
}

type WorkUpdateAnnouncementAttempt struct {
	AttemptedAt time.Time                            `json:"attemptedAt"`
	Detail      string                               `json:"detail"`
	Number      int                                  `json:"number"`
	Outcome     WorkUpdateAnnouncementAttemptOutcome `json:"outcome"`
	Run         int                                  `json:"run"`
	Status      *int                                 `json:"status,omitempty"`
	TookMs      int                                  `json:"tookMs"`
}

type WorkUpdateAnnouncementAttemptOutcome string

const (
	WorkUpdateAnnouncementAttemptOutcomeDelivered   WorkUpdateAnnouncementAttemptOutcome = "delivered"
	WorkUpdateAnnouncementAttemptOutcomeRefused     WorkUpdateAnnouncementAttemptOutcome = "refused"
	WorkUpdateAnnouncementAttemptOutcomeUnconfirmed WorkUpdateAnnouncementAttemptOutcome = "unconfirmed"
	WorkUpdateAnnouncementAttemptOutcomeUnreachable WorkUpdateAnnouncementAttemptOutcome = "unreachable"
)

type WorkUpdateAnnouncementList struct {
	Announcements []WorkUpdateAnnouncement `json:"announcements"`
}

type WorkUpdateAnnouncementSettledReason string

const (
	WorkUpdateAnnouncementSettledReasonArrived     WorkUpdateAnnouncementSettledReason = "arrived"
	WorkUpdateAnnouncementSettledReasonDeleted     WorkUpdateAnnouncementSettledReason = "deleted"
	WorkUpdateAnnouncementSettledReasonDisabled    WorkUpdateAnnouncementSettledReason = "disabled"
	WorkUpdateAnnouncementSettledReasonExhausted   WorkUpdateAnnouncementSettledReason = "exhausted"
	WorkUpdateAnnouncementSettledReasonGone        WorkUpdateAnnouncementSettledReason = "gone"
	WorkUpdateAnnouncementSettledReasonMoved       WorkUpdateAnnouncementSettledReason = "moved"
	WorkUpdateAnnouncementSettledReasonRefused     WorkUpdateAnnouncementSettledReason = "refused"
	WorkUpdateAnnouncementSettledReasonRemoved     WorkUpdateAnnouncementSettledReason = "removed"
	WorkUpdateAnnouncementSettledReasonUnconfirmed WorkUpdateAnnouncementSettledReason = "unconfirmed"
	WorkUpdateAnnouncementSettledReasonUnlisted    WorkUpdateAnnouncementSettledReason = "unlisted"
	WorkUpdateAnnouncementSettledReasonWithdrawn   WorkUpdateAnnouncementSettledReason = "withdrawn"
	WorkUpdateAnnouncementSettledReasonWithheld    WorkUpdateAnnouncementSettledReason = "withheld"
)

type WorkUpdateAnnouncementState string

const (
	WorkUpdateAnnouncementStateDelivered   WorkUpdateAnnouncementState = "delivered"
	WorkUpdateAnnouncementStateFailed      WorkUpdateAnnouncementState = "failed"
	WorkUpdateAnnouncementStatePending     WorkUpdateAnnouncementState = "pending"
	WorkUpdateAnnouncementStateSending     WorkUpdateAnnouncementState = "sending"
	WorkUpdateAnnouncementStateUnconfirmed WorkUpdateAnnouncementState = "unconfirmed"
)

type WorkUpdateChannel struct {
	ChannelId string `json:"channelId"`
	GuildId   string `json:"guildId"`
}

type WorkUpdateDestination struct {
	Address             string                     `json:"address"`
	Channel             *WorkUpdateChannel         `json:"channel,omitempty"`
	CreatedAt           time.Time                  `json:"createdAt"`
	DisabledAt          *time.Time                 `json:"disabledAt,omitempty"`
	Host                string                     `json:"host"`
	Id                  uuid.UUID                  `json:"id"`
	Type                WorkUpdateDestinationType  `json:"kind"`
	Name                string                     `json:"name"`
	PreviousSecretUntil *time.Time                 `json:"previousSecretUntil,omitempty"`
	SecretSetAt         *time.Time                 `json:"secretSetAt,omitempty"`
	State               WorkUpdateDestinationState `json:"state"`
	VerifiedAt          *time.Time                 `json:"verifiedAt,omitempty"`
}

type WorkUpdateDestinationState string

const (
	WorkUpdateDestinationStateActive     WorkUpdateDestinationState = "active"
	WorkUpdateDestinationStateDisabled   WorkUpdateDestinationState = "disabled"
	WorkUpdateDestinationStateUnverified WorkUpdateDestinationState = "unverified"
)

type WorkUpdateDestinationChoice struct {
	ByDefault bool                      `json:"byDefault"`
	Id        uuid.UUID                 `json:"id"`
	Type      WorkUpdateDestinationType `json:"kind"`
	Name      string                    `json:"name"`
}

type WorkUpdateDestinationChoices struct {
	Destinations []WorkUpdateDestinationChoice `json:"destinations"`
}

type WorkUpdateDestinationDefaultsRequest struct {
	DestinationIds []uuid.UUID `json:"destinationIds"`
}

type WorkUpdateDestinationType string

const (
	WorkUpdateDestinationTypeDiscord WorkUpdateDestinationType = "discord"
	WorkUpdateDestinationTypeWebhook WorkUpdateDestinationType = "webhook"
)

type WorkUpdateDestinationList struct {
	Destinations []WorkUpdateDestination `json:"destinations"`
}

type UpdateWorkUpdateDestinationRequest struct {
	Address *string `json:"address,omitempty"`
	Name    *string `json:"name,omitempty"`
}
