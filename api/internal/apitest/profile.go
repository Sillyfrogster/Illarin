package apitest

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type PublicProfile struct {
	ID           string `json:"id"`
	Handle       string `json:"handle"`
	DisplayName  string `json:"displayName"`
	Biography    string `json:"biography"`
	ContactEmail string `json:"contactEmail"`
	Avatar       *struct {
		URL    string `json:"url"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"avatar"`
	Links []struct {
		Label   string `json:"label"`
		Address string `json:"address"`
	} `json:"links"`
}

func SaveProfile(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPut, "/v1/account/profile", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return Send(t, r, Authorized(request, session))
}

func AvatarUploadRequest(t *testing.T, file []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	WriteFilePartNamed(t, form, "avatar.png", file)
	if err := form.Close(); err != nil {
		t.Fatalf("close avatar form: %v", err)
	}
	request := httptest.NewRequest(http.MethodPut, "/v1/account/profile/avatar", &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	return request
}
