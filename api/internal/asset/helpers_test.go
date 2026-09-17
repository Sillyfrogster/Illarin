package asset

import (
	"context"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestService(t *testing.T) (*Service, *pgxpool.Pool) {
	t.Helper()
	return newTestServiceWithRegistry(t, registryWithModule(t, opaqueTestModule{}))
}

func newTestServiceWithRegistry(t *testing.T, registry *format.Registry) (*Service, *pgxpool.Pool) {
	t.Helper()
	pool := testdb.Connect(t)
	blob, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	return NewService(pool, registry, blob), pool
}

func registryWithModule(t *testing.T, module format.Module) *format.Registry {
	t.Helper()
	registry := format.NewRegistry()
	if err := registry.Register(module); err != nil {
		t.Fatalf("register module: %v", err)
	}
	return registry
}

func testReaderDeclaration(id, kind string) format.Declaration {
	return format.Declaration{
		ID: id, Kind: kind, Direction: format.Direction{Read: true},
		Recognition: []format.Recognition{{
			Kind: format.RecognitionSignature, Containers: []format.Container{format.JSON},
			Required: map[string]format.ValueType{"payload": format.ValueBoolean},
		}},
		Limits: format.ContentLimits{
			PayloadBytes: block.MaxPayloadBytes, CollectionItems: block.MaxCollectionItems,
			ItemBytes: block.MaxItemBytes,
		},
		ConsumedKeys:  []string{"payload"},
		Preservation:  format.PreservationDeclaration{Body: "test"},
		TestedOrigins: []string{id},
	}
}

type claimsFirstPayload struct{}

func (claimsFirstPayload) Claim(file format.Inspection) (format.Claim, bool) {
	if len(file.Payloads) == 0 {
		return format.Claim{}, false
	}
	return format.CompatibilityClaim(file.Payloads[0]), true
}

type opaqueTestModule struct{}

func (opaqueTestModule) ID() string { return "test_opaque" }
func (opaqueTestModule) Declaration() format.Declaration {
	declaration := testReaderDeclaration("test_opaque", "character")
	declaration.Label = "Test format"
	declaration.Direction.Write = true
	declaration.Header = []format.HeaderField{format.HeaderName, format.HeaderAssetVersion}
	declaration.TestedOrigins = append(declaration.TestedOrigins, format.OriginIllarin)
	declaration.Roles = map[block.Role]format.DirectionalRoleSupport{
		block.RoleDescription: {
			Read:  format.RoleSupport{Grade: format.SupportFull},
			Write: format.RoleSupport{Grade: format.SupportFull},
		},
		block.RoleGreetings: {
			Read:  format.RoleSupport{Grade: format.SupportFull},
			Write: format.RoleSupport{Grade: format.SupportFull},
		},
	}
	return declaration
}
func (opaqueTestModule) Write(_ context.Context, written format.ExportAsset) (format.Artifact, error) {
	return format.Artifact{
		Body:      []byte(written.Text(block.RoleDescription)),
		MediaType: "text/plain", Extension: ".txt",
	}, nil
}
func (opaqueTestModule) Claim(file format.Inspection) (format.Claim, bool) {
	return format.WholeFileCompatibilityClaim(file), true
}
func (opaqueTestModule) Parse(context.Context, format.Inspection, format.Claim) (format.Parsed, error) {
	return format.Parsed{
		Kind: "character", Format: "test_opaque",
		Elements: []block.Element{
			{Type: block.TypeProse, Role: block.RoleDescription, Content: block.Prose{Text: "Test description"}},
			{Type: block.TypeTextSet, Role: block.RoleGreetings, Content: block.TextSet{Texts: []block.TextItem{{ID: block.NewItemID(), Text: "Hello"}}}},
		},
	}, nil
}

type recognizedModule struct {
	claimsFirstPayload
	parsed format.Parsed
}

func (recognizedModule) ID() string { return "recognized" }
func (recognizedModule) Declaration() format.Declaration {
	return testReaderDeclaration("recognized", "character")
}
func (module recognizedModule) Parse(context.Context, format.Inspection, format.Claim) (format.Parsed, error) {
	return module.parsed, nil
}

type replacingModule struct {
	claimsFirstPayload
	parsed *format.Parsed
}

func (replacingModule) ID() string { return "replacing" }
func (replacingModule) Declaration() format.Declaration {
	declaration := testReaderDeclaration("replacing", "character")
	declaration.Label = "Replacing format"
	declaration.Direction.Write = true
	declaration.Header = []format.HeaderField{format.HeaderName, format.HeaderAssetVersion}
	declaration.TestedOrigins = append(declaration.TestedOrigins, format.OriginIllarin)
	declaration.Roles = map[block.Role]format.DirectionalRoleSupport{
		block.RoleDescription: {
			Read:  format.RoleSupport{Grade: format.SupportFull},
			Write: format.RoleSupport{Grade: format.SupportFull},
		},
	}
	return declaration
}
func (module replacingModule) Parse(context.Context, format.Inspection, format.Claim) (format.Parsed, error) {
	return *module.parsed, nil
}
func (replacingModule) Write(context.Context, format.ExportAsset) (format.Artifact, error) {
	return format.Artifact{MediaType: "text/plain", Extension: ".txt"}, nil
}

func startedDraft(t *testing.T, service *Service) (uuid.UUID, uuid.UUID) {
	t.Helper()
	owner := uuid.New()
	draft, err := service.StartFromNothing(context.Background(), owner, "character", "")
	if err != nil {
		t.Fatalf("start a draft: %v", err)
	}
	return owner, draft
}

func contentGeneration(t *testing.T, pool *pgxpool.Pool, assetID uuid.UUID) int {
	t.Helper()
	var generation int
	if err := pool.QueryRow(context.Background(),
		`select content_generation from assets where id = $1`, assetID,
	).Scan(&generation); err != nil {
		t.Fatalf("read the content generation: %v", err)
	}
	return generation
}

func draftBlocks(t *testing.T, pool *pgxpool.Pool, assetID uuid.UUID) []block.Block {
	t.Helper()
	blocks, err := block.Read(context.Background(), pool, assetID)
	if err != nil {
		t.Fatalf("read the blocks: %v", err)
	}
	return blocks
}

func blockFor(t *testing.T, blocks []block.Block, definition block.DefinitionID) block.Block {
	t.Helper()
	for _, holder := range blocks {
		if holder.Definition == definition {
			return holder
		}
	}
	t.Fatalf("no %s block on the page", definition)
	return block.Block{}
}

func updateOf(holder block.Block) BlockUpdate {
	return BlockUpdate{
		Title: holder.Title, Layout: holder.Layout, Width: holder.Width,
		Elements: append([]block.Element(nil), holder.Elements...),
	}
}

func saveDescription(t *testing.T, service *Service, owner, draft uuid.UUID, pool *pgxpool.Pool, text string) {
	t.Helper()
	core := blockFor(t, draftBlocks(t, pool, draft), block.CharacterCore)
	update := updateOf(core)
	update.Elements[0].Content = block.Prose{Text: text}
	if _, err := service.SaveBlock(context.Background(), owner, draft, core.ID, update, currentCandidate(t, service, draft)); err != nil {
		t.Fatalf("save the description: %v", err)
	}
}

func saveGreeting(t *testing.T, service *Service, owner, draft uuid.UUID, pool *pgxpool.Pool, text string) {
	t.Helper()
	messages := blockFor(t, draftBlocks(t, pool, draft), block.Messages)
	update := updateOf(messages)
	update.Elements[0].Content = block.TextSet{
		Texts: []block.TextItem{{ID: block.NewItemID(), Text: text}},
	}
	if _, err := service.SaveBlock(context.Background(), owner, draft, messages.ID, update, currentCandidate(t, service, draft)); err != nil {
		t.Fatalf("save the greeting: %v", err)
	}
}
