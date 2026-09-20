package page

import (
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/google/uuid"
)

func (e WorkVisibilityRequestVisibility) Valid() bool {
	switch e {
	case WorkVisibilityRequestVisibilityListed:
		return true
	case WorkVisibilityRequestVisibilityUnlisted:
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
	Label string            `json:"label"`
	Type  block.ElementType `json:"type"`
}

type AppName struct {
	Id    string `json:"id"`
	Label string `json:"label"`
}

type AppFormat struct {
	Format string `json:"format"`
	Id     string `json:"id"`
	Label  string `json:"label"`
}

type WorkDetail struct {
	AddableBlocks         *[]AddableBlock          `json:"addableBlocks,omitempty"`
	AllowedApps           []AppName                `json:"allowedApps"`
	AppFormats            []AppFormat              `json:"appFormats"`
	Blocks                []block.WorkBlock        `json:"blocks"`
	Blurb                 string                   `json:"blurb"`
	CreatedAt             time.Time                `json:"createdAt"`
	Creator               string                   `json:"creator"`
	Visibility            WorkDetailVisibility     `json:"visibility"`
	Downloads             []DownloadFormat         `json:"downloads"`
	EligibleApps          []AppName                `json:"eligibleApps"`
	ExtensionDependencies []ExtensionDependency    `json:"extensionDependencies"`
	Id                    uuid.UUID                `json:"id"`
	Identifier            *string                  `json:"identifier,omitempty"`
	InstalledAppVersions  []string                 `json:"installedAppVersions"`
	IsNsfw                *bool                    `json:"isNsfw" tstype:"boolean | null,required"`
	IsOwner               bool                     `json:"isOwner"`
	Type                  WorkDetailType           `json:"type"`
	LatestVersion         *RecordedVersion         `json:"latestVersion,omitempty"`
	Lifecycle             WorkDetailLifecycle      `json:"lifecycle"`
	HasPrivatePrompts     bool                     `json:"hasPrivatePrompts"`
	Media                 []WorkImage              `json:"media"`
	Name                  string                   `json:"name"`
	Original              *OriginalUpload          `json:"original" tstype:"OriginalUpload | null,required"`
	Preview               *string                  `json:"preview" tstype:"string | null,required"`
	Readiness             *[]ReadinessItem         `json:"readiness,omitempty"`
	PreservedPrompts      *int                     `json:"preservedPrompts,omitempty"`
	Tags                  []WorkTag                `json:"tags"`
	UnpublishedChanges    *bool                    `json:"unpublishedChanges,omitempty"`
	NSFWPreference        WorkDetailNSFWPreference `json:"nsfwPreference"`
	Follow                *notify.WorkFollow       `json:"follow,omitempty"`
	Takedown              *WorkTakedown            `json:"takedown,omitempty"`
	DraftedChangesVersion *int64                   `json:"draftedChangesVersion,omitempty"`
}

type WorkDetailVisibility string

const (
	WorkDetailVisibilityListed   WorkDetailVisibility = "listed"
	WorkDetailVisibilityUnlisted WorkDetailVisibility = "unlisted"
)

type WorkDetailType string

const (
	WorkDetailTypeCharacter WorkDetailType = "character"
	WorkDetailTypeExtension WorkDetailType = "extension"
	WorkDetailTypeLorebook  WorkDetailType = "lorebook"
	WorkDetailTypePack      WorkDetailType = "pack"
	WorkDetailTypePreset    WorkDetailType = "preset"
	WorkDetailTypeTheme     WorkDetailType = "theme"
)

type WorkDetailLifecycle string

const (
	WorkDetailLifecycleDraft     WorkDetailLifecycle = "draft"
	WorkDetailLifecyclePublished WorkDetailLifecycle = "published"
)

type WorkDetailNSFWPreference string

const (
	WorkDetailNSFWPreferenceBlurred WorkDetailNSFWPreference = "blurred"
	WorkDetailNSFWPreferenceHidden  WorkDetailNSFWPreference = "hidden"
	WorkDetailNSFWPreferenceShown   WorkDetailNSFWPreference = "shown"
)

type WorkVisibilityRequest struct {
	Visibility WorkVisibilityRequestVisibility `json:"visibility"`
}

type WorkVisibilityRequestVisibility string

const (
	WorkVisibilityRequestVisibilityListed   WorkVisibilityRequestVisibility = "listed"
	WorkVisibilityRequestVisibilityUnlisted WorkVisibilityRequestVisibility = "unlisted"
)

type WorkImage struct {
	Bytes     int           `json:"bytes"`
	DetailUrl string        `json:"detailUrl"`
	Height    int           `json:"height"`
	Id        uuid.UUID     `json:"id"`
	IsCover   bool          `json:"isCover"`
	Role      WorkImageRole `json:"role"`
	ThumbUrl  string        `json:"thumbUrl"`
	Width     int           `json:"width"`
}

type WorkImageRole string

const (
	WorkImageRoleAvatar           WorkImageRole = "avatar"
	WorkImageRoleAvatarAlt        WorkImageRole = "avatar_alt"
	WorkImageRoleExpression       WorkImageRole = "expression"
	WorkImageRoleGallery          WorkImageRole = "gallery"
	WorkImageRolePackItem         WorkImageRole = "pack_item"
	WorkImageRolePerspectiveLayer WorkImageRole = "perspective_layer"
)

type WorkList struct {
	EmptyState     *WorkListEmptyState    `json:"emptyState" tstype:"WorkListEmptyState | null,required"`
	Facets         []BrowseFacet          `json:"facets"`
	Items          []BrowseWork           `json:"items"`
	NextCursor     *BrowseCursor          `json:"nextCursor,omitempty"`
	Apps           []BrowseOption         `json:"apps"`
	Suppressed     int                    `json:"suppressed"`
	Total          int                    `json:"total"`
	NSFWPreference WorkListNSFWPreference `json:"nsfwPreference"`
}

type WorkListEmptyState string

const (
	WorkListEmptyStateNothingPublished WorkListEmptyState = "nothing_published"
	WorkListEmptyStateLessThannil      WorkListEmptyState = "<nil>"
	WorkListEmptyStateNoMatches        WorkListEmptyState = "no_matches"
	WorkListEmptyStateSuppressed       WorkListEmptyState = "suppressed"
)

type WorkListNSFWPreference string

const (
	WorkListNSFWPreferenceBlurred WorkListNSFWPreference = "blurred"
	WorkListNSFWPreferenceHidden  WorkListNSFWPreference = "hidden"
	WorkListNSFWPreferenceShown   WorkListNSFWPreference = "shown"
)

type WorkTag struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type WorkTakedown struct {
	At     time.Time `json:"at"`
	Reason string    `json:"reason"`
}

type BrowseWork struct {
	Cover      *BrowseCover          `json:"cover" tstype:"BrowseCover | null,required"`
	Creator    string                `json:"creator"`
	Id         uuid.UUID             `json:"id"`
	IsNsfw     *bool                 `json:"isNsfw" tstype:"boolean | null,required"`
	Type       BrowseWorkType        `json:"type"`
	Name       string                `json:"name"`
	OwnerState *BrowseWorkOwnerState `json:"ownerState,omitempty"`
	Takedown   *WorkTakedown         `json:"takedown,omitempty"`
}

type BrowseWorkType string

const (
	BrowseWorkTypeCharacter BrowseWorkType = "character"
	BrowseWorkTypeExtension BrowseWorkType = "extension"
	BrowseWorkTypeLorebook  BrowseWorkType = "lorebook"
	BrowseWorkTypePack      BrowseWorkType = "pack"
	BrowseWorkTypePreset    BrowseWorkType = "preset"
	BrowseWorkTypeTheme     BrowseWorkType = "theme"
)

type BrowseWorkOwnerState string

const (
	BrowseWorkOwnerStateDraft     BrowseWorkOwnerState = "draft"
	BrowseWorkOwnerStateUnlisted  BrowseWorkOwnerState = "unlisted"
	BrowseWorkOwnerStateTakenDown BrowseWorkOwnerState = "taken_down"
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

type DeletedWork struct {
	DeletedAt        time.Time       `json:"deletedAt"`
	Id               uuid.UUID       `json:"id"`
	Type             DeletedWorkType `json:"type"`
	Name             string          `json:"name"`
	RecoverableUntil time.Time       `json:"recoverableUntil"`
}

type DeletedWorkType string

const (
	DeletedWorkTypeCharacter DeletedWorkType = "character"
	DeletedWorkTypeExtension DeletedWorkType = "extension"
	DeletedWorkTypeLorebook  DeletedWorkType = "lorebook"
	DeletedWorkTypePack      DeletedWorkType = "pack"
	DeletedWorkTypePreset    DeletedWorkType = "preset"
	DeletedWorkTypeTheme     DeletedWorkType = "theme"
)

type DeletedWorkList struct {
	Items []DeletedWork `json:"items"`
}

type DependencyWork struct {
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

type DownloadFormat struct {
	Format      string                `json:"format"`
	Label       string                `json:"label"`
	Recommended bool                  `json:"recommended"`
	Roles       []DownloadRoleVerdict `json:"roles"`
}

type ExtensionDependency struct {
	Works []DependencyWork `json:"works"`
	Name  string           `json:"name"`
}

type OriginalUpload struct {
	ArrivedAt time.Time `json:"arrivedAt"`
	Label     string    `json:"label"`
	MediaType string    `json:"mediaType"`
}

type ListWorksParams struct {
	Type     *ListWorksParamsType `json:"type,omitempty"`
	App      *string              `json:"app,omitempty"`
	Creator  *string              `json:"creator,omitempty"`
	Q        *string              `json:"q,omitempty"`
	Facet    *[]string            `json:"facet,omitempty"`
	Nsfw     *ListWorksParamsNsfw `json:"nsfw,omitempty"`
	Limit    *int                 `json:"limit,omitempty"`
	Before   *time.Time           `json:"before,omitempty"`
	BeforeId *uuid.UUID           `json:"beforeId,omitempty"`
}

type GetWorkParams struct {
	DraftedChanges *bool              `json:"draftedChanges,omitempty"`
	Nsfw           *GetWorkParamsNsfw `json:"nsfw,omitempty"`
}

type ListWorksParamsType string

const (
	ListWorksParamsTypeCharacter ListWorksParamsType = "character"
	ListWorksParamsTypeExtension ListWorksParamsType = "extension"
	ListWorksParamsTypeLorebook  ListWorksParamsType = "lorebook"
	ListWorksParamsTypePack      ListWorksParamsType = "pack"
	ListWorksParamsTypePreset    ListWorksParamsType = "preset"
	ListWorksParamsTypeTheme     ListWorksParamsType = "theme"
)

type ListWorksParamsNsfw string

const (
	ListWorksParamsNsfwBlurred ListWorksParamsNsfw = "blurred"
	ListWorksParamsNsfwHidden  ListWorksParamsNsfw = "hidden"
	ListWorksParamsNsfwShown   ListWorksParamsNsfw = "shown"
)

type GetWorkParamsNsfw string

const (
	GetWorkParamsNsfwBlurred GetWorkParamsNsfw = "blurred"
	GetWorkParamsNsfwHidden  GetWorkParamsNsfw = "hidden"
	GetWorkParamsNsfwShown   GetWorkParamsNsfw = "shown"
)

type WorkDetailsRequest struct {
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
	CandidateConflictCodeWorkFrozen             CandidateConflictCode = "work_frozen"
	CandidateConflictCodeDraftedChangesConflict CandidateConflictCode = "drafted_changes_conflict"
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

type PreservedData struct {
	Bytes int    `json:"bytes"`
	Label string `json:"label"`
	Name  string `json:"name"`
}
