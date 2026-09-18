package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

// FetchWork reads a work page, as its owner when a session is given
func FetchWork(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	workID string,
) StartedWork {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/works/"+workID, nil)
	if session != nil {
		request = Authorized(request, session)
	}
	response := Send(t, r, request)
	if response.Code != http.StatusOK {
		t.Fatalf("read the asset: %d %s", response.Code, response.Body.String())
	}
	var page StartedWork
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
	if got := SaveDetails(
		t, router, session, started.ID, `{"name":"`+name+`","blurb":"","isNsfw":false}`,
	); got.Code != http.StatusNoContent {
		t.Fatalf("save details status = %d, want 204: %s", got.Code, got.Body.String())
	}
	if got := PublishWork(t, router, session, started.ID); got.Code != http.StatusOK {
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
) StartedWork {
	t.Helper()
	started := StartPreset(t, router, session, "lumiverse")
	core := EditableBlock(BlockNamed(t, started.Blocks, "preset_core"))
	core.Elements[0].Content = SealedPresetPrompts(publicID, sealedID, publicText, sealedText)
	core.AllowedApps = &[]string{"lumiverse"}
	if got := SaveBlock(t, router, session, started.ID, started.Blocks[0].ID, core); got.Code != http.StatusOK {
		t.Fatalf("save the sealed prompts: %d %s", got.Code, got.Body.String())
	}
	if got := SaveDetails(t, router, session, started.ID,
		`{"name":"Sealed preset","blurb":"","isNsfw":false}`); got.Code != http.StatusNoContent {
		t.Fatalf("save the details: %d %s", got.Code, got.Body.String())
	}
	if got := PublishWork(t, router, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish the preset: %d %s", got.Code, got.Body.String())
	}
	return started
}

// AcceptReplacementPreview accepts the staged replacement at location, removing whatever the file cannot hold
func AcceptReplacementPreview(t *testing.T, r *gin.Engine, session *http.Cookie, workID, location string, exposeProtected ...bool) {
	t.Helper()
	preview := Send(t, r, Authorized(httptest.NewRequest(http.MethodGet, location, nil), session))
	if preview.Code != http.StatusOK {
		t.Fatalf("read replacement preview = %d: %s", preview.Code, preview.Body.String())
	}
	var operation struct {
		Status  string `json:"status"`
		Preview *struct {
			Unrepresentable []string `json:"unrepresentable"`
		} `json:"preview"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &operation); err != nil {
		t.Fatal(err)
	}
	if operation.Status != "preview" || operation.Preview == nil {
		t.Fatalf("replacement preview = %+v", operation)
	}
	decisions := make(map[string]string, len(operation.Preview.Unrepresentable))
	for _, role := range operation.Preview.Unrepresentable {
		decisions[role] = "remove"
	}
	body, err := json.Marshal(map[string]any{
		"unrepresentable": decisions,
		"exposeProtected": len(exposeProtected) > 0 && exposeProtected[0],
	})
	if err != nil {
		t.Fatal(err)
	}
	operationID := strings.TrimPrefix(location, "/v1/ingests/")
	request := httptest.NewRequest(http.MethodPost, "/v1/works/"+workID+"/revisions/"+operationID+"/accept", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	accepted := Send(t, r, Authorized(request, session))
	if accepted.Code != http.StatusOK {
		t.Fatalf("accept replacement preview = %d: %s", accepted.Code, accepted.Body.String())
	}
}

// SealEveryFragment marks every prompt in a prompt list private to the given apps
func SealEveryFragment(t *testing.T, body SaveBlockBody, apps []string) SaveBlockBody {
	t.Helper()
	var list struct {
		Groups    []json.RawMessage            `json:"groups"`
		Fragments []map[string]json.RawMessage `json:"fragments"`
	}
	if err := json.Unmarshal(body.Elements[0].Content, &list); err != nil {
		t.Fatalf("read the prompt list to seal: %v", err)
	}
	for index := range list.Fragments {
		list.Fragments[index]["protected"] = json.RawMessage("true")
	}
	sealed, err := json.Marshal(list)
	if err != nil {
		t.Fatalf("write the sealed prompt list: %v", err)
	}
	body.Elements[0].Content = sealed
	body.AllowedApps = &apps
	return body
}

// NeverClaimsModule is a format that recognises no file
type NeverClaimsModule struct{}

func (NeverClaimsModule) ID() string { return "never" }
func (NeverClaimsModule) Declaration() format.Declaration {
	return ReaderDeclaration("never", "character")
}
func (NeverClaimsModule) Claim(format.Inspection) (format.Claim, bool) { return format.Claim{}, false }
func (NeverClaimsModule) Parse(context.Context, format.Inspection, format.Claim) (format.Parsed, error) {
	return format.Parsed{}, errors.New("unreachable")
}

// KeyedSealedPreset is a Lumiverse preset whose one prompt is sealed by its key
const KeyedSealedPreset = `{
	"schemaVersion": 1,
	"name": "Keyed sealed preset",
	"blocks": [
		{"id":"public","name":"Public","role":"system","content":"Visible prompt.","enabled":true},
		{"id":"private","name":"Private","role":"system","content":"Exact private prompt.","enabled":true,"sealed":true,"sealedKey":"dialogue.frame"}
	]
}`
