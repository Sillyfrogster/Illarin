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

type blogStack struct {
	router   *gin.Engine
	pool     *pgxpool.Pool
	handlers apitest.Services
	outbox   *apitest.VerificationOutbox
	admin    *http.Cookie
}

type profileAvatar struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type blogCategory struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	Position int    `json:"position"`
	Retired  bool   `json:"retired"`
}

type blogCategoryList struct {
	Categories []blogCategory `json:"categories"`
}

type blogWorkspace struct {
	Handle string `json:"handle"`
	Writer bool   `json:"writer"`
}

type contributor struct {
	handle  string
	session *http.Cookie
}

func (s blogStack) contributor(t *testing.T, email, handle string) contributor {
	t.Helper()
	session := s.member(t, email, handle)
	switched := apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/blog/writers", `{"handle":"`+handle+`"}`,
	), s.admin))
	if switched.Code != http.StatusCreated {
		t.Fatalf("switch %s on status = %d: %s", handle, switched.Code, switched.Body.String())
	}
	return contributor{handle: handle, session: session}
}

func newBlogStack(t *testing.T) blogStack {
	t.Helper()
	outbox := &apitest.VerificationOutbox{}
	router, pool, handlers := harness.NewRouterWithSenderPoolAndServices(
		t, 1<<20, api.DefaultDeadlines(), outbox,
	)
	session := apitest.VerifiedSignUp(t, router, outbox, "admin@example.com", "blog.admin")
	apitest.HoldsAuthority(t, pool, "blog.admin")
	return blogStack{router: router, pool: pool, handlers: handlers, outbox: outbox, admin: session}
}

func (s blogStack) member(t *testing.T, email, handle string) *http.Cookie {
	t.Helper()
	return apitest.VerifiedSignUp(t, s.router, s.outbox, email, handle)
}

func jsonRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func (s blogStack) categories(t *testing.T) []blogCategory {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/blog/categories", nil), s.admin,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("list categories status = %d: %s", response.Code, response.Body.String())
	}
	var listed blogCategoryList
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode categories: %v", err)
	}
	return listed.Categories
}

func (s blogStack) categoryBySlug(t *testing.T, slug string) blogCategory {
	t.Helper()
	for _, category := range s.categories(t) {
		if category.Slug == slug {
			return category
		}
	}
	t.Fatalf("no seeded category with the slug %q", slug)
	return blogCategory{}
}

func (s blogStack) workspace(t *testing.T, session *http.Cookie) blogWorkspace {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/blog/workspace", nil), session,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("read workspace status = %d: %s", response.Code, response.Body.String())
	}
	var open blogWorkspace
	if err := json.Unmarshal(response.Body.Bytes(), &open); err != nil {
		t.Fatalf("decode workspace: %v", err)
	}
	return open
}

type postAuthor struct {
	Handle string `json:"handle"`
}

type postByline struct {
	Handle       string         `json:"handle"`
	DisplayName  string         `json:"displayName"`
	ContactEmail string         `json:"contactEmail"`
	Avatar       *profileAvatar `json:"avatar"`
}

type postBody struct {
	Version int              `json:"version"`
	Content []map[string]any `json:"content"`
}

type blogPost struct {
	ID              string            `json:"id"`
	Status          string            `json:"status"`
	Title           string            `json:"title"`
	Summary         string            `json:"summary"`
	Slug            string            `json:"slug"`
	Category        blogCategory      `json:"category"`
	Body            postBody          `json:"body"`
	BodyVersion     int               `json:"bodyVersion"`
	Header          *postHeader       `json:"header"`
	LinkCardMediaID string            `json:"linkCardMediaId"`
	PublicRevision  string            `json:"publicRevisionId"`
	Schedule        *postSchedule     `json:"schedule"`
	Unpublishing    *postUnpublishing `json:"unpublishing"`
	Deletion        *postDeletion     `json:"deletion"`
	Media           []postPicture     `json:"media"`
	Byline          *postByline       `json:"byline"`
	FormerAddresses []string          `json:"formerAddresses"`
	Version         int               `json:"version"`
	Author          postAuthor        `json:"author"`
	PublishedAt     *time.Time        `json:"publishedAt"`
	UpdatedPublicAt *time.Time        `json:"updatedPublicAt"`
}

type postList struct {
	Posts []blogPost `json:"posts"`
}

type publicPost struct {
	ID            string        `json:"id"`
	Slug          string        `json:"slug"`
	OriginalSlug  string        `json:"originalSlug"`
	Title         string        `json:"title"`
	Summary       string        `json:"summary"`
	Category      blogCategory  `json:"category"`
	Body          postBody      `json:"body"`
	Header        *postHeader   `json:"header"`
	LinkCardImage *postPicture  `json:"linkCardImage"`
	Media         []postPicture `json:"media"`
	Byline        postByline    `json:"byline"`
	Related       []postSummary `json:"related"`
	PublishedAt   time.Time     `json:"publishedAt"`
	UpdatedAt     *time.Time    `json:"updatedAt"`
}

func paragraph(words string) string {
	return fmt.Sprintf(
		`{"version":2,"content":[{"type":"paragraph","content":[{"type":"text","text":%q}]}]}`,
		words,
	)
}

func (s blogStack) start(
	t *testing.T,
	session *http.Cookie,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/blog/posts", body,
	), session))
}

func (s blogStack) started(t *testing.T, session *http.Cookie, body string) blogPost {
	t.Helper()
	response := s.start(t, session, body)
	if response.Code != http.StatusCreated {
		t.Fatalf("start post status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s blogStack) save(
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
		http.MethodPut, "/v1/blog/posts/"+id, string(body),
	), session))
}

func finished(draft blogPost, changes map[string]any) map[string]any {
	working := map[string]any{
		"version":    draft.Version,
		"categoryId": draft.Category.ID,
		"title":      draft.Title,
		"summary":    "What Illarin changed this week.",
		"slug":       draft.Slug,
		"body":       json.RawMessage(paragraph("Illarin now keeps its own writing.")),
	}
	for key, value := range changes {
		working[key] = value
	}
	return working
}

func (s blogStack) saved(
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

func (s blogStack) publish(
	t *testing.T,
	session *http.Cookie,
	id string,
) *httptest.ResponseRecorder {
	t.Helper()
	version := 1
	reading := apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/blog/posts/"+id, nil), session,
	))
	if reading.Code == http.StatusOK {
		version = decodePost(t, reading).Version
	}
	return s.publishAt(t, session, id, version)
}

func (s blogStack) publishAt(
	t *testing.T,
	session *http.Cookie,
	id string,
	version int,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/blog/posts/"+id+"/publish",
		fmt.Sprintf(`{"version":%d}`, version),
	), session))
}

func (s blogStack) working(t *testing.T, session *http.Cookie, id string) blogPost {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/blog/posts/"+id, nil), session,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("read post status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s blogStack) published(t *testing.T, session *http.Cookie, id string) blogPost {
	t.Helper()
	response := s.publish(t, session, id)
	if response.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s blogStack) publishedAt(
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

func (s blogStack) read(t *testing.T, slug string) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, httptest.NewRequest(http.MethodGet, "/v1/posts/"+slug, nil))
}

func (s blogStack) reader(t *testing.T, slug string) publicPost {
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

func (s blogStack) siteAdmin(t *testing.T, email, handle string) *http.Cookie {
	t.Helper()
	session := s.member(t, email, handle)
	apitest.SetRole(t, s.pool, handle, "admin")
	return session
}

func (s blogStack) illarinDraft(t *testing.T, session *http.Cookie, title string) blogPost {
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

type postUnpublishing struct {
	Reason      string    `json:"reason"`
	Explanation string    `json:"explanation"`
	By          string    `json:"by"`
	At          time.Time `json:"at"`
}

func (s blogStack) unpublish(
	t *testing.T,
	session *http.Cookie,
	id, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/blog/posts/"+id+"/unpublish", body,
	), session))
}

func (s blogStack) unpublished(
	t *testing.T,
	session *http.Cookie,
	id string,
	version int,
	reason, explanation string,
) blogPost {
	t.Helper()
	body := fmt.Sprintf(`{"version":%d,"reason":%q,"explanation":%q}`,
		version, reason, explanation)
	response := s.unpublish(t, session, id, body)
	if response.Code != http.StatusOK {
		t.Fatalf("unpublish status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s blogStack) republish(
	t *testing.T,
	session *http.Cookie,
	id, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/blog/posts/"+id+"/republish", body,
	), session))
}

func (s blogStack) republished(
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

func (s blogStack) gone(t *testing.T, slug string) (tombstone, string) {
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

func (s blogStack) events(t *testing.T, postID string) []string {
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
	ID           string       `json:"id"`
	Slug         string       `json:"slug"`
	OriginalSlug string       `json:"originalSlug"`
	Title        string       `json:"title"`
	Summary      string       `json:"summary"`
	Category     blogCategory `json:"category"`
	Byline       postByline   `json:"byline"`
	PublishedAt  time.Time    `json:"publishedAt"`
	UpdatedAt    *time.Time   `json:"updatedAt"`
}

type postDeletion struct {
	At    time.Time `json:"at"`
	Until time.Time `json:"until"`
	By    string    `json:"by"`
}
