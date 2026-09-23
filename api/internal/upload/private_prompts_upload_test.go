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
		Name    string `json:"name"`
		Text    string `json:"text"`
		Private bool   `json:"private"`
	} `json:"fragments"`
}

func promptListFromPage(t *testing.T, page apitest.StartedWork) promptListResponse {
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

func TestAKeyedPrivateUploadStoresAnOwnerPromptAndARedactedReaderStub(t *testing.T) {
	t.Parallel()
	router, session, works, _ := harness.NewVerifiedUploadRouterWithPool(t, apitest.LumiverseRegistry(t))
	metadata := apitest.ExampleMetadata("Keyed private prompt preset")
	metadata["filename"] = "keyed.json"
	finished := apitest.UploadAndFinish(t, router, session, works, metadata, []byte(apitest.KeyedPrivatePreset))
	workID := apitest.WorkIDFromUpload(t, finished)

	owner := apitest.FetchStartedWork(t, router, session, workID)
	if !owner.HasPrivatePrompts || len(owner.AllowedApps) != 1 || owner.AllowedApps[0].ID != "lumiverse" {
		t.Fatalf("owner policy = has private prompts %t, apps %v", owner.HasPrivatePrompts, owner.AllowedApps)
	}
	ownerPrompts := promptListFromPage(t, owner).Fragments
	if len(ownerPrompts) != 2 || ownerPrompts[1].Text != "Exact private prompt." ||
		!ownerPrompts[1].Private {
		t.Fatalf("owner prompts = %+v", ownerPrompts)
	}

	readerResponse := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/works/"+workID, nil))
	if readerResponse.Code != http.StatusOK {
		t.Fatalf("reader page = %d: %s", readerResponse.Code, readerResponse.Body.String())
	}
	if strings.Contains(readerResponse.Body.String(), "Exact private prompt.") {
		t.Fatal("the reader response contains the private text")
	}
	var reader apitest.StartedWork
	if err := json.Unmarshal(readerResponse.Body.Bytes(), &reader); err != nil {
		t.Fatalf("decode reader page: %v", err)
	}
	readerPrompts := promptListFromPage(t, reader).Fragments
	if len(readerPrompts) != 2 || readerPrompts[1].Text != "" || !readerPrompts[1].Private {
		t.Fatalf("reader prompts = %+v", readerPrompts)
	}
}

func TestAKeyedPlaceholderOriginalFileKeepsTheExistingPrivateText(t *testing.T) {
	t.Parallel()
	router, session, works, _ := harness.NewVerifiedUploadRouterWithPool(t, apitest.LumiverseRegistry(t))
	metadata := apitest.ExampleMetadata("Keyed private prompt preset")
	metadata["filename"] = "keyed.json"
	created := apitest.UploadAndFinish(t, router, session, works, metadata, []byte(apitest.KeyedPrivatePreset))
	workID := apitest.WorkIDFromUpload(t, created)

	placeholder := []byte(`{
		"schemaVersion": 1,
		"name": "Keyed private prompt preset revision",
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
		apitest.OriginalFileRequest(t, workID, "keyed-revision.json", placeholder), session,
	))
	if accepted.Code != http.StatusAccepted {
		t.Fatalf("revision upload = %d: %s", accepted.Code, accepted.Body.String())
	}
	if processed, err := apitest.Uploads(works).ProcessNextUpload(t.Context()); err != nil || !processed {
		t.Fatalf("process revision = %t, %v; want true, nil", processed, err)
	}
	apitest.AcceptReplacementPreview(t, router, session, workID, accepted.Header().Get("Location"))
	apitest.PollUploadWork(t, router, session, accepted.Header().Get("Location"))

	owner := apitest.FetchStartedWork(t, router, session, workID)
	prompts := promptListFromPage(t, owner).Fragments
	if len(prompts) != 1 || prompts[0].Name != "Private renamed" ||
		prompts[0].Text != "Exact private prompt." || !prompts[0].Private {
		t.Fatalf("owner prompts after placeholder revision = %+v", prompts)
	}
}

func TestReplacementNeedsConfirmationBeforeMakingPrivatePromptsPublic(t *testing.T) {
	t.Parallel()
	for _, madePrivateAfterPublication := range []bool{false, true} {
		name := "private on upload"
		if madePrivateAfterPublication {
			name = "private after publication"
		}
		t.Run(name, func(t *testing.T) {
			router, session, works, _ := harness.NewVerifiedUploadRouterWithPool(t, apitest.LumiverseRegistry(t))
			ordinary := strings.ReplaceAll(apitest.KeyedPrivatePreset, `,"sealed":true,"sealedKey":"dialogue.frame"`, "")
			initial := apitest.KeyedPrivatePreset
			if madePrivateAfterPublication {
				initial = ordinary
			}
			metadata := apitest.ExampleMetadata("Replacement privacy")
			metadata["filename"] = "keyed.json"
			created := apitest.UploadAndFinish(t, router, session, works, metadata, []byte(initial))
			workID := apitest.WorkIDFromUpload(t, created)
			if madePrivateAfterPublication {
				page := apitest.FetchStartedWork(t, router, session, workID)
				core := apitest.BlockNamed(t, page.Blocks, "preset_core")
				body := apitest.MakeEveryFragmentPrivate(t, apitest.EditableBlock(core), []string{"lumiverse"})
				if response := apitest.SaveBlock(t, router, session, workID, core.ID, body); response.Code != http.StatusOK {
					t.Fatalf("make published text private: %d %s", response.Code, response.Body.String())
				}
			}
			before := apitest.FetchStartedWork(t, router, session, workID)
			replacement := strings.ReplaceAll(ordinary, "Keyed private prompt preset", "Replacement preset")
			staged := apitest.Send(t, router, apitest.Authorized(apitest.OriginalFileRequest(t, workID, "replacement.json", []byte(replacement)), session))
			if staged.Code != http.StatusAccepted {
				t.Fatalf("stage replacement: %d %s", staged.Code, staged.Body.String())
			}
			if processed, err := apitest.Uploads(works).ProcessNextUpload(t.Context()); err != nil || !processed {
				t.Fatalf("process replacement: %t %v", processed, err)
			}
			operationID := strings.TrimPrefix(staged.Header().Get("Location"), "/v1/uploads/")
			path := "/v1/works/" + workID + "/original-file/" + operationID + "/accept"
			request := apitest.AuthorizedJSONRequest(t, http.MethodPost, path, `{"unrepresentable":{}}`, session)
			apitest.WithReviewedVersion(t, router, request)
			refused := apitest.Send(t, router, request)
			if refused.Code != http.StatusConflict || !strings.Contains(refused.Body.String(), `"code":"prompts_made_public"`) || !strings.Contains(refused.Body.String(), "Private") {
				t.Fatalf("unconfirmed replacement: %d %s", refused.Code, refused.Body.String())
			}
			after := apitest.FetchStartedWork(t, router, session, workID)
			if !after.HasPrivatePrompts || !reflect.DeepEqual(after.Blocks, before.Blocks) {
				t.Fatal("refused replacement changed the drafted changes or its private prompts")
			}
			reader := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/works/"+workID, nil))
			if reader.Code != http.StatusOK || strings.Contains(reader.Body.String(), "Exact private prompt.") {
				t.Fatalf("reader after refusal: %d %s", reader.Code, reader.Body.String())
			}
			confirmation := apitest.AuthorizedJSONRequest(t, http.MethodPost, path, `{"unrepresentable":{},"makePromptsPublic":true}`, session)
			confirmation.Header.Set("X-Drafted-Changes-Version", request.Header.Get("X-Drafted-Changes-Version"))
			confirmed := apitest.Send(t, router, confirmation)
			if confirmed.Code != http.StatusOK {
				t.Fatalf("confirmed replacement: %d %s", confirmed.Code, confirmed.Body.String())
			}
			if apitest.FetchStartedWork(t, router, session, workID).HasPrivatePrompts {
				t.Fatal("confirmed replacement kept the old private prompts")
			}
			if madePrivateAfterPublication {
				reader = apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/works/"+workID, nil))
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
			router, session, works, _ := harness.NewVerifiedUploadRouterWithPool(t, apitest.LumiverseRegistry(t))
			metadata := apitest.ExampleMetadata("Refused preset")
			metadata["filename"] = "refused.json"
			accepted := apitest.Send(t, router, apitest.Authorized(
				apitest.UploadRequest(t, metadata, []byte(test.file)), session,
			))
			if accepted.Code != http.StatusAccepted {
				t.Fatalf("upload = %d: %s", accepted.Code, accepted.Body.String())
			}
			if processed, err := apitest.Uploads(works).ProcessNextUpload(t.Context()); err != nil || !processed {
				t.Fatalf("process upload = %t, %v; want true, nil", processed, err)
			}
			poll := apitest.Send(t, router, apitest.Authorized(httptest.NewRequest(
				http.MethodGet, accepted.Header().Get("Location"), nil,
			), session))
			var operation struct {
				Status  string `json:"status"`
				Work    any    `json:"work"`
				Failure *struct {
					Reason string `json:"reason"`
				} `json:"failure"`
			}
			if err := json.Unmarshal(poll.Body.Bytes(), &operation); err != nil {
				t.Fatalf("decode failed upload: %v", err)
			}
			if operation.Status != "failed" || operation.Work != nil || operation.Failure == nil ||
				operation.Failure.Reason != test.wantReason {
				t.Fatalf("operation = %#v, want an unsaved %s failure", operation, test.wantReason)
			}
		})
	}
}

func TestAnOrdinaryLumiversePresetStillUploadsAsPublicContent(t *testing.T) {
	t.Parallel()
	router, session, works, _ := harness.NewVerifiedUploadRouterWithPool(t, apitest.LumiverseRegistry(t))
	metadata := apitest.ExampleMetadata("Ordinary preset")
	metadata["filename"] = "ordinary.json"
	finished := apitest.UploadAndFinish(t, router, session, works, metadata, []byte(`{
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
	workID := apitest.WorkIDFromUpload(t, finished)

	readerResponse := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/works/"+workID, nil))
	if readerResponse.Code != http.StatusOK {
		t.Fatalf("reader page = %d: %s", readerResponse.Code, readerResponse.Body.String())
	}
	var reader apitest.StartedWork
	if err := json.Unmarshal(readerResponse.Body.Bytes(), &reader); err != nil {
		t.Fatalf("decode reader page: %v", err)
	}
	prompts := promptListFromPage(t, reader).Fragments
	if reader.HasPrivatePrompts || len(reader.AllowedApps) != 0 ||
		len(prompts) != 1 || prompts[0].Private || prompts[0].Text != "Ordinary public prompt." {
		t.Fatalf("ordinary preset = has private prompts %t, apps %v, prompts %+v",
			reader.HasPrivatePrompts, reader.AllowedApps, prompts)
	}
	if len(reader.Downloads) != 1 || reader.Downloads[0].Format != "preset_lumiverse" {
		t.Fatalf("ordinary downloads = %+v, want the Lumiverse target", reader.Downloads)
	}
}
