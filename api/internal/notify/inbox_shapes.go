package notify

import (
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	Work      *NotificationWork      `json:"work,omitempty"`
	Creator   *NotificationCreator   `json:"creator,omitempty"`
	CreatedAt time.Time              `json:"createdAt"`
	Id        uuid.UUID              `json:"id"`
	ReadAt    *time.Time             `json:"readAt,omitempty"`
	Reason    *string                `json:"reason,omitempty"`
	SendTo    *[]NotificationSendApp `json:"sendTo,omitempty"`
	Type      NotificationType       `json:"type"`
	Update    *NotificationUpdate    `json:"update,omitempty"`
}

type NotificationWork struct {
	Id   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type NotificationCreator struct {
	Handle string `json:"handle"`
	Name   string `json:"name"`
}

type NotificationCursor struct {
	Before   time.Time `json:"before"`
	BeforeId uuid.UUID `json:"beforeId"`
}

type NotificationList struct {
	Items      []Notification      `json:"items"`
	NextCursor *NotificationCursor `json:"nextCursor,omitempty"`
}

type NotificationSendApp struct {
	AppName        string    `json:"appName"`
	ConnectedAppId uuid.UUID `json:"connectedAppId"`
	Name           string    `json:"name"`
	Waiting        bool      `json:"waiting"`
}

type NotificationType string

const (
	NotificationTypeWorkRestored      NotificationType = "work_restored"
	NotificationTypeWorkUpdated       NotificationType = "work_updated"
	NotificationTypeWorkPublished     NotificationType = "work_published"
	NotificationTypeWorkTakenDown     NotificationType = "work_taken_down"
	NotificationTypeProfileRestored   NotificationType = "profile_restored"
	NotificationTypeProfileRestricted NotificationType = "profile_restricted"
	NotificationTypeGitHubReleaseHeld NotificationType = "github_release_held"
)

type NotificationUpdate struct {
	Count        int     `json:"count"`
	Number       int     `json:"number"`
	Summary      string  `json:"summary"`
	VersionLabel *string `json:"versionLabel,omitempty"`
}

type UnreadNotifications struct {
	Count int `json:"count"`
}

type ListNotificationsParams struct {
	Limit    *int       `json:"limit,omitempty"`
	Before   *time.Time `json:"before,omitempty"`
	BeforeId *uuid.UUID `json:"beforeId,omitempty"`
}
