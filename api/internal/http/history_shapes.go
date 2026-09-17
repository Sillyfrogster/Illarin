package http

import (
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
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
	Version   work.RecordedVersion `json:"version"`
}

type ProtectionMismatchList struct {
	Items []ProtectionMismatch `json:"items"`
}

type RecordedVersionDownloads struct {
	AppTargets        []work.AppTarget             `json:"appTargets"`
	Blocks            []block.AssetBlock           `json:"blocks"`
	Downloads         []work.DownloadTarget        `json:"downloads"`
	Kind              RecordedVersionDownloadsKind `json:"kind"`
	LinkedInstallOnly bool                         `json:"linkedInstallOnly"`
	Media             []work.AssetImage            `json:"media"`
	Version           work.RecordedVersion         `json:"version"`
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
	Items []work.RecordedVersion `json:"items"`
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
	From            work.RecordedVersion `json:"from"`
	Groups          []VersionChangeGroup `json:"groups"`
	PromptsWithheld bool                 `json:"promptsWithheld"`
	To              work.RecordedVersion `json:"to"`
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
