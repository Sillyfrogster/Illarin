package work

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

type Visibility string

const (
	VisibilityListed   Visibility = "listed"
	VisibilityUnlisted Visibility = "unlisted"
)

func (d Visibility) Valid() bool {
	return d == VisibilityListed || d == VisibilityUnlisted
}

type Work struct {
	ID             uuid.UUID
	Type           string
	Format         string
	OriginalFormat *string
	WorkVersion    string
	CreditedAuthor string
	Nickname       string
	Name           string
	Blurb          string
	Tags           []string
	IsNSFW         *bool
	Visibility     Visibility
	Lifecycle      Lifecycle
	OriginalFileID uuid.UUID
	CreatedAt      time.Time
}

type NSFWPreference string

const (
	NSFWHidden  NSFWPreference = "hidden"
	NSFWBlurred NSFWPreference = "blurred"
	NSFWShown   NSFWPreference = "shown"
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
	Type   string
	Blocks []block.Block
}

type OriginalUpload struct {
	Label     string
	MediaType string
	ArrivedAt time.Time
}
