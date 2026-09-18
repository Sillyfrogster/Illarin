package notify

import (
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	Asset       *NotificationAsset        `json:"asset,omitempty"`
	CreatedAt   time.Time                 `json:"createdAt"`
	Id          uuid.UUID                 `json:"id"`
	ReadAt      *time.Time                `json:"readAt,omitempty"`
	Reason      *string                   `json:"reason,omitempty"`
	SendTargets *[]NotificationSendTarget `json:"sendTargets,omitempty"`
	Type        NotificationType          `json:"type"`
	Update      *NotificationUpdate       `json:"update,omitempty"`
}

type NotificationAsset struct {
	Id   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type NotificationCursor struct {
	Before   time.Time `json:"before"`
	BeforeId uuid.UUID `json:"beforeId"`
}

type NotificationList struct {
	Items      []Notification      `json:"items"`
	NextCursor *NotificationCursor `json:"nextCursor,omitempty"`
}

type NotificationSendTarget struct {
	ApplicationName string    `json:"applicationName"`
	InstanceId      uuid.UUID `json:"instanceId"`
	InstanceName    string    `json:"instanceName"`
	Waiting         bool      `json:"waiting"`
}

type NotificationType string

const (
	NotificationTypeAssetRestored     NotificationType = "asset_restored"
	NotificationTypeAssetUpdated      NotificationType = "asset_updated"
	NotificationTypeAssetWithheld     NotificationType = "asset_withheld"
	NotificationTypeProfileRestored   NotificationType = "profile_restored"
	NotificationTypeProfileRestricted NotificationType = "profile_restricted"
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
