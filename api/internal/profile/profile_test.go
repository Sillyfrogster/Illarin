package profile_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestOwnerFillsAPublicProfileAndAVisitorReadsIt(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)

	saved := apitest.SaveProfile(t, r, session, `{
		"displayName":"Wren Ashdown",
		"biography":"Writes lorebooks about weather.",
		"contactEmail":"hello@example.com",
		"links":[
			{"label":"Notes","address":"https://example.com/notes"},
			{"label":"Workshop","address":"https://example.org/workshop"}
		]
	}`)
	if saved.Code != http.StatusOK {
		t.Fatalf("save status = %d, want 200: %s", saved.Code, saved.Body.String())
	}

	visitor := readProfile(t, r, "verified.creator")
	if visitor.Handle != "verified.creator" || visitor.DisplayName != "Wren Ashdown" {
		t.Fatalf("profile = %+v", visitor)
	}
	if visitor.Biography != "Writes lorebooks about weather." {
		t.Fatalf("biography = %q", visitor.Biography)
	}
	if visitor.ContactEmail != "hello@example.com" {
		t.Fatalf("contact = %q", visitor.ContactEmail)
	}
	if len(visitor.Links) != 2 ||
		visitor.Links[0].Label != "Notes" ||
		visitor.Links[1].Address != "https://example.org/workshop" {
		t.Fatalf("links = %+v", visitor.Links)
	}
}

func TestClearingAProfileFieldRemovesItFromPublicView(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	apitest.SaveProfile(t, r, session, `{
		"displayName":"Wren Ashdown",
		"biography":"Writes lorebooks about weather.",
		"contactEmail":"hello@example.com",
		"links":[{"label":"Notes","address":"https://example.com/notes"}]
	}`)

	cleared := apitest.SaveProfile(t, r, session, `{
		"displayName":"",
		"biography":"",
		"contactEmail":"",
		"links":[]
	}`)
	if cleared.Code != http.StatusOK {
		t.Fatalf("clear status = %d, want 200: %s", cleared.Code, cleared.Body.String())
	}

	visitor := readProfile(t, r, "verified.creator")
	if visitor.DisplayName != "" || visitor.Biography != "" || visitor.ContactEmail != "" {
		t.Fatalf("cleared profile still shows fields: %+v", visitor)
	}
	if len(visitor.Links) != 0 {
		t.Fatalf("cleared profile still shows links: %+v", visitor.Links)
	}
	if visitor.Handle != "verified.creator" {
		t.Fatalf("handle changed to %q", visitor.Handle)
	}
}

func TestPublicContactIsNeverTakenFromASignInAddress(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)

	apitest.SaveProfile(t, r, session, `{
		"displayName":"Wren","biography":"","contactEmail":"","links":[]
	}`)

	visitor := readProfile(t, r, "verified.creator")
	if visitor.ContactEmail != "" {
		t.Fatalf("public contact filled itself in as %q", visitor.ContactEmail)
	}
}

func TestPublicProfileHidesEverySignInAndAuthorityField(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	apitest.SaveProfile(t, r, session, `{
		"displayName":"Wren","biography":"","contactEmail":"hello@example.com","links":[]
	}`)

	response := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/v1/profiles/verified.creator", nil))
	body := response.Body.String()
	for _, private := range []string{
		"verified@example.com", "emailVerified", "discordLinked", "hasPassword", "role",
	} {
		if strings.Contains(body, private) {
			t.Fatalf("public profile exposed %q: %s", private, body)
		}
	}
}

func TestAnUnverifiedAccountCannotEditItsPublicProfile(t *testing.T) {
	t.Parallel()
	r := harness.NewRouter(t)
	session := apitest.SignUp(t, r, "unverified@example.com", "unverified.one")

	response := apitest.SaveProfile(t, r, session, `{
		"displayName":"Wren","biography":"","contactEmail":"","links":[]
	}`)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", response.Code, response.Body.String())
	}
}

func TestAProfileRefusesFieldsThatAreTooLongOrNotHTTPS(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	tests := []struct {
		name string
		body string
	}{
		{
			name: "display name over the limit",
			body: `{"displayName":"` + strings.Repeat("n", 49) +
				`","biography":"","contactEmail":"","links":[]}`,
		},
		{
			name: "biography over the limit",
			body: `{"displayName":"","biography":"` + strings.Repeat("b", 401) +
				`","contactEmail":"","links":[]}`,
		},
		{
			name: "contact that is not an address",
			body: `{"displayName":"","biography":"","contactEmail":"not-an-address","links":[]}`,
		},
		{
			name: "link that is not https",
			body: `{"displayName":"","biography":"","contactEmail":"","links":[` +
				`{"label":"Notes","address":"http://example.com"}]}`,
		},
		{
			name: "link without a label",
			body: `{"displayName":"","biography":"","contactEmail":"","links":[` +
				`{"label":"","address":"https://example.com"}]}`,
		},
		{
			name: "more links than a profile shows",
			body: `{"displayName":"","biography":"","contactEmail":"","links":[` +
				strings.TrimSuffix(strings.Repeat(
					`{"label":"Notes","address":"https://example.com"},`, 7), ",") + `]}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := apitest.SaveProfile(t, r, session, test.body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestAvatarBytesTravelTheSharedMediaPathAndAreNotCatalogMedia(t *testing.T) {
	t.Parallel()
	r, session, _, pool := harness.NewVerifiedIngestRouterWithPool(t, apitest.Registry(t))

	uploaded := apitest.Send(t, r, apitest.Authorized(apitest.AvatarUploadRequest(t, apitest.PNG(t, 400, 400)), session))
	if uploaded.Code != http.StatusOK {
		t.Fatalf("avatar status = %d, want 200: %s", uploaded.Code, uploaded.Body.String())
	}
	var profile apitest.PublicProfile
	if err := json.Unmarshal(uploaded.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decode avatar response: %v", err)
	}
	if profile.Avatar == nil || profile.Avatar.Width != 400 || profile.Avatar.Height != 400 {
		t.Fatalf("avatar = %+v", profile.Avatar)
	}

	image := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, profile.Avatar.URL, nil))
	if image.Code != http.StatusOK {
		t.Fatalf("avatar image status = %d, want 200: %s", image.Code, image.Body.String())
	}
	if image.Header().Get("X-Accel-Redirect") == "" {
		t.Fatalf("avatar image did not hand off to the byte server: %+v", image.Header())
	}
	if cache := image.Header().Get("Cache-Control"); cache != "public, max-age=31536000, immutable" {
		t.Fatalf("avatar cache control = %q", cache)
	}

	var workMedia int
	if err := pool.QueryRow(t.Context(), `select count(*) from work_media`).Scan(&workMedia); err != nil {
		t.Fatalf("count catalog media: %v", err)
	}
	if workMedia != 0 {
		t.Fatalf("the avatar became %d catalog media rows", workMedia)
	}
}

func TestReplacingAnAvatarRetiresTheOneItReplaced(t *testing.T) {
	t.Parallel()
	r, session, _, pool := harness.NewVerifiedIngestRouterWithPool(t, apitest.Registry(t))

	first := apitest.Send(t, r, apitest.Authorized(apitest.AvatarUploadRequest(t, apitest.PNG(t, 200, 200)), session))
	if first.Code != http.StatusOK {
		t.Fatalf("first avatar status = %d: %s", first.Code, first.Body.String())
	}
	var before apitest.PublicProfile
	if err := json.Unmarshal(first.Body.Bytes(), &before); err != nil {
		t.Fatalf("decode first avatar: %v", err)
	}

	second := apitest.Send(t, r, apitest.Authorized(apitest.AvatarUploadRequest(t, apitest.PNG(t, 300, 300)), session))
	if second.Code != http.StatusOK {
		t.Fatalf("second avatar status = %d: %s", second.Code, second.Body.String())
	}
	var after apitest.PublicProfile
	if err := json.Unmarshal(second.Body.Bytes(), &after); err != nil {
		t.Fatalf("decode second avatar: %v", err)
	}
	if after.Avatar == nil || after.Avatar.URL == before.Avatar.URL {
		t.Fatalf("replacement kept the old address: %+v", after.Avatar)
	}

	var remaining int
	if err := pool.QueryRow(t.Context(), `select count(*) from profile_media`).Scan(&remaining); err != nil {
		t.Fatalf("count profile media: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("profile media rows = %d, want 1", remaining)
	}
	gone := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, before.Avatar.URL, nil))
	if gone.Code != http.StatusNotFound {
		t.Fatalf("retired avatar status = %d, want 404", gone.Code)
	}
}

func TestRemovingAnAvatarLeavesTheProfileWithoutOne(t *testing.T) {
	t.Parallel()
	r, session, _, _ := harness.NewVerifiedIngestRouterWithPool(t, apitest.Registry(t))

	apitest.Send(t, r, apitest.Authorized(apitest.AvatarUploadRequest(t, apitest.PNG(t, 200, 200)), session))
	removed := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/account/profile/avatar", nil), session,
	))
	if removed.Code != http.StatusOK {
		t.Fatalf("remove status = %d, want 200: %s", removed.Code, removed.Body.String())
	}
	var profile apitest.PublicProfile
	if err := json.Unmarshal(removed.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decode removal response: %v", err)
	}
	if profile.Avatar != nil {
		t.Fatalf("avatar survived removal: %+v", profile.Avatar)
	}

	again := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/account/profile/avatar", nil), session,
	))
	if again.Code != http.StatusNotFound {
		t.Fatalf("second removal status = %d, want 404", again.Code)
	}
}

func TestAnAvatarUploadRefusesSomethingThatIsNotAnImage(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)

	response := apitest.Send(t, r, apitest.Authorized(apitest.AvatarUploadRequest(t, []byte("not a picture")), session))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", response.Code, response.Body.String())
	}
}

func readProfile(t *testing.T, r http.Handler, handle string) apitest.PublicProfile {
	t.Helper()
	response := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/v1/profiles/"+handle, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("read profile status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var profile apitest.PublicProfile
	if err := json.Unmarshal(response.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	return profile
}
