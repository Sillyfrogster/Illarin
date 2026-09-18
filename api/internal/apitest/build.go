package apitest

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/block/edit"
	"github.com/Sillyfrogster/Illarin/api/internal/download"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
	"github.com/Sillyfrogster/Illarin/api/internal/upload"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Assets gives a test the shared service over a fresh database and blob store
func Assets(t *testing.T) (*work.Service, *pgxpool.Pool) {
	t.Helper()
	return AssetsWithRegistry(t, RegistryWith(t, OpaqueModule{}))
}

func AssetsWithRegistry(t *testing.T, registry *format.Registry) (*work.Service, *pgxpool.Pool) {
	t.Helper()
	pool := testdb.Connect(t)
	blob, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	return work.NewService(pool, registry, blob), pool
}

func RegistryWith(t *testing.T, modules ...format.Module) *format.Registry {
	t.Helper()
	registry := format.NewRegistry()
	for _, module := range modules {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %q: %v", module.ID(), err)
		}
	}
	return registry
}

func Works(assets *work.Service) *page.Service {
	return page.NewService(assets.Pool(), assets)
}

func Blocks(assets *work.Service) *edit.Service {
	return edit.NewService(assets.Pool(), assets)
}

func Downloads(assets *work.Service) *download.Service {
	return download.NewService(assets.Pool(), assets)
}

func Versions(assets *work.Service) *version.Service {
	return version.NewService(assets.Pool(), assets)
}

// StartedDraft makes an owner and an empty character draft
func StartedDraft(t *testing.T, assets *work.Service) (uuid.UUID, uuid.UUID) {
	t.Helper()
	owner := uuid.New()
	draft, err := Uploads(assets).StartFromNothing(context.Background(), owner, "character", "")
	if err != nil {
		t.Fatalf("start a draft: %v", err)
	}
	return owner, draft
}

// CurrentCandidate reads the working copy version a change must present
func CurrentCandidate(t *testing.T, assets *work.Service, id uuid.UUID) *work.Candidate {
	t.Helper()
	var candidate work.Candidate
	if err := assets.Pool().QueryRow(context.Background(),
		`select working_copy_version from assets where id = $1`, id,
	).Scan(&candidate.Version); err != nil {
		t.Fatalf("read the working copy version: %v", err)
	}
	return &candidate
}

func DraftBlocks(t *testing.T, pool *pgxpool.Pool, workID uuid.UUID) []block.Block {
	t.Helper()
	blocks, err := block.Read(context.Background(), pool, workID)
	if err != nil {
		t.Fatalf("read the blocks: %v", err)
	}
	return blocks
}

func BlockFor(t *testing.T, blocks []block.Block, definition block.DefinitionID) block.Block {
	t.Helper()
	for _, holder := range blocks {
		if holder.Definition == definition {
			return holder
		}
	}
	t.Fatalf("no %s block on the page", definition)
	return block.Block{}
}

func UpdateOf(holder block.Block) edit.BlockUpdate {
	return edit.BlockUpdate{
		Title: holder.Title, Layout: holder.Layout, Width: holder.Width,
		Elements: append([]block.Element(nil), holder.Elements...),
	}
}

func SaveDescription(t *testing.T, assets *work.Service, owner, draft uuid.UUID, text string) {
	t.Helper()
	core := BlockFor(t, DraftBlocks(t, assets.Pool(), draft), block.CharacterCore)
	update := UpdateOf(core)
	update.Elements[0].Content = block.Prose{Text: text}
	if _, err := Blocks(assets).SaveBlock(
		context.Background(), owner, draft, core.ID, update, CurrentCandidate(t, assets, draft),
	); err != nil {
		t.Fatalf("save the description: %v", err)
	}
}

func SaveGreeting(t *testing.T, assets *work.Service, owner, draft uuid.UUID, text string) {
	t.Helper()
	messages := BlockFor(t, DraftBlocks(t, assets.Pool(), draft), block.Messages)
	update := UpdateOf(messages)
	update.Elements[0].Content = block.TextSet{
		Texts: []block.TextItem{{ID: block.NewItemID(), Text: text}},
	}
	if _, err := Blocks(assets).SaveBlock(
		context.Background(), owner, draft, messages.ID, update, CurrentCandidate(t, assets, draft),
	); err != nil {
		t.Fatalf("save the greeting: %v", err)
	}
}

// Sweeper cleans up blobs the way the server's background sweeper does
func Sweeper(assets *work.Service) *storage.Sweeper {
	return storage.NewSweeper(assets.Pool(), assets.Store())
}

// SweeperAt cleans up on a clock the test moves
func SweeperAt(assets *work.Service, now func() time.Time) *storage.Sweeper {
	return storage.NewSweeperWithClock(assets.Pool(), assets.Store(), now)
}

// Owner makes an account that can own a work
func Owner(t *testing.T, assets *work.Service, handle string) uuid.UUID {
	t.Helper()
	ownerID := uuid.New()
	if _, err := assets.Pool().Exec(context.Background(),
		`insert into users (id, username) values ($1, $2)`, ownerID, handle,
	); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	return ownerID
}

// PublishedWork makes an owner and a published character with a description and a greeting
func PublishedWork(t *testing.T, assets *work.Service, handle string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	owner := Owner(t, assets, handle)
	id, err := Uploads(assets).StartFromNothing(context.Background(), owner, "character", "")
	if err != nil {
		t.Fatalf("start a draft: %v", err)
	}
	SaveDescription(t, assets, owner, id, "Published description")
	SaveGreeting(t, assets, owner, id, "Published greeting")
	adult := false
	works := Works(assets)
	if err := works.SetIdentity(context.Background(), page.Identity{
		OwnerID: owner, AssetID: id, Name: "Published name", IsNSFW: &adult,
	}, CurrentCandidate(t, assets, id)); err != nil {
		t.Fatalf("save the header: %v", err)
	}
	if _, err := works.Publish(context.Background(), owner, id, CurrentCandidate(t, assets, id)); err != nil {
		t.Fatalf("publish the asset: %v", err)
	}
	return owner, id
}

// IngestOne reads one file in and returns the work it made
func IngestOne(t *testing.T, assets *work.Service, ownerID uuid.UUID, filename string, file []byte) work.Asset {
	t.Helper()
	uploads := Uploads(assets)
	operation, err := uploads.AcceptIngest(context.Background(), upload.IngestInput{
		OwnerID: ownerID, Filename: filename, File: bytes.NewReader(file),
	})
	if err != nil {
		t.Fatalf("AcceptIngest: %v", err)
	}
	if processed, err := uploads.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("ProcessNextIngest = %v, %v; want true, nil", processed, err)
	}
	operation, err = uploads.GetIngest(context.Background(), ownerID, operation.ID)
	if err != nil {
		t.Fatalf("GetIngest: %v", err)
	}
	if operation.Asset == nil {
		t.Fatalf("ingest did not create an asset: %+v", operation)
	}
	return *operation.Asset
}

// PublishImported gives an imported work a name and publishes it
func PublishImported(t *testing.T, svc *work.Service, ownerID uuid.UUID, created work.Asset) {
	t.Helper()
	name := created.Name
	if name == "" {
		name = "Test asset"
	}
	nsfw := false
	works := Works(svc)
	if err := works.SetIdentity(context.Background(), page.Identity{
		OwnerID: ownerID, AssetID: created.ID, Name: name, Blurb: created.Blurb, IsNSFW: &nsfw,
	}, CurrentCandidate(t, svc, created.ID)); err != nil {
		t.Fatalf("SetIdentity imported asset: %v", err)
	}
	if _, err := works.Publish(
		context.Background(), ownerID, created.ID, CurrentCandidate(t, svc, created.ID),
	); err != nil {
		t.Fatalf("Publish imported asset: %v", err)
	}
}

// AddRevision uploads a replacement file and takes it all the way to a saved revision
func AddRevision(
	t *testing.T,
	assets *work.Service,
	ownerID, workID uuid.UUID,
	filename string,
	file []byte,
) upload.Operation {
	t.Helper()
	uploads := Uploads(assets)
	ctx := context.Background()
	operation, err := uploads.AcceptRevision(ctx, upload.RevisionInput{
		OwnerID: ownerID, AssetID: workID, Filename: filename, File: bytes.NewReader(file),
	}, CurrentCandidate(t, assets, workID))
	if err != nil {
		t.Fatalf("AcceptRevision: %v", err)
	}
	if processed, err := uploads.ProcessNextIngest(ctx); err != nil || !processed {
		t.Fatalf("ProcessNextIngest = %v, %v; want true, nil", processed, err)
	}
	got, err := uploads.GetIngest(ctx, ownerID, operation.ID)
	if err != nil {
		t.Fatalf("GetIngest: %v", err)
	}
	if got.Status == upload.IngestPreview {
		got, err = uploads.AcceptReplacement(
			ctx, ownerID, workID, operation.ID,
			CurrentCandidate(t, assets, workID), nil, false,
		)
		if err != nil {
			t.Fatalf("AcceptReplacement: %v", err)
		}
	}
	return got
}
