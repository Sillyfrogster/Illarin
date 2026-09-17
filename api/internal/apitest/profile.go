package apitest

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/upload"
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

type ProfileListingResponse struct {
	Items []struct {
		Name       string  `json:"name"`
		IsNsfw     *bool   `json:"isNsfw"`
		OwnerState *string `json:"ownerState"`
		Withhold   *struct {
			Reason string    `json:"reason"`
			At     time.Time `json:"at"`
		} `json:"withhold"`
	} `json:"items"`
	Total      int `json:"total"`
	Suppressed int `json:"suppressed"`
}

func CreateProfileAsset(
	t *testing.T,
	assets *asset.Service,
	ownerID uuid.UUID,
	name string,
	isNSFW bool,
	discovery asset.Discovery,
) uuid.UUID {
	t.Helper()
	created, err := Uploads(assets).Create(context.Background(), upload.CreateInput{
		OwnerID: ownerID, Kind: "theme", Filename: name + ".lumitheme",
		File: bytes.NewReader([]byte(name)), Name: name, IsNSFW: isNSFW,
		Discovery: discovery,
	})
	if err != nil {
		t.Fatalf("create profile asset %q: %v", name, err)
	}
	return created.ID
}
