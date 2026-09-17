package http

import (
	"time"

	"github.com/google/uuid"
)

type CorrectPostAddressRequest struct {
	Slug string `json:"slug"`
}

type CorrectPostBylineRequest struct {
	Handle string `json:"handle"`
}

type CreatePostRequest struct {
	CategoryId uuid.UUID  `json:"categoryId"`
	GrantId    *uuid.UUID `json:"grantId,omitempty"`
	Title      string     `json:"title"`
}

type Media struct {
	AssetId           uuid.UUID `json:"assetId"`
	DerivativeVersion int       `json:"derivativeVersion"`
	Height            int       `json:"height"`
	Id                uuid.UUID `json:"id"`
	Role              MediaRole `json:"role"`
	Width             int       `json:"width"`
}

type Post struct {
	App              *PublicationApp     `json:"app,omitempty"`
	Author           PostAuthor          `json:"author"`
	Byline           *PostByline         `json:"byline,omitempty"`
	Category         PublicationCategory `json:"category"`
	CreatedAt        time.Time           `json:"createdAt"`
	Deletion         *PostDeletion       `json:"deletion,omitempty"`
	Document         PostDocument        `json:"document"`
	DocumentVersion  int                 `json:"documentVersion"`
	FormerAddresses  []string            `json:"formerAddresses"`
	GrantId          *uuid.UUID          `json:"grantId,omitempty"`
	Header           *PostHeader         `json:"header,omitempty"`
	Id               uuid.UUID           `json:"id"`
	Media            []PostMedia         `json:"media"`
	PublicRevisionId *uuid.UUID          `json:"publicRevisionId,omitempty"`
	PublishedAt      *time.Time          `json:"publishedAt,omitempty"`
	Release          *PostRelease        `json:"release,omitempty"`
	Schedule         *PostSchedule       `json:"schedule,omitempty"`
	Slug             string              `json:"slug"`
	SocialMediaId    *uuid.UUID          `json:"socialMediaId,omitempty"`
	Status           PostStatus          `json:"status"`
	Summary          string              `json:"summary"`
	Title            string              `json:"title"`
	UpdatedAt        time.Time           `json:"updatedAt"`
	UpdatedPublicAt  *time.Time          `json:"updatedPublicAt,omitempty"`
	Version          int                 `json:"version"`
	Withdrawal       *PostWithdrawal     `json:"withdrawal,omitempty"`
}

type PostArchive struct {
	App      *PublicationApp      `json:"app,omitempty"`
	Category *PublicationCategory `json:"category,omitempty"`
	Page     int                  `json:"page"`
	Pages    int                  `json:"pages"`
	Posts    []PostSummary        `json:"posts"`
	Total    int                  `json:"total"`
}

type PostAuthor struct {
	Handle string `json:"handle"`
}

type PostByline struct {
	App          *PublicationApp `json:"app,omitempty"`
	Avatar       *ProfileAvatar  `json:"avatar,omitempty"`
	ContactEmail string          `json:"contactEmail"`
	DisplayName  string          `json:"displayName"`
	Handle       string          `json:"handle"`
	Historical   bool            `json:"historical"`
}

type PostConflict struct {
	Code      PublicationErrorCode `json:"code"`
	Error     string               `json:"error"`
	Field     *string              `json:"field,omitempty"`
	UpdatedAt *time.Time           `json:"updatedAt,omitempty"`
	Version   *int                 `json:"version,omitempty"`
}

type PostDocument struct {
	Content []map[string]interface{} `json:"content"`
	Version int                      `json:"version"`
}

type PostHeader struct {
	Alt     string    `json:"alt"`
	Caption *string   `json:"caption,omitempty"`
	MediaId uuid.UUID `json:"mediaId"`
}

type PostHeaderEdit struct {
	Alt     string    `json:"alt"`
	Caption *string   `json:"caption,omitempty"`
	MediaId uuid.UUID `json:"mediaId"`
}

type PostList struct {
	Posts []Post `json:"posts"`
}

type PostMedia struct {
	Height   int              `json:"height"`
	Id       uuid.UUID        `json:"id"`
	PostId   uuid.UUID        `json:"postId"`
	Purpose  PostMediaPurpose `json:"purpose"`
	ThumbUrl string           `json:"thumbUrl"`
	Url      string           `json:"url"`
	Width    int              `json:"width"`
}

type PostMediaPurpose string

const (
	Document PostMediaPurpose = "document"
	Header   PostMediaPurpose = "header"
	Social   PostMediaPurpose = "social"
)

type PostRelease struct {
	Address *string        `json:"address,omitempty"`
	App     PublicationApp `json:"app"`
	Version string         `json:"version"`
}

type PostReleaseEdit struct {
	Address *string   `json:"address,omitempty"`
	AppId   uuid.UUID `json:"appId"`
	Version string    `json:"version"`
}

type PostStatus string

const (
	PostStatusDraft     PostStatus = "draft"
	PostStatusPublished PostStatus = "published"
	PostStatusWithdrawn PostStatus = "withdrawn"
)

type PostSummary struct {
	App            *PublicationApp     `json:"app,omitempty"`
	Byline         PostByline          `json:"byline"`
	Category       PublicationCategory `json:"category"`
	Id             uuid.UUID           `json:"id"`
	OriginalSlug   string              `json:"originalSlug"`
	PublishedAt    time.Time           `json:"publishedAt"`
	ReleaseVersion *string             `json:"releaseVersion,omitempty"`
	Slug           string              `json:"slug"`
	Summary        string              `json:"summary"`
	Title          string              `json:"title"`
	UpdatedAt      *time.Time          `json:"updatedAt,omitempty"`
}

type ProfileAvatar struct {
	Height int    `json:"height"`
	Url    string `json:"url"`
	Width  int    `json:"width"`
}

type PublicPost struct {
	Byline       PostByline          `json:"byline"`
	Category     PublicationCategory `json:"category"`
	Document     PostDocument        `json:"document"`
	Header       *PostHeader         `json:"header,omitempty"`
	Id           uuid.UUID           `json:"id"`
	Media        []PostMedia         `json:"media"`
	OriginalSlug string              `json:"originalSlug"`
	PublishedAt  time.Time           `json:"publishedAt"`
	Related      []PostSummary       `json:"related"`
	Release      *PostRelease        `json:"release,omitempty"`
	Slug         string              `json:"slug"`
	SocialImage  *PostMedia          `json:"socialImage,omitempty"`
	Summary      string              `json:"summary"`
	Title        string              `json:"title"`
	UpdatedAt    *time.Time          `json:"updatedAt,omitempty"`
}

type PublicationAppList struct {
	Apps []PublicationApp `json:"apps"`
}

type PublishPostRequest struct {
	DestinationIds     *[]uuid.UUID `json:"destinationIds,omitempty"`
	Note               *string      `json:"note,omitempty"`
	RoleDestinationIds *[]uuid.UUID `json:"roleDestinationIds,omitempty"`
	Version            int          `json:"version"`
}

type SavePostRequest struct {
	CategoryId    uuid.UUID        `json:"categoryId"`
	Document      PostDocument     `json:"document"`
	Header        *PostHeaderEdit  `json:"header,omitempty"`
	Release       *PostReleaseEdit `json:"release,omitempty"`
	Slug          string           `json:"slug"`
	SocialMediaId *uuid.UUID       `json:"socialMediaId,omitempty"`
	Summary       string           `json:"summary"`
	Title         string           `json:"title"`
	Version       int              `json:"version"`
}

type ListPostsParams struct {
	Deleted *bool `json:"deleted,omitempty"`
}

type ListPublishedPostsParams struct {
	Page     *int    `json:"page,omitempty"`
	Category *string `json:"category,omitempty"`
	App      *string `json:"app,omitempty"`
}
