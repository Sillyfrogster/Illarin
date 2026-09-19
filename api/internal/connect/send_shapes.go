package connect

import (
	"time"

	"github.com/google/uuid"
)

type WorkInstance struct {
	ApplicationName  string          `json:"applicationName"`
	CanReceive       bool            `json:"canReceive"`
	Delivery         *QueuedDelivery `json:"delivery" tstype:"QueuedDelivery | null,required"`
	InstalledVersion *int            `json:"installedVersion" tstype:"number | null,required"`
	InstanceId       uuid.UUID       `json:"instanceId"`
	InstanceName     string          `json:"instanceName"`
	LastSeenAt       *time.Time      `json:"lastSeenAt" tstype:"string | null,required"`
	ReportsLibrary   bool            `json:"reportsLibrary"`
	UpdateAvailable  bool            `json:"updateAvailable"`
}

type WorkInstanceList struct {
	VersionNumber int            `json:"versionNumber"`
	Items         []WorkInstance `json:"items"`
}

type CollectDeliveries struct {
	Acknowledge []uuid.UUID `json:"acknowledge"`
}

type DeliveryFile struct {
	IsCover *bool            `json:"isCover,omitempty"`
	Type    DeliveryFileType `json:"type"`
	MediaId *uuid.UUID       `json:"mediaId,omitempty"`
	Role    *string          `json:"role,omitempty"`
	Url     string           `json:"url"`
}

type DeliveryFileType string

const (
	DeliveryFileTypeExport  DeliveryFileType = "export"
	DeliveryFileTypePicture DeliveryFileType = "picture"
)

type DeliveryWork struct {
	Files          []DeliveryFile `json:"files"`
	WorkId         uuid.UUID      `json:"workId"`
	VersionNumber  int            `json:"versionNumber"`
	Format         string         `json:"format"`
	Id             uuid.UUID      `json:"id"`
	Type           string         `json:"type"`
	Label          string         `json:"label"`
	LeaseExpiresAt time.Time      `json:"leaseExpiresAt"`
	Name           string         `json:"name"`
	QueuedAt       time.Time      `json:"queuedAt"`
}

type DeliveryWorkList struct {
	Deliveries []DeliveryWork   `json:"deliveries"`
	Withheld   []WithheldNotice `json:"withheld"`
}

type LibraryEntry struct {
	WorkId        uuid.UUID `json:"workId"`
	VersionNumber *int      `json:"versionNumber,omitempty"`
}

type LibraryReport struct {
	ApplicationVersion *string        `json:"applicationVersion,omitempty"`
	Entries            []LibraryEntry `json:"entries"`
	Removed            *[]uuid.UUID   `json:"removed,omitempty"`
	Snapshot           bool           `json:"snapshot"`
}

type LibraryReportResult struct {
	Accepted int              `json:"accepted"`
	Ignored  int              `json:"ignored"`
	Removed  int              `json:"removed"`
	Withheld []WithheldNotice `json:"withheld"`
}

type QueuedDelivery struct {
	WorkId         uuid.UUID             `json:"workId"`
	ExpiresAt      time.Time             `json:"expiresAt"`
	Id             uuid.UUID             `json:"id"`
	InstanceId     uuid.UUID             `json:"instanceId"`
	QueuedAt       time.Time             `json:"queuedAt"`
	Reason         *QueuedDeliveryReason `json:"reason,omitempty"`
	SettledAt      *time.Time            `json:"settledAt" tstype:"string | null,required"`
	State          QueuedDeliveryState   `json:"state"`
	UpdatesInstall bool                  `json:"updatesInstall"`
}

type QueuedDeliveryReason string

const (
	QueuedDeliveryReasonAbandoned   QueuedDeliveryReason = "abandoned"
	QueuedDeliveryReasonLessThannil QueuedDeliveryReason = "<nil>"
	QueuedDeliveryReasonUnsupported QueuedDeliveryReason = "unsupported"
	QueuedDeliveryReasonWithdrawn   QueuedDeliveryReason = "withdrawn"
)

type QueuedDeliveryState string

const (
	QueuedDeliveryStateDelivered QueuedDeliveryState = "delivered"
	QueuedDeliveryStateFailed    QueuedDeliveryState = "failed"
	QueuedDeliveryStateQueued    QueuedDeliveryState = "queued"
	QueuedDeliveryStateReleased  QueuedDeliveryState = "released"
)

type SendWorkRequest struct {
	InstanceId uuid.UUID `json:"instanceId"`
}

type WithheldNotice struct {
	WorkId     uuid.UUID `json:"workId"`
	Name       string    `json:"name"`
	WithheldAt time.Time `json:"withheldAt"`
}

type DownloadDeliveryExportParams struct {
	Expires   string `json:"expires"`
	Signature string `json:"signature"`
}
