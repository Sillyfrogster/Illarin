package http

import (
	"time"

	"github.com/google/uuid"
)

type AddAssetUpdateDestinationRequest struct {
	Address *string                    `json:"address,omitempty"`
	Kind    AssetUpdateDestinationKind `json:"kind"`
	Name    string                     `json:"name"`
}

type AddedAssetUpdateDestination struct {
	Destination AssetUpdateDestination `json:"destination"`
	Secret      *string                `json:"secret,omitempty"`
}

type AssetUpdateAnnouncement struct {
	Attempts      int                                   `json:"attempts"`
	Destination   string                                `json:"destination"`
	DueAt         time.Time                             `json:"dueAt"`
	EventId       uuid.UUID                             `json:"eventId"`
	Id            uuid.UUID                             `json:"id"`
	Kind          AssetUpdateDestinationKind            `json:"kind"`
	Last          *AssetUpdateAnnouncementAttempt       `json:"last,omitempty"`
	MessageId     string                                `json:"messageId"`
	OccurredAt    time.Time                             `json:"occurredAt"`
	Removed       bool                                  `json:"removed"`
	Run           int                                   `json:"run"`
	SettledAt     *time.Time                            `json:"settledAt,omitempty"`
	SettledReason *AssetUpdateAnnouncementSettledReason `json:"settledReason,omitempty"`
	State         AssetUpdateAnnouncementState          `json:"state"`
	UpdateId      uuid.UUID                             `json:"updateId"`
	UpdateNumber  int                                   `json:"updateNumber"`
}

type AssetUpdateAnnouncementAttempt struct {
	AttemptedAt time.Time                             `json:"attemptedAt"`
	Detail      string                                `json:"detail"`
	Number      int                                   `json:"number"`
	Outcome     AssetUpdateAnnouncementAttemptOutcome `json:"outcome"`
	Run         int                                   `json:"run"`
	Status      *int                                  `json:"status,omitempty"`
	TookMs      int                                   `json:"tookMs"`
}

type AssetUpdateAnnouncementAttemptOutcome string

const (
	AssetUpdateAnnouncementAttemptOutcomeDelivered   AssetUpdateAnnouncementAttemptOutcome = "delivered"
	AssetUpdateAnnouncementAttemptOutcomeRefused     AssetUpdateAnnouncementAttemptOutcome = "refused"
	AssetUpdateAnnouncementAttemptOutcomeUnconfirmed AssetUpdateAnnouncementAttemptOutcome = "unconfirmed"
	AssetUpdateAnnouncementAttemptOutcomeUnreachable AssetUpdateAnnouncementAttemptOutcome = "unreachable"
)

type AssetUpdateAnnouncementList struct {
	Announcements []AssetUpdateAnnouncement `json:"announcements"`
}

type AssetUpdateAnnouncementSettledReason string

const (
	AssetUpdateAnnouncementSettledReasonArrived     AssetUpdateAnnouncementSettledReason = "arrived"
	AssetUpdateAnnouncementSettledReasonDeleted     AssetUpdateAnnouncementSettledReason = "deleted"
	AssetUpdateAnnouncementSettledReasonDisabled    AssetUpdateAnnouncementSettledReason = "disabled"
	AssetUpdateAnnouncementSettledReasonExhausted   AssetUpdateAnnouncementSettledReason = "exhausted"
	AssetUpdateAnnouncementSettledReasonGone        AssetUpdateAnnouncementSettledReason = "gone"
	AssetUpdateAnnouncementSettledReasonMoved       AssetUpdateAnnouncementSettledReason = "moved"
	AssetUpdateAnnouncementSettledReasonRefused     AssetUpdateAnnouncementSettledReason = "refused"
	AssetUpdateAnnouncementSettledReasonRemoved     AssetUpdateAnnouncementSettledReason = "removed"
	AssetUpdateAnnouncementSettledReasonUnconfirmed AssetUpdateAnnouncementSettledReason = "unconfirmed"
	AssetUpdateAnnouncementSettledReasonUnlisted    AssetUpdateAnnouncementSettledReason = "unlisted"
	AssetUpdateAnnouncementSettledReasonWithdrawn   AssetUpdateAnnouncementSettledReason = "withdrawn"
	AssetUpdateAnnouncementSettledReasonWithheld    AssetUpdateAnnouncementSettledReason = "withheld"
)

type AssetUpdateAnnouncementState string

const (
	AssetUpdateAnnouncementStateDelivered   AssetUpdateAnnouncementState = "delivered"
	AssetUpdateAnnouncementStateFailed      AssetUpdateAnnouncementState = "failed"
	AssetUpdateAnnouncementStatePending     AssetUpdateAnnouncementState = "pending"
	AssetUpdateAnnouncementStateSending     AssetUpdateAnnouncementState = "sending"
	AssetUpdateAnnouncementStateUnconfirmed AssetUpdateAnnouncementState = "unconfirmed"
)

type AssetUpdateChannel struct {
	ChannelId string `json:"channelId"`
	GuildId   string `json:"guildId"`
}

type AssetUpdateDestination struct {
	Address             string                      `json:"address"`
	Channel             *AssetUpdateChannel         `json:"channel,omitempty"`
	CreatedAt           time.Time                   `json:"createdAt"`
	DisabledAt          *time.Time                  `json:"disabledAt,omitempty"`
	Host                string                      `json:"host"`
	Id                  uuid.UUID                   `json:"id"`
	Kind                AssetUpdateDestinationKind  `json:"kind"`
	Name                string                      `json:"name"`
	PreviousSecretUntil *time.Time                  `json:"previousSecretUntil,omitempty"`
	SecretSetAt         *time.Time                  `json:"secretSetAt,omitempty"`
	State               AssetUpdateDestinationState `json:"state"`
	VerifiedAt          *time.Time                  `json:"verifiedAt,omitempty"`
}

type AssetUpdateDestinationState string

const (
	AssetUpdateDestinationStateActive     AssetUpdateDestinationState = "active"
	AssetUpdateDestinationStateDisabled   AssetUpdateDestinationState = "disabled"
	AssetUpdateDestinationStateUnverified AssetUpdateDestinationState = "unverified"
)

type AssetUpdateDestinationChoice struct {
	ByDefault bool                       `json:"byDefault"`
	Id        uuid.UUID                  `json:"id"`
	Kind      AssetUpdateDestinationKind `json:"kind"`
	Name      string                     `json:"name"`
}

type AssetUpdateDestinationChoices struct {
	Destinations []AssetUpdateDestinationChoice `json:"destinations"`
}

type AssetUpdateDestinationDefaultsRequest struct {
	DestinationIds []uuid.UUID `json:"destinationIds"`
}

type AssetUpdateDestinationKind string

const (
	AssetUpdateDestinationKindDiscord AssetUpdateDestinationKind = "discord"
	AssetUpdateDestinationKindWebhook AssetUpdateDestinationKind = "webhook"
)

type AssetUpdateDestinationList struct {
	Destinations []AssetUpdateDestination `json:"destinations"`
}

type UpdateAssetUpdateDestinationRequest struct {
	Address *string `json:"address,omitempty"`
	Name    *string `json:"name,omitempty"`
}
