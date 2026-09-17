package http

import (
	"time"

	"github.com/google/uuid"
)

type AssetInstance struct {
	ApplicationName     string          `json:"applicationName"`
	CanReceive          bool            `json:"canReceive"`
	Delivery            *QueuedDelivery `json:"delivery"`
	InstalledGeneration *int            `json:"installedGeneration"`
	InstanceId          uuid.UUID       `json:"instanceId"`
	InstanceName        string          `json:"instanceName"`
	LastSeenAt          *time.Time      `json:"lastSeenAt"`
	ReportsLibrary      bool            `json:"reportsLibrary"`
	UpdateAvailable     bool            `json:"updateAvailable"`
}

type AssetInstanceList struct {
	ContentGeneration int             `json:"contentGeneration"`
	Items             []AssetInstance `json:"items"`
}

type CollectDeliveries struct {
	Acknowledge []uuid.UUID `json:"acknowledge"`
}

type DeliveryArtifact struct {
	IsCover *bool                `json:"isCover,omitempty"`
	Kind    DeliveryArtifactKind `json:"kind"`
	MediaId *uuid.UUID           `json:"mediaId,omitempty"`
	Role    *string              `json:"role,omitempty"`
	Url     string               `json:"url"`
}

type DeliveryArtifactKind string

const (
	Export  DeliveryArtifactKind = "export"
	Picture DeliveryArtifactKind = "picture"
)

type DeliveryWork struct {
	Artifacts         []DeliveryArtifact `json:"artifacts"`
	AssetId           uuid.UUID          `json:"assetId"`
	ContentGeneration int                `json:"contentGeneration"`
	Format            string             `json:"format"`
	Id                uuid.UUID          `json:"id"`
	Kind              string             `json:"kind"`
	Label             string             `json:"label"`
	LeaseExpiresAt    time.Time          `json:"leaseExpiresAt"`
	Name              string             `json:"name"`
	QueuedAt          time.Time          `json:"queuedAt"`
}

type DeliveryWorkList struct {
	Deliveries []DeliveryWork   `json:"deliveries"`
	Withheld   []WithheldNotice `json:"withheld"`
}

type LibraryEntry struct {
	AssetId           uuid.UUID `json:"assetId"`
	ContentGeneration *int      `json:"contentGeneration,omitempty"`
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
	AssetId        uuid.UUID             `json:"assetId"`
	ExpiresAt      time.Time             `json:"expiresAt"`
	Id             uuid.UUID             `json:"id"`
	InstanceId     uuid.UUID             `json:"instanceId"`
	QueuedAt       time.Time             `json:"queuedAt"`
	Reason         *QueuedDeliveryReason `json:"reason,omitempty"`
	SettledAt      *time.Time            `json:"settledAt"`
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

type SendAssetRequest struct {
	InstanceId uuid.UUID `json:"instanceId"`
}

type WithheldNotice struct {
	AssetId    uuid.UUID `json:"assetId"`
	Name       string    `json:"name"`
	WithheldAt time.Time `json:"withheldAt"`
}

type DownloadDeliveryExportParams struct {
	Expires   string `json:"expires"`
	Signature string `json:"signature"`
}
