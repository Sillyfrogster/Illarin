package upload

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/character"
	"github.com/Sillyfrogster/Illarin/api/internal/format/theme"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

type leasedModule struct {
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func TestImportPayloadLimitNamesTheLimitAndActualBytes(t *testing.T) {
	t.Parallel()
	pool := testdb.Connect(t)
	ownerID := uuid.New()
	if _, err := pool.Exec(context.Background(),
		`insert into users (id, username) values ($1, 'payload.limit.owner')`, ownerID); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	blobs, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	registry := format.NewRegistry()
	for _, module := range character.Modules() {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %s: %v", module.ID(), err)
		}
	}
	service := NewService(pool, work.NewService(pool, registry, blobs))
	payload, err := json.Marshal(map[string]any{
		"spec": "chara_card_v3", "spec_version": "3.0",
		"data": map[string]any{
			"description": strings.Repeat("x", block.MaxPayloadBytes), "first_mes": "Hello",
		},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	operation, err := service.AcceptIngest(context.Background(), IngestInput{
		OwnerID: ownerID, Filename: "too-large.json", File: bytes.NewReader(payload),
	})
	if err != nil {
		t.Fatalf("AcceptIngest: %v", err)
	}
	if processed, err := service.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("ProcessNextIngest = %v, %v", processed, err)
	}
	got, err := service.GetIngest(context.Background(), ownerID, operation.ID)
	if err != nil {
		t.Fatalf("GetIngest: %v", err)
	}
	if got.Failure == nil || got.Failure.Reason != string(format.FailureLimitExceeded) ||
		!strings.Contains(got.Failure.Message, strconv.Itoa(block.MaxPayloadBytes)) ||
		!strings.Contains(got.Failure.Message, strconv.Itoa(len(payload))) {
		t.Fatalf("failure = %+v, want limit %d and actual %d", got.Failure, block.MaxPayloadBytes, len(payload))
	}
	var works int
	if err := pool.QueryRow(context.Background(), `select count(*) from works`).Scan(&works); err != nil || works != 0 {
		t.Fatalf("works = %d, %v; over-limit import must store none", works, err)
	}
}

func TestUnrecognisedImportFailsTerminallyAndReleasesItsBlobReference(t *testing.T) {
	t.Parallel()
	pool := testdb.Connect(t)
	ownerID := uuid.New()
	if _, err := pool.Exec(context.Background(),
		`insert into users (id, username) values ($1, 'expiry.owner')`, ownerID); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	blobs, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	service := NewService(pool, work.NewService(pool, format.NewRegistry(), blobs))
	operation, err := service.AcceptIngest(context.Background(), IngestInput{
		OwnerID: ownerID, Filename: "mystery.bundle", File: bytes.NewReader([]byte("mystery")),
	})
	if err != nil {
		t.Fatalf("accept ingest: %v", err)
	}
	if processed, err := service.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("process ingest = %v, %v; want true, nil", processed, err)
	}

	got, err := service.GetIngest(context.Background(), ownerID, operation.ID)
	if err != nil {
		t.Fatalf("GetIngest: %v", err)
	}
	if got.Status != IngestFailed || got.Failure == nil ||
		got.Failure.Reason != string(format.FailureUnsupportedFormat) {
		t.Fatalf("operation = %+v, want terminal unsupported_format", got)
	}
	var blobReferences int
	if err := pool.QueryRow(context.Background(), `
		select count(*) from ingest_operations where id = $1 and blob_id is not null
	`, operation.ID).Scan(&blobReferences); err != nil {
		t.Fatalf("count references: %v", err)
	}
	if blobReferences != 0 {
		t.Fatalf("failed ingest blob references = %d, want 0", blobReferences)
	}
}

func (*leasedModule) ID() string { return "leased" }
func (*leasedModule) Declaration() format.Declaration {
	return testReaderDeclaration("leased", "character")
}
func (*leasedModule) Claim(file format.Inspection) (format.Claim, bool) {
	if len(file.Payloads) == 0 {
		return format.Claim{}, false
	}
	return format.CompatibilityClaim(file.Payloads[0]), true
}
func (m *leasedModule) Parse(context.Context, format.Inspection, format.Claim) (format.Parsed, error) {
	if m.calls.Add(1) == 1 {
		close(m.started)
		<-m.release
	}
	return format.Parsed{Type: "character", Format: "leased"}, nil
}

func TestExpiredLeaseIsReclaimedAndFinalizationIsIdempotent(t *testing.T) {
	t.Parallel()
	pool := testdb.Connect(t)
	ownerID := uuid.New()
	if _, err := pool.Exec(context.Background(),
		`insert into users (id, username) values ($1, 'lease.owner')`, ownerID); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	blobs, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	module := &leasedModule{started: make(chan struct{}), release: make(chan struct{})}
	registry := format.NewRegistry()
	if err := registry.Register(module); err != nil {
		t.Fatalf("register module: %v", err)
	}
	settings := work.DefaultIngestSettings()
	settings.LeaseDuration = time.Minute
	service := NewService(pool, work.NewServiceWithIngestSettings(pool, registry, blobs, settings))
	name := "Leased card"
	_, err = service.AcceptIngest(context.Background(), IngestInput{
		OwnerID: ownerID, Filename: "leased.json", File: bytes.NewReader([]byte(`{"value":true}`)),
		Name: &name, Visibility: work.VisibilityListed,
	})
	if err != nil {
		t.Fatalf("accept ingest: %v", err)
	}
	var clock atomic.Value
	clock.Store(time.Now().Add(time.Second))
	service.now = func() time.Time { return clock.Load().(time.Time) }

	firstDone := make(chan error, 1)
	go func() {
		_, processErr := service.ProcessNextIngest(context.Background())
		firstDone <- processErr
	}()
	<-module.started

	clock.Store(clock.Load().(time.Time).Add(2 * time.Minute))
	if processed, err := service.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("reclaimed process = %v, %v; want true, nil", processed, err)
	}
	close(module.release)
	if err := <-firstDone; err != nil {
		t.Fatalf("stale worker finalization: %v", err)
	}

	for _, table := range []string{"works", "work_revisions"} {
		var count int
		if err := pool.QueryRow(context.Background(), "select count(*) from "+table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("%s count = %d, want 1", table, count)
		}
	}
	var status string
	if err := pool.QueryRow(context.Background(),
		`select status from ingest_operations`).Scan(&status); err != nil {
		t.Fatalf("read operation: %v", err)
	}
	if status != "success" {
		t.Errorf("operation status = %q, want success", status)
	}
}

func TestAThemeArchiveOverItsFileLimitIsRefusedByName(t *testing.T) {
	t.Parallel()
	pool := testdb.Connect(t)
	ownerID := uuid.New()
	if _, err := pool.Exec(context.Background(),
		`insert into users (id, username) values ($1, 'file.limit.owner')`, ownerID); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	blobs, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	registry := format.NewRegistry()
	for _, module := range theme.Modules() {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %s: %v", module.ID(), err)
		}
	}
	var bundle bytes.Buffer
	archive := zip.NewWriter(&bundle)
	for index := range format.MaxArchiveFiles + 1 {
		name, content := fmt.Sprintf("assets/%d.css", index), ""
		if index == 0 {
			name, content = "theme.json", `{"format":3}`
		}
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close bundle: %v", err)
	}
	service := NewService(pool, work.NewService(pool, registry, blobs))
	operation, err := service.AcceptIngest(context.Background(), IngestInput{
		OwnerID: ownerID, Filename: "crowded.lumitheme", File: bytes.NewReader(bundle.Bytes()),
	})
	if err != nil {
		t.Fatalf("AcceptIngest: %v", err)
	}
	if processed, err := service.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("ProcessNextIngest = %v, %v", processed, err)
	}
	got, err := service.GetIngest(context.Background(), ownerID, operation.ID)
	if err != nil {
		t.Fatalf("GetIngest: %v", err)
	}
	want := fmt.Sprintf("holds %d files, and one may hold %d", format.MaxArchiveFiles+1, format.MaxArchiveFiles)
	if got.Failure == nil || got.Failure.Reason != string(format.FailureLimitExceeded) ||
		!strings.Contains(got.Failure.Message, want) || strings.Contains(got.Failure.Message, "limit_exceeded") {
		t.Fatalf("failure = %+v, want the file limit named: %q", got.Failure, want)
	}
}
