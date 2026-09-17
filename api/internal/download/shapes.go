package download

import (
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
)

type DownloadExportParams struct {
	Images  *string `json:"images,omitempty"`
	Version *int    `json:"version,omitempty"`
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

type GetRecordedVersionDownloadsParams struct {
	Nsfw *GetRecordedVersionDownloadsParamsNsfw `json:"nsfw,omitempty"`
}

type GetRecordedVersionDownloadsParamsNsfw string

const (
	GetRecordedVersionDownloadsParamsNsfwBlurred GetRecordedVersionDownloadsParamsNsfw = "blurred"
	GetRecordedVersionDownloadsParamsNsfwHidden  GetRecordedVersionDownloadsParamsNsfw = "hidden"
	GetRecordedVersionDownloadsParamsNsfwShown   GetRecordedVersionDownloadsParamsNsfw = "shown"
)
