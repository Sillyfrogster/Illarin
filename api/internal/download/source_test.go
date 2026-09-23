package download_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

func rasterSources(t *testing.T) map[string][]byte {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, 1, 1))
	picture.Set(0, 0, color.RGBA{R: 48, G: 76, B: 115, A: 255})

	encode := func(write func(*bytes.Buffer) error) []byte {
		var out bytes.Buffer
		if err := write(&out); err != nil {
			t.Fatalf("encode raster fixture: %v", err)
		}
		return out.Bytes()
	}
	webp, err := base64.StdEncoding.DecodeString("UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA==")
	if err != nil {
		t.Fatalf("decode WebP fixture: %v", err)
	}
	return map[string][]byte{
		"image/png":  encode(func(out *bytes.Buffer) error { return png.Encode(out, picture) }),
		"image/jpeg": encode(func(out *bytes.Buffer) error { return jpeg.Encode(out, picture, nil) }),
		"image/gif":  encode(func(out *bytes.Buffer) error { return gif.Encode(out, picture, nil) }),
		"image/webp": webp,
	}
}

func TestDownloadHandsTheCurrentSourceToNginx(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewVerifiedUploadRouter(t, format.NewRegistry())

	original := []byte{0x00, 0xff, 0xfe, 0x10, 0x80}

	metadata := apitest.ExampleMetadata("Exact")
	metadata["filename"] = "exact.lumitheme"
	rec := apitest.UploadAndFinish(t, r, session, works, metadata, original)

	var created struct {
		Work *struct {
			ID string `json:"id"`
		} `json:"work"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.Work == nil {
		t.Fatal("completed upload has no work")
	}

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/download/"+created.Work.ID, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("download status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("X-Accel-Redirect"); got == "" {
		t.Fatal("X-Accel-Redirect is missing")
	}
	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename=exact.lumitheme` {
		t.Fatalf("Content-Disposition = %q, want the uploaded filename", got)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("Go returned %d bytes instead of handing the file to nginx", rec.Body.Len())
	}
}

func TestAPublicSourceDownloadRecordsTheAuthorizedHandoff(t *testing.T) {
	t.Parallel()
	router, session, works, pool := harness.NewVerifiedUploadRouterWithPool(t, format.NewRegistry())
	workID := apitest.UploadVisibilityTestWork(t, router, session, works, work.VisibilityListed)

	before := time.Now()
	download := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/download/"+workID, nil))
	after := time.Now()
	if download.Code != http.StatusOK || download.Header().Get("X-Accel-Redirect") == "" {
		t.Fatalf("download = %d, headers %v", download.Code, download.Header())
	}

	var originalFileID, currentOriginalFileID uuid.UUID
	var formatID, access, visibility string
	var handedOffAt time.Time
	err := pool.QueryRow(context.Background(), `
		select original_file_id, format, handed_off_at, access, visibility
		  from download_records
		 where work_id = $1
	`, workID).Scan(&originalFileID, &formatID, &handedOffAt, &access, &visibility)
	if err != nil {
		t.Fatalf("read download record: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `
		select original_file_id from works where id = $1
	`, workID).Scan(&currentOriginalFileID); err != nil {
		t.Fatalf("read the original file: %v", err)
	}
	if originalFileID != currentOriginalFileID || formatID != "raw" ||
		access != "public" || visibility != "listed" {
		t.Fatalf(
			"download record = original file %s, format %q, access %q, visibility %q",
			originalFileID, formatID, access, visibility,
		)
	}
	if handedOffAt.Before(before) || handedOffAt.After(after) {
		t.Fatalf("handoff time %s is outside request interval %s to %s", handedOffAt, before, after)
	}
	var recorded int
	if err := pool.QueryRow(context.Background(), `
		select count(*) from download_records where work_id = $1
	`, workID).Scan(&recorded); err != nil {
		t.Fatalf("count download records: %v", err)
	}
	if recorded != 1 {
		t.Fatalf("one handoff wrote %d download records", recorded)
	}
}

func TestExportFromAnWorkMadeInIllarinRecordsTheHandoff(t *testing.T) {
	t.Parallel()
	router, session, _, pool := harness.NewCharacterUploadRouterWithPool(t)
	started := apitest.StartCharacter(t, router, session)
	apitest.WriteCharacterFloor(t, router, session, started)
	if published := apitest.PublishWork(t, router, session, started.ID); published.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200: %s", published.Code, published.Body.String())
	}

	download := apitest.Send(t, router, httptest.NewRequest(
		http.MethodGet, "/download/"+started.ID+"/chara_card_v3", nil,
	))
	if download.Code != http.StatusOK {
		t.Fatalf("download status = %d, want 200: %s", download.Code, download.Body.String())
	}

	var originalFileMissing bool
	var formatID, access string
	if err := pool.QueryRow(context.Background(), `
		select original_file_id is null, format, access
		  from download_records
		 where work_id = $1
	`, started.ID).Scan(&originalFileMissing, &formatID, &access); err != nil {
		t.Fatalf("read download record: %v", err)
	}
	if !originalFileMissing || formatID != "chara_card_v3" || access != "public" {
		t.Fatalf("record = original file missing %t, format %q, access %q",
			originalFileMissing, formatID, access)
	}
}

type blockingRedirectStore struct {
	storage.Store
	reached chan struct{}
	release chan struct{}
}

func (s *blockingRedirectStore) InternalRedirect(ctx context.Context, id uuid.UUID) (string, error) {
	s.reached <- struct{}{}
	select {
	case <-s.release:
		return s.Store.InternalRedirect(ctx, id)
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func TestDownloadSnapshotsVisibilityAtHandoff(t *testing.T) {
	t.Parallel()
	var blocker *blockingRedirectStore
	router, session, works, pool := harness.NewVerifiedUploadRouterWithStore(
		t,
		format.NewRegistry(),
		work.DefaultUploadSettings(),
		func(store storage.Store) storage.Store {
			blocker = &blockingRedirectStore{
				Store: store, reached: make(chan struct{}, 1), release: make(chan struct{}),
			}
			return blocker
		},
	)
	workID := apitest.UploadVisibilityTestWork(t, router, session, works, work.VisibilityListed)

	response := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		response <- apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/download/"+workID, nil))
	}()
	select {
	case <-blocker.reached:
	case <-time.After(time.Second):
		t.Fatal("download did not reach the handoff boundary")
	}
	changed := apitest.Send(t, router, apitest.AuthorizedJSONRequest(
		t, http.MethodPut, "/v1/works/"+workID+"/visibility",
		`{"visibility":"unlisted"}`, session,
	))
	if changed.Code != http.StatusNoContent {
		t.Fatalf("change visibility status = %d, want 204", changed.Code)
	}
	close(blocker.release)
	if download := <-response; download.Code != http.StatusOK {
		t.Fatalf("download status = %d, want 200", download.Code)
	}

	var visibility string
	if err := pool.QueryRow(context.Background(), `
		select visibility from download_records where work_id = $1
	`, workID).Scan(&visibility); err != nil {
		t.Fatalf("read download record: %v", err)
	}
	if visibility != "unlisted" {
		t.Fatalf("visibility at handoff = %q, want unlisted", visibility)
	}
}

func TestExportDownloadRecordsTheFormatItHandedOver(t *testing.T) {
	t.Parallel()
	router, session, works, pool := harness.NewVerifiedUploadRouterWithPool(t, format.NewRegistry())
	workID := apitest.UploadVisibilityTestWork(t, router, session, works, work.VisibilityListed)

	download := apitest.Send(t, router, httptest.NewRequest(
		http.MethodGet, "/download/"+workID+"/test_opaque", nil,
	))
	if download.Code != http.StatusOK {
		t.Fatalf("download status = %d, want 200: %s", download.Code, download.Body.String())
	}
	if got := download.Header().Get("X-Illarin-Format"); got != "test_opaque" {
		t.Fatalf("format header = %q, want the format handed over", got)
	}
	if redirect := download.Header().Get("X-Accel-Redirect"); redirect != "" {
		t.Fatalf("a generated export was served from disk at %q", redirect)
	}
	if got := download.Header().Get("Content-Disposition"); !strings.Contains(got, ".txt") {
		t.Fatalf("disposition = %q, want a filename the format chose", got)
	}

	var formatID string
	if err := pool.QueryRow(context.Background(), `
		select format from download_records where work_id = $1
	`, workID).Scan(&formatID); err != nil {
		t.Fatalf("read download record: %v", err)
	}
	if formatID != "test_opaque" {
		t.Fatalf("recorded format = %q, want the format handed over", formatID)
	}
}

func TestAFormatTheWorkIsNotOfferedInIs404(t *testing.T) {
	t.Parallel()
	router, session, works := harness.NewVerifiedUploadRouter(t, format.NewRegistry())
	workID := apitest.UploadVisibilityTestWork(t, router, session, works, work.VisibilityListed)

	response := apitest.Send(t, router, httptest.NewRequest(
		http.MethodGet, "/download/"+workID+"/chara_card_v2", nil,
	))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", response.Code, response.Body.String())
	}
}

func TestADownloadRecordsOneExclusiveAccess(t *testing.T) {
	t.Parallel()
	router, ownerSession, works, pool := harness.NewVerifiedUploadRouterWithPool(t, format.NewRegistry())
	workID := apitest.UploadVisibilityTestWork(t, router, ownerSession, works, work.VisibilityListed)
	readerSession := apitest.SignUp(t, router, "reader@example.com", "signed.reader")

	requests := []*http.Request{
		httptest.NewRequest(http.MethodGet, "/download/"+workID, nil),
		apitest.Authorized(httptest.NewRequest(http.MethodGet, "/download/"+workID, nil), ownerSession),
		apitest.Authorized(httptest.NewRequest(http.MethodGet, "/download/"+workID, nil), readerSession),
	}
	for _, request := range requests {
		if response := apitest.Send(t, router, request); response.Code != http.StatusOK {
			t.Fatalf("download status = %d, want 200", response.Code)
		}
	}

	rows, err := pool.Query(context.Background(), `
		select access from download_records order by id
	`)
	if err != nil {
		t.Fatalf("read access values: %v", err)
	}
	defer rows.Close()
	var accesses []string
	for rows.Next() {
		var access string
		if err := rows.Scan(&access); err != nil {
			t.Fatalf("scan access: %v", err)
		}
		accesses = append(accesses, access)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read access values: %v", err)
	}
	if want := []string{"public", "owner", "public"}; !slices.Equal(accesses, want) {
		t.Fatalf("access values = %v, want %v", accesses, want)
	}
}

func TestDownloadSnapshotsUnlistedAndOwnerTakenDownWorks(t *testing.T) {
	t.Parallel()
	router, ownerSession, works, pool := harness.NewVerifiedUploadRouterWithPool(t, format.NewRegistry())
	unlistedID := apitest.UploadVisibilityTestWork(
		t, router, ownerSession, works, work.VisibilityUnlisted,
	)
	takenDownID := apitest.UploadVisibilityTestWork(
		t, router, ownerSession, works, work.VisibilityListed,
	)
	if _, err := pool.Exec(context.Background(), `
		update works work
		   set taken_down_at = now(), taken_down_by = owner.id, taken_down_reason = 'review'
		  from users owner
		 where work.id = $1 and owner.username = 'verified.creator'
	`, takenDownID); err != nil {
		t.Fatalf("take down work: %v", err)
	}

	unlisted := apitest.Send(t, router, httptest.NewRequest(
		http.MethodGet, "/download/"+unlistedID, nil,
	))
	takenDownRequest := apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/download/"+takenDownID, nil,
	), ownerSession)
	takenDown := apitest.Send(t, router, takenDownRequest)
	if unlisted.Code != http.StatusOK || takenDown.Code != http.StatusOK {
		t.Fatalf("download statuses = unlisted %d, taken down owner %d", unlisted.Code, takenDown.Code)
	}

	for _, want := range []struct {
		workID        string
		visibility    string
		authorization string
	}{
		{workID: unlistedID, visibility: "unlisted", authorization: "public"},
		{workID: takenDownID, visibility: "listed", authorization: "owner"},
	} {
		var visibility, authorization string
		if err := pool.QueryRow(context.Background(), `
			select visibility, access
			  from download_records
			 where work_id = $1
		`, want.workID).Scan(&visibility, &authorization); err != nil {
			t.Fatalf("read download record for %s: %v", want.workID, err)
		}
		if visibility != want.visibility || authorization != want.authorization {
			t.Fatalf(
				"record for %s = visibility %q, access %q; want %q, %q",
				want.workID, visibility, authorization, want.visibility, want.authorization,
			)
		}
	}
}

func TestDownloadUnknownWorkIs404(t *testing.T) {
	t.Parallel()
	r, pool := harness.NewRouterWithSenderAndPool(
		t, 1<<20, api.DefaultDeadlines(), &apitest.VerificationOutbox{},
	)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/download/11111111-1111-1111-1111-111111111111", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var recorded int
	if err := pool.QueryRow(context.Background(), `select count(*) from download_records`).Scan(&recorded); err != nil {
		t.Fatalf("count download records: %v", err)
	}
	if recorded != 0 {
		t.Fatalf("rejected request wrote %d download records", recorded)
	}
}

func TestPrivateBlockEditsKeepThePublishedDownloadAndUpload(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewCharacterUploadRouter(t)
	source := []byte(`{
		"spec":"chara_card_v3","spec_version":"3.0",
		"data":{"name":"Ana","description":"Before","first_mes":"Hello",
			"extensions": { "third_party": { "keep": true, "order": [3,1,2] } }}
	}`)
	metadata := apitest.ExampleMetadata("Ana")
	metadata["filename"] = "ana.json"
	workID := apitest.WorkIDFromUpload(t, apitest.UploadAndFinish(t, r, session, works, metadata, source))

	page := apitest.FetchStartedWork(t, r, session, workID)
	core := apitest.EditableBlock(apitest.BlockNamed(t, page.Blocks, "character_core"))
	core.Elements[0].Content = json.RawMessage(`{"text":"After"}`)
	saved := apitest.SaveBlock(t, r, session, workID, apitest.BlockNamed(t, page.Blocks, "character_core").ID, core)
	if saved.Code != http.StatusOK {
		t.Fatalf("save the description: %d %s", saved.Code, saved.Body.String())
	}

	download := apitest.Send(t, r, httptest.NewRequest(
		http.MethodGet, "/download/"+workID+"/chara_card_v3", nil,
	))
	if download.Code != http.StatusOK {
		t.Fatalf("download status = %d, want 200: %s", download.Code, download.Body.String())
	}
	var card struct {
		Data struct {
			Description string          `json:"description"`
			Extensions  json.RawMessage `json:"extensions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(download.Body.Bytes(), &card); err != nil {
		t.Fatalf("read the downloaded card: %v", err)
	}
	if card.Data.Description != "Before" {
		t.Fatalf("downloaded description = %q, want the published text", card.Data.Description)
	}
	var sourceCard struct {
		Data struct {
			Extensions json.RawMessage `json:"extensions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(source, &sourceCard); err != nil {
		t.Fatalf("read the source fixture: %v", err)
	}
	if !bytes.Equal(apitest.CompactJSON(t, card.Data.Extensions), apitest.CompactJSON(t, sourceCard.Data.Extensions)) {
		t.Fatalf("third-party extensions changed\n got: %s\nwant: %s",
			card.Data.Extensions, sourceCard.Data.Extensions)
	}

	storedSource, err := works.OpenSource(context.Background(), uuid.MustParse(workID))
	if err != nil {
		t.Fatalf("OpenSource: %v", err)
	}
	storedBytes, readErr := io.ReadAll(storedSource)
	closeErr := storedSource.Close()
	if readErr != nil || closeErr != nil || !bytes.Equal(storedBytes, source) {
		t.Fatalf("the upload changed: read %v, close %v", readErr, closeErr)
	}
}

func TestUnverifiedSourceTypeDownloadsAsAnOpaqueAttachment(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewVerifiedUploadRouter(t, format.NewRegistry())

	payload := []byte(`<script>alert(1)</script>`)

	metadata := apitest.ExampleMetadata("Evil")
	metadata["filename"] = "evil.lumitheme"
	rec := apitest.UploadAndFinish(t, r, session, works, metadata, payload)

	var created struct {
		Work *struct {
			ID string `json:"id"`
		} `json:"work"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.Work == nil {
		t.Fatal("completed upload has no work")
	}

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/download/"+created.Work.ID, nil))

	if got := rec.Header().Get("Content-Type"); got != "application/octet-stream" {
		t.Errorf("Content-Type = %q, want application/octet-stream", got)
	}
	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename=evil.lumitheme` {
		t.Errorf("Content-Disposition = %q, want an attachment with the uploaded filename", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "" {
		t.Errorf("Cache-Control = %q, download URLs must not be immutable", got)
	}
}

func TestProbeVerifiedRasterSourcesMayRenderInline(t *testing.T) {
	t.Parallel()
	for wantType, source := range rasterSources(t) {
		t.Run(wantType, func(t *testing.T) {
			r, session, works := harness.NewVerifiedUploadRouter(t, format.NewRegistry())
			metadata := apitest.ExampleMetadata("Raster")
			metadata["filename"] = "misleading.lumitheme"
			created := apitest.UploadAndFinish(t, r, session, works, metadata, source)

			var operation struct {
				Work *struct {
					ID string `json:"id"`
				} `json:"work"`
			}
			if err := json.Unmarshal(created.Body.Bytes(), &operation); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if operation.Work == nil {
				t.Fatal("completed upload has no work")
			}

			download := apitest.Send(t, r, httptest.NewRequest(
				http.MethodGet, "/download/"+operation.Work.ID, nil,
			))
			if download.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", download.Code)
			}
			if got := download.Header().Get("Content-Type"); got != wantType {
				t.Errorf("Content-Type = %q, want %q", got, wantType)
			}
			if got := download.Header().Get("Content-Disposition"); got != `inline; filename=misleading.lumitheme` {
				t.Errorf("Content-Disposition = %q, want inline with the uploaded filename", got)
			}
		})
	}
}

func TestFilenameExtensionAndDeclaredTypeCannotMakeAnUnknownSVGImportable(t *testing.T) {
	t.Parallel()
	registry := format.NewRegistry()
	if err := registry.Register(apitest.NeverMatchesModule{}); err != nil {
		t.Fatalf("register non-matching module: %v", err)
	}
	r, session, works := harness.NewVerifiedUploadRouter(t, registry)
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	apitest.WriteMetadataPart(t, form, apitest.ExampleMetadata("Matched image"))
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="matched.png"`)
	header.Set("Content-Type", "image/jpeg")
	part, err := form.CreatePart(header)
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := part.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script /></svg>`)); err != nil {
		t.Fatalf("write file part: %v", err)
	}
	if err := form.Close(); err != nil {
		t.Fatalf("close form: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/works", body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	accepted := apitest.Send(t, r, apitest.Authorized(request, session))
	if accepted.Code != http.StatusAccepted {
		t.Fatalf("upload status = %d, want 202", accepted.Code)
	}
	if processed, err := apitest.Uploads(works).ProcessNextUpload(request.Context()); err != nil || !processed {
		t.Fatalf("process upload = %v, %v; want true, nil", processed, err)
	}

	removedCompletion := apitest.Send(t, r, apitest.AuthorizedJSONRequest(
		t, http.MethodPatch, accepted.Header().Get("Location"),
		`{"type":"theme","name":"Matched image"}`, session,
	))
	if removedCompletion.Code != http.StatusNotFound {
		t.Fatalf("removed completion route status = %d, want 404", removedCompletion.Code)
	}
	poll := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, accepted.Header().Get("Location"), nil), session,
	))
	var operation struct {
		Failure *struct {
			Reason string `json:"reason"`
		} `json:"failure"`
	}
	if err := json.Unmarshal(poll.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if operation.Failure == nil || operation.Failure.Reason != "unsupported_format" {
		t.Fatalf("failure = %#v, want unsupported_format", operation.Failure)
	}
}
