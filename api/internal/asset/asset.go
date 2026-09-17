package asset

import (
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
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

type SavedBlocks struct {
	Kind   string
	Blocks []block.Block
}
