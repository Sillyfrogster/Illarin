package profile_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const creatorHandle = "moon.creator"

var hexColor = regexp.MustCompile(`^#[0-9a-f]{6}$`)

func TestABannerUploadsWithItsTintAndRemoves(t *testing.T) {
	t.Parallel()
	s := newPortfolioStack(t)

	uploaded := apitest.Send(t, s.router, apitest.Authorized(apitest.BannerUploadRequest(t, apitest.PNG(t, 900, 300)), s.creator))
	if uploaded.Code != http.StatusOK {
		t.Fatalf("banner status = %d, want 200: %s", uploaded.Code, uploaded.Body.String())
	}
	shown := s.profile(t, nil)
	if shown.Banner == nil || shown.Banner.Width != 900 || shown.Banner.Height != 300 {
		t.Fatalf("banner = %+v", shown.Banner)
	}
	if !hexColor.MatchString(shown.Tint) {
		t.Fatalf("tint = %q, want a hex color", shown.Tint)
	}
	image := apitest.Send(t, s.router, httptest.NewRequest(http.MethodGet, shown.Banner.URL, nil))
	if image.Code != http.StatusOK {
		t.Fatalf("banner image status = %d, want 200: %s", image.Code, image.Body.String())
	}

	removed := apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/account/profile/banner", nil), s.creator,
	))
	if removed.Code != http.StatusOK {
		t.Fatalf("remove status = %d, want 200: %s", removed.Code, removed.Body.String())
	}
	after := s.profile(t, nil)
	if after.Banner != nil || after.Tint != "" {
		t.Fatalf("banner survived removal: %+v tint %q", after.Banner, after.Tint)
	}
	var remaining int
	if err := s.pool.QueryRow(t.Context(), `select count(*) from profile_media`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("profile media rows = %d, want 0", remaining)
	}
}

func TestAnOwnerFeaturesUpToFourPublishedWorksInOrder(t *testing.T) {
	t.Parallel()
	s := newPortfolioStack(t)
	ids := make([]string, 5)
	for i := range ids {
		ids[i] = apitest.PublishedCharacter(t, s.router, s.creator)
	}

	if got := s.feature(t, ids[3], ids[0], ids[2]); got.Code != http.StatusOK {
		t.Fatalf("feature status = %d, want 200: %s", got.Code, got.Body.String())
	}
	shown := s.profile(t, nil)
	if len(shown.Featured) != 3 || shown.Featured[0].ID != ids[3] ||
		shown.Featured[1].ID != ids[0] || shown.Featured[2].ID != ids[2] {
		t.Fatalf("featured = %+v", shown.Featured)
	}
	if shown.Works != 5 {
		t.Fatalf("works = %d, want 5", shown.Works)
	}

	if got := s.feature(t, ids...); got.Code != http.StatusBadRequest {
		t.Fatalf("five featured = %d, want 400: %s", got.Code, got.Body.String())
	}
	if got := s.feature(t); got.Code != http.StatusOK {
		t.Fatalf("clearing featured = %d, want 200: %s", got.Code, got.Body.String())
	}
	if cleared := s.profile(t, nil); len(cleared.Featured) != 0 {
		t.Fatalf("featured after clearing = %+v", cleared.Featured)
	}
}

func TestFeaturingRefusesADraftAnUnlistedWorkAndAnotherCreatorsWork(t *testing.T) {
	t.Parallel()
	s := newPortfolioStack(t)
	draft := apitest.StartCharacter(t, s.router, s.creator).ID
	unlisted := apitest.PublishedCharacter(t, s.router, s.creator)
	s.setVisibility(t, unlisted, "unlisted")
	other := s.reader(t, "other@example.com", "other.creator")
	theirs := apitest.PublishedCharacter(t, s.router, other)

	for name, id := range map[string]string{"draft": draft, "unlisted": unlisted, "another creator's": theirs} {
		if got := s.feature(t, id); got.Code != http.StatusBadRequest {
			t.Errorf("featuring a %s work = %d, want 400: %s", name, got.Code, got.Body.String())
		}
	}
}

func TestAFeaturedWorkThatIsUnlistedOrDeletedLeavesTheProfile(t *testing.T) {
	t.Parallel()
	s := newPortfolioStack(t)
	kept := apitest.PublishedCharacter(t, s.router, s.creator)
	hidden := apitest.PublishedCharacter(t, s.router, s.creator)
	gone := apitest.PublishedCharacter(t, s.router, s.creator)
	s.feature(t, kept, hidden, gone)

	s.setVisibility(t, hidden, "unlisted")
	deleted := apitest.Send(t, s.router, apitest.BrowserRequest(t, http.MethodDelete, "/v1/works/"+gone, nil, s.creator))
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204: %s", deleted.Code, deleted.Body.String())
	}

	shown := s.profile(t, nil)
	if len(shown.Featured) != 1 || shown.Featured[0].ID != kept {
		t.Fatalf("featured = %+v, want only the kept work", shown.Featured)
	}
	if shown.Works != 1 {
		t.Fatalf("works = %d, want 1", shown.Works)
	}
}

func TestAReaderFollowsACreatorAndTheOwnerCannotFollowThemselves(t *testing.T) {
	t.Parallel()
	s := newPortfolioStack(t)
	reader := s.reader(t, "reader@example.com", "moon.reader")

	if got := s.profile(t, nil); got.Following != nil || got.Followers != 0 {
		t.Fatalf("signed-out profile = following %v, followers %d", got.Following, got.Followers)
	}
	if got := s.profile(t, reader); got.Following == nil || *got.Following {
		t.Fatalf("reader's profile before following = %v", got.Following)
	}

	followed := s.follow(t, reader, http.MethodPut)
	if followed.Code != http.StatusOK {
		t.Fatalf("follow status = %d, want 200: %s", followed.Code, followed.Body.String())
	}
	if got := s.profile(t, reader); got.Following == nil || !*got.Following || got.Followers != 1 {
		t.Fatalf("after following = following %v, followers %d", got.Following, got.Followers)
	}
	if got := s.profile(t, s.creator); got.Following != nil || got.Followers != 1 || !got.IsOwner {
		t.Fatalf("owner's profile = following %v, followers %d, owner %v", got.Following, got.Followers, got.IsOwner)
	}

	if got := s.follow(t, reader, http.MethodDelete); got.Code != http.StatusOK {
		t.Fatalf("unfollow status = %d, want 200: %s", got.Code, got.Body.String())
	}
	if got := s.profile(t, reader); got.Following == nil || *got.Following || got.Followers != 0 {
		t.Fatalf("after unfollowing = following %v, followers %d", got.Following, got.Followers)
	}

	if got := s.follow(t, s.creator, http.MethodPut); got.Code != http.StatusForbidden {
		t.Fatalf("following yourself = %d, want 403: %s", got.Code, got.Body.String())
	}
	signedOut := apitest.Send(t, s.router, apitest.BrowserMutation(
		httptest.NewRequest(http.MethodPut, "/v1/profiles/"+creatorHandle+"/follow", nil)))
	if signedOut.Code != http.StatusUnauthorized {
		t.Fatalf("signed-out follow = %d, want 401", signedOut.Code)
	}
}

func TestACreatorsFollowersHearWhenTheyPublishANewListedWork(t *testing.T) {
	t.Parallel()
	s := newPortfolioStack(t)
	follower := s.reader(t, "follower@example.com", "moon.follower")
	s.follow(t, follower, http.MethodPut)
	bystander := s.reader(t, "bystander@example.com", "moon.bystander")

	quiet := apitest.StartCharacter(t, s.router, s.creator)
	apitest.WriteCharacterFloor(t, s.router, s.creator, quiet)
	if _, err := s.pool.Exec(t.Context(), `update works set visibility = 'unlisted' where id = $1`, quiet.ID); err != nil {
		t.Fatal(err)
	}
	if got := apitest.PublishWork(t, s.router, s.creator, quiet.ID); got.Code != http.StatusOK {
		t.Fatalf("publish unlisted = %d: %s", got.Code, got.Body.String())
	}
	published := apitest.PublishedCharacter(t, s.router, s.creator)
	if _, err := s.notifications.FanOut(t.Context(), time.Now()); err != nil {
		t.Fatal(err)
	}

	page := s.inbox(t, follower)
	if len(page.Items) != 1 {
		t.Fatalf("follower has %d entries, want 1: %+v", len(page.Items), page.Items)
	}
	entry := page.Items[0]
	if entry.Type != "work_published" || entry.Work == nil || entry.Work.ID != published ||
		entry.Creator == nil || entry.Creator.Handle != creatorHandle {
		t.Fatalf("follower's entry = %+v", entry)
	}
	for name, session := range map[string]*http.Cookie{"owner": s.creator, "bystander": bystander} {
		if got := s.inbox(t, session); len(got.Items) != 0 {
			t.Errorf("the %s has %d entries, want none: %+v", name, len(got.Items), got.Items)
		}
	}
}

func TestAProfileListsRecentVersionsAcrossPublicWorksNewestFirst(t *testing.T) {
	t.Parallel()
	s := newPortfolioStack(t)
	first := apitest.PublishedCharacter(t, s.router, s.creator)
	second := apitest.PublishedCharacter(t, s.router, s.creator)
	unlisted := apitest.PublishedCharacter(t, s.router, s.creator)
	s.setVisibility(t, unlisted, "unlisted")
	s.describe(t, first, "She keeps the books that remember themselves.")
	if got := apitest.PublishWorkVersion(t, s.router, s.creator, first, `{"summary":"Rewrote her opening"}`); got.Code != http.StatusOK {
		t.Fatalf("publish a version = %d: %s", got.Code, got.Body.String())
	}

	shown := s.profile(t, nil)
	if len(shown.RecentVersions) != 3 {
		t.Fatalf("recent versions = %+v, want 3", shown.RecentVersions)
	}
	newest := shown.RecentVersions[0]
	if newest.WorkID != first || newest.Number != 2 || newest.Summary != "Rewrote her opening" {
		t.Fatalf("newest version = %+v", newest)
	}
	if shown.RecentVersions[1].WorkID != second || shown.RecentVersions[1].Number != 1 ||
		shown.RecentVersions[2].WorkID != first {
		t.Fatalf("older versions = %+v", shown.RecentVersions[1:])
	}
	for _, version := range shown.RecentVersions {
		if version.WorkID == unlisted {
			t.Fatalf("an unlisted work's version reached the feed: %+v", version)
		}
	}
}

func TestARestrictedProfileHidesItsBannerAndFeaturedWorks(t *testing.T) {
	t.Parallel()
	s := newPortfolioStack(t)
	apitest.Send(t, s.router, apitest.Authorized(apitest.BannerUploadRequest(t, apitest.PNG(t, 600, 200)), s.creator))
	s.feature(t, apitest.PublishedCharacter(t, s.router, s.creator))
	staff := s.reader(t, "staff@example.com", "moon.staff")
	apitest.SetRole(t, s.pool, "moon.staff", "admin")
	restricted := apitest.Send(t, s.router, apitest.AuthorizedJSONRequest(
		t, http.MethodPut, "/v1/profiles/"+creatorHandle+"/restricted", `{"reason":"Links to a scam."}`, staff,
	))
	if restricted.Code != http.StatusOK && restricted.Code != http.StatusNoContent {
		t.Fatalf("restrict status = %d: %s", restricted.Code, restricted.Body.String())
	}

	shown := s.profile(t, nil)
	if !shown.Restricted || shown.Banner != nil || shown.Tint != "" || len(shown.Featured) != 0 {
		t.Fatalf("restricted profile still shows banner %+v tint %q featured %+v", shown.Banner, shown.Tint, shown.Featured)
	}
	if shown.Works != 1 {
		t.Fatalf("works = %d, want the published work still counted", shown.Works)
	}
}

type portfolioStack struct {
	router        *gin.Engine
	pool          *pgxpool.Pool
	notifications *notify.Service
	outbox        *apitest.VerificationOutbox
	creator       *http.Cookie
}

type portfolioProfile struct {
	apitest.PublicProfile
	Banner *struct {
		URL    string `json:"url"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"banner"`
	Tint     string `json:"tint"`
	Featured []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"featured"`
	RecentVersions []struct {
		WorkID  string `json:"workId"`
		Number  int    `json:"number"`
		Summary string `json:"summary"`
	} `json:"recentVersions"`
	Works      int   `json:"works"`
	Followers  int   `json:"followers"`
	Following  *bool `json:"following"`
	IsOwner    bool  `json:"isOwner"`
	Restricted bool  `json:"restricted"`
}

type portfolioInbox struct {
	Items []struct {
		Type string `json:"type"`
		Work *struct {
			ID string `json:"id"`
		} `json:"work"`
		Creator *struct {
			Handle string `json:"handle"`
		} `json:"creator"`
	} `json:"items"`
}

func newPortfolioStack(t *testing.T) portfolioStack {
	t.Helper()
	outbox := &apitest.VerificationOutbox{}
	router, pool, services := harness.NewRouterWithSenderPoolAndServices(t, 1<<20, api.DefaultDeadlines(), outbox)
	creator := apitest.VerifiedSignUp(t, router, outbox, "creator@example.com", creatorHandle)
	return portfolioStack{
		router: router, pool: pool, notifications: services.Notifications, outbox: outbox, creator: creator,
	}
}

func (s portfolioStack) reader(t *testing.T, email, handle string) *http.Cookie {
	t.Helper()
	return apitest.VerifiedSignUp(t, s.router, s.outbox, email, handle)
}

func (s portfolioStack) profile(t *testing.T, session *http.Cookie) portfolioProfile {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/profiles/"+creatorHandle, nil)
	if session != nil {
		request = apitest.Authorized(request, session)
	}
	response := apitest.Send(t, s.router, request)
	if response.Code != http.StatusOK {
		t.Fatalf("profile status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var shown portfolioProfile
	if err := json.Unmarshal(response.Body.Bytes(), &shown); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	return shown
}

func (s portfolioStack) feature(t *testing.T, ids ...string) *httptest.ResponseRecorder {
	t.Helper()
	quoted := make([]string, 0, len(ids))
	for _, id := range ids {
		quoted = append(quoted, fmt.Sprintf("%q", id))
	}
	return apitest.Send(t, s.router, apitest.AuthorizedJSONRequest(
		t, http.MethodPut, "/v1/account/profile/featured",
		`{"workIds":[`+strings.Join(quoted, ",")+`]}`, s.creator,
	))
}

func (s portfolioStack) follow(t *testing.T, session *http.Cookie, method string) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(method, "/v1/profiles/"+creatorHandle+"/follow", nil), session,
	))
}

func (s portfolioStack) setVisibility(t *testing.T, workID, visibility string) {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.AuthorizedJSONRequest(
		t, http.MethodPut, "/v1/works/"+workID+"/visibility", `{"visibility":"`+visibility+`"}`, s.creator,
	))
	if response.Code != http.StatusNoContent {
		t.Fatalf("set visibility = %d, want 204: %s", response.Code, response.Body.String())
	}
}

func (s portfolioStack) describe(t *testing.T, workID, text string) {
	t.Helper()
	started := apitest.FetchStartedWork(t, s.router, s.creator, workID)
	coreBlock := apitest.BlockNamed(t, started.Blocks, "character_core")
	core := apitest.EditableBlock(coreBlock)
	core.Elements[0].Content = json.RawMessage(fmt.Sprintf(`{"text":%q}`, text))
	if got := apitest.SaveBlock(t, s.router, s.creator, workID, coreBlock.ID, core); got.Code != http.StatusOK {
		t.Fatalf("save the description = %d, want 200: %s", got.Code, got.Body.String())
	}
}

func (s portfolioStack) inbox(t *testing.T, session *http.Cookie) portfolioInbox {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(httptest.NewRequest(http.MethodGet, "/v1/notifications", nil), session))
	if response.Code != http.StatusOK {
		t.Fatalf("inbox status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var page portfolioInbox
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode inbox: %v", err)
	}
	return page
}
