package http

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/character"
	"github.com/Sillyfrogster/Illarin/api/internal/probe"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sys/unix"
)

type parseFailureModule struct{}

type neverClaimsModule struct{}

func (neverClaimsModule) ID() string { return "never" }
func (neverClaimsModule) Declaration() format.Declaration {
	return apitest.ReaderDeclaration("never", "character")
}
func (neverClaimsModule) Claim(probe.Inspection) (format.Claim, bool) { return format.Claim{}, false }
func (neverClaimsModule) Parse(context.Context, probe.Inspection, format.Claim) (format.Parsed, error) {
	return format.Parsed{}, errors.New("unreachable")
}

func (parseFailureModule) ID() string { return "claimed" }
func (parseFailureModule) Declaration() format.Declaration {
	return apitest.ReaderDeclaration("claimed", "character")
}
func (parseFailureModule) Claim(file probe.Inspection) (format.Claim, bool) {
	if len(file.Payloads) == 0 {
		return format.Claim{}, false
	}
	return format.CompatibilityClaim(file.Payloads[0]), true
}

type typedParseFailureModule struct {
	err error
}

func (typedParseFailureModule) ID() string { return "typed_failure" }
func (typedParseFailureModule) Declaration() format.Declaration {
	return apitest.ReaderDeclaration("typed_failure", "character")
}
func (typedParseFailureModule) Claim(file probe.Inspection) (format.Claim, bool) {
	if len(file.Payloads) == 0 {
		return format.Claim{}, false
	}
	return format.CompatibilityClaim(file.Payloads[0]), true
}
func (m typedParseFailureModule) Parse(context.Context, probe.Inspection, format.Claim) (format.Parsed, error) {
	return format.Parsed{}, m.err
}
func (parseFailureModule) Parse(context.Context, probe.Inspection, format.Claim) (format.Parsed, error) {
	return format.Parsed{}, errors.New("the claimed payload is malformed")
}

func TestUploadReturnsAPendingOperation(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)

	rec := apitest.Send(t, r, apitest.Authorized(
		apitest.UploadRequest(t, apitest.ExampleMetadata("Evening Theme"), []byte("theme bytes")),
		session,
	))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202. body: %s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "/v1/ingests/") {
		t.Fatalf("Location = %q, want an ingest operation URL", location)
	}

	var operation struct {
		Status string `json:"status"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode operation: %v", err)
	}
	if operation.Status != "pending" {
		t.Errorf("status = %q, want pending", operation.Status)
	}
	if operation.URL != location {
		t.Errorf("url = %q, want Location %q", operation.URL, location)
	}
}

func TestUploadWaitsWhenItsMaximumWriteWouldCrossTheStorageReserve(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	var filesystem unix.Statfs_t
	if err := unix.Statfs(root, &filesystem); err != nil {
		t.Fatalf("read free space: %v", err)
	}
	available := int64(filesystem.Bavail) * filesystem.Bsize
	const headroom = int64(8 << 20)
	if available <= headroom {
		t.Skip("test filesystem has less than 8 MB free")
	}

	r, session, _, pool := harness.NewVerifiedIngestRouterWithStoreFactory(
		t, format.NewRegistry(), asset.DefaultIngestSettings(),
		func(pool *pgxpool.Pool) (storage.Store, error) {
			return storage.NewStoreWithCapacity(pool, root, storage.Capacity{
				FreeSpaceReserveBytes: available - headroom,
				MaximumBlobWriteBytes: 32 << 20,
			})
		},
	)

	response := apitest.Send(t, r, apitest.Authorized(
		apitest.UploadRequest(t, apitest.ExampleMetadata("Waiting theme"), []byte("small upload")),
		session,
	))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Location") != "" {
		t.Fatalf("refused upload has Location %q", response.Header().Get("Location"))
	}
	var operations int
	if err := pool.QueryRow(context.Background(), `select count(*) from ingest_operations`).Scan(&operations); err != nil {
		t.Fatalf("count ingest operations: %v", err)
	}
	if operations != 0 {
		t.Fatalf("refused upload recorded %d ingest operations", operations)
	}
}

func TestAccountStorageCapChargesSharedBytesPerAccountButNotRepeatedUse(t *testing.T) {
	t.Parallel()
	shared := []byte("shared canonical bytes")
	root := t.TempDir()
	var blobs storage.Store
	r, firstSession, assets, pool := harness.NewVerifiedIngestRouterWithStoreFactory(
		t, format.NewRegistry(), asset.DefaultIngestSettings(),
		func(pool *pgxpool.Pool) (storage.Store, error) {
			var err error
			blobs, err = storage.NewStore(pool, root)
			return blobs, err
		},
	)
	seed := apitest.Send(t, r, apitest.Authorized(
		apitest.UploadRequest(t, apitest.ExampleMetadata("First use"), shared), firstSession,
	))
	if seed.Code != http.StatusAccepted {
		t.Fatalf("seed upload status = %d, want 202: %s", seed.Code, seed.Body.String())
	}
	if processed, err := assets.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("process seed ingest = %v, %v; want true, nil", processed, err)
	}
	created := pollIngestAsset(t, r, firstSession, seed.Header().Get("Location"))

	settings := asset.DefaultIngestSettings()
	settings.AccountStorageCapBytes = int64(len(shared) - 1)
	limitedAssets := asset.NewServiceWithIngestSettings(
		pool, format.NewRegistry(), blobs, settings,
	)
	outbox := &apitest.VerificationOutbox{}
	handlers := apitest.NewServicesOver(pool, blobs, limitedAssets, outbox, nil)
	limitedRouter := harness.RegisterRouter(t, handlers, api.DefaultDeadlines())

	repeated := apitest.Send(t, limitedRouter, apitest.Authorized(
		apitest.UploadRequest(t, apitest.ExampleMetadata("Repeated use"), shared), firstSession,
	))
	if repeated.Code != http.StatusAccepted {
		t.Fatalf("repeated upload status = %d, want 202: %s", repeated.Code, repeated.Body.String())
	}
	repeatedRevision := apitest.Send(t, limitedRouter, apitest.Authorized(
		revisionRequest(t, created.ID, "same.bin", shared), firstSession,
	))
	if repeatedRevision.Code != http.StatusAccepted {
		t.Fatalf("repeated revision status = %d, want 202: %s", repeatedRevision.Code, repeatedRevision.Body.String())
	}
	distinctRevision := apitest.Send(t, limitedRouter, apitest.Authorized(
		revisionRequest(t, created.ID, "different.bin", []byte("different canonical bytes")), firstSession,
	))
	if distinctRevision.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("distinct revision status = %d, want 413: %s", distinctRevision.Code, distinctRevision.Body.String())
	}

	secondSession := apitest.SignUp(t, limitedRouter, "second@example.com", "second.creator")
	verificationURL, err := url.Parse(outbox.Messages[0].Link)
	if err != nil {
		t.Fatalf("parse second verification link: %v", err)
	}
	verified := apitest.SendJSON(t, limitedRouter, http.MethodPost, "/v1/auth/verify-email",
		`{"token":"`+verificationURL.Query().Get("token")+`"}`)
	if verified.Code != http.StatusOK {
		t.Fatalf("verify second account: %d %s", verified.Code, verified.Body.String())
	}
	charged := apitest.Send(t, limitedRouter, apitest.Authorized(
		apitest.UploadRequest(t, apitest.ExampleMetadata("Somebody else's use"), shared), secondSession,
	))
	if charged.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("cross-account upload status = %d, want 413: %s", charged.Code, charged.Body.String())
	}
}

func TestCreatorCanPollTheirPendingIngest(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	upload := apitest.Send(t, r, apitest.Authorized(
		apitest.UploadRequest(t, apitest.ExampleMetadata("Evening Theme"), []byte("theme bytes")),
		session,
	))

	poll := httptest.NewRequest(http.MethodGet, upload.Header().Get("Location"), nil)
	rec := apitest.Send(t, r, apitest.Authorized(poll, session))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}
	var operation struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode operation: %v", err)
	}
	if operation.Status != "pending" {
		t.Errorf("status = %q, want pending", operation.Status)
	}
}

func TestCharacterUploadLandsOnABuiltDraftPage(t *testing.T) {
	t.Parallel()
	registry := format.NewRegistry()
	for _, module := range character.Modules() {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %s: %v", module.ID(), err)
		}
	}
	r, session, assets, pool := harness.NewVerifiedIngestRouterWithPool(t, registry)
	metadata := apitest.ExampleMetadata("Ana")
	metadata["filename"] = "ana.json"
	metadata["_keepDraft"] = true
	card := []byte(`{
		"spec":"chara_card_v3","spec_version":"3.0",
		"data":{
			"name":"Ana","nickname":"Archivist","character_version":"main","creator":"A. Writer",
			"description":"Keeps the archive.","personality":"Patient",
			"scenario":"After closing","first_mes":"Welcome back.",
			"group_only_greetings":["All of you made it."],
			"system_prompt":"Stay in character.","future_structure":{"kept":"whole"}
		}
	}`)
	finished := apitest.UploadAndFinish(t, r, session, assets, metadata, card)
	assetID := apitest.AssetIDFromIngest(t, finished)

	pageResponse := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/assets/"+assetID, nil), session,
	))
	if pageResponse.Code != http.StatusOK {
		t.Fatalf("page status = %d, want 200: %s", pageResponse.Code, pageResponse.Body.String())
	}
	var page apitest.StartedAsset
	if err := json.Unmarshal(pageResponse.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode imported page: %v", err)
	}
	if page.Lifecycle != "draft" {
		t.Errorf("lifecycle = %q, want draft", page.Lifecycle)
	}
	if len(page.Blocks) != 3 {
		t.Fatalf("blocks = %d, want character core, messages and model instructions", len(page.Blocks))
	}
	core := apitest.BlockNamed(t, page.Blocks, "character_core")
	if core.Layout != "stack-3" || core.Width != "two_thirds" {
		t.Errorf("character core = %s at %s", core.Layout, core.Width)
	}
	if string(core.Elements[0].Content) != `{"text":"Keeps the archive."}` {
		t.Errorf("description = %s", core.Elements[0].Content)
	}
	messages := apitest.BlockNamed(t, page.Blocks, "messages")
	if messages.Layout != "stack-3" || len(messages.Elements) != 3 {
		t.Errorf("messages = %+v, want three roles in stack-3", messages)
	}
	instructions := apitest.BlockNamed(t, page.Blocks, "model_instructions")
	if instructions.Width != "half" || len(instructions.Elements) != 1 ||
		instructions.Elements[0].Role != "system_prompt" {
		t.Errorf("model instructions = %+v", instructions)
	}
	var origin, version, author, nickname string
	if err := pool.QueryRow(context.Background(), `
		select origin_format, asset_version, credited_author, nickname
		  from assets where id = $1
	`, assetID).Scan(&origin, &version, &author, &nickname); err != nil {
		t.Fatalf("read imported header: %v", err)
	}
	if origin != character.V3 || version != "main" || author != "A. Writer" || nickname != "Archivist" {
		t.Errorf("origin and header = %q, %q, %q, %q", origin, version, author, nickname)
	}
	var preserved []byte
	if err := pool.QueryRow(context.Background(), `
		select payload from asset_preserved_data where asset_id = $1 and namespace = 'card'
	`, assetID).Scan(&preserved); err != nil {
		t.Fatalf("read preserved remainder: %v", err)
	}
	if !bytes.Contains(preserved, []byte(`"future_structure"`)) {
		t.Errorf("preserved remainder = %s", preserved)
	}
}

func TestEveryCharacterReaderBuildsTheCatalogPage(t *testing.T) {
	t.Parallel()
	cardBody := func(spec, description string) []byte {
		version := "3.0"
		if spec == character.V2 {
			version = "2.0"
		}
		return []byte(fmt.Sprintf(`{
			"spec":%q,"spec_version":%q,
			"data":{"name":"Ana","description":%q,"first_mes":"Hello"}
		}`, spec, version, description))
	}
	tests := []struct {
		name, filename, origin, description string
		file                                []byte
	}{
		{name: "CCv2", filename: "ana-v2.json", origin: character.V2,
			description: "From V2", file: cardBody(character.V2, "From V2")},
		{name: "CCv3", filename: "ana-v3.json", origin: character.V3,
			description: "From V3", file: cardBody(character.V3, "From V3")},
		{name: "CharX", filename: "ana.charx", origin: character.CharX,
			description: "From CharX", file: zipCharacterCard(t, cardBody(character.V3, "From CharX"))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := format.NewRegistry()
			for _, module := range character.Modules() {
				if err := registry.Register(module); err != nil {
					t.Fatalf("register %s: %v", module.ID(), err)
				}
			}
			r, session, assets, pool := harness.NewVerifiedIngestRouterWithPool(t, registry)
			metadata := apitest.ExampleMetadata("Ana")
			metadata["filename"] = test.filename
			metadata["_keepDraft"] = true
			assetID := apitest.AssetIDFromIngest(t, apitest.UploadAndFinish(t, r, session, assets, metadata, test.file))
			response := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
				http.MethodGet, "/v1/assets/"+assetID, nil,
			), session))
			var page apitest.StartedAsset
			if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &page) != nil {
				t.Fatalf("page = %d: %s", response.Code, response.Body.String())
			}
			if page.Lifecycle != "draft" || len(page.Blocks) != 2 {
				t.Fatalf("page = lifecycle %q, blocks %+v", page.Lifecycle, page.Blocks)
			}
			core := apitest.BlockNamed(t, page.Blocks, "character_core")
			if core.Layout != "stack-3" || core.Width != "two_thirds" ||
				string(core.Elements[0].Content) != fmt.Sprintf(`{"text":%q}`, test.description) {
				t.Errorf("character core = %+v", core)
			}
			messages := apitest.BlockNamed(t, page.Blocks, "messages")
			if messages.Layout != "stack-2" || messages.Width != "full" || len(messages.Elements) != 2 {
				t.Errorf("messages = %+v", messages)
			}
			var origin string
			if err := pool.QueryRow(context.Background(),
				`select origin_format from assets where id = $1`, assetID,
			).Scan(&origin); err != nil || origin != test.origin {
				t.Errorf("origin = %q, %v; want %q", origin, err, test.origin)
			}
		})
	}
}

func TestAnUnreadableOptionalCharXImageDoesNotRejectTheCharacter(t *testing.T) {
	t.Parallel()
	registry := format.NewRegistry()
	for _, module := range character.Modules() {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %s: %v", module.ID(), err)
		}
	}
	r, session, assets, pool := harness.NewVerifiedIngestRouterWithPool(t, registry)
	card := []byte(`{
		"spec":"chara_card_v3","spec_version":"3.0",
		"data":{"name":"Ana","description":"Quiet","first_mes":"Hello",
			"assets":[{"type":"emotion","uri":"embeded://assets/bad.png","name":"bad","ext":"png"}]}
	}`)
	file := zipCharacterCardWithFiles(t, card, map[string][]byte{
		"assets/bad.png": []byte("not an image"),
	})
	marker := []byte("not an image")
	position := bytes.Index(file, marker)
	if position < 0 {
		t.Fatal("stored image bytes are missing from the CharX fixture")
	}
	file[position] ^= 0xff
	metadata := apitest.ExampleMetadata("Ana")
	metadata["filename"] = "ana.charx"
	metadata["_keepDraft"] = true
	assetID := apitest.AssetIDFromIngest(t, apitest.UploadAndFinish(t, r, session, assets, metadata, file))

	var mediaCount int
	if err := pool.QueryRow(context.Background(),
		`select count(*) from asset_media where asset_id = $1`, assetID,
	).Scan(&mediaCount); err != nil {
		t.Fatalf("count extracted media: %v", err)
	}
	if mediaCount != 0 {
		t.Errorf("extracted media = %d, want the unreadable optional image skipped", mediaCount)
	}
	var preserved []byte
	if err := pool.QueryRow(context.Background(), `
		select payload from asset_preserved_data where asset_id = $1 and namespace = 'card'
	`, assetID).Scan(&preserved); err != nil {
		t.Fatalf("read preserved assets: %v", err)
	}
	var cardRemainder map[string]json.RawMessage
	if err := json.Unmarshal(preserved, &cardRemainder); err != nil {
		t.Fatalf("decode preserved card data: %v", err)
	}
	assetsRemainder, ok := cardRemainder["assets"]
	if !ok || !bytes.Contains(assetsRemainder, []byte("bad.png")) {
		t.Fatalf("preserved card data = %s", preserved)
	}
}

func TestExtractedMediaCannotTakeTheAccountPastItsStorageCap(t *testing.T) {
	t.Parallel()
	registry := format.NewRegistry()
	for _, module := range character.Modules() {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %s: %v", module.ID(), err)
		}
	}
	image := apitest.PNG(t, 120, 60)
	card := []byte(`{
		"spec":"chara_card_v3","spec_version":"3.0",
		"data":{"name":"Ana","description":"Quiet","first_mes":"Hello",
			"assets":[{"type":"emotion","uri":"embeded://assets/happy.png","name":"happy","ext":"png"}]}
	}`)
	file := zipCharacterCardWithFiles(t, card, map[string][]byte{"assets/happy.png": image})
	settings := asset.DefaultIngestSettings()
	settings.AccountStorageCapBytes = int64(len(file) + len(image) - 1)
	r, session, assets, pool := harness.NewVerifiedIngestRouterWithSettings(t, registry, settings)
	metadata := apitest.ExampleMetadata("Ana")
	metadata["filename"] = "ana.charx"
	upload := apitest.Send(t, r, apitest.Authorized(apitest.UploadRequest(t, metadata, file), session))
	if upload.Code != http.StatusAccepted {
		t.Fatalf("upload status = %d, want 202: %s", upload.Code, upload.Body.String())
	}

	if processed, err := assets.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("process ingest = %v, %v; want true, nil", processed, err)
	}
	poll := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, upload.Header().Get("Location"), nil), session,
	))
	var operation struct {
		Status  string `json:"status"`
		Failure *struct {
			Reason string `json:"reason"`
		} `json:"failure"`
	}
	if err := json.Unmarshal(poll.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode ingest: %v", err)
	}
	if operation.Status != "failed" || operation.Failure == nil || operation.Failure.Reason != "limit_exceeded" {
		t.Fatalf("operation = %#v, want a storage-limit failure", operation)
	}
	var assetCount int
	if err := pool.QueryRow(context.Background(), `select count(*) from assets`).Scan(&assetCount); err != nil {
		t.Fatalf("count assets: %v", err)
	}
	if assetCount != 0 {
		t.Fatalf("over-cap ingest recorded %d assets", assetCount)
	}
}

func zipCharacterCard(t *testing.T, card []byte) []byte {
	return zipCharacterCardWithFiles(t, card, nil)
}

func zipCharacterCardWithFiles(t *testing.T, card []byte, files map[string][]byte) []byte {
	t.Helper()
	var data bytes.Buffer
	archive := zip.NewWriter(&data)
	part, err := archive.Create("card.json")
	if err != nil {
		t.Fatalf("create card.json: %v", err)
	}
	if _, err := part.Write(card); err != nil {
		t.Fatalf("write card.json: %v", err)
	}
	for name, content := range files {
		part, err := archive.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close CharX: %v", err)
	}
	return data.Bytes()
}

func TestUnknownUploadIsRefusedAndNothingIsStored(t *testing.T) {
	t.Parallel()
	registry := format.NewRegistry()
	if err := registry.Register(neverClaimsModule{}); err != nil {
		t.Fatalf("register non-claiming module: %v", err)
	}
	r, session, assets := harness.NewVerifiedIngestRouter(t, registry)
	metadata := apitest.ExampleMetadata("Unknown")
	metadata["filename"] = "velvet-night.bundle"
	upload := apitest.Send(t, r, apitest.Authorized(
		apitest.UploadRequest(t, metadata, []byte("unrecognised bytes")), session,
	))
	if processed, err := assets.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("process ingest = %v, %v; want true, nil", processed, err)
	}

	poll := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, upload.Header().Get("Location"), nil), session,
	))
	var operation struct {
		Status  string `json:"status"`
		Asset   any    `json:"asset"`
		Failure *struct {
			Reason  string `json:"reason"`
			Message string `json:"message"`
		} `json:"failure"`
	}
	if err := json.Unmarshal(poll.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode refused operation: %v", err)
	}
	if operation.Status != "failed" || operation.Asset != nil || operation.Failure == nil {
		t.Fatalf("operation = %#v, want a refusal with no asset", operation)
	}
	if operation.Failure.Reason != "unsupported_format" ||
		!strings.Contains(operation.Failure.Message, "start from nothing") {
		t.Errorf("failure = %+v", operation.Failure)
	}
}

func TestClaimedFileThatFailsToParseIsRejectedWithoutAnAsset(t *testing.T) {
	t.Parallel()
	registry := format.NewRegistry()
	if err := registry.Register(parseFailureModule{}); err != nil {
		t.Fatalf("register module: %v", err)
	}
	r, session, assets := harness.NewVerifiedIngestRouter(t, registry)
	metadata := apitest.ExampleMetadata("Broken card")
	metadata["filename"] = "broken.json"
	upload := apitest.Send(t, r, apitest.Authorized(
		apitest.UploadRequest(t, metadata, []byte(`{"payload":true}`)), session,
	))
	if processed, err := assets.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("process ingest = %v, %v; want true, nil", processed, err)
	}

	poll := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, upload.Header().Get("Location"), nil), session,
	))
	var operation struct {
		Status  string `json:"status"`
		Failure *struct {
			Reason string `json:"reason"`
		} `json:"failure"`
		Asset any `json:"asset"`
	}
	if err := json.Unmarshal(poll.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode failed operation: %v", err)
	}
	if operation.Status != "failed" || operation.Failure == nil ||
		operation.Failure.Reason != "malformed_input" {
		t.Fatalf("operation = %#v, want malformed_input", operation)
	}
	if operation.Asset != nil {
		t.Fatalf("failed ingest returned asset %#v", operation.Asset)
	}
	if listed := apitest.ListItems(t, r, "/v1/assets"); len(listed) != 0 {
		t.Fatalf("browse found %d assets after a failed parse, want none", len(listed))
	}
}

func TestTerminalIngestFailuresStayDistinct(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		registry   func(t *testing.T) *format.Registry
		file       []byte
		wantReason string
	}{
		{
			name: "unsupported format",
			registry: func(t *testing.T) *format.Registry {
				registry := format.NewRegistry()
				if err := registry.Register(neverClaimsModule{}); err != nil {
					t.Fatalf("register non-claiming module: %v", err)
				}
				return registry
			},
			file:       []byte(`{"spec":"future_card"}`),
			wantReason: "unsupported_format",
		},
		{
			name: "unsupported version",
			registry: func(t *testing.T) *format.Registry {
				registry := format.NewRegistry()
				err := registry.Register(typedParseFailureModule{
					err: format.UnsupportedVersion(errors.New("version 99")),
				})
				if err != nil {
					t.Fatalf("register module: %v", err)
				}
				return registry
			},
			file:       []byte(`{"payload":true}`),
			wantReason: "unsupported_version",
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			r, session, assets := harness.NewVerifiedIngestRouter(t, test.registry(t))
			metadata := apitest.ExampleMetadata("Refused")
			metadata["filename"] = "refused.json"
			upload := apitest.Send(t, r, apitest.Authorized(apitest.UploadRequest(t, metadata, test.file), session))
			if processed, err := assets.ProcessNextIngest(context.Background()); err != nil || !processed {
				t.Fatalf("process ingest = %v, %v; want true, nil", processed, err)
			}
			poll := apitest.Send(t, r, apitest.Authorized(
				httptest.NewRequest(http.MethodGet, upload.Header().Get("Location"), nil), session,
			))
			var operation struct {
				Status  string `json:"status"`
				Failure *struct {
					Reason string `json:"reason"`
				} `json:"failure"`
			}
			if err := json.Unmarshal(poll.Body.Bytes(), &operation); err != nil {
				t.Fatalf("decode operation: %v", err)
			}
			if operation.Status != "failed" || operation.Failure == nil ||
				operation.Failure.Reason != test.wantReason {
				t.Fatalf("operation = %#v, want %s", operation, test.wantReason)
			}
		})
	}
}

func makeZIP(t *testing.T, header *zip.FileHeader) []byte {
	t.Helper()
	var data bytes.Buffer
	archive := zip.NewWriter(&data)
	part, err := archive.CreateHeader(header)
	if err != nil {
		t.Fatalf("create ZIP entry: %v", err)
	}
	if _, err := part.Write([]byte("safe bytes")); err != nil {
		t.Fatalf("write ZIP entry: %v", err)
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close ZIP: %v", err)
	}
	return data.Bytes()
}

func encryptedZIP(t *testing.T) []byte {
	t.Helper()
	data := makeZIP(t, &zip.FileHeader{Name: "theme.json", Method: zip.Store})
	binary.LittleEndian.PutUint16(data[6:8], binary.LittleEndian.Uint16(data[6:8])|1)
	central := bytes.Index(data, []byte("PK\x01\x02"))
	if central < 0 {
		t.Fatal("ZIP has no central directory")
	}
	binary.LittleEndian.PutUint16(
		data[central+8:central+10],
		binary.LittleEndian.Uint16(data[central+8:central+10])|1,
	)
	return data
}

func TestArchiveStructuralFailuresAreReportedFromTheWorker(t *testing.T) {
	t.Parallel()
	symlink := &zip.FileHeader{Name: "theme.json", Method: zip.Store}
	symlink.SetMode(os.ModeSymlink | 0o777)
	cases := []struct {
		name       string
		file       func(t *testing.T) []byte
		wantReason string
		wantRule   string
	}{
		{"traversal", func(t *testing.T) []byte {
			return makeZIP(t, &zip.FileHeader{Name: "../escape", Method: zip.Store})
		}, "safety_violation", `"../escape" leads outside the archive`},
		{"absolute path", func(t *testing.T) []byte {
			return makeZIP(t, &zip.FileHeader{Name: "/escape", Method: zip.Store})
		}, "safety_violation", `"/escape" leads outside the archive`},
		{"symlink", func(t *testing.T) []byte { return makeZIP(t, symlink) }, "safety_violation", `"theme.json" is a symbolic link`},
		{"encrypted", encryptedZIP, "safety_violation", "is encrypted"},
		{"malformed", func(*testing.T) []byte { return []byte("PK\x03\x04broken") }, "malformed_input", ""},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			r, session, assets := harness.NewVerifiedIngestRouter(t, format.NewRegistry())
			metadata := apitest.ExampleMetadata("Unsafe theme")
			metadata["filename"] = "unsafe.lumitheme"
			upload := apitest.Send(t, r, apitest.Authorized(apitest.UploadRequest(t, metadata, test.file(t)), session))
			if processed, err := assets.ProcessNextIngest(context.Background()); err != nil || !processed {
				t.Fatalf("process ingest = %v, %v; want true, nil", processed, err)
			}
			poll := apitest.Send(t, r, apitest.Authorized(
				httptest.NewRequest(http.MethodGet, upload.Header().Get("Location"), nil), session,
			))
			var operation struct {
				Failure *struct {
					Reason  string `json:"reason"`
					Message string `json:"message"`
				} `json:"failure"`
			}
			if err := json.Unmarshal(poll.Body.Bytes(), &operation); err != nil {
				t.Fatalf("decode operation: %v", err)
			}
			if operation.Failure == nil || operation.Failure.Reason != test.wantReason {
				t.Fatalf("failure = %#v, want %s", operation.Failure, test.wantReason)
			}
			if !strings.Contains(operation.Failure.Message, test.wantRule) {
				t.Errorf("refusal = %q, want it to name the rule %q", operation.Failure.Message, test.wantRule)
			}
		})
	}
}

type internalFailureModule struct {
	failuresLeft int
}

type catalogModule struct{}

type invalidFinalizationModule struct{}

func (invalidFinalizationModule) ID() string { return "invalid_finalization" }
func (invalidFinalizationModule) Declaration() format.Declaration {
	return apitest.ReaderDeclaration("invalid_finalization", "not-a-kind")
}
func (invalidFinalizationModule) Claim(file probe.Inspection) (format.Claim, bool) {
	if len(file.Payloads) == 0 {
		return format.Claim{}, false
	}
	return format.CompatibilityClaim(file.Payloads[0]), true
}
func (invalidFinalizationModule) Parse(context.Context, probe.Inspection, format.Claim) (format.Parsed, error) {
	return format.Parsed{Kind: "not-a-kind", Format: "invalid_finalization"}, nil
}

func (catalogModule) ID() string { return "catalog" }
func (catalogModule) Declaration() format.Declaration {
	return apitest.ReaderDeclaration("catalog", "character")
}
func (catalogModule) Claim(file probe.Inspection) (format.Claim, bool) {
	if len(file.Payloads) == 0 {
		return format.Claim{}, false
	}
	return format.CompatibilityClaim(file.Payloads[0]), true
}
func (catalogModule) Parse(context.Context, probe.Inspection, format.Claim) (format.Parsed, error) {
	nsfw := true
	return format.Parsed{
		Kind: "character", Format: "catalog",
		Header: format.Header{
			Name: "Moonlit Visitor", Blurb: "A quiet visitor from the edge of the wood.",
		},
		Tags: []string{"folklore", "gentle"}, IsNSFW: &nsfw,
		Elements: []block.Element{
			{Type: block.TypeProse, Role: block.RoleDescription, Content: block.Prose{Text: "A quiet visitor."}},
			{Type: block.TypeTextSet, Role: block.RoleGreetings, Content: block.TextSet{Texts: []block.TextItem{{ID: block.NewItemID(), Text: "Good evening."}}}},
		},
	}, nil
}

func (*internalFailureModule) ID() string { return "internal_failure" }
func (*internalFailureModule) Declaration() format.Declaration {
	return apitest.ReaderDeclaration("internal_failure", "character")
}
func (*internalFailureModule) Claim(file probe.Inspection) (format.Claim, bool) {
	if len(file.Payloads) == 0 {
		return format.Claim{}, false
	}
	return format.CompatibilityClaim(file.Payloads[0]), true
}
func (m *internalFailureModule) Parse(context.Context, probe.Inspection, format.Claim) (format.Parsed, error) {
	if m.failuresLeft > 0 {
		m.failuresLeft--
		return format.Parsed{}, format.InternalFailure(errors.New("temporary module failure"))
	}
	return format.Parsed{Kind: "character", Format: "internal_failure"}, nil
}

func TestOnlyInternalFailuresRetry(t *testing.T) {
	t.Parallel()
	settings := asset.DefaultIngestSettings()
	settings.RetryBase = 0
	settings.MaxAttempts = 2
	module := &internalFailureModule{failuresLeft: 1}
	registry := format.NewRegistry()
	if err := registry.Register(module); err != nil {
		t.Fatalf("register module: %v", err)
	}
	r, session, assets, _ := harness.NewVerifiedIngestRouterWithSettings(t, registry, settings)
	metadata := apitest.ExampleMetadata("Recovered card")
	metadata["filename"] = "recovered.json"
	upload := apitest.Send(t, r, apitest.Authorized(
		apitest.UploadRequest(t, metadata, []byte(`{"payload":true}`)), session,
	))

	if processed, err := assets.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("first process = %v, %v; want true, nil", processed, err)
	}
	firstPoll := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, upload.Header().Get("Location"), nil), session,
	))
	var first struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(firstPoll.Body.Bytes(), &first); err != nil {
		t.Fatalf("decode first poll: %v", err)
	}
	if first.Status != "pending" {
		t.Fatalf("status after retryable failure = %q, want pending", first.Status)
	}

	if processed, err := assets.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("second process = %v, %v; want true, nil", processed, err)
	}
	secondPoll := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, upload.Header().Get("Location"), nil), session,
	))
	var second struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(secondPoll.Body.Bytes(), &second); err != nil {
		t.Fatalf("decode second poll: %v", err)
	}
	if second.Status != "success" {
		t.Fatalf("status after retry = %q, want success. body: %s", second.Status, secondPoll.Body.String())
	}
}

func TestExhaustedInternalFailureIsReported(t *testing.T) {
	t.Parallel()
	settings := asset.DefaultIngestSettings()
	settings.RetryBase = 0
	settings.MaxAttempts = 2
	registry := format.NewRegistry()
	if err := registry.Register(&internalFailureModule{failuresLeft: 3}); err != nil {
		t.Fatalf("register module: %v", err)
	}
	r, session, assets, _ := harness.NewVerifiedIngestRouterWithSettings(t, registry, settings)
	metadata := apitest.ExampleMetadata("Still broken")
	metadata["filename"] = "broken.json"
	upload := apitest.Send(t, r, apitest.Authorized(
		apitest.UploadRequest(t, metadata, []byte(`{"payload":true}`)), session,
	))
	for attempt := 0; attempt < settings.MaxAttempts; attempt++ {
		if processed, err := assets.ProcessNextIngest(context.Background()); err != nil || !processed {
			t.Fatalf("process %d = %v, %v; want true, nil", attempt+1, processed, err)
		}
	}
	poll := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, upload.Header().Get("Location"), nil), session,
	))
	var operation struct {
		Failure *struct {
			Reason string `json:"reason"`
		} `json:"failure"`
	}
	if err := json.Unmarshal(poll.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode operation: %v", err)
	}
	if operation.Failure == nil || operation.Failure.Reason != "internal_failure" {
		t.Fatalf("failure = %#v, want internal_failure", operation.Failure)
	}
}

func TestAClaimedKindWithoutABlockCatalogIsRefused(t *testing.T) {
	t.Parallel()
	settings := asset.DefaultIngestSettings()
	settings.RetryBase = 0
	settings.MaxAttempts = 2
	registry := format.NewRegistry()
	if err := registry.Register(invalidFinalizationModule{}); err != nil {
		t.Fatalf("register module: %v", err)
	}
	r, session, assets, _ := harness.NewVerifiedIngestRouterWithSettings(t, registry, settings)
	metadata := apitest.ExampleMetadata("Invalid finalization")
	metadata["filename"] = "invalid.json"
	upload := apitest.Send(t, r, apitest.Authorized(
		apitest.UploadRequest(t, metadata, []byte(`{"payload":true}`)), session,
	))

	if processed, err := assets.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("process = %v, %v; want true, nil", processed, err)
	}
	poll := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, upload.Header().Get("Location"), nil), session,
	))
	var operation struct {
		Status  string `json:"status"`
		Failure *struct {
			Reason string `json:"reason"`
		} `json:"failure"`
	}
	if err := json.Unmarshal(poll.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode operation: %v", err)
	}
	if operation.Status != "failed" || operation.Failure == nil ||
		operation.Failure.Reason != "unsupported_format" {
		t.Fatalf("operation = %#v, want an unsupported-format refusal", operation)
	}
}

func TestCatalogMetadataSeedsFromParseWithoutChangingTheFile(t *testing.T) {
	t.Parallel()
	registry := format.NewRegistry()
	if err := registry.Register(catalogModule{}); err != nil {
		t.Fatalf("register module: %v", err)
	}
	r, session, assets := harness.NewVerifiedIngestRouter(t, registry)
	file := []byte(`{"payload":"original"}`)
	upload := apitest.Send(t, r, apitest.Authorized(apitest.UploadRequest(t, map[string]any{
		"filename": "visitor.json", "confirmed": true,
	}, file), session))
	if processed, err := assets.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("process ingest = %v, %v; want true, nil", processed, err)
	}
	poll := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, upload.Header().Get("Location"), nil), session,
	))
	var operation struct {
		Asset *struct {
			ID     string   `json:"id"`
			Name   string   `json:"name"`
			Blurb  string   `json:"blurb"`
			Tags   []string `json:"tags"`
			IsNSFW bool     `json:"isNsfw"`
		} `json:"asset"`
	}
	if err := json.Unmarshal(poll.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode operation: %v", err)
	}
	if operation.Asset == nil || operation.Asset.Name != "Moonlit Visitor" ||
		operation.Asset.Blurb != "A quiet visitor from the edge of the wood." ||
		!operation.Asset.IsNSFW || strings.Join(operation.Asset.Tags, ",") != "folklore,gentle" {
		t.Fatalf("asset metadata = %#v, want the parsed catalog seed", operation.Asset)
	}

	assetID, err := uuid.Parse(operation.Asset.ID)
	if err != nil {
		t.Fatalf("parse asset id: %v", err)
	}
	published := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodPost, "/v1/assets/"+operation.Asset.ID+"/publish", nil,
	), session))
	if published.Code != http.StatusOK {
		t.Fatalf("publish imported asset = %d: %s", published.Code, published.Body.String())
	}
	stored, err := assets.OpenSource(context.Background(), assetID)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	got, err := io.ReadAll(stored)
	stored.Close()
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if !bytes.Equal(got, file) {
		t.Fatalf("stored source = %q, want the original bytes", got)
	}
}

type blockingModule struct {
	started chan struct{}
	release chan struct{}
}

func (*blockingModule) ID() string { return "blocking" }
func (*blockingModule) Declaration() format.Declaration {
	return apitest.ReaderDeclaration("blocking", "character")
}
func (*blockingModule) Claim(file probe.Inspection) (format.Claim, bool) {
	if len(file.Payloads) == 0 {
		return format.Claim{}, false
	}
	return format.CompatibilityClaim(file.Payloads[0]), true
}
func (m *blockingModule) Parse(context.Context, probe.Inspection, format.Claim) (format.Parsed, error) {
	close(m.started)
	<-m.release
	return format.Parsed{Kind: "character", Format: "blocking"}, nil
}

func TestPollingReportsProcessingWhileAWorkerHoldsTheLease(t *testing.T) {
	t.Parallel()
	module := &blockingModule{started: make(chan struct{}), release: make(chan struct{})}
	registry := format.NewRegistry()
	if err := registry.Register(module); err != nil {
		t.Fatalf("register module: %v", err)
	}
	r, session, assets := harness.NewVerifiedIngestRouter(t, registry)
	metadata := apitest.ExampleMetadata("Patient card")
	metadata["filename"] = "patient.json"
	upload := apitest.Send(t, r, apitest.Authorized(
		apitest.UploadRequest(t, metadata, []byte(`{"payload":true}`)), session,
	))
	done := make(chan error, 1)
	go func() {
		_, err := assets.ProcessNextIngest(context.Background())
		done <- err
	}()
	<-module.started

	poll := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, upload.Header().Get("Location"), nil), session,
	))
	var operation struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(poll.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode operation: %v", err)
	}
	if operation.Status != "processing" {
		t.Fatalf("status = %q, want processing", operation.Status)
	}
	close(module.release)
	if err := <-done; err != nil {
		t.Fatalf("finish ingest: %v", err)
	}
}

func TestIngestContinuesAfterTheUploadConnectionCloses(t *testing.T) {
	t.Parallel()
	r, session, assets := harness.NewVerifiedIngestRouter(t, format.NewRegistry())
	workerContext, stopWorkers := context.WithCancel(context.Background())
	workersDone := make(chan struct{})
	go func() {
		assets.RunIngestWorkers(workerContext, 1, nil)
		close(workersDone)
	}()
	defer func() {
		stopWorkers()
		<-workersDone
	}()

	metadata := apitest.ExampleMetadata("Background theme")
	metadata["filename"] = "background.lumitheme"
	requestContext, closeConnection := context.WithCancel(context.Background())
	req := apitest.UploadRequest(t, metadata, []byte("theme bytes")).WithContext(requestContext)
	upload := apitest.Send(t, r, apitest.Authorized(req, session))
	closeConnection()

	deadline := time.Now().Add(3 * time.Second)
	for {
		poll := apitest.Send(t, r, apitest.Authorized(
			httptest.NewRequest(http.MethodGet, upload.Header().Get("Location"), nil), session,
		))
		var operation struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(poll.Body.Bytes(), &operation); err != nil {
			t.Fatalf("decode operation: %v", err)
		}
		if operation.Status == "success" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("operation stayed %q after the request closed", operation.Status)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func revisionRequest(t *testing.T, assetID, filename string, file []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	apitest.WriteFilePartNamed(t, form, filename, file)
	if err := form.Close(); err != nil {
		t.Fatalf("close form: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/assets/"+assetID+"/revisions", body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	return req
}

func TestARevisionUploadKeepsThePublishedBytesAndCatalogEntry(t *testing.T) {
	t.Parallel()
	r, session, assets := harness.NewVerifiedIngestRouter(t, format.NewRegistry())
	metadata := apitest.ExampleMetadata("Evening Theme")
	metadata["filename"] = "evening.lumitheme"
	upload := apitest.Send(t, r, apitest.Authorized(apitest.UploadRequest(t, metadata, []byte("first bytes")), session))
	if _, err := assets.ProcessNextIngest(context.Background()); err != nil {
		t.Fatalf("process ingest: %v", err)
	}
	created := pollIngestAsset(t, r, session, upload.Header().Get("Location"))
	published := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodPost, "/v1/assets/"+created.ID+"/publish", nil,
	), session))
	if published.Code != http.StatusOK {
		t.Fatalf("publish initial import = %d: %s", published.Code, published.Body.String())
	}
	firstFile := servedSourcePath(t, r, created.ID)

	revision := apitest.Send(t, r, apitest.Authorized(
		revisionRequest(t, created.ID, "evening.lumitheme", []byte("second bytes")), session,
	))
	if revision.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202. body: %s", revision.Code, revision.Body.String())
	}
	if _, err := assets.ProcessNextIngest(context.Background()); err != nil {
		t.Fatalf("process revision: %v", err)
	}
	acceptReplacementPreview(t, r, session, created.ID, revision.Header().Get("Location"))
	updated := pollIngestAsset(t, r, session, revision.Header().Get("Location"))
	if updated.ID != created.ID {
		t.Fatalf("revision made asset %s, want %s", updated.ID, created.ID)
	}
	if updated.Name != created.Name {
		t.Fatalf("name = %q, want the creator's own %q", updated.Name, created.Name)
	}

	if servedSourcePath(t, r, created.ID) != firstFile {
		t.Fatal("the private replacement changed the published source")
	}
}

func servedSourcePath(t *testing.T, r *gin.Engine, assetID string) string {
	t.Helper()
	rec := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/download/"+assetID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("download status = %d, want 200", rec.Code)
	}
	path := rec.Header().Get("X-Accel-Redirect")
	if path == "" {
		t.Fatal("X-Accel-Redirect is missing")
	}
	return path
}

func TestARevisionForSomebodyElsesAssetIsNotFound(t *testing.T) {
	t.Parallel()
	r, session, assets := harness.NewVerifiedIngestRouter(t, format.NewRegistry())
	metadata := apitest.ExampleMetadata("Evening Theme")
	metadata["filename"] = "evening.lumitheme"
	upload := apitest.Send(t, r, apitest.Authorized(apitest.UploadRequest(t, metadata, []byte("first bytes")), session))
	if _, err := assets.ProcessNextIngest(context.Background()); err != nil {
		t.Fatalf("process ingest: %v", err)
	}
	created := pollIngestAsset(t, r, session, upload.Header().Get("Location"))

	stranger := apitest.Send(t, r, revisionRequest(t, created.ID, "evening.lumitheme", []byte("second")))
	if stranger.Code != http.StatusUnauthorized {
		t.Fatalf("signed-out status = %d, want 401", stranger.Code)
	}
}

func pollIngestAsset(t *testing.T, r *gin.Engine, session *http.Cookie, location string) struct {
	ID   string `json:"id"`
	Name string `json:"name"`
} {
	t.Helper()
	rec := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(http.MethodGet, location, nil), session))
	if rec.Code != http.StatusOK {
		t.Fatalf("poll status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}
	var operation struct {
		Status string `json:"status"`
		Asset  *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"asset"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode operation: %v", err)
	}
	if operation.Status != "success" || operation.Asset == nil {
		t.Fatalf("operation = %#v, want a successful asset", operation)
	}
	return *operation.Asset
}

func acceptReplacementPreview(t *testing.T, r *gin.Engine, session *http.Cookie, assetID, location string, exposeProtected ...bool) {
	t.Helper()
	preview := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(http.MethodGet, location, nil), session))
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
	request := httptest.NewRequest(http.MethodPost, "/v1/assets/"+assetID+"/revisions/"+operationID+"/accept", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	accepted := apitest.Send(t, r, apitest.Authorized(request, session))
	if accepted.Code != http.StatusOK {
		t.Fatalf("accept replacement preview = %d: %s", accepted.Code, accepted.Body.String())
	}
}
