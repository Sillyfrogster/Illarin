package upload

import (
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/google/uuid"
)

type Work struct {
	Blurb      string         `json:"blurb"`
	CreatedAt  time.Time      `json:"createdAt"`
	Visibility WorkVisibility `json:"discovery"`
	Format     string         `json:"format"`
	Id         uuid.UUID      `json:"id"`
	IsNsfw     *bool          `json:"isNsfw" tstype:"boolean | null,required"`
	Type       string         `json:"kind"`
	Name       string         `json:"name"`
	Tags       []string       `json:"tags"`
}

type WorkVisibility string

const (
	WorkVisibilityListed   WorkVisibility = "listed"
	WorkVisibilityUnlisted WorkVisibility = "unlisted"
)

type IngestFailure struct {
	Message string              `json:"message"`
	Reason  IngestFailureReason `json:"reason"`
}

type IngestFailureReason string

const (
	IngestFailureReasonWorkUnavailable     IngestFailureReason = "asset_unavailable"
	IngestFailureReasonInternalFailure     IngestFailureReason = "internal_failure"
	IngestFailureReasonLimitExceeded       IngestFailureReason = "limit_exceeded"
	IngestFailureReasonMalformedInput      IngestFailureReason = "malformed_input"
	IngestFailureReasonSafetyViolation     IngestFailureReason = "safety_violation"
	IngestFailureReasonUnsupportedFormat   IngestFailureReason = "unsupported_format"
	IngestFailureReasonUnsupportedVersion  IngestFailureReason = "unsupported_version"
	IngestFailureReasonWorkingCopyConflict IngestFailureReason = "working_copy_conflict"
	IngestFailureReasonWrongType           IngestFailureReason = "wrong_kind"
)

type IngestOperation struct {
	Work    *Work                 `json:"asset,omitempty"`
	Failure *IngestFailure        `json:"failure,omitempty"`
	Id      uuid.UUID             `json:"id"`
	Preview *ReplacementPreview   `json:"preview,omitempty"`
	Status  IngestOperationStatus `json:"status"`
	Url     string                `json:"url"`
}

type IngestOperationStatus string

const (
	IngestOperationStatusCancelled  IngestOperationStatus = "cancelled"
	IngestOperationStatusFailed     IngestOperationStatus = "failed"
	IngestOperationStatusPending    IngestOperationStatus = "pending"
	IngestOperationStatusPreview    IngestOperationStatus = "preview"
	IngestOperationStatusProcessing IngestOperationStatus = "processing"
	IngestOperationStatusSuccess    IngestOperationStatus = "success"
)

type ReplacementAcceptance struct {
	ExposeProtected *bool                                           `json:"exposeProtected,omitempty"`
	Unrepresentable map[string]ReplacementAcceptanceUnrepresentable `json:"unrepresentable"`
}

type ReplacementAcceptanceUnrepresentable string

const (
	ReplacementAcceptanceUnrepresentableKeep   ReplacementAcceptanceUnrepresentable = "keep"
	ReplacementAcceptanceUnrepresentableRemove ReplacementAcceptanceUnrepresentable = "remove"
)

type ReplacementPreview struct {
	Conflicts       []string                     `json:"conflicts"`
	Format          string                       `json:"format"`
	Groups          []version.VersionChangeGroup `json:"groups"`
	MissingWording  []string                     `json:"missingWording"`
	Seals           int                          `json:"seals"`
	Unrepresentable []string                     `json:"unrepresentable"`
}

type CreateWorkRequest struct {
	Blurb      *string                      `json:"blurb,omitempty"`
	Confirmed  bool                         `json:"confirmed"`
	Visibility *CreateWorkRequestVisibility `json:"discovery,omitempty"`
	IsNsfw     *bool                        `json:"isNsfw,omitempty"`
	Name       *string                      `json:"name,omitempty"`
	Tags       *[]string                    `json:"tags,omitempty"`
}

type CreateWorkRequestVisibility string

const (
	CreateWorkRequestVisibilityListed   CreateWorkRequestVisibility = "listed"
	CreateWorkRequestVisibilityUnlisted CreateWorkRequestVisibility = "unlisted"
)
