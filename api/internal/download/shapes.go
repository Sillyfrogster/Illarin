package download

import (
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
)

type DownloadExportParams struct {
	Images  *string `json:"images,omitempty"`
	Version *int    `json:"version,omitempty"`
}

type RecordedVersionDownloads struct {
	AppTargets        []page.AppTarget             `json:"appTargets"`
	Blocks            []block.AssetBlock           `json:"blocks"`
	Downloads         []page.DownloadTarget        `json:"downloads"`
	Kind              RecordedVersionDownloadsKind `json:"kind"`
	LinkedInstallOnly bool                         `json:"linkedInstallOnly"`
	Media             []page.AssetImage            `json:"media"`
	Version           page.RecordedVersion         `json:"version"`
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
