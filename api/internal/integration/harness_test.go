package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest/full"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

var harness = full.Harness

type publicationStack struct {
	router    *gin.Engine
	pool      *pgxpool.Pool
	handlers  apitest.Services
	outbox    *apitest.VerificationOutbox
	authority *http.Cookie
}

type profileAvatar struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type publicationApp struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Home     string `json:"home"`
	Position int    `json:"position"`
	Retired  bool   `json:"retired"`
}

type publicationAppList struct {
	Apps []publicationApp `json:"apps"`
}

type publicationCategory struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	Position int    `json:"position"`
	Retired  bool   `json:"retired"`
}

type publicationCategoryList struct {
	Categories []publicationCategory `json:"categories"`
}

type publicationGrantHolder struct {
	Handle      string         `json:"handle"`
	DisplayName string         `json:"displayName"`
	Avatar      *profileAvatar `json:"avatar"`
	Restricted  bool           `json:"restricted"`
}

type publicationGrant struct {
	ID              string                 `json:"id"`
	Holder          publicationGrantHolder `json:"holder"`
	App             publicationApp         `json:"app"`
	Categories      []publicationCategory  `json:"categories"`
	DefaultCategory publicationCategory    `json:"defaultCategory"`
	GrantedAt       time.Time              `json:"grantedAt"`
	RevokedAt       *time.Time             `json:"revokedAt"`
	Active          bool                   `json:"active"`
}

type publicationGrantList struct {
	Grants []publicationGrant `json:"grants"`
}

type publicationWorkspace struct {
	Handle string             `json:"handle"`
	Grants []publicationGrant `json:"grants"`
}

type contributor struct {
	handle  string
	session *http.Cookie
	grant   publicationGrant
}

func (s publicationStack) contributor(t *testing.T, email, handle string) contributor {
	t.Helper()
	session := s.member(t, email, handle)
	illarin := s.appBySlug(t, "illarin")
	announcement := s.categoryBySlug(t, "announcement")
	made := s.approved(t, handle, illarin.ID, []string{announcement.ID}, announcement.ID)
	return contributor{handle: handle, session: session, grant: made}
}

func newPublicationStack(t *testing.T) publicationStack {
	t.Helper()
	outbox := &apitest.VerificationOutbox{}
	router, pool, handlers := harness.NewRouterWithSenderPoolAndServices(
		t, 1<<20, api.DefaultDeadlines(), outbox,
	)
	session := apitest.VerifiedSignUp(t, router, outbox, "authority@example.com", "publication.authority")
	apitest.HoldsAuthority(t, pool, "publication.authority")
	return publicationStack{
		router: router, pool: pool, handlers: handlers, outbox: outbox, authority: session,
	}
}

func (s publicationStack) member(t *testing.T, email, handle string) *http.Cookie {
	t.Helper()
	return apitest.VerifiedSignUp(t, s.router, s.outbox, email, handle)
}

func jsonRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func (s publicationStack) apps(t *testing.T) []publicationApp {
	t.Helper()
	rows, err := s.pool.Query(context.Background(), `
		select id::text, slug, name, home_url, position, retired_at is not null
		  from publication_apps
		 order by position, created_at
	`)
	if err != nil {
		t.Fatalf("read apps: %v", err)
	}
	defer rows.Close()
	listed := []publicationApp{}
	for rows.Next() {
		var one publicationApp
		if err := rows.Scan(&one.ID, &one.Slug, &one.Name, &one.Home, &one.Position, &one.Retired); err != nil {
			t.Fatalf("read an app: %v", err)
		}
		listed = append(listed, one)
	}
	return listed
}

func (s publicationStack) categories(t *testing.T) []publicationCategory {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/categories", nil), s.authority,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("list categories status = %d: %s", response.Code, response.Body.String())
	}
	var listed publicationCategoryList
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode categories: %v", err)
	}
	return listed.Categories
}

func (s publicationStack) categoryBySlug(t *testing.T, slug string) publicationCategory {
	t.Helper()
	for _, category := range s.categories(t) {
		if category.Slug == slug {
			return category
		}
	}
	t.Fatalf("no seeded category with the slug %q", slug)
	return publicationCategory{}
}

func (s publicationStack) appBySlug(t *testing.T, slug string) publicationApp {
	t.Helper()
	for _, app := range s.apps(t) {
		if app.Slug == slug {
			return app
		}
	}
	t.Fatalf("no configured app with the slug %q", slug)
	return publicationApp{}
}

func (s publicationStack) configureApp(t *testing.T, slug, name, home string) publicationApp {
	t.Helper()
	_, err := s.pool.Exec(context.Background(), `
		insert into publication_apps (id, slug, name, home_url, position)
		values (gen_random_uuid(), $1, $2, $3,
		        (select coalesce(max(position) + 1, 0) from publication_apps))
	`, slug, name, home)
	if err != nil {
		t.Fatalf("configure %s: %v", slug, err)
	}
	return s.appBySlug(t, slug)
}

func (s publicationStack) approve(
	t *testing.T,
	session *http.Cookie,
	handle, appID string,
	categoryIDs []string,
	defaultID string,
) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"handle": handle, "appId": appID,
		"categoryIds": categoryIDs, "defaultCategoryId": defaultID,
	})
	if err != nil {
		t.Fatalf("encode approval: %v", err)
	}
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/grants", string(body),
	), session))
}

func (s publicationStack) approved(
	t *testing.T,
	handle, appID string,
	categoryIDs []string,
	defaultID string,
) publicationGrant {
	t.Helper()
	response := s.approve(t, s.authority, handle, appID, categoryIDs, defaultID)
	if response.Code != http.StatusCreated {
		t.Fatalf("approve %s status = %d: %s", handle, response.Code, response.Body.String())
	}
	var made publicationGrant
	if err := json.Unmarshal(response.Body.Bytes(), &made); err != nil {
		t.Fatalf("decode grant: %v", err)
	}
	return made
}

func (s publicationStack) workspace(t *testing.T, session *http.Cookie) publicationWorkspace {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/workspace", nil), session,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("read workspace status = %d: %s", response.Code, response.Body.String())
	}
	var open publicationWorkspace
	if err := json.Unmarshal(response.Body.Bytes(), &open); err != nil {
		t.Fatalf("decode workspace: %v", err)
	}
	return open
}

type postAuthor struct {
	Handle string `json:"handle"`
}

type postRelease struct {
	App     publicationApp `json:"app"`
	Version string         `json:"version"`
	Address string         `json:"address"`
}

type postByline struct {
	Handle       string          `json:"handle"`
	DisplayName  string          `json:"displayName"`
	ContactEmail string          `json:"contactEmail"`
	Avatar       *profileAvatar  `json:"avatar"`
	App          *publicationApp `json:"app"`
}

type postDocument struct {
	Version int              `json:"version"`
	Content []map[string]any `json:"content"`
}

type blogPost struct {
	ID              string              `json:"id"`
	Status          string              `json:"status"`
	Title           string              `json:"title"`
	Summary         string              `json:"summary"`
	Slug            string              `json:"slug"`
	Category        publicationCategory `json:"category"`
	Document        postDocument        `json:"document"`
	DocumentVersion int                 `json:"documentVersion"`
	Release         *postRelease        `json:"release"`
	Header          *postHeader         `json:"header"`
	SocialMediaID   string              `json:"socialMediaId"`
	PublicRevision  string              `json:"publicRevisionId"`
	Schedule        *postSchedule       `json:"schedule"`
	Withdrawal      *postWithdrawal     `json:"withdrawal"`
	Deletion        *postDeletion       `json:"deletion"`
	Media           []postPicture       `json:"media"`
	Byline          *postByline         `json:"byline"`
	FormerAddresses []string            `json:"formerAddresses"`
	App             *publicationApp     `json:"app"`
	GrantID         string              `json:"grantId"`
	Version         int                 `json:"version"`
	Author          postAuthor          `json:"author"`
	PublishedAt     *time.Time          `json:"publishedAt"`
	UpdatedPublicAt *time.Time          `json:"updatedPublicAt"`
}

type postList struct {
	Posts []blogPost `json:"posts"`
}

type publicPost struct {
	ID           string              `json:"id"`
	Slug         string              `json:"slug"`
	OriginalSlug string              `json:"originalSlug"`
	Title        string              `json:"title"`
	Summary      string              `json:"summary"`
	Category     publicationCategory `json:"category"`
	Document     postDocument        `json:"document"`
	Release      *postRelease        `json:"release"`
	Header       *postHeader         `json:"header"`
	SocialImage  *postPicture        `json:"socialImage"`
	Media        []postPicture       `json:"media"`
	Byline       postByline          `json:"byline"`
	Related      []postSummary       `json:"related"`
	PublishedAt  time.Time           `json:"publishedAt"`
	UpdatedAt    *time.Time          `json:"updatedAt"`
}

func paragraph(words string) string {
	return fmt.Sprintf(
		`{"version":2,"content":[{"type":"paragraph","content":[{"type":"text","text":%q}]}]}`,
		words,
	)
}

func (s publicationStack) start(
	t *testing.T,
	session *http.Cookie,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts", body,
	), session))
}

func (s publicationStack) started(t *testing.T, session *http.Cookie, body string) blogPost {
	t.Helper()
	response := s.start(t, session, body)
	if response.Code != http.StatusCreated {
		t.Fatalf("start post status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s publicationStack) save(
	t *testing.T,
	session *http.Cookie,
	id string,
	working map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(working)
	if err != nil {
		t.Fatalf("encode drafted changes: %v", err)
	}
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPut, "/v1/publication/posts/"+id, string(body),
	), session))
}

func finished(draft blogPost, changes map[string]any) map[string]any {
	working := map[string]any{
		"version":    draft.Version,
		"categoryId": draft.Category.ID,
		"title":      draft.Title,
		"summary":    "What Illarin changed this week.",
		"slug":       draft.Slug,
		"document":   json.RawMessage(paragraph("Illarin now keeps its own writing.")),
	}
	for key, value := range changes {
		working[key] = value
	}
	return working
}

func (s publicationStack) saved(
	t *testing.T,
	session *http.Cookie,
	id string,
	working map[string]any,
) blogPost {
	t.Helper()
	response := s.save(t, session, id, working)
	if response.Code != http.StatusOK {
		t.Fatalf("save post status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s publicationStack) publish(
	t *testing.T,
	session *http.Cookie,
	id string,
) *httptest.ResponseRecorder {
	t.Helper()
	version := 1
	reading := apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/posts/"+id, nil), session,
	))
	if reading.Code == http.StatusOK {
		version = decodePost(t, reading).Version
	}
	return s.publishAt(t, session, id, version)
}

func (s publicationStack) publishAt(
	t *testing.T,
	session *http.Cookie,
	id string,
	version int,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+id+"/publish",
		fmt.Sprintf(`{"version":%d}`, version),
	), session))
}

func (s publicationStack) working(t *testing.T, session *http.Cookie, id string) blogPost {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/posts/"+id, nil), session,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("read post status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s publicationStack) published(t *testing.T, session *http.Cookie, id string) blogPost {
	t.Helper()
	response := s.publish(t, session, id)
	if response.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s publicationStack) publishedAt(
	t *testing.T,
	session *http.Cookie,
	id string,
	version int,
) blogPost {
	t.Helper()
	response := s.publishAt(t, session, id, version)
	if response.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s publicationStack) read(t *testing.T, slug string) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, httptest.NewRequest(http.MethodGet, "/v1/posts/"+slug, nil))
}

func (s publicationStack) reader(t *testing.T, slug string) publicPost {
	t.Helper()
	response := s.read(t, slug)
	if response.Code != http.StatusOK {
		t.Fatalf("read %s status = %d: %s", slug, response.Code, response.Body.String())
	}
	var found publicPost
	if err := json.Unmarshal(response.Body.Bytes(), &found); err != nil {
		t.Fatalf("decode published post: %v", err)
	}
	return found
}

func decodePost(t *testing.T, response *httptest.ResponseRecorder) blogPost {
	t.Helper()
	var found blogPost
	if err := json.Unmarshal(response.Body.Bytes(), &found); err != nil {
		t.Fatalf("decode post: %v", err)
	}
	return found
}

func (s publicationStack) admin(t *testing.T, email, handle string) *http.Cookie {
	t.Helper()
	session := s.member(t, email, handle)
	apitest.SetRole(t, s.pool, handle, "admin")
	return session
}

func (s publicationStack) illarinDraft(t *testing.T, session *http.Cookie, title string) blogPost {
	t.Helper()
	announcement := s.categoryBySlug(t, "announcement")
	return s.started(t, session, fmt.Sprintf(
		`{"categoryId":%q,"title":%q}`, announcement.ID, title,
	))
}

type tombstone struct {
	Slug        string `json:"slug"`
	Explanation string `json:"explanation"`
}

type postWithdrawal struct {
	Reason      string    `json:"reason"`
	Explanation string    `json:"explanation"`
	By          string    `json:"by"`
	At          time.Time `json:"at"`
}

func (s publicationStack) withdraw(
	t *testing.T,
	session *http.Cookie,
	id, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+id+"/withdraw", body,
	), session))
}

func (s publicationStack) withdrawn(
	t *testing.T,
	session *http.Cookie,
	id string,
	version int,
	reason, explanation string,
) blogPost {
	t.Helper()
	body := fmt.Sprintf(`{"version":%d,"reason":%q,"explanation":%q}`,
		version, reason, explanation)
	response := s.withdraw(t, session, id, body)
	if response.Code != http.StatusOK {
		t.Fatalf("withdraw status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s publicationStack) republish(
	t *testing.T,
	session *http.Cookie,
	id, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+id+"/republish", body,
	), session))
}

func (s publicationStack) republished(
	t *testing.T,
	session *http.Cookie,
	id string,
	version int,
	revisionID string,
) blogPost {
	t.Helper()
	response := s.republish(t, session, id, fmt.Sprintf(
		`{"version":%d,"revisionId":%q}`, version, revisionID,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("republish status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s publicationStack) gone(t *testing.T, slug string) (tombstone, string) {
	t.Helper()
	response := s.read(t, slug)
	if response.Code != http.StatusGone {
		t.Fatalf("read %s status = %d, want 410: %s", slug, response.Code, response.Body.String())
	}
	var found tombstone
	if err := json.Unmarshal(response.Body.Bytes(), &found); err != nil {
		t.Fatalf("decode tombstone: %v", err)
	}
	return found, response.Body.String()
}

func (s publicationStack) events(t *testing.T, postID string) []string {
	t.Helper()
	rows, err := s.pool.Query(context.Background(), `
		select type from blog_announcements where post_id = $1 order by occurred_at, id
	`, postID)
	if err != nil {
		t.Fatalf("read the announcements: %v", err)
	}
	defer rows.Close()
	recorded := make([]string, 0, 4)
	for rows.Next() {
		var one string
		if err := rows.Scan(&one); err != nil {
			t.Fatalf("read an announcement: %v", err)
		}
		recorded = append(recorded, one)
	}
	return recorded
}

type postSchedule struct {
	ID             string    `json:"id"`
	RevisionID     string    `json:"revisionId"`
	RevisionNumber int       `json:"revisionNumber"`
	At             time.Time `json:"at"`
	State          string    `json:"state"`
	StoppedBecause string    `json:"stoppedBecause"`
	CreatedBy      string    `json:"createdBy"`
	CreatedAt      time.Time `json:"createdAt"`
}

type postPicture struct {
	ID       string `json:"id"`
	PostID   string `json:"postId"`
	Purpose  string `json:"purpose"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumbUrl"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type postHeader struct {
	MediaID string `json:"mediaId"`
	Alt     string `json:"alt"`
	Caption string `json:"caption"`
}

type postSummary struct {
	ID             string              `json:"id"`
	Slug           string              `json:"slug"`
	OriginalSlug   string              `json:"originalSlug"`
	Title          string              `json:"title"`
	Summary        string              `json:"summary"`
	Category       publicationCategory `json:"category"`
	App            *publicationApp     `json:"app"`
	ReleaseVersion string              `json:"releaseVersion"`
	Byline         postByline          `json:"byline"`
	PublishedAt    time.Time           `json:"publishedAt"`
	UpdatedAt      *time.Time          `json:"updatedAt"`
}

type postDeletion struct {
	At    time.Time `json:"at"`
	Until time.Time `json:"until"`
	By    string    `json:"by"`
}
