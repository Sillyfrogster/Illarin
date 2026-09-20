package blog

import "github.com/google/uuid"

type BlogIntegrationChoice struct {
	ByDefault     bool                   `json:"byDefault"`
	Announcements []BlogAnnouncementType `json:"announcements"`
	Id            uuid.UUID              `json:"id"`
	Type          BlogIntegrationType    `json:"type"`
	Name          string                 `json:"name"`
	Role          string                 `json:"role"`
	State         BlogIntegrationState   `json:"state"`
}

type BlogIntegrationType string

const (
	BlogIntegrationTypeDiscord BlogIntegrationType = "discord"
	BlogIntegrationTypeWebhook BlogIntegrationType = "webhook"
)

type BlogIntegrationState string

const (
	BlogIntegrationStateActive     BlogIntegrationState = "active"
	BlogIntegrationStateDisabled   BlogIntegrationState = "disabled"
	BlogIntegrationStateUnverified BlogIntegrationState = "unverified"
)

type BlogAnnouncementType string

const (
	BlogAnnouncementTypePostPublished   BlogAnnouncementType = "blog.post.published.v1"
	BlogAnnouncementTypePostUpdated     BlogAnnouncementType = "blog.post.updated.v1"
	BlogAnnouncementTypePostUnpublished BlogAnnouncementType = "blog.post.unpublished.v1"
)

func ChoiceRows(held []Choice) []BlogIntegrationChoice {
	listed := make([]BlogIntegrationChoice, 0, len(held))
	for _, one := range held {
		announced := make([]BlogAnnouncementType, 0, len(one.Announcements))
		for _, name := range one.Announcements {
			announced = append(announced, BlogAnnouncementType(name))
		}
		listed = append(listed, BlogIntegrationChoice{
			Id:            one.ID,
			Name:          one.Name,
			Type:          BlogIntegrationType(one.Type),
			State:         BlogIntegrationState(one.State),
			Announcements: announced,
			Role:          one.Role,
			ByDefault:     one.ByDefault,
		})
	}
	return listed
}
