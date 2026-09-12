package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/preset"
)

const keyedSealedPreset = `{
	"schemaVersion": 1,
	"name": "Keyed sealed preset",
	"blocks": [
		{"id":"public","name":"Public","role":"system","content":"Visible prompt.","enabled":true},
		{"id":"private","name":"Private","role":"system","content":"Exact private prompt.","enabled":true,"sealed":true,"sealedKey":"dialogue.frame"}
	]
}`

type promptListResponse struct {
	Fragments []struct {
		Name      string `json:"name"`
		Text      string `json:"text"`
		Protected bool   `json:"protected"`
	} `json:"fragments"`
}

func lumiverseIngestRegistry(t *testing.T) *format.Registry {
	t.Helper()
	registry := format.NewRegistry()
	if err := registry.Register(preset.LumiverseModule{}); err != nil {
		t.Fatalf("register Lumiverse preset format: %v", err)
	}
	return registry
}

func promptListFromPage(t *testing.T, page startedAsset) promptListResponse {
	t.Helper()
	core := blockNamed(t, page.Blocks, "preset_core")
	if len(core.Elements) != 1 {
		t.Fatalf("preset core elements = %d, want one prompt list", len(core.Elements))
	}
	var list promptListResponse
	if err := json.Unmarshal(core.Elements[0].Content, &list); err != nil {
		t.Fatalf("decode prompt list: %v", err)
	}
	return list
}

func TestAKeyedSealedUploadStoresAnOwnerPromptAndARedactedReaderStub(t *testing.T) {
	router, session, assets, _ := newVerifiedIngestRouterWithPool(t, lumiverseIngestRegistry(t))
	metadata := exampleMetadata("Keyed sealed preset")
	metadata["filename"] = "keyed.json"
	finished := uploadAndFinish(t, router, session, assets, metadata, []byte(keyedSealedPreset))
	assetID := assetIDFromIngest(t, finished)

	owner := fetchStartedAsset(t, router, session, assetID)
	if !owner.LinkedInstallOnly || len(owner.AllowedApps) != 1 || owner.AllowedApps[0] != "lumiverse" {
		t.Fatalf("owner policy = linked install only %t, apps %v", owner.LinkedInstallOnly, owner.AllowedApps)
	}
	ownerPrompts := promptListFromPage(t, owner).Fragments
	if len(ownerPrompts) != 2 || ownerPrompts[1].Text != "Exact private prompt." ||
		!ownerPrompts[1].Protected {
		t.Fatalf("owner prompts = %+v", ownerPrompts)
	}

	readerResponse := send(t, router, httptest.NewRequest(http.MethodGet, "/v1/assets/"+assetID, nil))
	if readerResponse.Code != http.StatusOK {
		t.Fatalf("reader page = %d: %s", readerResponse.Code, readerResponse.Body.String())
	}
	if strings.Contains(readerResponse.Body.String(), "Exact private prompt.") {
		t.Fatal("the reader response contains the protected text")
	}
	var reader startedAsset
	if err := json.Unmarshal(readerResponse.Body.Bytes(), &reader); err != nil {
		t.Fatalf("decode reader page: %v", err)
	}
	readerPrompts := promptListFromPage(t, reader).Fragments
	if len(readerPrompts) != 2 || readerPrompts[1].Text != "" || !readerPrompts[1].Protected {
		t.Fatalf("reader prompts = %+v", readerPrompts)
	}
}

func TestAKeyedPlaceholderRevisionKeepsTheExistingPrivateText(t *testing.T) {
	router, session, assets, _ := newVerifiedIngestRouterWithPool(t, lumiverseIngestRegistry(t))
	metadata := exampleMetadata("Keyed sealed preset")
	metadata["filename"] = "keyed.json"
	created := uploadAndFinish(t, router, session, assets, metadata, []byte(keyedSealedPreset))
	assetID := assetIDFromIngest(t, created)

	placeholder := []byte(`{
		"schemaVersion": 1,
		"name": "Keyed sealed preset revision",
		"blocks": [{
			"id":"private-revision",
			"name":"Private renamed",
			"role":"system",
			"content":"{{presetBlock::dialogue.frame}}",
			"enabled":true,
			"sealed":true,
			"sealedKey":"dialogue.frame"
		}]
	}`)
	accepted := send(t, router, authorized(
		revisionRequest(t, assetID, "keyed-revision.json", placeholder), session,
	))
	if accepted.Code != http.StatusAccepted {
		t.Fatalf("revision upload = %d: %s", accepted.Code, accepted.Body.String())
	}
	if processed, err := assets.ProcessNextIngest(t.Context()); err != nil || !processed {
		t.Fatalf("process revision = %t, %v; want true, nil", processed, err)
	}
	acceptReplacementPreview(t, router, session, assetID, accepted.Header().Get("Location"))
	pollIngestAsset(t, router, session, accepted.Header().Get("Location"))

	owner := fetchStartedAsset(t, router, session, assetID)
	prompts := promptListFromPage(t, owner).Fragments
	if len(prompts) != 1 || prompts[0].Name != "Private renamed" ||
		prompts[0].Text != "Exact private prompt." || !prompts[0].Protected {
		t.Fatalf("owner prompts after placeholder revision = %+v", prompts)
	}
}

func TestReplacementNeedsConfirmationBeforeRemovingPromptProtection(t *testing.T) {
	for _, sealedAfterPublication := range []bool{false, true} {
		name := "sealed on upload"
		if sealedAfterPublication {
			name = "sealed after publication"
		}
		t.Run(name, func(t *testing.T) {
			router, session, assets, _ := newVerifiedIngestRouterWithPool(t, lumiverseIngestRegistry(t))
			ordinary := strings.ReplaceAll(keyedSealedPreset, `,"sealed":true,"sealedKey":"dialogue.frame"`, "")
			initial := keyedSealedPreset
			if sealedAfterPublication {
				initial = ordinary
			}
			metadata := exampleMetadata("Replacement protection")
			metadata["filename"] = "keyed.json"
			created := uploadAndFinish(t, router, session, assets, metadata, []byte(initial))
			assetID := assetIDFromIngest(t, created)
			if sealedAfterPublication {
				page := fetchStartedAsset(t, router, session, assetID)
				core := blockNamed(t, page.Blocks, "preset_core")
				body := sealEveryFragment(t, editableBlock(core), []string{"lumiverse"})
				if response := saveBlock(t, router, session, assetID, core.ID, body); response.Code != http.StatusOK {
					t.Fatalf("seal published text: %d %s", response.Code, response.Body.String())
				}
			}
			before := fetchStartedAsset(t, router, session, assetID)
			replacement := strings.ReplaceAll(ordinary, "Keyed sealed preset", "Replacement preset")
			staged := send(t, router, authorized(revisionRequest(t, assetID, "replacement.json", []byte(replacement)), session))
			if staged.Code != http.StatusAccepted {
				t.Fatalf("stage replacement: %d %s", staged.Code, staged.Body.String())
			}
			if processed, err := assets.ProcessNextIngest(t.Context()); err != nil || !processed {
				t.Fatalf("process replacement: %t %v", processed, err)
			}
			operationID := strings.TrimPrefix(staged.Header().Get("Location"), "/v1/ingests/")
			path := "/v1/assets/" + assetID + "/revisions/" + operationID + "/accept"
			request := authorizedJSONRequest(t, http.MethodPost, path, `{"unrepresentable":{}}`, session)
			withReviewedVersion(t, router, request)
			refused := send(t, router, request)
			if refused.Code != http.StatusConflict || !strings.Contains(refused.Body.String(), `"code":"sealed_exposure"`) || !strings.Contains(refused.Body.String(), "Private") {
				t.Fatalf("unconfirmed replacement: %d %s", refused.Code, refused.Body.String())
			}
			after := fetchStartedAsset(t, router, session, assetID)
			if !after.LinkedInstallOnly || !reflect.DeepEqual(after.Blocks, before.Blocks) {
				t.Fatal("refused replacement changed the working copy or its protection")
			}
			reader := send(t, router, httptest.NewRequest(http.MethodGet, "/v1/assets/"+assetID, nil))
			if reader.Code != http.StatusOK || strings.Contains(reader.Body.String(), "Exact private prompt.") {
				t.Fatalf("reader after refusal: %d %s", reader.Code, reader.Body.String())
			}
			confirmation := authorizedJSONRequest(t, http.MethodPost, path, `{"unrepresentable":{},"exposeProtected":true}`, session)
			confirmation.Header.Set("X-Working-Copy-Version", request.Header.Get("X-Working-Copy-Version"))
			confirmed := send(t, router, confirmation)
			if confirmed.Code != http.StatusOK {
				t.Fatalf("confirmed replacement: %d %s", confirmed.Code, confirmed.Body.String())
			}
			if fetchStartedAsset(t, router, session, assetID).LinkedInstallOnly {
				t.Fatal("confirmed replacement kept the old protection")
			}
			if sealedAfterPublication {
				reader = send(t, router, httptest.NewRequest(http.MethodGet, "/v1/assets/"+assetID, nil))
				if !strings.Contains(reader.Body.String(), "Exact private prompt.") {
					t.Fatal("confirmed removal did not restore access to previously public text")
				}
			}
		})
	}
}

func TestANewKeyedPlaceholderAndDuplicateKeysAreMalformedInputs(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		wantReason string
	}{
		{
			name: "new placeholder",
			file: `{
				"schemaVersion":1,
				"blocks":[{
					"id":"private",
					"content":"{{presetBlock::unknown}}",
					"enabled":true,
					"sealed":true,
					"sealedKey":"unknown"
				}]
			}`,
			wantReason: "malformed_input",
		},
		{
			name: "duplicate key",
			file: `{
				"schemaVersion":1,
				"blocks":[
					{"id":"one","content":"First","enabled":true,"sealed":true,"sealedKey":"same"},
					{"id":"two","content":"Second","enabled":true,"sealed":true,"sealedKey":"same"}
				]
			}`,
			wantReason: "malformed_input",
		},
		{
			name:       "unsupported format",
			file:       `{"not":"a preset"}`,
			wantReason: "unsupported_format",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router, session, assets, _ := newVerifiedIngestRouterWithPool(t, lumiverseIngestRegistry(t))
			metadata := exampleMetadata("Refused preset")
			metadata["filename"] = "refused.json"
			accepted := send(t, router, authorized(
				uploadRequest(t, metadata, []byte(test.file)), session,
			))
			if accepted.Code != http.StatusAccepted {
				t.Fatalf("upload = %d: %s", accepted.Code, accepted.Body.String())
			}
			if processed, err := assets.ProcessNextIngest(t.Context()); err != nil || !processed {
				t.Fatalf("process ingest = %t, %v; want true, nil", processed, err)
			}
			poll := send(t, router, authorized(httptest.NewRequest(
				http.MethodGet, accepted.Header().Get("Location"), nil,
			), session))
			var operation struct {
				Status  string `json:"status"`
				Asset   any    `json:"asset"`
				Failure *struct {
					Reason string `json:"reason"`
				} `json:"failure"`
			}
			if err := json.Unmarshal(poll.Body.Bytes(), &operation); err != nil {
				t.Fatalf("decode failed ingest: %v", err)
			}
			if operation.Status != "failed" || operation.Asset != nil || operation.Failure == nil ||
				operation.Failure.Reason != test.wantReason {
				t.Fatalf("operation = %#v, want an unsaved %s failure", operation, test.wantReason)
			}
		})
	}
}

func TestAnOrdinaryLumiversePresetStillIngestsAsPublicContent(t *testing.T) {
	router, session, assets, _ := newVerifiedIngestRouterWithPool(t, lumiverseIngestRegistry(t))
	metadata := exampleMetadata("Ordinary preset")
	metadata["filename"] = "ordinary.json"
	finished := uploadAndFinish(t, router, session, assets, metadata, []byte(`{
		"schemaVersion":1,
		"name":"Ordinary preset",
		"blocks":[
			{"id":"group","name":"Core","marker":"category","sealed":false},
			{
				"id":"public",
				"name":"Public",
				"role":"system",
				"content":"Ordinary public prompt.",
				"enabled":true,
				"sealed":false
			}
		]
	}`))
	assetID := assetIDFromIngest(t, finished)

	readerResponse := send(t, router, httptest.NewRequest(http.MethodGet, "/v1/assets/"+assetID, nil))
	if readerResponse.Code != http.StatusOK {
		t.Fatalf("reader page = %d: %s", readerResponse.Code, readerResponse.Body.String())
	}
	var reader startedAsset
	if err := json.Unmarshal(readerResponse.Body.Bytes(), &reader); err != nil {
		t.Fatalf("decode reader page: %v", err)
	}
	prompts := promptListFromPage(t, reader).Fragments
	if reader.LinkedInstallOnly || len(reader.AllowedApps) != 0 ||
		len(prompts) != 1 || prompts[0].Protected || prompts[0].Text != "Ordinary public prompt." {
		t.Fatalf("ordinary preset = linked install only %t, apps %v, prompts %+v",
			reader.LinkedInstallOnly, reader.AllowedApps, prompts)
	}
	if len(reader.Downloads) != 1 || reader.Downloads[0].Format != "preset_lumiverse" {
		t.Fatalf("ordinary downloads = %+v, want the Lumiverse target", reader.Downloads)
	}
}
