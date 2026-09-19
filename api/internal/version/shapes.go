package version

import (
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/google/uuid"
)

type WorkVersionNotesRequest struct {
	Notes   *string `json:"notes,omitempty"`
	Summary string  `json:"summary"`
}

type WorkVersionWithdrawalRequest struct {
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

type PrivatePromptMismatch struct {
	Recorded  []NamedPrompt        `json:"recorded"`
	Unmatched []NamedPrompt        `json:"unmatched"`
	Version   page.RecordedVersion `json:"version"`
}

type PrivatePromptMismatchList struct {
	Items []PrivatePromptMismatch `json:"items"`
}

type RecordedVersionList struct {
	Items []page.RecordedVersion `json:"items"`
}

type VersionChange struct {
	After        *string           `json:"after,omitempty"`
	AfterImage   *string           `json:"afterImage,omitempty"`
	Before       *string           `json:"before,omitempty"`
	BeforeImage  *string           `json:"beforeImage,omitempty"`
	Type         VersionChangeType `json:"type"`
	Name         string            `json:"name"`
	Note         *string           `json:"note,omitempty"`
	PreviousName *string           `json:"previousName,omitempty"`
}

type VersionChangeType string

const (
	VersionChangeTypeAddition VersionChangeType = "addition"
	VersionChangeTypeChange   VersionChangeType = "change"
	VersionChangeTypeRemoval  VersionChangeType = "removal"
)

type VersionComparison struct {
	From            page.RecordedVersion `json:"from"`
	Groups          []VersionChangeGroup `json:"groups"`
	PromptsWithheld bool                 `json:"promptsWithheld"`
	To              page.RecordedVersion `json:"to"`
	Unavailable     *string              `json:"unavailable,omitempty"`
}

type CompareWorkVersionsParams struct {
	From *int `json:"from,omitempty"`
	To   *int `json:"to,omitempty"`
}

type VersionChangeGroup struct {
	Changes []VersionChange `json:"changes"`
	Label   string          `json:"label"`
	Subject string          `json:"subject"`
}
