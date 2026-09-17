package upload_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

type promptListResponse struct {
	Fragments []struct {
		Name      string `json:"name"`
		Text      string `json:"text"`
		Protected bool   `json:"protected"`
	} `json:"fragments"`
}

func promptListFromPage(t *testing.T, page apitest.StartedAsset) promptListResponse {
	t.Helper()
	core := apitest.BlockNamed(t, page.Blocks, "preset_core")
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
	t.Parallel()
	router, session, assets, _ := harness.NewVerifiedIngestRouterWithPool(t, apitest.LumiverseRegistry(t))
	metadata := apitest.ExampleMetadata("Keyed sealed preset")
	metadata["filename"] = "keyed.json"
	finished := apitest.UploadAndFinish(t, router, session, assets, metadata, []byte(apitest.KeyedSealedPreset))
	assetID := apitest.AssetIDFromIngest(t, finished)

	owner := apitest.FetchStartedAsset(t, router, session, assetID)
	if !owner.LinkedInstallOnly || len(owner.AllowedApps) != 1 || owner.AllowedApps[0] != "lumiverse" {
		t.Fatalf("owner policy = linked install only %t, apps %v", owner.LinkedInstallOnly, owner.AllowedApps)
	}
	ownerPrompts := promptListFromPage(t, owner).Fragments
	if len(ownerPrompts) != 2 || ownerPrompts[1].Text != "Exact private prompt." ||
		!ownerPrompts[1].Protected {
		t.Fatalf("owner prompts = %+v", ownerPrompts)
	}

	readerResponse := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/assets/"+assetID, nil))
	if readerResponse.Code != http.StatusOK {
		t.Fatalf("reader page = %d: %s", readerResponse.Code, readerResponse.Body.String())
	}
	if strings.Contains(readerResponse.Body.String(), "Exact private prompt.") {
		t.Fatal("the reader response contains the protected text")
	}
	var reader apitest.StartedAsset
	if err := json.Unmarshal(readerResponse.Body.Bytes(), &reader); err != nil {
		t.Fatalf("decode reader page: %v", err)
	}
	readerPrompts := promptListFromPage(t, reader).Fragments
	if len(readerPrompts) != 2 || readerPrompts[1].Text != "" || !readerPrompts[1].Protected {
		t.Fatalf("reader prompts = %+v", readerPrompts)
	}
}

func TestAKeyedPlaceholderRevisionKeepsTheExistingPrivateText(t *testing.T) {
	t.Parallel()
	router, session, assets, _ := harness.NewVerifiedIngestRouterWithPool(t, apitest.LumiverseRegistry(t))
	metadata := apitest.ExampleMetadata("Keyed sealed preset")
	metadata["filename"] = "keyed.json"
	created := apitest.UploadAndFinish(t, router, session, assets, metadata, []byte(apitest.KeyedSealedPreset))
	assetID := apitest.AssetIDFromIngest(t, created)

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
	accepted := apitest.Send(t, router, apitest.Authorized(
		apitest.RevisionRequest(t, assetID, "keyed-revision.json", placeholder), session,
	))
	if accepted.Code != http.StatusAccepted {
		t.Fatalf("revision upload = %d: %s", accepted.Code, accepted.Body.String())
	}
	if processed, err := apitest.Uploads(assets).ProcessNextIngest(t.Context()); err != nil || !processed {
		t.Fatalf("process revision = %t, %v; want true, nil", processed, err)
	}
	apitest.AcceptReplacementPreview(t, router, session, assetID, accepted.Header().Get("Location"))
	apitest.PollIngestAsset(t, router, session, accepted.Header().Get("Location"))

	owner := apitest.FetchStartedAsset(t, router, session, assetID)
	prompts := promptListFromPage(t, owner).Fragments
	if len(prompts) != 1 || prompts[0].Name != "Private renamed" ||
		prompts[0].Text != "Exact private prompt." || !prompts[0].Protected {
		t.Fatalf("owner prompts after placeholder revision = %+v", prompts)
	}
}

func TestReplacementNeedsConfirmationBeforeRemovingPromptProtection(t *testing.T) {
	t.Parallel()
	for _, sealedAfterPublication := range []bool{false, true} {
		name := "sealed on upload"
		if sealedAfterPublication {
			name = "sealed after publication"
		}
		t.Run(name, func(t *testing.T) {
			router, session, assets, _ := harness.NewVerifiedIngestRouterWithPool(t, apitest.LumiverseRegistry(t))
			ordinary := strings.ReplaceAll(apitest.KeyedSealedPreset, `,"sealed":true,"sealedKey":"dialogue.frame"`, "")
			initial := apitest.KeyedSealedPreset
			if sealedAfterPublication {
				initial = ordinary
			}
			metadata := apitest.ExampleMetadata("Replacement protection")
			metadata["filename"] = "keyed.json"
			created := apitest.UploadAndFinish(t, router, session, assets, metadata, []byte(initial))
			assetID := apitest.AssetIDFromIngest(t, created)
			if sealedAfterPublication {
				page := apitest.FetchStartedAsset(t, router, session, assetID)
				core := apitest.BlockNamed(t, page.Blocks, "preset_core")
				body := apitest.SealEveryFragment(t, apitest.EditableBlock(core), []string{"lumiverse"})
				if response := apitest.SaveBlock(t, router, session, assetID, core.ID, body); response.Code != http.StatusOK {
					t.Fatalf("seal published text: %d %s", response.Code, response.Body.String())
				}
			}
			before := apitest.FetchStartedAsset(t, router, session, assetID)
			replacement := strings.ReplaceAll(ordinary, "Keyed sealed preset", "Replacement preset")
			staged := apitest.Send(t, router, apitest.Authorized(apitest.RevisionRequest(t, assetID, "replacement.json", []byte(replacement)), session))
			if staged.Code != http.StatusAccepted {
				t.Fatalf("stage replacement: %d %s", staged.Code, staged.Body.String())
			}
			if processed, err := apitest.Uploads(assets).ProcessNextIngest(t.Context()); err != nil || !processed {
				t.Fatalf("process replacement: %t %v", processed, err)
			}
			operationID := strings.TrimPrefix(staged.Header().Get("Location"), "/v1/ingests/")
			path := "/v1/assets/" + assetID + "/revisions/" + operationID + "/accept"
			request := apitest.AuthorizedJSONRequest(t, http.MethodPost, path, `{"unrepresentable":{}}`, session)
			apitest.WithReviewedVersion(t, router, request)
			refused := apitest.Send(t, router, request)
			if refused.Code != http.StatusConflict || !strings.Contains(refused.Body.String(), `"code":"sealed_exposure"`) || !strings.Contains(refused.Body.String(), "Private") {
				t.Fatalf("unconfirmed replacement: %d %s", refused.Code, refused.Body.String())
			}
			after := apitest.FetchStartedAsset(t, router, session, assetID)
			if !after.LinkedInstallOnly || !reflect.DeepEqual(after.Blocks, before.Blocks) {
				t.Fatal("refused replacement changed the working copy or its protection")
			}
			reader := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/assets/"+assetID, nil))
			if reader.Code != http.StatusOK || strings.Contains(reader.Body.String(), "Exact private prompt.") {
				t.Fatalf("reader after refusal: %d %s", reader.Code, reader.Body.String())
			}
			confirmation := apitest.AuthorizedJSONRequest(t, http.MethodPost, path, `{"unrepresentable":{},"exposeProtected":true}`, session)
			confirmation.Header.Set("X-Working-Copy-Version", request.Header.Get("X-Working-Copy-Version"))
			confirmed := apitest.Send(t, router, confirmation)
			if confirmed.Code != http.StatusOK {
				t.Fatalf("confirmed replacement: %d %s", confirmed.Code, confirmed.Body.String())
			}
			if apitest.FetchStartedAsset(t, router, session, assetID).LinkedInstallOnly {
				t.Fatal("confirmed replacement kept the old protection")
			}
			if sealedAfterPublication {
				reader = apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/assets/"+assetID, nil))
				if !strings.Contains(reader.Body.String(), "Exact private prompt.") {
					t.Fatal("confirmed removal did not restore access to previously public text")
				}
			}
		})
	}
}

func TestANewKeyedPlaceholderAndDuplicateKeysAreMalformedInputs(t *testing.T) {
	t.Parallel()
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
			router, session, assets, _ := harness.NewVerifiedIngestRouterWithPool(t, apitest.LumiverseRegistry(t))
			metadata := apitest.ExampleMetadata("Refused preset")
			metadata["filename"] = "refused.json"
			accepted := apitest.Send(t, router, apitest.Authorized(
				apitest.UploadRequest(t, metadata, []byte(test.file)), session,
			))
			if accepted.Code != http.StatusAccepted {
				t.Fatalf("upload = %d: %s", accepted.Code, accepted.Body.String())
			}
			if processed, err := apitest.Uploads(assets).ProcessNextIngest(t.Context()); err != nil || !processed {
				t.Fatalf("process ingest = %t, %v; want true, nil", processed, err)
			}
			poll := apitest.Send(t, router, apitest.Authorized(httptest.NewRequest(
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
	t.Parallel()
	router, session, assets, _ := harness.NewVerifiedIngestRouterWithPool(t, apitest.LumiverseRegistry(t))
	metadata := apitest.ExampleMetadata("Ordinary preset")
	metadata["filename"] = "ordinary.json"
	finished := apitest.UploadAndFinish(t, router, session, assets, metadata, []byte(`{
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
	assetID := apitest.AssetIDFromIngest(t, finished)

	readerResponse := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/assets/"+assetID, nil))
	if readerResponse.Code != http.StatusOK {
		t.Fatalf("reader page = %d: %s", readerResponse.Code, readerResponse.Body.String())
	}
	var reader apitest.StartedAsset
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
