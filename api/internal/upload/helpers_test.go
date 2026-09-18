package upload

import (
	"archive/zip"
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
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
	return NewService(pool, work.NewService(pool, registry, blob)), pool
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

func currentCandidate(t *testing.T, svc *Service, id uuid.UUID) *work.Candidate {
	t.Helper()
	var candidate work.Candidate
	if err := svc.pool.QueryRow(context.Background(), `select working_copy_version from assets where id = $1`, id).Scan(&candidate.Version); err != nil {
		t.Fatal(err)
	}
	return &candidate
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

func works(s *Service) *page.Service {
	return page.NewService(s.pool, s.assets)
}

func testPNG(t *testing.T, width, height int, fill color.Color) []byte {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			picture.Set(x, y, fill)
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, picture); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	return encoded.Bytes()
}

func archiveWithImage(t *testing.T, picture []byte) []byte {
	t.Helper()
	var file bytes.Buffer
	archive := zip.NewWriter(&file)
	for _, entry := range []struct{ name, body string }{
		{name: "card.json", body: `{"spec":"chara_card_v3"}`},
		{name: "assets/icon/main.png", body: string(picture)},
	} {
		writer, err := archive.Create(entry.name)
		if err != nil {
			t.Fatalf("create archive entry: %v", err)
		}
		if _, err := io.WriteString(writer, entry.body); err != nil {
			t.Fatalf("write archive entry: %v", err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	return file.Bytes()
}
