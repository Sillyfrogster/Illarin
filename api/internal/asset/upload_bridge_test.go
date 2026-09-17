package asset

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
)

// UploadBridge reaches the upload package, which imports this one, and is filled in by upload_external_test.go
var UploadBridge struct {
	StartFromNothing  func(context.Context, *Service, uuid.UUID, string, string) (uuid.UUID, error)
	Create            func(context.Context, *Service, CreateInput) (Asset, error)
	AcceptIngest      func(context.Context, *Service, IngestInput) (IngestOperation, error)
	ProcessNextIngest func(context.Context, *Service) (bool, error)
	GetIngest         func(context.Context, *Service, uuid.UUID, uuid.UUID) (IngestOperation, error)
	AcceptRevision    func(context.Context, *Service, RevisionInput, *Candidate) (IngestOperation, error)
	AcceptReplacement func(context.Context, *Service, uuid.UUID, uuid.UUID, uuid.UUID, *Candidate, map[string]string, bool) (IngestOperation, error)
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
	IngestPreview IngestStatus = "preview"
	IngestFailed  IngestStatus = "failed"
	IngestSuccess IngestStatus = "success"
)

type IngestOperation struct {
	ID      uuid.UUID
	Status  IngestStatus
	Failure *IngestFailure
	Asset   *Asset
}

type IngestFailure struct {
	Reason  string
	Message string
}

func (s *Service) StartFromNothing(ctx context.Context, ownerID uuid.UUID, kind, app string) (uuid.UUID, error) {
	return UploadBridge.StartFromNothing(ctx, s, ownerID, kind, app)
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Asset, error) {
	return UploadBridge.Create(ctx, s, in)
}

func (s *Service) AcceptIngest(ctx context.Context, in IngestInput) (IngestOperation, error) {
	return UploadBridge.AcceptIngest(ctx, s, in)
}

func (s *Service) ProcessNextIngest(ctx context.Context) (bool, error) {
	return UploadBridge.ProcessNextIngest(ctx, s)
}

func (s *Service) GetIngest(ctx context.Context, ownerID, id uuid.UUID) (IngestOperation, error) {
	return UploadBridge.GetIngest(ctx, s, ownerID, id)
}

func (s *Service) AcceptRevision(ctx context.Context, in RevisionInput, candidate *Candidate) (IngestOperation, error) {
	return UploadBridge.AcceptRevision(ctx, s, in, candidate)
}

func (s *Service) AcceptReplacement(ctx context.Context, ownerID, assetID, operationID uuid.UUID, candidate *Candidate, decisions map[string]string, exposeProtected bool) (IngestOperation, error) {
	return UploadBridge.AcceptReplacement(ctx, s, ownerID, assetID, operationID, candidate, decisions, exposeProtected)
}

func currentCandidate(t *testing.T, svc *Service, id uuid.UUID) *Candidate {
	t.Helper()
	var candidate Candidate
	if err := svc.pool.QueryRow(context.Background(), `select working_copy_version from assets where id = $1`, id).Scan(&candidate.Version); err != nil {
		t.Fatal(err)
	}
	return &candidate
}

func revisionOwner(t *testing.T, svc *Service, handle string) uuid.UUID {
	t.Helper()
	ownerID := uuid.New()
	if _, err := svc.pool.Exec(context.Background(),
		`insert into users (id, username) values ($1, $2)`, ownerID, handle); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	return ownerID
}

func ingestOne(t *testing.T, svc *Service, ownerID uuid.UUID, filename string, file []byte) Asset {
	t.Helper()
	operation, err := svc.AcceptIngest(context.Background(), IngestInput{
		OwnerID: ownerID, Filename: filename, File: bytes.NewReader(file),
	})
	if err != nil {
		t.Fatalf("AcceptIngest: %v", err)
	}
	if processed, err := svc.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("ProcessNextIngest = %v, %v; want true, nil", processed, err)
	}
	operation, err = svc.GetIngest(context.Background(), ownerID, operation.ID)
	if err != nil {
		t.Fatalf("GetIngest: %v", err)
	}
	if operation.Asset == nil {
		t.Fatalf("ingest did not create an asset: %+v", operation)
	}
	return *operation.Asset
}

func publishImported(t *testing.T, svc *Service, ownerID uuid.UUID, created Asset) {
	t.Helper()
	name := created.Name
	if name == "" {
		name = "Test asset"
	}
	nsfw := false
	if err := svc.SetIdentity(context.Background(), Identity{
		OwnerID: ownerID, AssetID: created.ID, Name: name, Blurb: created.Blurb, IsNSFW: &nsfw,
	}, currentCandidate(t, svc, created.ID)); err != nil {
		t.Fatalf("SetIdentity imported asset: %v", err)
	}
	if _, err := svc.Publish(context.Background(), ownerID, created.ID, currentCandidate(t, svc, created.ID)); err != nil {
		t.Fatalf("Publish imported asset: %v", err)
	}
}

func addRevision(
	t *testing.T,
	svc *Service,
	ownerID, assetID uuid.UUID,
	filename string,
	file []byte,
) IngestOperation {
	t.Helper()
	operation, err := svc.AcceptRevision(context.Background(), RevisionInput{
		OwnerID: ownerID, AssetID: assetID, Filename: filename, File: bytes.NewReader(file),
	}, currentCandidate(t, svc, assetID))
	if err != nil {
		t.Fatalf("AcceptRevision: %v", err)
	}
	if processed, err := svc.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("ProcessNextIngest = %v, %v; want true, nil", processed, err)
	}
	got, err := svc.GetIngest(context.Background(), ownerID, operation.ID)
	if err != nil {
		t.Fatalf("GetIngest: %v", err)
	}
	if got.Status == IngestPreview {
		got, err = svc.AcceptReplacement(context.Background(), ownerID, assetID, operation.ID, currentCandidate(t, svc, assetID), nil, false)
		if err != nil {
			t.Fatalf("AcceptReplacement: %v", err)
		}
	}
	return got
}
