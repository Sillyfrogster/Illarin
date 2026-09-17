package http

import (
	"github.com/google/uuid"
)

type AssetDiscovery string

const (
	AssetDiscoveryListed   AssetDiscovery = "listed"
	AssetDiscoveryUnlisted AssetDiscovery = "unlisted"
)

type IngestFailure struct {
	Message string              `json:"message"`
	Reason  IngestFailureReason `json:"reason"`
}

type IngestFailureReason string

const (
	IngestFailureReasonAssetUnavailable    IngestFailureReason = "asset_unavailable"
	IngestFailureReasonInternalFailure     IngestFailureReason = "internal_failure"
	IngestFailureReasonLimitExceeded       IngestFailureReason = "limit_exceeded"
	IngestFailureReasonMalformedInput      IngestFailureReason = "malformed_input"
	IngestFailureReasonSafetyViolation     IngestFailureReason = "safety_violation"
	IngestFailureReasonUnsupportedFormat   IngestFailureReason = "unsupported_format"
	IngestFailureReasonUnsupportedVersion  IngestFailureReason = "unsupported_version"
	IngestFailureReasonWorkingCopyConflict IngestFailureReason = "working_copy_conflict"
	IngestFailureReasonWrongKind           IngestFailureReason = "wrong_kind"
)

type IngestOperation struct {
	Asset   *Asset                `json:"asset,omitempty"`
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

type MediaRole string

const (
	MediaRoleAvatar           MediaRole = "avatar"
	MediaRoleAvatarAlt        MediaRole = "avatar_alt"
	MediaRoleExpression       MediaRole = "expression"
	MediaRoleGallery          MediaRole = "gallery"
	MediaRolePackItem         MediaRole = "pack_item"
	MediaRolePerspectiveLayer MediaRole = "perspective_layer"
)

type MediaList struct {
	Items []Media `json:"items"`
}

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
	Conflicts       []string             `json:"conflicts"`
	Format          string               `json:"format"`
	Groups          []VersionChangeGroup `json:"groups"`
	MissingWording  []string             `json:"missingWording"`
	Seals           int                  `json:"seals"`
	Unrepresentable []string             `json:"unrepresentable"`
}

type SealedExposureRefusal struct {
	Code    SealedExposureRefusalCode `json:"code" tstype:"'sealed_exposure',required"`
	Error   string                    `json:"error"`
	Prompts []string                  `json:"prompts"`
}

type SealedExposureRefusalCode string

const (
	SealedExposureRefusalCodeSealedExposure SealedExposureRefusalCode = "sealed_exposure"
)

type VersionChangeGroup struct {
	Changes []VersionChange `json:"changes"`
	Label   string          `json:"label"`
	Subject string          `json:"subject"`
}

type WithholdAssetRequest struct {
	Reason string `json:"reason"`
}

type WorkingCopyVersion = int64
