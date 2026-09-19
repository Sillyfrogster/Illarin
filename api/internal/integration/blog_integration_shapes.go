package integration

import (
	"time"

	announcements "github.com/Sillyfrogster/Illarin/api/internal/integration/blog"

	"github.com/google/uuid"
)

type AddBlogIntegrationRequest struct {
	Address       string                  `json:"address"`
	Announcements *[]BlogAnnouncementType `json:"announcements,omitempty"`
	Name          string                  `json:"name"`
}

type AddedBlogIntegration struct {
	Integration BlogIntegration `json:"integration"`
	Secret      string          `json:"secret"`
}

type IntegrationPolicyRequest struct {
	DefaultIntegrationIds *[]uuid.UUID `json:"defaultIntegrationIds,omitempty"`
	IntegrationIds        *[]uuid.UUID `json:"integrationIds,omitempty"`
}

type BlogAnnouncementAttempt struct {
	Tries            int                                   `json:"tries"`
	Integration      string                                `json:"integration"`
	DueAt            time.Time                             `json:"dueAt"`
	AnnouncementId   uuid.UUID                             `json:"announcementId"`
	AnnouncementType string                                `json:"announcementType"`
	Id               uuid.UUID                             `json:"id"`
	Type             BlogAnnouncementAttemptType           `json:"type"`
	Last             *BlogAnnouncementTry                  `json:"last,omitempty"`
	MessageId        string                                `json:"messageId"`
	OccurredAt       time.Time                             `json:"occurredAt"`
	PostId           uuid.UUID                             `json:"postId"`
	PostTitle        string                                `json:"postTitle"`
	Removed          bool                                  `json:"removed"`
	RevisionId       uuid.UUID                             `json:"revisionId"`
	Run              int                                   `json:"run"`
	SettledAt        *time.Time                            `json:"settledAt,omitempty"`
	SettledReason    *BlogAnnouncementAttemptSettledReason `json:"settledReason,omitempty"`
	State            BlogAnnouncementAttemptState          `json:"state"`
}

type BlogAnnouncementAttemptType string

const (
	BlogAnnouncementAttemptTypeDiscord BlogAnnouncementAttemptType = "discord"
	BlogAnnouncementAttemptTypeEmpty   BlogAnnouncementAttemptType = ""
	BlogAnnouncementAttemptTypeWebhook BlogAnnouncementAttemptType = "webhook"
)

type BlogAnnouncementTry struct {
	AttemptedAt time.Time                  `json:"attemptedAt"`
	Detail      string                     `json:"detail"`
	Number      int                        `json:"number"`
	Outcome     BlogAnnouncementTryOutcome `json:"outcome"`
	Run         int                        `json:"run"`
	Status      *int                       `json:"status,omitempty"`
	TookMs      int                        `json:"tookMs"`
}

type BlogAnnouncementTryList struct {
	Tries []BlogAnnouncementTry `json:"tries"`
}

type BlogAnnouncementAttemptList struct {
	Attempts []BlogAnnouncementAttempt `json:"attempts"`
}

type BlogAnnouncementTryOutcome string

const (
	BlogAnnouncementTryOutcomeDelivered   BlogAnnouncementTryOutcome = "delivered"
	BlogAnnouncementTryOutcomeRefused     BlogAnnouncementTryOutcome = "refused"
	BlogAnnouncementTryOutcomeUnconfirmed BlogAnnouncementTryOutcome = "unconfirmed"
	BlogAnnouncementTryOutcomeUnreachable BlogAnnouncementTryOutcome = "unreachable"
)

type BlogAnnouncementAttemptSettledReason string

const (
	BlogAnnouncementAttemptSettledReasonArrived     BlogAnnouncementAttemptSettledReason = "arrived"
	BlogAnnouncementAttemptSettledReasonDisabled    BlogAnnouncementAttemptSettledReason = "disabled"
	BlogAnnouncementAttemptSettledReasonExhausted   BlogAnnouncementAttemptSettledReason = "exhausted"
	BlogAnnouncementAttemptSettledReasonGone        BlogAnnouncementAttemptSettledReason = "gone"
	BlogAnnouncementAttemptSettledReasonMoved       BlogAnnouncementAttemptSettledReason = "moved"
	BlogAnnouncementAttemptSettledReasonRefused     BlogAnnouncementAttemptSettledReason = "refused"
	BlogAnnouncementAttemptSettledReasonRemoved     BlogAnnouncementAttemptSettledReason = "removed"
	BlogAnnouncementAttemptSettledReasonUnconfirmed BlogAnnouncementAttemptSettledReason = "unconfirmed"
)

type BlogAnnouncementAttemptState string

const (
	BlogAnnouncementAttemptStateDelivered   BlogAnnouncementAttemptState = "delivered"
	BlogAnnouncementAttemptStateFailed      BlogAnnouncementAttemptState = "failed"
	BlogAnnouncementAttemptStatePending     BlogAnnouncementAttemptState = "pending"
	BlogAnnouncementAttemptStateSending     BlogAnnouncementAttemptState = "sending"
	BlogAnnouncementAttemptStateUnconfirmed BlogAnnouncementAttemptState = "unconfirmed"
)

type BlogChannel struct {
	ChannelId   string `json:"channelId"`
	GuildId     string `json:"guildId"`
	RoleId      string `json:"roleId"`
	RoleName    string `json:"roleName"`
	WebhookName string `json:"webhookName"`
}

type BlogChannelRequest struct {
	Address  *string `json:"address,omitempty"`
	Name     string  `json:"name"`
	RoleId   *string `json:"roleId,omitempty"`
	RoleName *string `json:"roleName,omitempty"`
}

type BlogIntegration struct {
	Address             string                 `json:"address"`
	Channel             *BlogChannel           `json:"channel,omitempty"`
	CreatedAt           time.Time              `json:"createdAt"`
	DisabledAt          *time.Time             `json:"disabledAt,omitempty"`
	Announcements       []BlogAnnouncementType `json:"announcements"`
	Host                string                 `json:"host"`
	Id                  uuid.UUID              `json:"id"`
	Type                BlogIntegrationType    `json:"type"`
	Name                string                 `json:"name"`
	PreviousSecretUntil *time.Time             `json:"previousSecretUntil,omitempty"`
	SecretSetAt         time.Time              `json:"secretSetAt"`
	State               BlogIntegrationState   `json:"state"`
	VerifiedAt          *time.Time             `json:"verifiedAt,omitempty"`
}

type BlogIntegrationChoice = announcements.BlogIntegrationChoice

type BlogIntegrationChoiceList struct {
	Integrations []BlogIntegrationChoice `json:"integrations"`
	Inherited    bool                    `json:"inherited"`
}

type BlogIntegrationType = announcements.BlogIntegrationType

type BlogIntegrationList struct {
	Integrations []BlogIntegration `json:"integrations"`
}

type BlogIntegrationState = announcements.BlogIntegrationState

type BlogAnnouncementType = announcements.BlogAnnouncementType

type BlogAnnouncementApp struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Url  string `json:"url"`
}

type BlogAnnouncement struct {
	Id         uuid.UUID            `json:"id"`
	Note       *string              `json:"note,omitempty"`
	OccurredAt time.Time            `json:"occurredAt"`
	Post       BlogAnnouncementPost `json:"post"`
	Type       BlogAnnouncementType `json:"type"`
}

type BlogAnnouncementPost struct {
	Byline struct {
		App    *BlogAnnouncementApp `json:"app,omitempty"`
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
		App     BlogAnnouncementApp `json:"app"`
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

type RotatedBlogSecret struct {
	Integration         BlogIntegration `json:"integration"`
	PreviousSecretUntil time.Time       `json:"previousSecretUntil"`
	Secret              string          `json:"secret"`
}

type UpdateBlogIntegrationRequest struct {
	Address       *string                 `json:"address,omitempty"`
	Announcements *[]BlogAnnouncementType `json:"announcements,omitempty"`
	Name          *string                 `json:"name,omitempty"`
}

type ListBlogAnnouncementAttemptsParams struct {
	State *BlogAnnouncementAttemptState `json:"state,omitempty"`
	Limit *int                          `json:"limit,omitempty"`
}
