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
	AppFormats        []page.AppFormat             `json:"appFormats"`
	Blocks            []block.WorkBlock            `json:"blocks"`
	Downloads         []page.DownloadFormat        `json:"downloads"`
	Type              RecordedVersionDownloadsType `json:"type"`
	LinkedInstallOnly bool                         `json:"linkedInstallOnly"`
	Media             []page.WorkImage             `json:"media"`
	Version           page.RecordedVersion         `json:"version"`
}

type RecordedVersionDownloadsType string

const (
	RecordedVersionDownloadsTypeCharacter RecordedVersionDownloadsType = "character"
	RecordedVersionDownloadsTypeExtension RecordedVersionDownloadsType = "extension"
	RecordedVersionDownloadsTypeLorebook  RecordedVersionDownloadsType = "lorebook"
	RecordedVersionDownloadsTypePack      RecordedVersionDownloadsType = "pack"
	RecordedVersionDownloadsTypePreset    RecordedVersionDownloadsType = "preset"
	RecordedVersionDownloadsTypeTheme     RecordedVersionDownloadsType = "theme"
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
