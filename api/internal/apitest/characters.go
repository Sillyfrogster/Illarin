package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/character"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const PlainCard = `{
	"spec":"chara_card_v3","spec_version":"3.0",
	"data":{"name":"Ana","description":"Keeps the archive.","first_mes":"Hello"}
}`

const CardWithThirdPartyNamespaces = `{
	"spec":"chara_card_v3","spec_version":"3.0",
	"data":{
		"name":"Ana","description":"Keeps the archive.","personality":"Patient",
		"scenario":"After closing","first_mes":"Welcome back.",
		"tags":["archivist"],
		"character_book":{"name":"Ana's world","scan_depth":4,
			"entries":[
				{"keys":["ledger"],"content":"A debt.","uid":91,"probability":75},
				{"keys":["mira"],"content":"A name.","uid":92,"group":"people"}]},
		"extensions":{
			"chub":{"full_path":"ana/quiet","related_lorebooks":[]},
			"tavern_helper":{"scripts":[{"name":"Opening"}]},
			"depth_prompt":{"depth":4,"prompt":"","role":"system"},
			"talkativeness":"0.5","fav":false,"world":""
		}
	}
}`

func GiveExpressions(t *testing.T, r http.Handler, session *http.Cookie, assetID string) {
	t.Helper()
	GivePictures(t, r, session, assetID, "expression", "expressions")
}

func GivePictures(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	assetID, mediaRole, definition string,
) {
	t.Helper()
	mediaID := UploadedImageID(t, r, session, assetID, mediaRole, PNG(t, 64, 64))
	block := AddedBlock(t, AddBlock(t, r, session, assetID, definition, "image_set"))
	body := EditableBlock(block)
	body.Elements[0].Content = json.RawMessage(
		`{"images":[{"mediaId":"` + mediaID + `","name":"happy"}]}`,
	)
	if saved := SaveBlock(t, r, session, assetID, block.ID, body); saved.Code != http.StatusOK {
		t.Fatalf("save the %s block: %d %s", definition, saved.Code, saved.Body.String())
	}
}

func PublishCharacter(t *testing.T, r http.Handler, session *http.Cookie, assetID string) {
	t.Helper()
	if got := SaveIdentity(t, r, session, assetID,
		`{"name":"Ana","blurb":"","isNsfw":false}`); got.Code != http.StatusNoContent {
		t.Fatalf("save identity: %d %s", got.Code, got.Body.String())
	}
	if got := PublishAsset(t, r, session, assetID); got.Code != http.StatusOK {
		t.Fatalf("publish: %d %s", got.Code, got.Body.String())
	}
}

func (h Harness) NewCharacterIngestRouter(t *testing.T) (*gin.Engine, *http.Cookie, *work.Service) {
	t.Helper()
	router, session, assets, _ := h.NewCharacterIngestRouterWithPool(t)
	return router, session, assets
}

func (h Harness) NewCharacterIngestRouterWithPool(
	t *testing.T,
) (*gin.Engine, *http.Cookie, *work.Service, *pgxpool.Pool) {
	t.Helper()
	registry := format.NewRegistry()
	for _, module := range character.Modules() {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %s: %v", module.ID(), err)
		}
	}
	return h.NewVerifiedIngestRouterWithPool(t, registry)
}

func ProjectionComputedAt(t *testing.T, pool *pgxpool.Pool, assetID string) time.Time {
	t.Helper()
	var computedAt time.Time
	if err := pool.QueryRow(context.Background(), `
		select export_computed_at from asset_projections where asset_id = $1
	`, assetID).Scan(&computedAt); err != nil {
		t.Fatalf("read the export projection: %v", err)
	}
	return computedAt
}

func ContainsBytes(haystack, needle []byte) bool {
	return bytes.Contains(haystack, needle)
}

func UploadedCharacterID(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	assets *work.Service,
	card string,
) string {
	t.Helper()
	metadata := ExampleMetadata("Ana")
	metadata["filename"] = "ana.json"
	metadata["_keepDraft"] = true
	return AssetIDFromIngest(t, UploadAndFinish(t, r, session, assets, metadata, []byte(card)))
}

func NamespacesOf(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	found := make(map[string]json.RawMessage)
	if len(raw) == 0 {
		return found
	}
	if err := json.Unmarshal(raw, &found); err != nil {
		t.Fatalf("read %s as an object: %v", raw, err)
	}
	return found
}

func CardBodyOf(t *testing.T, card []byte) map[string]json.RawMessage {
	t.Helper()
	var read struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(card, &read); err != nil {
		t.Fatalf("read a card: %v", err)
	}
	return read.Data
}
