package version

import (
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/google/uuid"
)

type AssetVersionNotesRequest struct {
	Notes   *string `json:"notes,omitempty"`
	Summary string  `json:"summary"`
}

type AssetVersionWithdrawalRequest struct {
	Explanation string `json:"explanation"`
}

type NamedPrompt struct {
	Id   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type PromptCorrespondenceRequest struct {
	Matches []struct {
		Current  uuid.UUID  `json:"current"`
		Recorded *uuid.UUID `json:"recorded,omitempty"`
	} `json:"matches"`
}

type ProtectionMismatch struct {
	Recorded  []NamedPrompt        `json:"recorded"`
	Unmatched []NamedPrompt        `json:"unmatched"`
	Version   page.RecordedVersion `json:"version"`
}

type ProtectionMismatchList struct {
	Items []ProtectionMismatch `json:"items"`
}

type RecordedVersionList struct {
	Items []page.RecordedVersion `json:"items"`
}

type VersionChange struct {
	After        *string           `json:"after,omitempty"`
	AfterImage   *string           `json:"afterImage,omitempty"`
	Before       *string           `json:"before,omitempty"`
	BeforeImage  *string           `json:"beforeImage,omitempty"`
	Kind         VersionChangeKind `json:"kind"`
	Name         string            `json:"name"`
	Note         *string           `json:"note,omitempty"`
	PreviousName *string           `json:"previousName,omitempty"`
}

type VersionChangeKind string

const (
	VersionChangeKindAddition VersionChangeKind = "addition"
	VersionChangeKindChange   VersionChangeKind = "change"
	VersionChangeKindRemoval  VersionChangeKind = "removal"
)

type VersionComparison struct {
	From            page.RecordedVersion `json:"from"`
	Groups          []VersionChangeGroup `json:"groups"`
	PromptsWithheld bool                 `json:"promptsWithheld"`
	To              page.RecordedVersion `json:"to"`
	Unavailable     *string              `json:"unavailable,omitempty"`
}

type CompareAssetVersionsParams struct {
	From *int `json:"from,omitempty"`
	To   *int `json:"to,omitempty"`
}

type VersionChangeGroup struct {
	Changes []VersionChange `json:"changes"`
	Label   string          `json:"label"`
	Subject string          `json:"subject"`
}
