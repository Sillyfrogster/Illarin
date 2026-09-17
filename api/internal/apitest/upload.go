package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
)

func ExampleMetadata(name string) map[string]any {
	return map[string]any{
		"filename":  name + ".bin",
		"name":      name,
		"confirmed": true,
		"discovery": "listed",
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

	req := httptest.NewRequest(http.MethodPost, "/v1/assets", body)
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
	assets *asset.Service,
	metadata map[string]any,
	file []byte,
) *httptest.ResponseRecorder {
	t.Helper()
	accepted := Send(t, r, Authorized(UploadRequest(t, metadata, file), session))
	if accepted.Code != http.StatusAccepted {
		t.Fatalf("upload status = %d, want 202. body: %s", accepted.Code, accepted.Body.String())
	}
	if processed, err := assets.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("process ingest = %v, %v; want true, nil", processed, err)
	}
	finished := Send(t, r, Authorized(
		httptest.NewRequest(http.MethodGet, accepted.Header().Get("Location"), nil), session,
	))
	if keep, _ := metadata["_keepDraft"].(bool); !keep {
		var operation struct {
			Status string `json:"status"`
			Asset  *struct {
				ID string `json:"id"`
			} `json:"asset"`
		}
		if json.Unmarshal(finished.Body.Bytes(), &operation) == nil &&
			operation.Status == "success" && operation.Asset != nil {
			_ = Send(t, r, Authorized(httptest.NewRequest(
				http.MethodPost, "/v1/assets/"+operation.Asset.ID+"/publish", nil,
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

func MediaUploadRequest(t *testing.T, assetID, role string, file []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	WriteMetadataPart(t, form, map[string]any{"role": role})
	WriteFilePartNamed(t, form, "screenshot.png", file)
	if err := form.Close(); err != nil {
		t.Fatalf("close media form: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/assets/"+assetID+"/media", &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	return request
}

func AssetIDFromIngest(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var operation struct {
		Asset *struct {
			ID string `json:"id"`
		} `json:"asset"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode ingest response: %v", err)
	}
	if operation.Asset == nil {
		t.Fatal("ingest response has no asset")
	}
	return operation.Asset.ID
}

func UploadDiscoveryTestAsset(
	t *testing.T,
	router http.Handler,
	session *http.Cookie,
	assets *asset.Service,
	discovery asset.Discovery,
) string {
	t.Helper()
	metadata := ExampleMetadata("A quiet draft")
	metadata["filename"] = "quiet-draft.lumitheme"
	if discovery == "" {
		delete(metadata, "discovery")
	} else {
		metadata["discovery"] = discovery
	}
	return AssetIDFromIngest(
		t, UploadAndFinish(t, router, session, assets, metadata, []byte("theme")),
	)
}

func UploadedImageID(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	assetID, role string,
	file []byte,
) string {
	t.Helper()
	added := Send(t, r, Authorized(MediaUploadRequest(t, assetID, role, file), session))
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
