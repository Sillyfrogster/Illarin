package http

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func writeFilePart(t *testing.T, form *multipart.Writer, file []byte) {
	t.Helper()
	apitest.WriteFilePartNamed(t, form, "upload.bin", file)
}

func TestUploadOverTheCeilingIsRefused(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouterWith(t, 64, api.DefaultDeadlines())

	rec := apitest.Send(t, r, apitest.Authorized(
		apitest.UploadRequest(t, apitest.ExampleMetadata("Huge"), bytes.Repeat([]byte("a"), 1024)), session,
	))

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413. body: %s", rec.Code, rec.Body.String())
	}
}

func TestUploadAtTheCeilingIsAccepted(t *testing.T) {
	t.Parallel()
	req := apitest.UploadRequest(t, apitest.ExampleMetadata("Exactly"), bytes.Repeat([]byte("a"), 1024))
	r, session := harness.NewVerifiedRouterWith(t, 1024, api.DefaultDeadlines())

	rec := apitest.Send(t, r, apitest.Authorized(req, session))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202: a file of exactly the ceiling fits. body: %s",
			rec.Code, rec.Body.String())
	}
}

func TestUploadIsCutOffWhenItsLengthIsUnknown(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouterWith(t, 512, api.DefaultDeadlines())

	req := apitest.UploadRequest(t, apitest.ExampleMetadata("Unstated"), bytes.Repeat([]byte("a"), 4096))
	req.ContentLength = -1
	apitest.Authorized(req, session)

	rec := apitest.Send(t, r, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413. body: %s", rec.Code, rec.Body.String())
	}
}

func formRequest(t *testing.T, write func(*multipart.Writer)) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	write(form)
	if err := form.Close(); err != nil {
		t.Fatalf("close form: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/assets", body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	return req
}

func TestFormDataThatCannotBeReadIsRefused(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		request func(t *testing.T) *http.Request
		says    string
	}{
		{
			name: "not form data at all",
			request: func(t *testing.T) *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/v1/assets",
					strings.NewReader("kind=character"))
				req.Header.Set("Content-Type", "text/plain")
				return req
			},
			says: "form data",
		},
		{
			name: "the metadata part is missing",
			request: func(t *testing.T) *http.Request {
				return formRequest(t, func(form *multipart.Writer) {
					writeFilePart(t, form, []byte("bytes"))
				})
			},
			says: metadataPart,
		},
		{
			name: "the file part is missing",
			request: func(t *testing.T) *http.Request {
				return formRequest(t, func(form *multipart.Writer) {
					apitest.WriteMetadataPart(t, form, apitest.ExampleMetadata("Fileless"))
				})
			},
			says: filePart,
		},
		{
			name: "the metadata part is not JSON",
			request: func(t *testing.T) *http.Request {
				return formRequest(t, func(form *multipart.Writer) {
					if err := form.WriteField(metadataPart, "kind=character"); err != nil {
						t.Fatalf("write metadata part: %v", err)
					}
					writeFilePart(t, form, []byte("bytes"))
				})
			},
			says: "JSON",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, session := harness.NewVerifiedRouter(t)

			rec := apitest.Send(t, r, apitest.Authorized(c.request(t), session))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400. body: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), c.says) {
				t.Errorf("refusal %s does not say what was wrong, expected it to mention %q",
					rec.Body.String(), c.says)
			}
		})
	}
}
