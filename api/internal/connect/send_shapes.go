package connect

import (
	"time"

	"github.com/google/uuid"
)

type WorkConnectedApp struct {
	ConnectedAppId   uuid.UUID   `json:"connectedAppId"`
	AppName          string      `json:"appName"`
	Name             string      `json:"name"`
	CanReceive       bool        `json:"canReceive"`
	Send             *QueuedSend `json:"send" tstype:"QueuedSend | null,required"`
	InstalledVersion *int        `json:"installedVersion" tstype:"number | null,required"`
	LastSeenAt       *time.Time  `json:"lastSeenAt" tstype:"string | null,required"`
	ReportsLibrary   bool        `json:"reportsLibrary"`
	UpdateAvailable  bool        `json:"updateAvailable"`
}

type WorkConnectedAppList struct {
	VersionNumber int                `json:"versionNumber"`
	Items         []WorkConnectedApp `json:"items"`
}

type CollectSends struct {
	Acknowledge []uuid.UUID `json:"acknowledge"`
}

type SendFile struct {
	IsCover *bool        `json:"isCover,omitempty"`
	Type    SendFileType `json:"type"`
	MediaId *uuid.UUID   `json:"mediaId,omitempty"`
	Role    *string      `json:"role,omitempty"`
	Url     string       `json:"url"`
}

type SendFileType string

const (
	SendFileTypeExport  SendFileType = "export"
	SendFileTypePicture SendFileType = "picture"
)

type CollectedSend struct {
	Files          []SendFile `json:"files"`
	WorkId         uuid.UUID  `json:"workId"`
	VersionNumber  int        `json:"versionNumber"`
	Format         string     `json:"format"`
	Id             uuid.UUID  `json:"id"`
	Type           string     `json:"type"`
	Label          string     `json:"label"`
	LeaseExpiresAt time.Time  `json:"leaseExpiresAt"`
	Name           string     `json:"name"`
	QueuedAt       time.Time  `json:"queuedAt"`
}

type CollectedSends struct {
	Sends    []CollectedSend  `json:"sends"`
	Withheld []WithheldNotice `json:"withheld"`
}

type LibraryEntry struct {
	WorkId        uuid.UUID `json:"workId"`
	VersionNumber *int      `json:"versionNumber,omitempty"`
}

type LibraryReport struct {
	AppVersion *string        `json:"appVersion,omitempty"`
	Entries    []LibraryEntry `json:"entries"`
	Removed    *[]uuid.UUID   `json:"removed,omitempty"`
	Snapshot   bool           `json:"snapshot"`
}

type LibraryReportResult struct {
	Accepted int              `json:"accepted"`
	Ignored  int              `json:"ignored"`
	Removed  int              `json:"removed"`
	Withheld []WithheldNotice `json:"withheld"`
}

type QueuedSend struct {
	WorkId         uuid.UUID         `json:"workId"`
	ExpiresAt      time.Time         `json:"expiresAt"`
	Id             uuid.UUID         `json:"id"`
	ConnectedAppId uuid.UUID         `json:"connectedAppId"`
	QueuedAt       time.Time         `json:"queuedAt"`
	Reason         *QueuedSendReason `json:"reason,omitempty"`
	SettledAt      *time.Time        `json:"settledAt" tstype:"string | null,required"`
	State          QueuedSendState   `json:"state"`
	UpdatesInstall bool              `json:"updatesInstall"`
}

type QueuedSendReason string

const (
	QueuedSendReasonAbandoned   QueuedSendReason = "abandoned"
	QueuedSendReasonUnsupported QueuedSendReason = "unsupported"
	QueuedSendReasonWithdrawn   QueuedSendReason = "withdrawn"
)

type QueuedSendState string

const (
	QueuedSendStateDelivered QueuedSendState = "delivered"
	QueuedSendStateFailed    QueuedSendState = "failed"
	QueuedSendStateQueued    QueuedSendState = "queued"
	QueuedSendStateReleased  QueuedSendState = "released"
)

type SendWorkRequest struct {
	ConnectedAppId uuid.UUID `json:"connectedAppId"`
}

type WithheldNotice struct {
	WorkId     uuid.UUID `json:"workId"`
	Name       string    `json:"name"`
	WithheldAt time.Time `json:"withheldAt"`
}

type SendFileParams struct {
	Expires   string `json:"expires"`
	Signature string `json:"signature"`
}
