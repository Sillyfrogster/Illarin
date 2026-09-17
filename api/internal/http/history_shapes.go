package http

import (
	"time"

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
	Recorded  []NamedPrompt   `json:"recorded"`
	Unmatched []NamedPrompt   `json:"unmatched"`
	Version   RecordedVersion `json:"version"`
}

type ProtectionMismatchList struct {
	Items []ProtectionMismatch `json:"items"`
}

type RecordedVersion struct {
	Id                    uuid.UUID  `json:"id"`
	Initial               bool       `json:"initial"`
	Notes                 string     `json:"notes"`
	NotesEditedAt         *time.Time `json:"notesEditedAt,omitempty"`
	Number                int        `json:"number"`
	RecordedAt            time.Time  `json:"recordedAt"`
	Summary               string     `json:"summary"`
	VersionLabel          string     `json:"versionLabel"`
	WithdrawalExplanation *string    `json:"withdrawalExplanation,omitempty"`
	WithdrawnAt           *time.Time `json:"withdrawnAt,omitempty"`
}

type RecordedVersionDownloads struct {
	AppTargets        []AppTarget                  `json:"appTargets"`
	Blocks            []AssetBlock                 `json:"blocks"`
	Downloads         []DownloadTarget             `json:"downloads"`
	Kind              RecordedVersionDownloadsKind `json:"kind"`
	LinkedInstallOnly bool                         `json:"linkedInstallOnly"`
	Media             []AssetImage                 `json:"media"`
	Version           RecordedVersion              `json:"version"`
}

type RecordedVersionDownloadsKind string

const (
	RecordedVersionDownloadsKindCharacter RecordedVersionDownloadsKind = "character"
	RecordedVersionDownloadsKindExtension RecordedVersionDownloadsKind = "extension"
	RecordedVersionDownloadsKindLorebook  RecordedVersionDownloadsKind = "lorebook"
	RecordedVersionDownloadsKindPack      RecordedVersionDownloadsKind = "pack"
	RecordedVersionDownloadsKindPreset    RecordedVersionDownloadsKind = "preset"
	RecordedVersionDownloadsKindTheme     RecordedVersionDownloadsKind = "theme"
)

type RecordedVersionList struct {
	Items []RecordedVersion `json:"items"`
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
	Addition VersionChangeKind = "addition"
	Change   VersionChangeKind = "change"
	Removal  VersionChangeKind = "removal"
)

type VersionComparison struct {
	From            RecordedVersion      `json:"from"`
	Groups          []VersionChangeGroup `json:"groups"`
	PromptsWithheld bool                 `json:"promptsWithheld"`
	To              RecordedVersion      `json:"to"`
	Unavailable     *string              `json:"unavailable,omitempty"`
}

type CompareAssetVersionsParams struct {
	From *int `json:"from,omitempty"`
	To   *int `json:"to,omitempty"`
}

type GetRecordedVersionDownloadsParams struct {
	Nsfw *GetRecordedVersionDownloadsParamsNsfw `json:"nsfw,omitempty"`
}

type GetRecordedVersionDownloadsParamsNsfw string

const (
	GetRecordedVersionDownloadsParamsNsfwBlurred GetRecordedVersionDownloadsParamsNsfw = "blurred"
	GetRecordedVersionDownloadsParamsNsfwHidden  GetRecordedVersionDownloadsParamsNsfw = "hidden"
	GetRecordedVersionDownloadsParamsNsfwShown   GetRecordedVersionDownloadsParamsNsfw = "shown"
)
