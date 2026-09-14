package http

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/character"
	"github.com/Sillyfrogster/Illarin/api/internal/format/lorebook"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type writtenCard struct {
	Data struct {
		Description string `json:"description"`
	} `json:"data"`
}

func describeBlock(t *testing.T, r http.Handler, session *http.Cookie, started startedAsset, text string) {
	t.Helper()
	coreBlock := blockNamed(t, started.Blocks, "character_core")
	core := editableBlock(coreBlock)
	core.Elements[0].Content = json.RawMessage(`{"text":"` + text + `"}`)
	if got := saveBlock(t, r, session, started.ID, coreBlock.ID, core); got.Code != http.StatusOK {
		t.Fatalf("save the description: %d %s", got.Code, got.Body.String())
	}
}

func downloadVersion(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	assetID, target, query string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/download/"+assetID+"/"+target+query, nil)
	if session != nil {
		request = authorized(request, session)
	}
	return send(t, r, request)
}

func archivedCard(t *testing.T, archive []byte) []byte {
	t.Helper()
	opened, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatalf("open the CharX: %v", err)
	}
	for _, entry := range opened.File {
		if entry.Name != "card.json" {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			t.Fatalf("open the card: %v", err)
		}
		defer reader.Close()
		var held bytes.Buffer
		if _, err := held.ReadFrom(reader); err != nil {
			t.Fatalf("read the card: %v", err)
		}
		return held.Bytes()
	}
	t.Fatal("the CharX holds no card")
	return nil
}

func archivedDescription(t *testing.T, archive []byte) string {
	t.Helper()
	var card writtenCard
	if err := json.Unmarshal(archivedCard(t, archive), &card); err != nil {
		t.Fatalf("read the written card: %v", err)
	}
	return card.Data.Description
}

func archivedIcon(t *testing.T, archive []byte) []byte {
	t.Helper()
	for _, image := range archivedCardImages(t, archive) {
		if image.kind == "icon" {
			return image.data
		}
	}
	return nil
}

// publishTwoCoveredVersions publishes a character, then an update with a new description and cover.
func publishTwoCoveredVersions(
	t *testing.T,
	r *gin.Engine,
	session *http.Cookie,
) (startedAsset, []byte, []byte) {
	t.Helper()
	started := startCharacter(t, r, session)
	writeCharacterFloor(t, r, session, started)
	firstCover := httpTestPNG(t, 32, 32)
	uploadedImageID(t, r, session, started.ID, "avatar", firstCover)
	if got := publishAsset(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish: %d %s", got.Code, got.Body.String())
	}
	secondCover := httpTestPNG(t, 40, 40)
	uploadedImageID(t, r, session, started.ID, "avatar", secondCover)
	describeBlock(t, r, session, started, "She has moved to the east shelf.")
	if got := publishAssetUpdate(t, r, session, started.ID,
		`{"summary":"Moved her to the east shelf"}`); got.Code != http.StatusOK {
		t.Fatalf("publish the update: %d %s", got.Code, got.Body.String())
	}
	return started, firstCover, secondCover
}

func TestAnOlderVersionDownloadsWhatItRecordedAndTheNewestWhatReadersHave(t *testing.T) {
	t.Parallel()
	r, session, _, _ := newCharacterIngestRouterWithPool(t)
	started, firstCover, secondCover := publishTwoCoveredVersions(t, r, session)

	older := downloadVersion(t, r, nil, started.ID, "charx", "?version=1")
	if older.Code != http.StatusOK {
		t.Fatalf("download version 1: %d %s", older.Code, older.Body.String())
	}
	if got := archivedDescription(t, older.Body.Bytes()); got != "She keeps the books that forget themselves." {
		t.Errorf("version 1 description = %q, want the one it recorded", got)
	}
	if !bytes.Equal(archivedIcon(t, older.Body.Bytes()), firstCover) {
		t.Error("version 1 does not carry the cover it recorded")
	}
	if got := older.Header().Get("Content-Disposition"); !strings.Contains(got, "update-1") {
		t.Errorf("disposition = %q, want the update named in the filename", got)
	}

	newest := downloadVersion(t, r, nil, started.ID, "charx", "")
	if newest.Code != http.StatusOK {
		t.Fatalf("download the newest: %d %s", newest.Code, newest.Body.String())
	}
	if got := archivedDescription(t, newest.Body.Bytes()); got != "She has moved to the east shelf." {
		t.Errorf("newest description = %q, want the published one", got)
	}
	if !bytes.Equal(archivedIcon(t, newest.Body.Bytes()), secondCover) {
		t.Error("the newest version does not carry the current cover")
	}
	if got := newest.Header().Get("Content-Disposition"); strings.Contains(got, "update-") {
		t.Errorf("disposition = %q, want no update named on the current file", got)
	}
}

func TestAnOlderVersionsPicturesOutliveTheirReplacement(t *testing.T) {
	t.Parallel()
	r, session, assets, pool := newCharacterIngestRouterWithPool(t)
	started, firstCover, _ := publishTwoCoveredVersions(t, r, session)

	for range 2 {
		if _, err := assets.Sweep(t.Context()); err != nil {
			t.Fatalf("sweep: %v", err)
		}
		if _, err := pool.Exec(t.Context(),
			`update blob_sweep_marks set marked_at = marked_at - interval '2 days'`); err != nil {
			t.Fatalf("age the sweep marks: %v", err)
		}
	}

	older := downloadVersion(t, r, nil, started.ID, "charx", "?version=1")
	if older.Code != http.StatusOK {
		t.Fatalf("download version 1 after the sweep: %d %s", older.Code, older.Body.String())
	}
	if !bytes.Equal(archivedIcon(t, older.Body.Bytes()), firstCover) {
		t.Error("the sweep took the cover version 1 recorded")
	}
}

func TestAHistoricalDownloadNeverCarriesUnpublishedWork(t *testing.T) {
	t.Parallel()
	r, session, _, _ := newCharacterIngestRouterWithPool(t)
	started, firstCover, secondCover := publishTwoCoveredVersions(t, r, session)
	describeBlock(t, r, session, started, "She is thinking about the north shelf.")
	uploadedImageID(t, r, session, started.ID, "avatar", httpTestPNG(t, 48, 48))

	for _, reader := range []struct {
		name    string
		session *http.Cookie
	}{{"a stranger", nil}, {"the owner", session}} {
		for _, version := range []struct {
			query       string
			description string
			cover       []byte
		}{
			{"?version=1", "She keeps the books that forget themselves.", firstCover},
			{"?version=2", "She has moved to the east shelf.", secondCover},
			{"", "She has moved to the east shelf.", secondCover},
		} {
			download := downloadVersion(t, r, reader.session, started.ID, "charx", version.query)
			if download.Code != http.StatusOK {
				t.Fatalf("%s downloads %q: %d %s", reader.name, version.query, download.Code, download.Body.String())
			}
			if got := archivedDescription(t, download.Body.Bytes()); got != version.description {
				t.Errorf("%s downloads %q with description %q, want %q", reader.name, version.query, got, version.description)
			}
			if !bytes.Equal(archivedIcon(t, download.Body.Bytes()), version.cover) {
				t.Errorf("%s downloads %q with a cover that is not the recorded one", reader.name, version.query)
			}
		}
	}
}

func TestAnOlderVersionKeepsThePreservedDataItRecorded(t *testing.T) {
	t.Parallel()
	r, session, assets := newCharacterIngestRouter(t)
	assetID := uploadedCharacterID(t, r, session, assets, aCardCarryingThirdPartyNamespaces)
	publishCharacter(t, r, session, assetID)
	removed := send(t, r, authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/assets/"+assetID+"/preserved/chub", nil), session))
	if removed.Code != http.StatusNoContent {
		t.Fatalf("delete chub: %d %s", removed.Code, removed.Body.String())
	}
	if got := publishAssetUpdate(t, r, session, assetID,
		`{"summary":"Dropped the chub data"}`); got.Code != http.StatusOK {
		t.Fatalf("publish the update: %d %s", got.Code, got.Body.String())
	}

	older := downloadVersion(t, r, nil, assetID, "chara_card_v3", "?version=1")
	if older.Code != http.StatusOK {
		t.Fatalf("download version 1: %d %s", older.Code, older.Body.String())
	}
	if _, kept := namespacesOf(t, cardBodyOf(t, older.Body.Bytes())["extensions"])["chub"]; !kept {
		t.Error("version 1 lost the chub namespace it recorded")
	}
	newest := downloadVersion(t, r, nil, assetID, "chara_card_v3", "")
	if newest.Code != http.StatusOK {
		t.Fatalf("download the newest: %d %s", newest.Code, newest.Body.String())
	}
	if _, kept := namespacesOf(t, cardBodyOf(t, newest.Body.Bytes())["extensions"])["chub"]; kept {
		t.Error("the newest version still carries the chub namespace the creator removed")
	}
}

func TestAHistoricalDownloadHoldsTheCurrentProtection(t *testing.T) {
	t.Parallel()
	router, session := newVerifiedTestRouter(t)
	publicID, sealedID := uuid.New(), uuid.New()
	const firstSecret = "The first private instruction."
	const secondSecret = "The second private instruction."
	started := publishTwoPromptPreset(t, router, session, publicID, sealedID, "Answer plainly.", firstSecret)
	owner := fetchStartedAsset(t, router, session, started.ID)
	core := editableBlock(blockNamed(t, owner.Blocks, "preset_core"))
	core.Elements[0].Content = sealedPresetPrompts(publicID, sealedID, "Answer plainly.", secondSecret)
	core.AllowedApps = &[]string{"lumiverse"}
	if got := saveBlock(t, router, session, started.ID, owner.Blocks[0].ID, core); got.Code != http.StatusOK {
		t.Fatalf("edit the sealed prompt: %d %s", got.Code, got.Body.String())
	}
	if got := publishAssetUpdate(t, router, session, started.ID,
		`{"summary":"Reworded the private instruction"}`); got.Code != http.StatusOK {
		t.Fatalf("publish the update: %d %s", got.Code, got.Body.String())
	}

	for _, reader := range []struct {
		name    string
		session *http.Cookie
	}{{"a stranger", nil}, {"the owner", session}} {
		for _, query := range []string{"?version=1", "?version=2"} {
			refused := downloadVersion(t, router, reader.session, started.ID, "preset_lumiverse", query)
			if refused.Code != http.StatusNotFound {
				t.Fatalf("%s wrote a sealed preset %q: %d", reader.name, query, refused.Code)
			}
		}
	}

	owner = fetchStartedAsset(t, router, session, started.ID)
	core = editableBlock(blockNamed(t, owner.Blocks, "preset_core"))
	core.Elements[0].Content = sealedPresetPrompts(publicID, sealedID, "Answer plainly.", secondSecret)
	core.Elements[0].Content = json.RawMessage(strings.ReplaceAll(
		string(core.Elements[0].Content), `"protected":true`, `"protected":false`))
	core.AllowedApps = &[]string{}
	confirmed := true
	core.ExposeProtected = &confirmed
	if got := saveBlock(t, router, session, started.ID, owner.Blocks[0].ID, core); got.Code != http.StatusOK {
		t.Fatalf("unseal the prompt: %d %s", got.Code, got.Body.String())
	}

	first := downloadVersion(t, router, nil, started.ID, "preset_lumiverse", "?version=1")
	if first.Code != http.StatusOK {
		t.Fatalf("download version 1 once public: %d %s", first.Code, first.Body.String())
	}
	if !strings.Contains(first.Body.String(), firstSecret) || strings.Contains(first.Body.String(), secondSecret) {
		t.Error("version 1 does not carry the instruction it recorded, and only that one")
	}
	second := downloadVersion(t, router, nil, started.ID, "preset_lumiverse", "?version=2")
	if second.Code != http.StatusOK {
		t.Fatalf("download version 2 once public: %d %s", second.Code, second.Body.String())
	}
	if !strings.Contains(second.Body.String(), secondSecret) {
		t.Error("version 2 does not carry the instruction it recorded")
	}
}

func TestAVersionThatRecordedASealedPromptStaysUnwritableAfterItsRemoval(t *testing.T) {
	t.Parallel()
	router, session := newVerifiedTestRouter(t)
	publicID, sealedID := uuid.New(), uuid.New()
	const secret = "Words that were sealed when version 1 was recorded."
	started := publishTwoPromptPreset(t, router, session, publicID, sealedID, "Answer plainly.", secret)
	owner := fetchStartedAsset(t, router, session, started.ID)
	core := editableBlock(blockNamed(t, owner.Blocks, "preset_core"))
	core.Elements[0].Content = json.RawMessage(`{"groups":[],"fragments":[` +
		`{"id":"` + publicID.String() + `","name":"House rule","role":"system","text":"Answer plainly.","enabled":true}]}`)
	core.AllowedApps = &[]string{}
	if got := saveBlock(t, router, session, started.ID, owner.Blocks[0].ID, core); got.Code != http.StatusOK {
		t.Fatalf("remove the sealed prompt: %d %s", got.Code, got.Body.String())
	}
	if got := publishAssetUpdate(t, router, session, started.ID,
		`{"summary":"Removed the private instruction"}`); got.Code != http.StatusOK {
		t.Fatalf("publish the update: %d %s", got.Code, got.Body.String())
	}

	newest := downloadVersion(t, router, nil, started.ID, "preset_lumiverse", "")
	if newest.Code != http.StatusOK {
		t.Fatalf("download the newest: %d %s", newest.Code, newest.Body.String())
	}
	older := downloadVersion(t, router, nil, started.ID, "preset_lumiverse", "?version=1")
	if older.Code != http.StatusNotFound {
		t.Fatalf("version 1 was written with a prompt that was sealed when it was recorded: %d", older.Code)
	}
	if strings.Contains(older.Body.String(), secret) {
		t.Fatal("the refusal carried the sealed text")
	}
}

const aSillyTavernBook = `{"entries":{"0":{"uid":0,"key":["Timeline"],"comment":"Story Timeline",
	"content":"It is widely believed the story takes place in 2167.","order":100,"position":0,"disable":false}}}`

const aLumiverseBook = `{"name":"Zenless lore","entries":[{"keys":["Timeline"],
	"content":"It is widely believed the story takes place in 2167.","enabled":true,"insertion_order":100}]}`

func TestAVersionIsOfferedTheFormatsItsOwnRecordedOriginEarns(t *testing.T) {
	t.Parallel()
	registry := format.NewRegistry()
	for _, module := range lorebook.Modules() {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %s: %v", module.ID(), err)
		}
	}
	r, session, assets, pool := newVerifiedIngestRouterWithSettings(t, registry, asset.DefaultIngestSettings())
	metadata := exampleMetadata("Zenless lore")
	metadata["filename"] = "world-info.json"
	metadata["isNsfw"] = false
	assetID := assetIDFromIngest(t, uploadAndFinish(t, r, session, assets, metadata, []byte(aSillyTavernBook)))
	revision := send(t, r, authorized(revisionRequest(t, assetID, "lore.json", []byte(aLumiverseBook)), session))
	if revision.Code != http.StatusAccepted {
		t.Fatalf("upload the replacement: %d %s", revision.Code, revision.Body.String())
	}
	if _, err := assets.ProcessNextIngest(t.Context()); err != nil {
		t.Fatalf("process the replacement: %v", err)
	}
	acceptReplacementPreview(t, r, session, assetID, revision.Header().Get("Location"))
	if got := publishAssetUpdate(t, r, session, assetID,
		`{"summary":"Moved the book to the Lumiverse format"}`); got.Code != http.StatusOK {
		t.Fatalf("publish the update: %d %s", got.Code, got.Body.String())
	}

	for _, want := range []struct {
		query  string
		target string
		status int
	}{
		{"?version=1", "lorebook_sillytavern", http.StatusOK},
		{"?version=1", "lorebook", http.StatusNotFound},
		{"", "lorebook", http.StatusOK},
		{"", "lorebook_sillytavern", http.StatusNotFound},
		{"?version=2", "lorebook", http.StatusOK},
		{"?version=3", "lorebook", http.StatusNotFound},
		{"?version=0", "lorebook", http.StatusNotFound},
	} {
		got := downloadVersion(t, r, nil, assetID, want.target, want.query)
		if got.Code != want.status {
			t.Errorf("%s %q = %d, want %d: %s", want.target, want.query, got.Code, want.status, got.Body.String())
		}
	}

	var revisions int
	if err := pool.QueryRow(t.Context(), `select count(distinct revision_id) from download_events
		where asset_id = $1 and revision_id is not null`, assetID).Scan(&revisions); err != nil || revisions != 2 {
		t.Fatalf("revisions the download events name = %d, error = %v; want each version's own", revisions, err)
	}
}

func noisyPNG(t *testing.T, side int, seed int64) []byte {
	t.Helper()
	random := rand.New(rand.NewSource(seed))
	picture := image.NewRGBA(image.Rect(0, 0, side, side))
	for y := range side {
		for x := range side {
			picture.Set(x, y, color.RGBA{
				R: uint8(random.Intn(256)), G: uint8(random.Intn(256)), B: uint8(random.Intn(256)), A: 255,
			})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, picture); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	return encoded.Bytes()
}

func TestAFullAccountRefusesNewPicturesRatherThanForgettingRecordedOnes(t *testing.T) {
	t.Parallel()
	firstCover, secondCover, third := noisyPNG(t, 52, 1), noisyPNG(t, 37, 2), noisyPNG(t, 41, 3)
	registry := format.NewRegistry()
	for _, module := range character.Modules() {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %s: %v", module.ID(), err)
		}
	}
	settings := asset.DefaultIngestSettings()
	settings.AccountStorageCapBytes = int64(len(firstCover) + len(secondCover) + len(third) - 1)
	r, session, _, _ := newVerifiedIngestRouterWithSettings(t, registry, settings)

	started := startCharacter(t, r, session)
	writeCharacterFloor(t, r, session, started)
	uploadedImageID(t, r, session, started.ID, "avatar", firstCover)
	if got := publishAsset(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish: %d %s", got.Code, got.Body.String())
	}
	uploadedImageID(t, r, session, started.ID, "avatar", secondCover)
	if got := publishAssetUpdate(t, r, session, started.ID,
		`{"summary":"A new cover"}`); got.Code != http.StatusOK {
		t.Fatalf("publish the update: %d %s", got.Code, got.Body.String())
	}

	refused := send(t, r, authorized(mediaUploadRequest(t, started.ID, "avatar", third), session))
	if refused.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("a third cover past the cap = %d, want 413: %s", refused.Code, refused.Body.String())
	}
	for _, version := range []struct {
		query string
		cover []byte
	}{{"?version=1", firstCover}, {"?version=2", secondCover}} {
		download := downloadVersion(t, r, nil, started.ID, "charx", version.query)
		if download.Code != http.StatusOK {
			t.Fatalf("download %q after the refusal: %d %s", version.query, download.Code, download.Body.String())
		}
		if !bytes.Equal(archivedIcon(t, download.Body.Bytes()), version.cover) {
			t.Errorf("%q lost its cover to make room", version.query)
		}
	}
}

func TestHistoryFollowsTheAssetThroughDeletionRecoveryAndPurge(t *testing.T) {
	t.Parallel()
	r, session, assets, pool := newCharacterIngestRouterWithPool(t)
	started, firstCover, _ := publishTwoCoveredVersions(t, r, session)

	deleted := send(t, r, authorized(httptest.NewRequest(http.MethodDelete, "/v1/assets/"+started.ID, nil), session))
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", deleted.Code, deleted.Body.String())
	}
	for _, reader := range []*http.Cookie{nil, session} {
		if got := downloadVersion(t, r, reader, started.ID, "charx", "?version=1"); got.Code != http.StatusNotFound {
			t.Fatalf("a deleted asset still wrote version 1: %d", got.Code)
		}
	}
	restored := send(t, r, authorized(httptest.NewRequest(http.MethodPost, "/v1/assets/"+started.ID+"/restore", nil), session))
	if restored.Code != http.StatusNoContent {
		t.Fatalf("restore: %d %s", restored.Code, restored.Body.String())
	}
	recovered := downloadVersion(t, r, nil, started.ID, "charx", "?version=1")
	if recovered.Code != http.StatusOK || !bytes.Equal(archivedIcon(t, recovered.Body.Bytes()), firstCover) {
		t.Fatalf("version 1 after recovery = %d, want it whole again", recovered.Code)
	}

	if err := assets.Purge(t.Context(), sha256.Sum256(firstCover), "test_purge", uuid.New()); err != nil {
		t.Fatalf("purge the first cover: %v", err)
	}
	purged := downloadVersion(t, r, nil, started.ID, "charx", "?version=1")
	if purged.Code != http.StatusOK {
		t.Fatalf("version 1 after the purge: %d %s", purged.Code, purged.Body.String())
	}
	if archivedIcon(t, purged.Body.Bytes()) != nil {
		t.Fatal("version 1 still carries the purged cover")
	}

	deleted = send(t, r, authorized(httptest.NewRequest(http.MethodDelete, "/v1/assets/"+started.ID, nil), session))
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete again: %d %s", deleted.Code, deleted.Body.String())
	}
	if _, err := pool.Exec(t.Context(),
		`update assets set recoverable_until = now() - interval '1 second' where id = $1`, started.ID); err != nil {
		t.Fatalf("expire the recovery window: %v", err)
	}
	for range 2 {
		if _, err := assets.Sweep(t.Context()); err != nil {
			t.Fatalf("sweep: %v", err)
		}
		if _, err := pool.Exec(t.Context(),
			`update blob_sweep_marks set marked_at = marked_at - interval '2 days'`); err != nil {
			t.Fatalf("age the sweep marks: %v", err)
		}
	}
	if got := downloadVersion(t, r, session, started.ID, "charx", "?version=2"); got.Code != http.StatusNotFound {
		t.Fatalf("a finally deleted asset still wrote version 2: %d", got.Code)
	}
	var kept int
	if err := pool.QueryRow(t.Context(), `select count(*) from asset_snapshots where asset_id = $1`, started.ID).Scan(&kept); err != nil || kept != 0 {
		t.Fatalf("recorded versions after final deletion = %d, error = %v; want none", kept, err)
	}
	var blobs int
	if err := pool.QueryRow(t.Context(), `select count(*) from blobs blob
		where exists (select 1 from asset_media media where media.blob_id = blob.id and media.asset_id = $1)`, started.ID).Scan(&blobs); err != nil || blobs != 0 {
		t.Fatalf("picture blobs after final deletion = %d, error = %v; want none", blobs, err)
	}
}

type recordedDownloadsBody struct {
	Version           recordedVersionBody `json:"version"`
	Kind              string              `json:"kind"`
	LinkedInstallOnly bool                `json:"linkedInstallOnly"`
	Downloads         []downloadTarget    `json:"downloads"`
	AppTargets        []appTarget         `json:"appTargets"`
	Blocks            []startedBlock      `json:"blocks"`
	Media             []struct {
		ID       string `json:"id"`
		IsCover  bool   `json:"isCover"`
		ThumbURL string `json:"thumbUrl"`
		Bytes    int    `json:"bytes"`
	} `json:"media"`
}

func readVersionDownloads(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	assetID string,
	number int,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet,
		"/v1/assets/"+assetID+"/updates/"+strconv.Itoa(number)+"/downloads", nil)
	if session != nil {
		request = authorized(request, session)
	}
	return send(t, r, request)
}

func offeredFormats(targets []downloadTarget) []string {
	formats := make([]string, 0, len(targets))
	for _, target := range targets {
		formats = append(formats, target.Format)
	}
	return formats
}

func TestAVersionSaysWhichFilesItCanBeWrittenAsToday(t *testing.T) {
	t.Parallel()
	registry := format.NewRegistry()
	for _, module := range lorebook.Modules() {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %s: %v", module.ID(), err)
		}
	}
	r, session, assets := newVerifiedIngestRouter(t, registry)
	metadata := exampleMetadata("Zenless lore")
	metadata["filename"] = "world-info.json"
	metadata["isNsfw"] = false
	assetID := assetIDFromIngest(t, uploadAndFinish(t, r, session, assets, metadata, []byte(aSillyTavernBook)))
	revision := send(t, r, authorized(revisionRequest(t, assetID, "lore.json", []byte(aLumiverseBook)), session))
	if _, err := assets.ProcessNextIngest(t.Context()); err != nil {
		t.Fatalf("process the replacement: %v", err)
	}
	acceptReplacementPreview(t, r, session, assetID, revision.Header().Get("Location"))
	if got := publishAssetUpdate(t, r, session, assetID,
		`{"summary":"Moved the book to the Lumiverse format"}`); got.Code != http.StatusOK {
		t.Fatalf("publish the update: %d %s", got.Code, got.Body.String())
	}

	for number, want := range map[int][]string{1: {"lorebook_sillytavern"}, 2: {"lorebook"}} {
		answer := readVersionDownloads(t, r, nil, assetID, number)
		if answer.Code != http.StatusOK {
			t.Fatalf("version %d downloads: %d %s", number, answer.Code, answer.Body.String())
		}
		offered := decodeResponse[recordedDownloadsBody](t, answer)
		if offered.Version.Number != number || offered.Kind != "lorebook" || offered.LinkedInstallOnly {
			t.Errorf("version %d = %+v", number, offered)
		}
		if got := offeredFormats(offered.Downloads); !slices.Equal(got, want) {
			t.Errorf("version %d is offered %v, want %v", number, got, want)
		}
	}
	if missing := readVersionDownloads(t, r, nil, assetID, 3); missing.Code != http.StatusNotFound {
		t.Fatalf("a version never recorded = %d, want 404", missing.Code)
	}
}

func TestASealedVersionOffersNoFileAndSaysWhy(t *testing.T) {
	t.Parallel()
	setupRouter, router, session, _ := newVerifiedTestRoutersWithService(t, 1<<20, DefaultDeadlines())
	publicID, sealedID := uuid.New(), uuid.New()
	const secret = "Not for a file."
	started := publishTwoPromptPreset(t, router, session, publicID, sealedID, "Answer plainly.", secret)

	for _, reader := range []*http.Cookie{nil, session} {
		answer := readVersionDownloads(t, router, reader, started.ID, 1)
		if answer.Code != http.StatusOK {
			t.Fatalf("sealed version downloads: %d %s", answer.Code, answer.Body.String())
		}
		offered := decodeResponse[recordedDownloadsBody](t, answer)
		if !offered.LinkedInstallOnly || len(offered.Downloads) != 0 || len(offered.AppTargets) != 0 {
			t.Fatalf("a sealed version offered a file: %+v", offered)
		}
		if strings.Contains(answer.Body.String(), secret) {
			t.Fatal("the sealed text reached the download choices")
		}
	}

	other := signUp(t, setupRouter, "onlooker@example.com", "onlooker.reader")
	withheld := send(t, router, authorized(httptest.NewRequest(http.MethodDelete, "/v1/assets/"+started.ID, nil), session))
	if withheld.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", withheld.Code, withheld.Body.String())
	}
	if got := readVersionDownloads(t, router, other, started.ID, 1); got.Code != http.StatusNotFound {
		t.Fatalf("a deleted asset answered a stranger: %d", got.Code)
	}
}

func TestAVersionListsThePicturesItRecorded(t *testing.T) {
	t.Parallel()
	r, session, _, _ := newCharacterIngestRouterWithPool(t)
	started, _, _ := publishTwoCoveredVersions(t, r, session)

	answer := readVersionDownloads(t, r, nil, started.ID, 1)
	if answer.Code != http.StatusOK {
		t.Fatalf("version 1 downloads: %d %s", answer.Code, answer.Body.String())
	}
	offered := decodeResponse[recordedDownloadsBody](t, answer)
	if len(offered.Media) != 1 || !offered.Media[0].IsCover || offered.Media[0].Bytes == 0 {
		t.Fatalf("version 1 pictures = %+v, want its one cover", offered.Media)
	}
	if len(offered.Blocks) == 0 || len(offered.Downloads) == 0 {
		t.Fatalf("version 1 carries %d blocks and %d formats", len(offered.Blocks), len(offered.Downloads))
	}
	picture := send(t, r, httptest.NewRequest(http.MethodGet, offered.Media[0].ThumbURL, nil))
	if picture.Code != http.StatusOK || picture.Header().Get("X-Accel-Redirect") == "" {
		t.Fatalf("the recorded cover is not reachable: %d %s", picture.Code, picture.Body.String())
	}
}
