package apitest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/preset"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RecordedVersionBody is the id and number a response names a recorded version by
type RecordedVersionBody struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
}

// FetchAsset reads a work page, as its owner when a session is given
func FetchAsset(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	assetID string,
) StartedAsset {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/assets/"+assetID, nil)
	if session != nil {
		request = Authorized(request, session)
	}
	response := Send(t, r, request)
	if response.Code != http.StatusOK {
		t.Fatalf("read the asset: %d %s", response.Code, response.Body.String())
	}
	var page StartedAsset
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode the asset: %v", err)
	}
	return page
}

// LumiverseRegistry reads Lumiverse presets and nothing else
func LumiverseRegistry(t *testing.T) *format.Registry {
	t.Helper()
	registry := format.NewRegistry()
	if err := registry.Register(preset.LumiverseModule{}); err != nil {
		t.Fatalf("register Lumiverse preset format: %v", err)
	}
	return registry
}

// PublishSealedPreset publishes a Lumiverse preset whose one prompt is sealed
func PublishSealedPreset(
	t *testing.T,
	router *gin.Engine,
	session *http.Cookie,
	name string,
	privateText string,
) string {
	t.Helper()
	started := StartPreset(t, router, session, "lumiverse")
	core := EditableBlock(BlockNamed(t, started.Blocks, "preset_core"))
	core.Elements[0].Content = json.RawMessage(
		`{"groups":[],"fragments":[{"name":"Private instructions","role":"system","text":"` +
			privateText + `","protected":true,"enabled":true}]}`)
	core.AllowedApps = &[]string{"lumiverse"}
	if got := SaveBlock(t, router, session, started.ID, started.Blocks[0].ID, core); got.Code != http.StatusOK {
		t.Fatalf("save sealed prompt status = %d, want 200: %s", got.Code, got.Body.String())
	}
	if got := SaveIdentity(
		t, router, session, started.ID, `{"name":"`+name+`","blurb":"","isNsfw":false}`,
	); got.Code != http.StatusNoContent {
		t.Fatalf("save identity status = %d, want 204: %s", got.Code, got.Body.String())
	}
	if got := PublishAsset(t, router, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200: %s", got.Code, got.Body.String())
	}
	return started.ID
}

// SealedPresetPrompts is a prompt list with one public and one sealed prompt
func SealedPresetPrompts(publicID, sealedID uuid.UUID, publicText, sealedText string) json.RawMessage {
	return json.RawMessage(`{"groups":[],"fragments":[` +
		`{"id":"` + publicID.String() + `","name":"House rule","role":"system","text":"` +
		publicText + `","enabled":true},` +
		`{"id":"` + sealedID.String() + `","name":"Private instructions","role":"system","text":"` +
		sealedText + `","protected":true,"enabled":true}]}`)
}

// PublishTwoPromptPreset publishes a Lumiverse preset with one public and one sealed prompt
func PublishTwoPromptPreset(
	t *testing.T,
	router *gin.Engine,
	session *http.Cookie,
	publicID, sealedID uuid.UUID,
	publicText, sealedText string,
) StartedAsset {
	t.Helper()
	started := StartPreset(t, router, session, "lumiverse")
	core := EditableBlock(BlockNamed(t, started.Blocks, "preset_core"))
	core.Elements[0].Content = SealedPresetPrompts(publicID, sealedID, publicText, sealedText)
	core.AllowedApps = &[]string{"lumiverse"}
	if got := SaveBlock(t, router, session, started.ID, started.Blocks[0].ID, core); got.Code != http.StatusOK {
		t.Fatalf("save the sealed prompts: %d %s", got.Code, got.Body.String())
	}
	if got := SaveIdentity(t, router, session, started.ID,
		`{"name":"Sealed preset","blurb":"","isNsfw":false}`); got.Code != http.StatusNoContent {
		t.Fatalf("save the identity: %d %s", got.Code, got.Body.String())
	}
	if got := PublishAsset(t, router, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish the preset: %d %s", got.Code, got.Body.String())
	}
	return started
}
