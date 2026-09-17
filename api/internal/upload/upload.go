package upload

import (
	"errors"
	"io"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrIngestNotFound   = errors.New("ingest operation not found")
	ErrKindNotBuildable = errors.New("that kind cannot be built yet")
	ErrAppNotAnswered   = errors.New("that kind needs to know which app it is for")
)

type CreateInput struct {
	OwnerID   uuid.UUID
	Kind      string
	Filename  string
	File      io.Reader
	Name      string
	Blurb     string
	Tags      []string
	IsNSFW    bool
	Discovery asset.Discovery
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
	Discovery asset.Discovery
}

type RevisionInput struct {
	OwnerID  uuid.UUID
	AssetID  uuid.UUID
	Filename string
	File     io.Reader
}

type Status string

const (
	IngestPending    Status = "pending"
	IngestProcessing Status = "processing"
	IngestPreview    Status = "preview"
	IngestCancelled  Status = "cancelled"
	IngestFailed     Status = "failed"
	IngestSuccess    Status = "success"
)

type Operation struct {
	ID      uuid.UUID
	Status  Status
	Failure *Failure
	Asset   *asset.Asset
	Preview *Preview
}

type Failure struct {
	Reason  string
	Message string
}

type Preview struct {
	Format          string
	Groups          []asset.ChangeGroup
	Conflicts       []string
	Unrepresentable []string
	MissingWording  []string
	Seals           int
}

func uuidFromPgtype(p pgtype.UUID) uuid.UUID {
	return p.Bytes
}

func textToPointer(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}
