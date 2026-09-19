package integration

import (
	"time"

	"github.com/google/uuid"
)

type AddWorkIntegrationRequest struct {
	Address *string             `json:"address,omitempty"`
	Type    WorkIntegrationType `json:"type"`
	Name    string              `json:"name"`
}

type AddedWorkIntegration struct {
	Integration WorkIntegration `json:"integration"`
	Secret      *string         `json:"secret,omitempty"`
}

type WorkAnnouncementAttempt struct {
	Tries          int                                   `json:"tries"`
	Integration    string                                `json:"integration"`
	DueAt          time.Time                             `json:"dueAt"`
	AnnouncementId uuid.UUID                             `json:"announcementId"`
	Id             uuid.UUID                             `json:"id"`
	Type           WorkIntegrationType                   `json:"type"`
	Last           *WorkAnnouncementTry                  `json:"last,omitempty"`
	MessageId      string                                `json:"messageId"`
	OccurredAt     time.Time                             `json:"occurredAt"`
	Removed        bool                                  `json:"removed"`
	Run            int                                   `json:"run"`
	SettledAt      *time.Time                            `json:"settledAt,omitempty"`
	SettledReason  *WorkAnnouncementAttemptSettledReason `json:"settledReason,omitempty"`
	State          WorkAnnouncementAttemptState          `json:"state"`
	VersionId      uuid.UUID                             `json:"versionId"`
	VersionNumber  int                                   `json:"versionNumber"`
}

type WorkAnnouncementTry struct {
	AttemptedAt time.Time                  `json:"attemptedAt"`
	Detail      string                     `json:"detail"`
	Number      int                        `json:"number"`
	Outcome     WorkAnnouncementTryOutcome `json:"outcome"`
	Run         int                        `json:"run"`
	Status      *int                       `json:"status,omitempty"`
	TookMs      int                        `json:"tookMs"`
}

type WorkAnnouncementTryOutcome string

const (
	WorkAnnouncementTryOutcomeDelivered   WorkAnnouncementTryOutcome = "delivered"
	WorkAnnouncementTryOutcomeRefused     WorkAnnouncementTryOutcome = "refused"
	WorkAnnouncementTryOutcomeUnconfirmed WorkAnnouncementTryOutcome = "unconfirmed"
	WorkAnnouncementTryOutcomeUnreachable WorkAnnouncementTryOutcome = "unreachable"
)

type WorkAnnouncementAttemptList struct {
	Attempts []WorkAnnouncementAttempt `json:"attempts"`
}

type WorkAnnouncementAttemptSettledReason string

const (
	WorkAnnouncementAttemptSettledReasonArrived     WorkAnnouncementAttemptSettledReason = "arrived"
	WorkAnnouncementAttemptSettledReasonDeleted     WorkAnnouncementAttemptSettledReason = "deleted"
	WorkAnnouncementAttemptSettledReasonDisabled    WorkAnnouncementAttemptSettledReason = "disabled"
	WorkAnnouncementAttemptSettledReasonExhausted   WorkAnnouncementAttemptSettledReason = "exhausted"
	WorkAnnouncementAttemptSettledReasonGone        WorkAnnouncementAttemptSettledReason = "gone"
	WorkAnnouncementAttemptSettledReasonMoved       WorkAnnouncementAttemptSettledReason = "moved"
	WorkAnnouncementAttemptSettledReasonRefused     WorkAnnouncementAttemptSettledReason = "refused"
	WorkAnnouncementAttemptSettledReasonRemoved     WorkAnnouncementAttemptSettledReason = "removed"
	WorkAnnouncementAttemptSettledReasonUnconfirmed WorkAnnouncementAttemptSettledReason = "unconfirmed"
	WorkAnnouncementAttemptSettledReasonUnlisted    WorkAnnouncementAttemptSettledReason = "unlisted"
	WorkAnnouncementAttemptSettledReasonWithdrawn   WorkAnnouncementAttemptSettledReason = "withdrawn"
	WorkAnnouncementAttemptSettledReasonWithheld    WorkAnnouncementAttemptSettledReason = "withheld"
)

type WorkAnnouncementAttemptState string

const (
	WorkAnnouncementAttemptStateDelivered   WorkAnnouncementAttemptState = "delivered"
	WorkAnnouncementAttemptStateFailed      WorkAnnouncementAttemptState = "failed"
	WorkAnnouncementAttemptStatePending     WorkAnnouncementAttemptState = "pending"
	WorkAnnouncementAttemptStateSending     WorkAnnouncementAttemptState = "sending"
	WorkAnnouncementAttemptStateUnconfirmed WorkAnnouncementAttemptState = "unconfirmed"
)

type WorkIntegrationChannel struct {
	ChannelId string `json:"channelId"`
	GuildId   string `json:"guildId"`
}

type WorkIntegration struct {
	Address             string                  `json:"address"`
	Channel             *WorkIntegrationChannel `json:"channel,omitempty"`
	CreatedAt           time.Time               `json:"createdAt"`
	DisabledAt          *time.Time              `json:"disabledAt,omitempty"`
	Host                string                  `json:"host"`
	Id                  uuid.UUID               `json:"id"`
	Type                WorkIntegrationType     `json:"type"`
	Name                string                  `json:"name"`
	PreviousSecretUntil *time.Time              `json:"previousSecretUntil,omitempty"`
	SecretSetAt         *time.Time              `json:"secretSetAt,omitempty"`
	State               WorkIntegrationState    `json:"state"`
	VerifiedAt          *time.Time              `json:"verifiedAt,omitempty"`
}

type WorkIntegrationState string

const (
	WorkIntegrationStateActive     WorkIntegrationState = "active"
	WorkIntegrationStateDisabled   WorkIntegrationState = "disabled"
	WorkIntegrationStateUnverified WorkIntegrationState = "unverified"
)

type WorkIntegrationChoice struct {
	ByDefault bool                `json:"byDefault"`
	Id        uuid.UUID           `json:"id"`
	Type      WorkIntegrationType `json:"type"`
	Name      string              `json:"name"`
}

type WorkIntegrationChoices struct {
	Integrations []WorkIntegrationChoice `json:"integrations"`
}

type WorkIntegrationDefaultsRequest struct {
	IntegrationIds []uuid.UUID `json:"integrationIds"`
}

type WorkIntegrationType string

const (
	WorkIntegrationTypeDiscord WorkIntegrationType = "discord"
	WorkIntegrationTypeWebhook WorkIntegrationType = "webhook"
)

type WorkIntegrationList struct {
	Integrations []WorkIntegration `json:"integrations"`
}

type UpdateWorkIntegrationRequest struct {
	Address *string `json:"address,omitempty"`
	Name    *string `json:"name,omitempty"`
}
