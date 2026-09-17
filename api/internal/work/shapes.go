package work

import (
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/google/uuid"
)

func (e AssetDiscoveryRequestDiscovery) Valid() bool {
	switch e {
	case AssetDiscoveryRequestDiscoveryListed:
		return true
	case AssetDiscoveryRequestDiscoveryUnlisted:
		return true
	default:
		return false
	}
}

type AddableBlock struct {
	Choices    []AddableBlockChoice `json:"choices"`
	Definition string               `json:"definition"`
	Group      AddableBlockGroup    `json:"group"`
	GroupTitle string               `json:"groupTitle"`
	Repeatable bool                 `json:"repeatable"`
	Summary    string               `json:"summary"`
	Title      string               `json:"title"`
}

type AddableBlockGroup string

const (
	AddableBlockGroupFile   AddableBlockGroup = "file"
	AddableBlockGroupOther  AddableBlockGroup = "other"
	AddableBlockGroupReader AddableBlockGroup = "reader"
	AddableBlockGroupWork   AddableBlockGroup = "work"
)

type AddableBlockChoice struct {
	Label string      `json:"label"`
	Type  ElementType `json:"type"`
}

type AppTarget struct {
	Format string `json:"format"`
	Id     string `json:"id"`
	Label  string `json:"label"`
}

type AssetDetail struct {
	AddableBlocks         *[]AddableBlock           `json:"addableBlocks,omitempty"`
	AllowedApps           []AssetDetailAllowedApps  `json:"allowedApps" tstype:"'lumiverse'[],required"`
	AppTargets            []AppTarget               `json:"appTargets"`
	Blocks                []AssetBlock              `json:"blocks"`
	Blurb                 string                    `json:"blurb"`
	CreatedAt             time.Time                 `json:"createdAt"`
	Creator               string                    `json:"creator"`
	Discovery             AssetDetailDiscovery      `json:"discovery"`
	Downloads             []DownloadTarget          `json:"downloads"`
	EligibleApps          []AssetDetailEligibleApps `json:"eligibleApps" tstype:"'lumiverse'[],required"`
	ExtensionDependencies []ExtensionDependency     `json:"extensionDependencies"`
	Id                    uuid.UUID                 `json:"id"`
	Identifier            *string                   `json:"identifier,omitempty"`
	InstalledAppVersions  []string                  `json:"installedAppVersions"`
	IsNsfw                *bool                     `json:"isNsfw" tstype:"boolean | null,required"`
	IsOwner               bool                      `json:"isOwner"`
	Kind                  AssetDetailKind           `json:"kind"`
	LatestUpdate          *RecordedVersion          `json:"latestUpdate,omitempty"`
	Lifecycle             AssetDetailLifecycle      `json:"lifecycle"`
	LinkedInstallOnly     bool                      `json:"linkedInstallOnly"`
	Media                 []AssetImage              `json:"media"`
	Name                  string                    `json:"name"`
	Original              *OriginalUpload           `json:"original" tstype:"OriginalUpload | null,required"`
	Preview               *string                   `json:"preview" tstype:"string | null,required"`
	Readiness             *[]ReadinessItem          `json:"readiness,omitempty"`
	SealedBlocks          *int                      `json:"sealedBlocks,omitempty"`
	Tags                  []AssetTag                `json:"tags"`
	UnpublishedChanges    *bool                     `json:"unpublishedChanges,omitempty"`
	Visibility            AssetDetailVisibility     `json:"visibility"`
	Watch                 *notify.AssetWatch        `json:"watch,omitempty"`
	Withhold              *AssetWithhold            `json:"withhold,omitempty"`
	WorkingCopyVersion    *int64                    `json:"workingCopyVersion,omitempty"`
}

type AssetDetailAllowedApps string

const (
	AssetDetailAllowedAppsLumiverse AssetDetailAllowedApps = "lumiverse"
)

type AssetDetailDiscovery string

const (
	AssetDetailDiscoveryListed   AssetDetailDiscovery = "listed"
	AssetDetailDiscoveryUnlisted AssetDetailDiscovery = "unlisted"
)

type AssetDetailEligibleApps string

const (
	AssetDetailEligibleAppsLumiverse AssetDetailEligibleApps = "lumiverse"
)

type AssetDetailKind string

const (
	AssetDetailKindCharacter AssetDetailKind = "character"
	AssetDetailKindExtension AssetDetailKind = "extension"
	AssetDetailKindLorebook  AssetDetailKind = "lorebook"
	AssetDetailKindPack      AssetDetailKind = "pack"
	AssetDetailKindPreset    AssetDetailKind = "preset"
	AssetDetailKindTheme     AssetDetailKind = "theme"
)

type AssetDetailLifecycle string

const (
	AssetDetailLifecycleDraft     AssetDetailLifecycle = "draft"
	AssetDetailLifecyclePublished AssetDetailLifecycle = "published"
)

type AssetDetailVisibility string

const (
	AssetDetailVisibilityBlurred AssetDetailVisibility = "blurred"
	AssetDetailVisibilityHidden  AssetDetailVisibility = "hidden"
	AssetDetailVisibilityShown   AssetDetailVisibility = "shown"
)

type AssetDiscoveryRequest struct {
	Discovery AssetDiscoveryRequestDiscovery `json:"discovery"`
}

type AssetDiscoveryRequestDiscovery string

const (
	AssetDiscoveryRequestDiscoveryListed   AssetDiscoveryRequestDiscovery = "listed"
	AssetDiscoveryRequestDiscoveryUnlisted AssetDiscoveryRequestDiscovery = "unlisted"
)

type AssetImage struct {
	Bytes     int            `json:"bytes"`
	DetailUrl string         `json:"detailUrl"`
	Height    int            `json:"height"`
	Id        uuid.UUID      `json:"id"`
	IsCover   bool           `json:"isCover"`
	Role      AssetImageRole `json:"role"`
	ThumbUrl  string         `json:"thumbUrl"`
	Width     int            `json:"width"`
}

type AssetImageRole string

const (
	AssetImageRoleAvatar           AssetImageRole = "avatar"
	AssetImageRoleAvatarAlt        AssetImageRole = "avatar_alt"
	AssetImageRoleExpression       AssetImageRole = "expression"
	AssetImageRoleGallery          AssetImageRole = "gallery"
	AssetImageRolePackItem         AssetImageRole = "pack_item"
	AssetImageRolePerspectiveLayer AssetImageRole = "perspective_layer"
)

type AssetList struct {
	EmptyState *AssetListEmptyState `json:"emptyState" tstype:"AssetListEmptyState | null,required"`
	Facets     []BrowseFacet        `json:"facets"`
	Items      []BrowseAsset        `json:"items"`
	NextCursor *BrowseCursor        `json:"nextCursor,omitempty"`
	Platforms  []BrowseOption       `json:"platforms"`
	Suppressed int                  `json:"suppressed"`
	Total      int                  `json:"total"`
	Visibility AssetListVisibility  `json:"visibility"`
}

type AssetListEmptyState string

const (
	AssetListEmptyStateCatalog     AssetListEmptyState = "catalog"
	AssetListEmptyStateLessThannil AssetListEmptyState = "<nil>"
	AssetListEmptyStateNoMatches   AssetListEmptyState = "no_matches"
	AssetListEmptyStateSuppressed  AssetListEmptyState = "suppressed"
)

type AssetListVisibility string

const (
	AssetListVisibilityBlurred AssetListVisibility = "blurred"
	AssetListVisibilityHidden  AssetListVisibility = "hidden"
	AssetListVisibilityShown   AssetListVisibility = "shown"
)

type AssetTag struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type AssetWithhold struct {
	At     time.Time `json:"at"`
	Reason string    `json:"reason"`
}

type BrowseAsset struct {
	Cover      *BrowseCover           `json:"cover" tstype:"BrowseCover | null,required"`
	Creator    string                 `json:"creator"`
	Id         uuid.UUID              `json:"id"`
	IsNsfw     *bool                  `json:"isNsfw" tstype:"boolean | null,required"`
	Kind       BrowseAssetKind        `json:"kind"`
	Name       string                 `json:"name"`
	OwnerState *BrowseAssetOwnerState `json:"ownerState,omitempty"`
	Withhold   *AssetWithhold         `json:"withhold,omitempty"`
}

type BrowseAssetKind string

const (
	BrowseAssetKindCharacter BrowseAssetKind = "character"
	BrowseAssetKindExtension BrowseAssetKind = "extension"
	BrowseAssetKindLorebook  BrowseAssetKind = "lorebook"
	BrowseAssetKindPack      BrowseAssetKind = "pack"
	BrowseAssetKindPreset    BrowseAssetKind = "preset"
	BrowseAssetKindTheme     BrowseAssetKind = "theme"
)

type BrowseAssetOwnerState string

const (
	BrowseAssetOwnerStateDraft    BrowseAssetOwnerState = "draft"
	BrowseAssetOwnerStateUnlisted BrowseAssetOwnerState = "unlisted"
	BrowseAssetOwnerStateWithheld BrowseAssetOwnerState = "withheld"
)

type BrowseCover struct {
	Height int    `json:"height"`
	Url    string `json:"url"`
	Width  int    `json:"width"`
}

type BrowseCursor struct {
	Before   time.Time `json:"before"`
	BeforeId uuid.UUID `json:"beforeId"`
}

type BrowseFacet struct {
	Key     string         `json:"key"`
	Label   string         `json:"label"`
	Options []BrowseOption `json:"options"`
}

type BrowseOption struct {
	Count    int    `json:"count"`
	Label    string `json:"label"`
	Selected bool   `json:"selected"`
	Value    string `json:"value"`
}

type DeletedAsset struct {
	DeletedAt        time.Time        `json:"deletedAt"`
	Id               uuid.UUID        `json:"id"`
	Kind             DeletedAssetKind `json:"kind"`
	Name             string           `json:"name"`
	RecoverableUntil time.Time        `json:"recoverableUntil"`
}

type DeletedAssetKind string

const (
	DeletedAssetKindCharacter DeletedAssetKind = "character"
	DeletedAssetKindExtension DeletedAssetKind = "extension"
	DeletedAssetKindLorebook  DeletedAssetKind = "lorebook"
	DeletedAssetKindPack      DeletedAssetKind = "pack"
	DeletedAssetKindPreset    DeletedAssetKind = "preset"
	DeletedAssetKindTheme     DeletedAssetKind = "theme"
)

type DeletedAssetList struct {
	Items []DeletedAsset `json:"items"`
}

type DependencyAsset struct {
	Creator string    `json:"creator"`
	Id      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
}

type DownloadRoleVerdict struct {
	Destination *string                    `json:"destination,omitempty"`
	Label       string                     `json:"label"`
	Reason      *string                    `json:"reason,omitempty"`
	Role        string                     `json:"role"`
	Sample      DownloadSample             `json:"sample"`
	ShownBy     *[]string                  `json:"shownBy,omitempty"`
	Verdict     DownloadRoleVerdictVerdict `json:"verdict"`
}

type DownloadRoleVerdictVerdict string

const (
	DownloadRoleVerdictVerdictCarried DownloadRoleVerdictVerdict = "carried"
	DownloadRoleVerdictVerdictDropped DownloadRoleVerdictVerdict = "dropped"
	DownloadRoleVerdictVerdictReduced DownloadRoleVerdictVerdict = "reduced"
)

type DownloadSample struct {
	Count  int          `json:"count"`
	Images *[]uuid.UUID `json:"images,omitempty"`
	Texts  *[]string    `json:"texts,omitempty"`
}

type DownloadTarget struct {
	Format      string                `json:"format"`
	Label       string                `json:"label"`
	Recommended bool                  `json:"recommended"`
	Roles       []DownloadRoleVerdict `json:"roles"`
}

type ElementType string

const (
	ElementTypeColorSet       ElementType = "color_set"
	ElementTypeDialogueSample ElementType = "dialogue_sample"
	ElementTypeEntryTable     ElementType = "entry_table"
	ElementTypeFieldList      ElementType = "field_list"
	ElementTypeImageSet       ElementType = "image_set"
	ElementTypeLinkList       ElementType = "link_list"
	ElementTypePromptList     ElementType = "prompt_list"
	ElementTypeProse          ElementType = "prose"
	ElementTypeRecordList     ElementType = "record_list"
	ElementTypeScriptList     ElementType = "script_list"
	ElementTypeSettingGroup   ElementType = "setting_group"
	ElementTypeStylesheetSet  ElementType = "stylesheet_set"
	ElementTypeTextSet        ElementType = "text_set"
	ElementTypeVariableSchema ElementType = "variable_schema"
)

type ExtensionDependency struct {
	Assets []DependencyAsset `json:"assets"`
	Name   string            `json:"name"`
}

type OriginalUpload struct {
	ArrivedAt time.Time `json:"arrivedAt"`
	Label     string    `json:"label"`
	MediaType string    `json:"mediaType"`
}

type ListAssetsParams struct {
	Kind     *ListAssetsParamsKind `json:"kind,omitempty"`
	Platform *string               `json:"platform,omitempty"`
	Creator  *string               `json:"creator,omitempty"`
	Q        *string               `json:"q,omitempty"`
	Facet    *[]string             `json:"facet,omitempty"`
	Nsfw     *ListAssetsParamsNsfw `json:"nsfw,omitempty"`
	Limit    *int                  `json:"limit,omitempty"`
	Before   *time.Time            `json:"before,omitempty"`
	BeforeId *uuid.UUID            `json:"beforeId,omitempty"`
}

type GetAssetParams struct {
	WorkingCopy *bool               `json:"workingCopy,omitempty"`
	Nsfw        *GetAssetParamsNsfw `json:"nsfw,omitempty"`
}

type ListAssetsParamsKind string

const (
	ListAssetsParamsKindCharacter ListAssetsParamsKind = "character"
	ListAssetsParamsKindExtension ListAssetsParamsKind = "extension"
	ListAssetsParamsKindLorebook  ListAssetsParamsKind = "lorebook"
	ListAssetsParamsKindPack      ListAssetsParamsKind = "pack"
	ListAssetsParamsKindPreset    ListAssetsParamsKind = "preset"
	ListAssetsParamsKindTheme     ListAssetsParamsKind = "theme"
)

type ListAssetsParamsNsfw string

const (
	ListAssetsParamsNsfwBlurred ListAssetsParamsNsfw = "blurred"
	ListAssetsParamsNsfwHidden  ListAssetsParamsNsfw = "hidden"
	ListAssetsParamsNsfwShown   ListAssetsParamsNsfw = "shown"
)

type GetAssetParamsNsfw string

const (
	GetAssetParamsNsfwBlurred GetAssetParamsNsfw = "blurred"
	GetAssetParamsNsfwHidden  GetAssetParamsNsfw = "hidden"
	GetAssetParamsNsfwShown   GetAssetParamsNsfw = "shown"
)

type AssetIdentityRequest struct {
	Blurb  string `json:"blurb"`
	IsNsfw *bool  `json:"isNsfw" tstype:"boolean | null,required"`
	Name   string `json:"name"`
}

type PublishRefusal struct {
	Code      *PublishRefusalCode `json:"code,omitempty"`
	Error     string              `json:"error"`
	Readiness *[]ReadinessItem    `json:"readiness,omitempty"`
}

type PublishRefusalCode string

const (
	PublishRefusalCodeAlreadyPublished PublishRefusalCode = "already_published"
	PublishRefusalCodeNoChanges        PublishRefusalCode = "no_changes"
	PublishRefusalCodeNotReady         PublishRefusalCode = "not_ready"
)

type ReadinessItem struct {
	BlockId *uuid.UUID `json:"blockId,omitempty"`
	Detail  string     `json:"detail"`
	Id      string     `json:"id"`
	Label   string     `json:"label"`
	Met     bool       `json:"met"`
}

type CandidateConflict struct {
	Code           CandidateConflictCode `json:"code"`
	CurrentVersion *int64                `json:"currentVersion,omitempty"`
	Error          string                `json:"error"`
}

type CandidateConflictCode string

const (
	CandidateConflictCodeAssetFrozen         CandidateConflictCode = "asset_frozen"
	CandidateConflictCodeWorkingCopyConflict CandidateConflictCode = "working_copy_conflict"
)

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
