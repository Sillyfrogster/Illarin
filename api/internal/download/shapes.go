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
	HasPrivatePrompts bool                         `json:"hasPrivatePrompts"`
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

type FormatComparison struct {
	Apps    []page.AppName `json:"apps"`
	Formats []FormatColumn `json:"formats"`
}

type FormatColumn struct {
	Id          string         `json:"id"`
	Label       string         `json:"label"`
	Type        string         `json:"type"`
	ReadBy      []string       `json:"readBy"`
	KeepsUpload bool           `json:"keepsUpload"`
	Fields      []FieldSupport `json:"fields"`
}

type FieldSupport struct {
	Field block.Role `json:"field"`
	Label string     `json:"label"`
	Grade string     `json:"grade"`
	Note  string     `json:"note"`
}
