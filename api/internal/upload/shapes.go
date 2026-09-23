package upload

import (
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/google/uuid"
)

type Work struct {
	Blurb      string         `json:"blurb"`
	CreatedAt  time.Time      `json:"createdAt"`
	Visibility WorkVisibility `json:"visibility"`
	Format     string         `json:"format"`
	Id         uuid.UUID      `json:"id"`
	IsNsfw     *bool          `json:"isNsfw" tstype:"boolean | null,required"`
	Type       string         `json:"type"`
	Name       string         `json:"name"`
	Tags       []string       `json:"tags"`
}

type WorkVisibility string

const (
	WorkVisibilityListed   WorkVisibility = "listed"
	WorkVisibilityUnlisted WorkVisibility = "unlisted"
)

type UploadFailure struct {
	Message string              `json:"message"`
	Reason  UploadFailureReason `json:"reason"`
}

type UploadFailureReason string

const (
	UploadFailureReasonWorkUnavailable        UploadFailureReason = "work_unavailable"
	UploadFailureReasonInternalFailure        UploadFailureReason = "internal_failure"
	UploadFailureReasonLimitExceeded          UploadFailureReason = "limit_exceeded"
	UploadFailureReasonMalformedInput         UploadFailureReason = "malformed_input"
	UploadFailureReasonSafetyViolation        UploadFailureReason = "safety_violation"
	UploadFailureReasonUnsupportedFormat      UploadFailureReason = "unsupported_format"
	UploadFailureReasonUnsupportedVersion     UploadFailureReason = "unsupported_version"
	UploadFailureReasonDraftedChangesConflict UploadFailureReason = "drafted_changes_conflict"
	UploadFailureReasonWrongType              UploadFailureReason = "wrong_type"
)

type UploadOperation struct {
	Work    *Work                 `json:"work,omitempty"`
	Failure *UploadFailure        `json:"failure,omitempty"`
	Id      uuid.UUID             `json:"id"`
	Preview *ReplacementPreview   `json:"preview,omitempty"`
	Status  UploadOperationStatus `json:"status"`
	Url     string                `json:"url"`
}

type UploadOperationStatus string

const (
	UploadOperationStatusCancelled  UploadOperationStatus = "cancelled"
	UploadOperationStatusFailed     UploadOperationStatus = "failed"
	UploadOperationStatusPending    UploadOperationStatus = "pending"
	UploadOperationStatusPreview    UploadOperationStatus = "preview"
	UploadOperationStatusProcessing UploadOperationStatus = "processing"
	UploadOperationStatusSuccess    UploadOperationStatus = "success"
)

type ReplacementAcceptance struct {
	MakePromptsPublic *bool                                           `json:"makePromptsPublic,omitempty"`
	Unrepresentable   map[string]ReplacementAcceptanceUnrepresentable `json:"unrepresentable"`
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
	PrivatePrompts  int                          `json:"privatePrompts"`
	Unrepresentable []string                     `json:"unrepresentable"`
}

type CreateWorkRequest struct {
	Blurb      *string                      `json:"blurb,omitempty"`
	Confirmed  bool                         `json:"confirmed"`
	Visibility *CreateWorkRequestVisibility `json:"visibility,omitempty"`
	IsNsfw     *bool                        `json:"isNsfw,omitempty"`
	Name       *string                      `json:"name,omitempty"`
	Tags       *[]string                    `json:"tags,omitempty"`
}

type CreateWorkRequestVisibility string

const (
	CreateWorkRequestVisibilityListed   CreateWorkRequestVisibility = "listed"
	CreateWorkRequestVisibilityUnlisted CreateWorkRequestVisibility = "unlisted"
)
