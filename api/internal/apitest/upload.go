package apitest

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/upload"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func ExampleMetadata(name string) map[string]any {
	return map[string]any{
		"filename":   name + ".bin",
		"name":       name,
		"confirmed":  true,
		"visibility": "listed",
	}
}

func UploadRequest(t *testing.T, metadata map[string]any, file []byte) *http.Request {
	t.Helper()

	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	filename, _ := metadata["filename"].(string)
	WriteMetadataPart(t, form, metadata)
	WriteFilePartNamed(t, form, filename, file)
	if err := form.Close(); err != nil {
		t.Fatalf("close form: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/works", body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	return req
}

func WriteMetadataPart(t *testing.T, form *multipart.Writer, metadata map[string]any) {
	t.Helper()
	fields := make(map[string]any, len(metadata))
	for key, value := range metadata {
		if key != "filename" && !strings.HasPrefix(key, "_") {
			fields[key] = value
		}
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		t.Fatalf("encode metadata: %v", err)
	}
	if err := form.WriteField("metadata", string(encoded)); err != nil {
		t.Fatalf("write metadata part: %v", err)
	}
}

func WriteFilePartNamed(t *testing.T, form *multipart.Writer, filename string, file []byte) {
	t.Helper()
	part, err := form.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := part.Write(file); err != nil {
		t.Fatalf("write file part: %v", err)
	}
}

func UploadAndFinish(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	works *work.Service,
	metadata map[string]any,
	file []byte,
) *httptest.ResponseRecorder {
	t.Helper()
	accepted := Send(t, r, Authorized(UploadRequest(t, metadata, file), session))
	if accepted.Code != http.StatusAccepted {
		t.Fatalf("upload status = %d, want 202. body: %s", accepted.Code, accepted.Body.String())
	}
	if processed, err := Uploads(works).ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("process ingest = %v, %v; want true, nil", processed, err)
	}
	finished := Send(t, r, Authorized(
		httptest.NewRequest(http.MethodGet, accepted.Header().Get("Location"), nil), session,
	))
	if keep, _ := metadata["_keepDraft"].(bool); !keep {
		var operation struct {
			Status string `json:"status"`
			Work   *struct {
				ID string `json:"id"`
			} `json:"work"`
		}
		if json.Unmarshal(finished.Body.Bytes(), &operation) == nil &&
			operation.Status == "success" && operation.Work != nil {
			_ = Send(t, r, Authorized(httptest.NewRequest(
				http.MethodPost, "/v1/works/"+operation.Work.ID+"/publish", nil,
			), session))
		}
	}
	return finished
}

func PNG(t *testing.T, width, height int) []byte {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			picture.Set(x, y, color.RGBA{R: 20, G: 60, B: 100, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, picture); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	return encoded.Bytes()
}

func MediaUploadRequest(t *testing.T, workID, role string, file []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	WriteMetadataPart(t, form, map[string]any{"role": role})
	WriteFilePartNamed(t, form, "screenshot.png", file)
	if err := form.Close(); err != nil {
		t.Fatalf("close media form: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/works/"+workID+"/media", &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	return request
}

func WorkIDFromIngest(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var operation struct {
		Work *struct {
			ID string `json:"id"`
		} `json:"work"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode ingest response: %v", err)
	}
	if operation.Work == nil {
		t.Fatalf("ingest response has no work: %s", response.Body.String())
	}
	return operation.Work.ID
}

func UploadVisibilityTestWork(
	t *testing.T,
	router http.Handler,
	session *http.Cookie,
	works *work.Service,
	visibility work.Visibility,
) string {
	t.Helper()
	metadata := ExampleMetadata("A quiet draft")
	metadata["filename"] = "quiet-draft.lumitheme"
	if visibility == "" {
		delete(metadata, "visibility")
	} else {
		metadata["visibility"] = visibility
	}
	return WorkIDFromIngest(
		t, UploadAndFinish(t, router, session, works, metadata, []byte("theme")),
	)
}

func UploadedImageID(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	workID, role string,
	file []byte,
) string {
	t.Helper()
	added := Send(t, r, Authorized(MediaUploadRequest(t, workID, role, file), session))
	if added.Code != http.StatusCreated {
		t.Fatalf("add a %s image: %d %s", role, added.Code, added.Body.String())
	}
	var picture struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(added.Body.Bytes(), &picture); err != nil {
		t.Fatalf("decode the added picture: %v", err)
	}
	return picture.ID
}

// Uploads reads files in over the same database and store as works
func Uploads(works *work.Service) *upload.Service {
	return upload.NewService(works.Pool(), works)
}

// RevisionRequest uploads file as a new version of the work
func RevisionRequest(t *testing.T, workID, filename string, file []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	WriteFilePartNamed(t, form, filename, file)
	if err := form.Close(); err != nil {
		t.Fatalf("close form: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/works/"+workID+"/revisions", body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	return req
}

// UploadExtension uploads an extension archive and leaves it a draft
func UploadExtension(t *testing.T, r http.Handler, session *http.Cookie, works *work.Service, file []byte) string {
	t.Helper()
	metadata := ExampleMetadata("Quiet Toolbox")
	metadata["filename"] = "toolbox.zip"
	metadata["_keepDraft"] = true
	finished := UploadAndFinish(t, r, session, works, metadata, file)
	if !strings.Contains(finished.Body.String(), `"success"`) {
		t.Fatalf("extension ingest did not succeed: %s", finished.Body.String())
	}
	return WorkIDFromIngest(t, finished)
}

// ExtensionZip packs files into a ZIP archive
func ExtensionZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var file bytes.Buffer
	archive := zip.NewWriter(&file)
	for name, content := range files {
		entry, err := archive.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
		if err != nil {
			t.Fatalf("create %q: %v", name, err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("write %q: %v", name, err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	return file.Bytes()
}

// PollIngestWork reads a finished upload and fails the test unless it made a work
func PollIngestWork(t *testing.T, r *gin.Engine, session *http.Cookie, location string) struct {
	ID   string `json:"id"`
	Name string `json:"name"`
} {
	t.Helper()
	rec := Send(t, r, Authorized(httptest.NewRequest(http.MethodGet, location, nil), session))
	if rec.Code != http.StatusOK {
		t.Fatalf("poll status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}
	var operation struct {
		Status string `json:"status"`
		Work   *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"work"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode operation: %v", err)
	}
	if operation.Status != "success" || operation.Work == nil {
		t.Fatalf("operation = %#v, want a successful asset", operation)
	}
	return *operation.Work
}

// ToolboxManifest is the manifest of a small SillyTavern extension
const ToolboxManifest = `{
	"version": "1.0.0",
	"name": "Quiet Toolbox",
	"identifier": "quiet_toolbox",
	"author": "A developer",
	"github": "https://github.com/example/quiet_toolbox",
	"homepage": "https://github.com/example/quiet_toolbox",
	"description": "Small tools for a calmer chat.",
	"permissions": ["ui_panels", "generation"],
	"entry_frontend": "dist/frontend.js",
	"minimum_lumiverse_version": "0.1.0"
}`

// PublishExtension gives an uploaded extension its catalog details and publishes it
func PublishExtension(t *testing.T, r http.Handler, session *http.Cookie, works *work.Service, name string, file []byte) string {
	t.Helper()
	workID := UploadExtension(t, r, session, works, file)
	identity := fmt.Sprintf(`{"name":%q,"blurb":"","isNsfw":false}`, name)
	if saved := SaveIdentity(t, r, session, workID, identity); saved.Code != http.StatusNoContent {
		t.Fatalf("save identity = %d: %s", saved.Code, saved.Body.String())
	}
	if published := PublishWork(t, r, session, workID); published.Code != http.StatusOK {
		t.Fatalf("publish %s = %d: %s", name, published.Code, published.Body.String())
	}
	return workID
}
