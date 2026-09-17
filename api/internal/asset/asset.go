package asset

import (
	"io"
	"time"

	"github.com/google/uuid"
)

type Lifecycle string

const (
	LifecycleDraft     Lifecycle = "draft"
	LifecyclePublished Lifecycle = "published"
)

type Discovery string

const (
	DiscoveryListed   Discovery = "listed"
	DiscoveryUnlisted Discovery = "unlisted"
)

func (d Discovery) Valid() bool {
	return d == DiscoveryListed || d == DiscoveryUnlisted
}

type Asset struct {
	ID                uuid.UUID
	Kind              string
	Format            string
	OriginFormat      *string
	AssetVersion      string
	CreditedAuthor    string
	Nickname          string
	Name              string
	Blurb             string
	Tags              []string
	IsNSFW            *bool
	Discovery         Discovery
	Lifecycle         Lifecycle
	CurrentRevisionID uuid.UUID
	CreatedAt         time.Time
}

type CreateInput struct {
	OwnerID   uuid.UUID
	Kind      string
	Filename  string
	File      io.Reader
	Name      string
	Blurb     string
	Tags      []string
	IsNSFW    bool
	Discovery Discovery
	CreatedAt *time.Time
}

type IngestInput struct {
	OwnerID   uuid.UUID
	Filename  string
	File      io.Reader
	Name      *string
	Blurb     *string
	Tags      *[]string
	IsNSFW    *bool
	Discovery Discovery
}

type RevisionInput struct {
	OwnerID  uuid.UUID
	AssetID  uuid.UUID
	Filename string
	File     io.Reader
}

type IngestStatus string

const (
	IngestPending    IngestStatus = "pending"
	IngestProcessing IngestStatus = "processing"
	IngestPreview    IngestStatus = "preview"
	IngestCancelled  IngestStatus = "cancelled"
	IngestFailed     IngestStatus = "failed"
	IngestSuccess    IngestStatus = "success"
)

type IngestOperation struct {
	ID      uuid.UUID
	Status  IngestStatus
	Failure *IngestFailure
	Asset   *Asset
	Preview *ReplacementPreview
}

type IngestFailure struct {
	Reason  string
	Message string
}

type ReplacementPreview struct {
	Format          string
	Groups          []ChangeGroup
	Conflicts       []string
	Unrepresentable []string
	MissingWording  []string
	Seals           int
}

type ContentVisibility string

const (
	ContentHidden  ContentVisibility = "hidden"
	ContentBlurred ContentVisibility = "blurred"
	ContentShown   ContentVisibility = "shown"
)

type DetailImage struct {
	ID        uuid.UUID
	Role      MediaRole
	IsCover   bool
	DetailURL string
	ThumbURL  string
	Width     int
	Height    int
	Bytes     int64
}
