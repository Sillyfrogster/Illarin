package blog_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

type writer struct {
	AccountID   string         `json:"accountId"`
	Handle      string         `json:"handle"`
	DisplayName string         `json:"displayName"`
	Avatar      *profileAvatar `json:"avatar"`
	Restricted  bool           `json:"restricted"`
	Since       time.Time      `json:"since"`
}

type writerList struct {
	Writers []writer `json:"writers"`
}

type blogWorkspace struct {
	Handle     string         `json:"handle"`
	Admin      bool           `json:"admin"`
	Writer     bool           `json:"writer"`
	Categories []blogCategory `json:"categories"`
}

type contributor struct {
	handle  string
	session *http.Cookie
	writer  writer
}

func (s blogStack) contributor(t *testing.T, email, handle string) contributor {
	t.Helper()
	session := s.member(t, email, handle)
	return contributor{handle: handle, session: session, writer: s.switchedOn(t, handle)}
}

func newBlogStack(t *testing.T) blogStack {
	t.Helper()
	outbox := &apitest.VerificationOutbox{}
	router, pool, handlers := harness.NewRouterWithSenderPoolAndServices(
		t, 1<<20, api.DefaultDeadlines(), outbox,
	)
	session := apitest.VerifiedSignUp(t, router, outbox, "admin@example.com", "blog.admin")
	apitest.SetRole(t, pool, "blog.admin", "admin")
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

func (s blogStack) switchOn(t *testing.T, session *http.Cookie, handle string) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/blog/writers", `{"handle":"`+handle+`"}`,
	), session))
}

func (s blogStack) switchedOn(t *testing.T, handle string) writer {
	t.Helper()
	response := s.switchOn(t, s.admin, handle)
	if response.Code != http.StatusCreated {
		t.Fatalf("switch %s on status = %d: %s", handle, response.Code, response.Body.String())
	}
	var made writer
	if err := json.Unmarshal(response.Body.Bytes(), &made); err != nil {
		t.Fatalf("decode writer: %v", err)
	}
	return made
}

func (s blogStack) switchOff(t *testing.T, session *http.Cookie, accountID string) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/blog/writers/"+accountID, nil), session,
	))
}

func (s blogStack) writers(t *testing.T) []writer {
	t.Helper()
	listed := apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/blog/writers", nil), s.admin,
	))
	if listed.Code != http.StatusOK {
		t.Fatalf("list writers status = %d: %s", listed.Code, listed.Body.String())
	}
	var found writerList
	if err := json.Unmarshal(listed.Body.Bytes(), &found); err != nil {
		t.Fatalf("decode writers: %v", err)
	}
	return found.Writers
}

func (s blogStack) listPosts(t *testing.T, session *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/blog/posts", nil), session,
	))
}

func (s blogStack) readPost(t *testing.T, session *http.Cookie, id string) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/blog/posts/"+id, nil), session,
	))
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

func TestTheThreeCategoriesAreSeeded(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)

	seeded := stack.categories(t)
	if len(seeded) != 3 ||
		seeded[0].Slug != "announcement" ||
		seeded[1].Slug != "release" ||
		seeded[2].Slug != "article" {
		t.Fatalf("seeded categories = %+v", seeded)
	}
}

func TestOnlyAnAdminManagesCategoriesAndWriters(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	outsider := stack.member(t, "outsider@example.com", "blog.outsider")
	announcement := stack.categoryBySlug(t, "announcement")

	for _, role := range []string{"user", "moderator"} {
		apitest.SetRole(t, stack.pool, "blog.outsider", role)
		for _, path := range []string{"/v1/blog/categories", "/v1/blog/writers"} {
			refused := apitest.Send(t, stack.router, apitest.Authorized(
				httptest.NewRequest(http.MethodGet, path, nil), outsider,
			))
			if refused.Code != http.StatusForbidden {
				t.Fatalf("%s read of %s = %d, want 403", role, path, refused.Code)
			}
		}
		relabelled := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
			http.MethodPatch, "/v1/blog/categories/"+announcement.ID, `{"label":"News"}`,
		), outsider))
		if relabelled.Code != http.StatusForbidden {
			t.Fatalf("%s relabel category = %d, want 403", role, relabelled.Code)
		}
		if switched := stack.switchOn(t, outsider, "blog.outsider"); switched.Code != http.StatusForbidden {
			t.Fatalf("%s switch on = %d, want 403: %s", role, switched.Code, switched.Body.String())
		}
	}
}

func TestTheBlogAdminRelabelsAndOrdersCategories(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	announcement := stack.categoryBySlug(t, "announcement")
	article := stack.categoryBySlug(t, "article")
	release := stack.categoryBySlug(t, "release")

	relabelled := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPatch, "/v1/blog/categories/"+announcement.ID, `{"label":"Update"}`,
	), stack.admin))
	if relabelled.Code != http.StatusOK {
		t.Fatalf("relabel status = %d: %s", relabelled.Code, relabelled.Body.String())
	}

	ordered := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPut, "/v1/blog/categories",
		`{"categoryIds":["`+article.ID+`","`+release.ID+`","`+announcement.ID+`"]}`,
	), stack.admin))
	if ordered.Code != http.StatusOK {
		t.Fatalf("order categories status = %d: %s", ordered.Code, ordered.Body.String())
	}

	after := stack.categories(t)
	if after[0].Slug != "article" || after[2].Slug != "announcement" {
		t.Fatalf("categories after ordering = %+v", after)
	}
	if after[2].Label != "Update" || after[2].ID != announcement.ID {
		t.Fatalf("relabelled category = %+v", after[2])
	}
}

func TestTheWriterSwitchNeedsAVerifiedAccountAndStaysOnWhenTurnedOnTwice(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	stack.member(t, "writer@example.com", "verified.writer")
	apitest.SignUp(t, stack.router, "unverified@example.com", "unverified.writer")

	if unverified := stack.switchOn(t, stack.admin, "unverified.writer"); unverified.Code != http.StatusBadRequest {
		t.Fatalf("unverified switch on status = %d, want 400: %s", unverified.Code, unverified.Body.String())
	}
	if missing := stack.switchOn(t, stack.admin, "nobody.here"); missing.Code != http.StatusNotFound {
		t.Fatalf("unknown handle status = %d, want 404", missing.Code)
	}

	first := stack.switchedOn(t, "verified.writer")
	again := stack.switchedOn(t, "verified.writer")
	if again.AccountID != first.AccountID || !again.Since.Equal(first.Since) {
		t.Fatalf("switching on twice made a second writer: %+v then %+v", first, again)
	}
	if listed := stack.writers(t); len(listed) != 1 {
		t.Fatalf("writers = %+v, want one", listed)
	}
}

func TestTheWriterSwitchOpensAndClosesYourPostsAndTheEditor(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	session := stack.member(t, "switched@example.com", "switched.writer")
	announcement := stack.categoryBySlug(t, "announcement")
	start := `{"categoryId":"` + announcement.ID + `","title":"Hello"}`

	if open := stack.workspace(t, session); open.Writer || open.Admin {
		t.Fatalf("workspace before the switch = %+v", open)
	}
	if posts := stack.listPosts(t, session); posts.Code != http.StatusForbidden {
		t.Fatalf("Your posts before the switch = %d, want 403", posts.Code)
	}
	if started := stack.start(t, session, start); started.Code != http.StatusForbidden {
		t.Fatalf("editor before the switch = %d, want 403", started.Code)
	}

	made := stack.switchedOn(t, "switched.writer")
	if open := stack.workspace(t, session); !open.Writer || len(open.Categories) != 3 {
		t.Fatalf("workspace with the switch on = %+v", open)
	}
	if posts := stack.listPosts(t, session); posts.Code != http.StatusOK {
		t.Fatalf("Your posts with the switch on = %d: %s", posts.Code, posts.Body.String())
	}
	draft := stack.started(t, session, start)
	if draft.Author.Handle != "switched.writer" {
		t.Fatalf("draft author = %+v", draft.Author)
	}

	if off := stack.switchOff(t, stack.admin, made.AccountID); off.Code != http.StatusNoContent {
		t.Fatalf("switch off status = %d: %s", off.Code, off.Body.String())
	}
	if twice := stack.switchOff(t, stack.admin, made.AccountID); twice.Code != http.StatusNotFound {
		t.Fatalf("second switch off status = %d, want 404", twice.Code)
	}
	if open := stack.workspace(t, session); open.Writer {
		t.Fatalf("workspace after the switch = %+v", open)
	}
	if posts := stack.listPosts(t, session); posts.Code != http.StatusForbidden {
		t.Fatalf("Your posts after the switch = %d, want 403", posts.Code)
	}
	if reopened := stack.readPost(t, session, draft.ID); reopened.Code != http.StatusForbidden {
		t.Fatalf("editor after the switch = %d, want 403", reopened.Code)
	}
	if listed := stack.writers(t); len(listed) != 0 {
		t.Fatalf("writers after the switch = %+v", listed)
	}

	var role string
	var verified bool
	err := stack.pool.QueryRow(context.Background(), `
		select role, email_verified_at is not null from users where username = 'switched.writer'
	`).Scan(&role, &verified)
	if err != nil {
		t.Fatalf("read the account: %v", err)
	}
	if role != "user" || !verified {
		t.Fatalf("the switch changed the account: role %q, verified %v", role, verified)
	}
}

func TestAWriterSeesOnlyTheirOwnPostsAndNoAdministration(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	one := stack.contributor(t, "one@example.com", "one.writer")
	other := stack.contributor(t, "other@example.com", "other.writer")
	announcement := stack.categoryBySlug(t, "announcement")
	mine := stack.started(t, one.session, `{"categoryId":"`+announcement.ID+`","title":"Mine"}`)
	stack.started(t, other.session, `{"categoryId":"`+announcement.ID+`","title":"Theirs"}`)

	listed := stack.listPosts(t, one.session)
	var held postList
	if err := json.Unmarshal(listed.Body.Bytes(), &held); err != nil {
		t.Fatalf("decode posts: %v", err)
	}
	if len(held.Posts) != 1 || held.Posts[0].ID != mine.ID {
		t.Fatalf("one writer's posts = %+v", held.Posts)
	}
	if theirs := stack.readPost(t, other.session, mine.ID); theirs.Code != http.StatusForbidden {
		t.Fatalf("another writer opened a post that is not theirs: %d", theirs.Code)
	}
	for _, path := range []string{"/v1/blog/categories", "/v1/blog/writers"} {
		refused := apitest.Send(t, stack.router, apitest.Authorized(
			httptest.NewRequest(http.MethodGet, path, nil), one.session,
		))
		if refused.Code != http.StatusForbidden {
			t.Fatalf("a writer read %s: %d", path, refused.Code)
		}
	}
}

func TestSwitchingAWriterLeavesSafeIdentifiersBehind(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	stack.member(t, "audited@example.com", "audited.writer")
	made := stack.switchedOn(t, "audited.writer")
	if off := stack.switchOff(t, stack.admin, made.AccountID); off.Code != http.StatusNoContent {
		t.Fatalf("switch off status = %d", off.Code)
	}

	rows, err := stack.pool.Query(context.Background(), `
		select action, subject_id::text from blog_activity_log order by recorded_at
	`)
	if err != nil {
		t.Fatalf("read the activity log: %v", err)
	}
	defer rows.Close()
	recorded := make([]string, 0, 2)
	for rows.Next() {
		var action, subject string
		if err := rows.Scan(&action, &subject); err != nil {
			t.Fatalf("read an entry: %v", err)
		}
		if subject != made.AccountID {
			t.Fatalf("%s names %s, want the writer %s", action, subject, made.AccountID)
		}
		recorded = append(recorded, action)
	}
	if strings.Join(recorded, ",") != "writer.on,writer.off" {
		t.Fatalf("activity log = %v", recorded)
	}

	var columns int
	err = stack.pool.QueryRow(context.Background(), `
		select count(*) from information_schema.columns
		 where table_name = 'blog_activity_log'
		   and data_type not in ('uuid', 'timestamp with time zone')
		   and column_name not in ('action', 'credential', 'before_state', 'after_state')
	`).Scan(&columns)
	if err != nil {
		t.Fatalf("read the activity log columns: %v", err)
	}
	if columns != 0 {
		t.Fatalf("the activity log has %d columns that could hold copied content", columns)
	}
	for _, pinned := range []string{
		"blog_activity_log_credential_check",
		"blog_activity_log_state_check",
		"blog_activity_log_next_state_check",
	} {
		var exists bool
		err = stack.pool.QueryRow(context.Background(), `
			select exists (select 1 from pg_constraint where conname = $1)
		`, pinned).Scan(&exists)
		if err != nil {
			t.Fatalf("read %s: %v", pinned, err)
		}
		if !exists {
			t.Errorf("%s does not pin its column to a closed set of words", pinned)
		}
	}
}

func TestAWriterIsNamedTheWayTheirProfileIs(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	session := stack.member(t, "named@example.com", "named.writer")

	saved := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPut, "/v1/account/profile",
		`{"displayName":"Kestrel","biography":"","contactEmail":"","links":[]}`,
	), session))
	if saved.Code != http.StatusOK {
		t.Fatalf("save profile status = %d: %s", saved.Code, saved.Body.String())
	}

	made := stack.switchedOn(t, "named.writer")
	if made.Handle != "named.writer" || made.DisplayName != "Kestrel" {
		t.Fatalf("writer = %+v", made)
	}

	admin := stack.member(t, "restrictor@example.com", "restricting.admin")
	apitest.SetRole(t, stack.pool, "restricting.admin", "admin")
	restricted := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPut, "/v1/profiles/named.writer/restricted",
		`{"reason":"Under review"}`,
	), admin))
	if restricted.Code != http.StatusOK && restricted.Code != http.StatusNoContent {
		t.Fatalf("restrict status = %d: %s", restricted.Code, restricted.Body.String())
	}

	shown := stack.writers(t)[0]
	if !shown.Restricted || shown.DisplayName != "" || shown.Avatar != nil {
		t.Fatalf("a restricted writer still reads as %+v", shown)
	}
	if shown.Handle != "named.writer" {
		t.Fatalf("a restricted writer lost their handle: %+v", shown)
	}
}
