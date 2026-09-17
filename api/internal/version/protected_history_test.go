package version_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type versionComparisonBody struct {
	From            apitest.RecordedVersionBody `json:"from"`
	To              apitest.RecordedVersionBody `json:"to"`
	PromptsWithheld bool                        `json:"promptsWithheld"`
	Groups          []struct {
		Subject string `json:"subject"`
	} `json:"groups"`
}

type protectionMismatchBody struct {
	Items []struct {
		Version   apitest.RecordedVersionBody `json:"version"`
		Unmatched []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"unmatched"`
		Recorded []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"recorded"`
	} `json:"items"`
}

func compareVersions(t *testing.T, router *gin.Engine, assetID, query string, session *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet,
		"/v1/assets/"+assetID+"/updates/comparison"+query, nil)
	if session != nil {
		request = apitest.Authorized(request, session)
	}
	return apitest.Send(t, router, request)
}

func TestRecordedPromptsAreReadUnderTheCurrentProtection(t *testing.T) {
	t.Parallel()
	setupRouter, router, session, _ := harness.NewVerifiedRoutersWithService(t, 1<<20, api.DefaultDeadlines())
	publicID, sealedID := uuid.New(), uuid.New()
	const firstSecret = "Never hand these words to a reader."
	const secondSecret = "Nor these ones either."
	started := apitest.PublishTwoPromptPreset(t, router, session, publicID, sealedID,
		"Answer plainly.", firstSecret)

	owner := apitest.FetchStartedAsset(t, router, session, started.ID)
	core := apitest.EditableBlock(apitest.BlockNamed(t, owner.Blocks, "preset_core"))
	core.Elements[0].Content = apitest.SealedPresetPrompts(publicID, sealedID,
		"Answer plainly and briefly.", secondSecret)
	core.AllowedApps = &[]string{"lumiverse"}
	if got := apitest.SaveBlock(t, router, session, started.ID, owner.Blocks[0].ID, core); got.Code != http.StatusOK {
		t.Fatalf("edit the prompts: %d %s", got.Code, got.Body.String())
	}
	if got := apitest.PublishAssetUpdate(t, router, session, started.ID,
		`{"summary":"Tightened the house rule"}`); got.Code != http.StatusOK {
		t.Fatalf("publish the update: %d %s", got.Code, got.Body.String())
	}

	history := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/assets/"+started.ID+"/updates", nil))
	if history.Code != http.StatusOK {
		t.Fatalf("read the history: %d %s", history.Code, history.Body.String())
	}
	var recorded struct {
		Items []apitest.RecordedVersionBody `json:"items"`
	}
	if err := json.Unmarshal(history.Body.Bytes(), &recorded); err != nil {
		t.Fatalf("decode the history: %v", err)
	}
	if len(recorded.Items) != 2 || recorded.Items[0].Number != 2 {
		t.Fatalf("history = %+v", recorded.Items)
	}

	reader := compareVersions(t, router, started.ID, "", nil)
	if reader.Code != http.StatusOK {
		t.Fatalf("public comparison: %d %s", reader.Code, reader.Body.String())
	}
	for _, secret := range []string{firstSecret, secondSecret} {
		if strings.Contains(reader.Body.String(), secret) {
			t.Fatalf("the public comparison carried %q", secret)
		}
	}
	if !strings.Contains(reader.Body.String(), "Answer plainly and briefly.") {
		t.Fatal("the public comparison lost the ordinary prompt change")
	}

	ownerView := compareVersions(t, router, started.ID, "", session)
	if ownerView.Code != http.StatusOK {
		t.Fatalf("owner comparison: %d %s", ownerView.Code, ownerView.Body.String())
	}
	for _, secret := range []string{firstSecret, secondSecret} {
		if !strings.Contains(ownerView.Body.String(), secret) {
			t.Fatalf("the owner's comparison lost %q", secret)
		}
	}

	guessedVersion := compareVersions(t, router, started.ID, "?from=1&to=99", nil)
	if guessedVersion.Code != http.StatusNotFound {
		t.Fatalf("guessed version = %d, want 404", guessedVersion.Code)
	}
	guessedAsset := compareVersions(t, router, uuid.NewString(), "", nil)
	if guessedAsset.Code != http.StatusNotFound {
		t.Fatalf("guessed asset = %d, want 404", guessedAsset.Code)
	}

	source := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/download/"+started.ID, nil))
	if source.Code != http.StatusNotFound {
		t.Fatalf("public source download = %d, want 404: %s", source.Code, source.Body.String())
	}

	other := apitest.SignUp(t, setupRouter, "onlooker@example.com", "onlooker.reader")
	crossOwner := apitest.Send(t, router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/assets/"+started.ID+"/updates/protection", nil), other))
	if crossOwner.Code != http.StatusNotFound {
		t.Fatalf("another account read the sealed prompts: %d %s", crossOwner.Code, crossOwner.Body.String())
	}

	owner = apitest.FetchStartedAsset(t, router, session, started.ID)
	core = apitest.EditableBlock(apitest.BlockNamed(t, owner.Blocks, "preset_core"))
	core.Elements[0].Content = json.RawMessage(strings.ReplaceAll(
		string(core.Elements[0].Content), `"protected":true`, `"protected":false`))
	core.AllowedApps = &[]string{}
	if got := apitest.SaveBlock(t, router, session, started.ID, owner.Blocks[0].ID, core); got.Code != http.StatusConflict {
		t.Fatalf("unseal without confirming = %d, want 409: %s", got.Code, got.Body.String())
	}
	confirmed := true
	core.ExposeProtected = &confirmed
	if got := apitest.SaveBlock(t, router, session, started.ID, owner.Blocks[0].ID, core); got.Code != http.StatusOK {
		t.Fatalf("unseal the prompt: %d %s", got.Code, got.Body.String())
	}

	reader = compareVersions(t, router, started.ID, "", nil)
	for _, secret := range []string{firstSecret, secondSecret} {
		if !strings.Contains(reader.Body.String(), secret) {
			t.Fatalf("history stayed sealed after the owner made %q public", secret)
		}
	}
}

func TestChangedPromptIdsHoldRecordedPromptsUntilTheOwnerSettlesThem(t *testing.T) {
	t.Parallel()
	router, session := harness.NewVerifiedRouter(t)
	publicID, sealedID := uuid.New(), uuid.New()
	const secret = "The reader must never receive these words."
	const houseRule = "Answer plainly."
	started := apitest.PublishTwoPromptPreset(t, router, session, publicID, sealedID, houseRule, secret)

	owner := apitest.FetchStartedAsset(t, router, session, started.ID)
	core := apitest.EditableBlock(apitest.BlockNamed(t, owner.Blocks, "preset_core"))
	core.Elements[0].Content = apitest.SealedPresetPrompts(publicID, sealedID, houseRule+" And briefly.", secret)
	core.AllowedApps = &[]string{"lumiverse"}
	if got := apitest.SaveBlock(t, router, session, started.ID, owner.Blocks[0].ID, core); got.Code != http.StatusOK {
		t.Fatalf("edit the house rule: %d %s", got.Code, got.Body.String())
	}
	if got := apitest.PublishAssetUpdate(t, router, session, started.ID,
		`{"summary":"Tightened the house rule"}`); got.Code != http.StatusOK {
		t.Fatalf("publish the update: %d %s", got.Code, got.Body.String())
	}

	reimportedPublic, reimportedSealed := uuid.New(), uuid.New()
	owner = apitest.FetchStartedAsset(t, router, session, started.ID)
	core = apitest.EditableBlock(apitest.BlockNamed(t, owner.Blocks, "preset_core"))
	core.Elements[0].Content = apitest.SealedPresetPrompts(
		reimportedPublic, reimportedSealed, houseRule+" And briefly.", secret)
	core.AllowedApps = &[]string{"lumiverse"}
	if got := apitest.SaveBlock(t, router, session, started.ID, owner.Blocks[0].ID, core); got.Code != http.StatusOK {
		t.Fatalf("reimport the prompts under new ids: %d %s", got.Code, got.Body.String())
	}

	page := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/assets/"+started.ID, nil))
	if page.Code != http.StatusOK {
		t.Fatalf("public page: %d %s", page.Code, page.Body.String())
	}
	for _, held := range []string{secret, houseRule} {
		if strings.Contains(page.Body.String(), held) {
			t.Fatalf("an unmatched recorded prompt was shown: %q", held)
		}
	}
	source := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/download/"+started.ID, nil))
	if source.Code != http.StatusNotFound {
		t.Fatalf("public source download = %d, want 404", source.Code)
	}

	held := compareVersions(t, router, started.ID, "", nil)
	if held.Code != http.StatusOK {
		t.Fatalf("public comparison: %d %s", held.Code, held.Body.String())
	}
	if !apitest.DecodeResponse[versionComparisonBody](t, held).PromptsWithheld {
		t.Fatal("a comparison over unmatched prompts did not say they were withheld")
	}

	mismatches := apitest.Send(t, router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/assets/"+started.ID+"/updates/protection", nil), session))
	if mismatches.Code != http.StatusOK {
		t.Fatalf("read the mismatches: %d %s", mismatches.Code, mismatches.Body.String())
	}
	listed := apitest.DecodeResponse[protectionMismatchBody](t, mismatches)
	if len(listed.Items) != 2 {
		t.Fatalf("mismatched versions = %+v", listed.Items)
	}
	settling := listed.Items[0]
	if settling.Version.Number != 2 || len(settling.Unmatched) != 1 ||
		settling.Unmatched[0].ID != reimportedSealed.String() {
		t.Fatalf("version 2 mismatch = %+v", settling)
	}
	if len(settling.Recorded) != 2 {
		t.Fatalf("version 2 choices = %+v", settling.Recorded)
	}

	stranger := resolveCorrespondence(t, router, session, started.ID, 2,
		`{"matches":[{"current":"`+reimportedSealed.String()+`","recorded":"`+uuid.NewString()+`"}]}`)
	if stranger.Code != http.StatusBadRequest {
		t.Fatalf("a match onto a prompt the version never held = %d, want 400", stranger.Code)
	}
	settled := resolveCorrespondence(t, router, session, started.ID, 2,
		`{"matches":[{"current":"`+reimportedSealed.String()+`","recorded":"`+sealedID.String()+`"}]}`)
	if settled.Code != http.StatusNoContent {
		t.Fatalf("settle version 2: %d %s", settled.Code, settled.Body.String())
	}

	page = apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/assets/"+started.ID, nil))
	if !strings.Contains(page.Body.String(), houseRule) {
		t.Fatal("the settled version still hid its ordinary prompt")
	}
	if strings.Contains(page.Body.String(), secret) {
		t.Fatal("settling the correspondence made the sealed prompt public")
	}

	owner = apitest.FetchStartedAsset(t, router, session, started.ID)
	core = apitest.EditableBlock(apitest.BlockNamed(t, owner.Blocks, "preset_core"))
	core.Elements[0].Content = json.RawMessage(strings.ReplaceAll(
		string(core.Elements[0].Content), `"protected":true`, `"protected":false`))
	core.AllowedApps = &[]string{}
	confirmed := true
	core.ExposeProtected = &confirmed
	if got := apitest.SaveBlock(t, router, session, started.ID, owner.Blocks[0].ID, core); got.Code != http.StatusOK {
		t.Fatalf("unseal the settled prompt: %d %s", got.Code, got.Body.String())
	}
	page = apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/v1/assets/"+started.ID, nil))
	if !strings.Contains(page.Body.String(), secret) {
		t.Fatal("the settled version stayed sealed after the owner made its prompt public")
	}
}

func resolveCorrespondence(
	t *testing.T,
	router *gin.Engine,
	session *http.Cookie,
	assetID string,
	number int,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPut,
		"/v1/assets/"+assetID+"/updates/"+strconv.Itoa(number)+"/protection", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return apitest.Send(t, router, apitest.Authorized(request, session))
}

func TestMediaRecordedInAnOlderVersionStaysPublic(t *testing.T) {
	t.Parallel()
	router, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, router, session)
	apitest.WriteCharacterFloor(t, router, session, started)
	if got := apitest.Send(t, router, apitest.Authorized(
		apitest.MediaUploadRequest(t, started.ID, "avatar", apitest.PNG(t, 40, 60)), session,
	)); got.Code != http.StatusCreated {
		t.Fatalf("upload the first cover: %d", got.Code)
	}
	if got := apitest.PublishAsset(t, router, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish: %d %s", got.Code, got.Body.String())
	}
	recorded := apitest.FetchAssetPage(t, router, "/v1/assets/"+started.ID).Media[0]
	if got := apitest.Send(t, router, apitest.Authorized(
		apitest.MediaUploadRequest(t, started.ID, "avatar", apitest.PNG(t, 80, 120)), session,
	)); got.Code != http.StatusCreated {
		t.Fatalf("upload the replacement cover: %d", got.Code)
	}
	if got := apitest.PublishAssetUpdate(t, router, session, started.ID,
		`{"summary":"Replaced the cover"}`); got.Code != http.StatusOK {
		t.Fatalf("publish the update: %d %s", got.Code, got.Body.String())
	}
	current := apitest.FetchAssetPage(t, router, "/v1/assets/"+started.ID).Media[0]
	if current.ID == recorded.ID {
		t.Fatal("the update did not replace the cover")
	}
	served := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, recorded.ThumbURL, nil))
	if served.Code != http.StatusOK || served.Header().Get("Cache-Control") == "private, no-store" {
		t.Fatalf("recorded media = %d %q", served.Code, served.Header().Get("Cache-Control"))
	}
}
