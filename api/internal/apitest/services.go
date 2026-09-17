package apitest

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/preset"
	"github.com/Sillyfrogster/Illarin/api/internal/integration"
	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/Sillyfrogster/Illarin/api/internal/secrets"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func Registry(t *testing.T) *format.Registry {
	t.Helper()
	registry := format.NewRegistry()
	if err := registry.Register(OpaqueModule{}); err != nil {
		t.Fatalf("register test format: %v", err)
	}
	if err := registry.Register(preset.LumiverseModule{}); err != nil {
		t.Fatalf("register Lumiverse preset format: %v", err)
	}
	return registry
}

func NewNotifications(pool *pgxpool.Pool) *notify.Service {
	return notify.NewService(pool)
}

func NewAccounts(
	pool *pgxpool.Pool,
	sender account.EmailSender,
	provider account.DiscordProvider,
	library *mediaproc.Library,
) *account.Service {
	return account.NewService(pool, sender, provider, library, "http://localhost:3000").
		WithPasswordCost(bcrypt.MinCost)
}

func MediaLibrary(store storage.Store) *mediaproc.Library {
	return mediaproc.NewLibrary(store, mediaproc.NewProcessor(mediaproc.DefaultLimits()), 1)
}

func NewPublicationService(pool *pgxpool.Pool, store storage.Store) *publication.Service {
	return publication.NewService(
		pool, MediaLibrary(store), Publishing(nil),
	)
}

func NewUpdateDestinations(pool *pgxpool.Pool) *integration.Service {
	return integration.NewService(pool, SealingKey(), Publishing(nil).Sender, "http://localhost:3000")
}

func Publishing(to publication.Sender) publication.Publishing {
	if to == nil {
		to = ClosedSender{}
	}
	return publication.Publishing{
		Sealing: SealingKey(),
		Sender:  to,
		Site:    "http://localhost:3000",
		Blog:    "http://blog.localhost:3000",
	}
}

func SealingKey() secrets.Key {
	key, err := secrets.NewKey(bytes.Repeat([]byte{3}, secrets.KeyBytes))
	if err != nil {
		panic(err)
	}
	return key
}

type ClosedSender struct{}

func (ClosedSender) Check(address string) (string, error) {
	return dispatch.NewCaller(dispatch.DefaultLimits()).Check(address)
}

func (ClosedSender) Get(context.Context, string) (dispatch.Answer, error) {
	return dispatch.Answer{}, errors.New("this test stack sends nowhere")
}

func (ClosedSender) Post(
	context.Context, string, map[string]string, []byte,
) (dispatch.Answer, error) {
	return dispatch.Answer{}, errors.New("this test stack sends nowhere")
}

func NewLinkingService(pool *pgxpool.Pool) *connect.Apps {
	return connect.NewApps(pool, "http://localhost:3000", []byte("01234567890123456789012345678901"))
}

func NewDeliveryService(
	pool *pgxpool.Pool,
	assets *asset.Service,
	links *connect.Apps,
) *connect.Sends {
	return connect.NewSends(pool, assets, links, DeliverySettings())
}

func DeliverySettings() connect.Settings {
	settings := connect.DefaultSettings()
	settings.HoldFloor = 50 * time.Millisecond
	settings.HoldCeiling = 80 * time.Millisecond
	settings.Recheck = 20 * time.Millisecond
	return settings
}

type OpaqueModule struct{}

func (OpaqueModule) ID() string { return "test_opaque" }

func (OpaqueModule) Declaration() format.Declaration {
	declaration := ReaderDeclaration("test_opaque", "character")
	declaration.Label = "Test format"
	declaration.Direction.Write = true
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

func (OpaqueModule) Write(_ context.Context, written format.ExportAsset) (format.Artifact, error) {
	return format.Artifact{
		Body:      []byte(written.Text(block.RoleDescription)),
		MediaType: "text/plain", Extension: ".txt",
	}, nil
}

func (OpaqueModule) Claim(file format.Inspection) (format.Claim, bool) {
	return format.WholeFileCompatibilityClaim(file), true
}

func (OpaqueModule) Parse(context.Context, format.Inspection, format.Claim) (format.Parsed, error) {
	return format.Parsed{
		Kind: "character", Format: "test_opaque",
		Elements: []block.Element{
			{Type: block.TypeProse, Role: block.RoleDescription, Content: block.Prose{Text: "Test description"}},
			{Type: block.TypeTextSet, Role: block.RoleGreetings, Content: block.TextSet{Texts: []block.TextItem{{ID: block.NewItemID(), Text: "Hello"}}}},
		},
	}, nil
}

func ReaderDeclaration(id, kind string) format.Declaration {
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
