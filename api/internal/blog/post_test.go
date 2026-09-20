package blog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

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
		t.Fatalf("encode the drafted changes: %v", err)
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

func TestAnAdminWritesAndPublishesTheFirstPost(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	session := stack.siteAdmin(t, "editor@example.com", "illarin.editor")

	draft := stack.illarinDraft(t, session, "Illarin has a blog again")
	if draft.Status != "draft" {
		t.Errorf("a new post is %q, want draft", draft.Status)
	}
	if draft.Slug != "illarin-has-a-blog-again" {
		t.Errorf("candidate address = %q", draft.Slug)
	}
	if draft.Version != 1 {
		t.Errorf("new drafted changes are version %d, want 1", draft.Version)
	}

	written := stack.saved(t, session, draft.ID, finished(draft, nil))
	if written.Version != 2 {
		t.Errorf("saving left the drafted changes at version %d, want 2", written.Version)
	}
	if stack.read(t, draft.Slug).Code != http.StatusNotFound {
		t.Error("an unpublished post answers on its address")
	}

	live := stack.published(t, session, draft.ID)
	if live.Status != "published" || live.PublishedAt == nil {
		t.Fatalf("publishing left the post %q with published at %v", live.Status, live.PublishedAt)
	}
	if live.UpdatedPublicAt != nil {
		t.Error("a first publication records an updated date")
	}

	found := stack.reader(t, draft.Slug)
	if found.Title != draft.Title || found.Summary != "What Illarin changed this week." {
		t.Errorf("public post = %q / %q", found.Title, found.Summary)
	}
	if len(found.Body.Content) != 1 || found.Body.Version != 2 {
		t.Errorf("public body = %+v", found.Body)
	}
	if found.Byline.Handle != "illarin.editor" {
		t.Errorf("byline = %+v", found.Byline)
	}
}

func TestAWriterPublishesUnderTheirOwnByline(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	announcement := stack.categoryBySlug(t, "announcement")
	session := stack.member(t, "dev@example.com", "lumiverse.dev")

	alone := stack.start(t, session, fmt.Sprintf(
		`{"categoryId":%q,"title":"Not a writer yet"}`, announcement.ID,
	))
	if alone.Code != http.StatusForbidden {
		t.Fatalf("writing without the switch returned %d", alone.Code)
	}

	stack.switchedOn(t, "lumiverse.dev")
	draft := stack.started(t, session, fmt.Sprintf(
		`{"categoryId":%q,"title":"Lumiverse 3 is out"}`, announcement.ID,
	))
	stack.saved(t, session, draft.ID, finished(draft, nil))
	stack.published(t, session, draft.ID)

	found := stack.reader(t, draft.Slug)
	if found.Byline.Handle != "lumiverse.dev" {
		t.Errorf("byline handle = %q", found.Byline.Handle)
	}
}

func TestOneContributorNeverReachesAnothersPost(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	first := stack.contributor(t, "first@example.com", "first.dev")
	announcement := stack.categoryBySlug(t, "announcement")
	second := stack.contributor(t, "second@example.com", "second.dev").session

	draft := stack.started(t, first.session, fmt.Sprintf(
		`{"categoryId":%q,"title":"First post"}`, announcement.ID,
	))

	reading := apitest.Send(t, stack.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/blog/posts/"+draft.ID, nil), second,
	))
	if reading.Code != http.StatusForbidden {
		t.Errorf("another contributor read the post: %d", reading.Code)
	}
	if code := stack.save(t, second, draft.ID, finished(draft, nil)).Code; code != http.StatusForbidden {
		t.Errorf("another contributor saved the post: %d", code)
	}
	if code := stack.publish(t, second, draft.ID).Code; code != http.StatusForbidden {
		t.Errorf("another contributor published the post: %d", code)
	}

	listed := apitest.Send(t, stack.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/blog/posts", nil), second,
	))
	var mine postList
	if err := json.Unmarshal(listed.Body.Bytes(), &mine); err != nil {
		t.Fatalf("decode posts: %v", err)
	}
	if len(mine.Posts) != 0 {
		t.Errorf("another contributor sees %d posts", len(mine.Posts))
	}
}

func TestAModeratorAndAnOrdinaryAccountReachNoPostAtAll(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	writer := stack.contributor(t, "writer@example.com", "writer.dev")
	announcement := stack.categoryBySlug(t, "announcement")
	draft := stack.started(t, writer.session, fmt.Sprintf(
		`{"categoryId":%q,"title":"Quiet news"}`, announcement.ID,
	))

	moderator := stack.member(t, "mod@example.com", "the.moderator")
	apitest.SetRole(t, stack.pool, "the.moderator", "moderator")
	ordinary := stack.member(t, "reader@example.com", "just.reading")

	for name, session := range map[string]*http.Cookie{
		"moderator": moderator, "ordinary account": ordinary,
	} {
		response := apitest.Send(t, stack.router, apitest.Authorized(
			httptest.NewRequest(http.MethodGet, "/v1/blog/posts/"+draft.ID, nil), session,
		))
		if response.Code != http.StatusForbidden {
			t.Errorf("a %s read the draft: %d", name, response.Code)
		}
		if code := stack.start(t, session, fmt.Sprintf(
			`{"categoryId":%q,"title":"Mine now"}`, announcement.ID,
		)).Code; code != http.StatusForbidden {
			t.Errorf("a %s started a post: %d", name, code)
		}
	}
}

func TestAStaleSaveIsRefusedAndLeavesTheNewerDraftedChanges(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	session := stack.siteAdmin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "Two editors one post")

	first := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"title": "The newer title",
	}))

	response := stack.save(t, session, draft.ID, finished(draft, map[string]any{
		"title": "The older title",
	}))
	if response.Code != http.StatusConflict {
		t.Fatalf("a stale save returned %d: %s", response.Code, response.Body.String())
	}
	var conflict struct {
		Error   string `json:"error"`
		Version int    `json:"version"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &conflict); err != nil {
		t.Fatalf("decode conflict: %v", err)
	}
	if conflict.Version != first.Version {
		t.Errorf("conflict names version %d, want %d", conflict.Version, first.Version)
	}
	if conflict.Error == "" {
		t.Error("the conflict says nothing about what to do")
	}

	current := apitest.Send(t, stack.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/blog/posts/"+draft.ID, nil), session,
	))
	if title := decodePost(t, current).Title; title != "The newer title" {
		t.Errorf("the stale save overwrote the drafted changes: %q", title)
	}
}

func TestPublicationRefusesAPostThatIsNotFinished(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	session := stack.siteAdmin(t, "editor@example.com", "illarin.editor")

	draft := stack.illarinDraft(t, session, "Nothing written yet")
	if code := stack.publish(t, session, draft.ID).Code; code != http.StatusBadRequest {
		t.Errorf("an empty post published: %d", code)
	}

	stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"summary": "",
	}))
	if code := stack.publish(t, session, draft.ID).Code; code != http.StatusBadRequest {
		t.Errorf("a post with no summary published: %d", code)
	}
}

func TestARefusedBodyNamesWhereItWentWrong(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	session := stack.siteAdmin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "Nothing dangerous here")

	response := stack.save(t, session, draft.ID, finished(draft, map[string]any{
		"body": json.RawMessage(
			`{"version":2,"content":[{"type":"paragraph","content":[` +
				`{"type":"text","text":"Go","marks":[{"type":"link","href":"javascript:alert(1)"}]}]}]}`,
		),
	}))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("a script link saved: %d", response.Code)
	}
	var refusal struct {
		Error string `json:"error"`
		Field string `json:"field"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &refusal); err != nil {
		t.Fatalf("decode refusal: %v", err)
	}
	if !strings.Contains(refusal.Field, "href") {
		t.Errorf("refusal field = %q", refusal.Field)
	}
	if refusal.Error == "" {
		t.Error("the refusal says nothing about what to fix")
	}
}

func TestAPublishedRevisionIsTheOneReadersGetUntilItIsPublishedAgain(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	session := stack.siteAdmin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "First edition")

	written := stack.saved(t, session, draft.ID, finished(draft, nil))
	stack.published(t, session, draft.ID)

	edited := stack.saved(t, session, draft.ID, finished(written, map[string]any{
		"version": written.Version,
		"title":   "Second edition",
		"summary": "Now with more detail.",
	}))
	if edited.Title != "Second edition" {
		t.Fatalf("the drafted changes did not change: %q", edited.Title)
	}
	if still := stack.reader(t, draft.Slug); still.Title != "First edition" {
		t.Errorf("a reader already sees %q", still.Title)
	}

	live := stack.published(t, session, draft.ID)
	if live.UpdatedPublicAt == nil {
		t.Error("republishing left no updated date")
	}
	if now := stack.reader(t, draft.Slug); now.Title != "Second edition" {
		t.Errorf("a reader sees %q after the second publication", now.Title)
	}

	var revisions int
	err := stack.pool.QueryRow(context.Background(),
		`select count(*) from post_revisions where post_id = $1`, draft.ID).Scan(&revisions)
	if err != nil {
		t.Fatalf("count revisions: %v", err)
	}
	if revisions != 2 {
		t.Errorf("two posts left %d revisions", revisions)
	}
}

func TestABylineIsCopiedOnceAndSurvivesAProfileChange(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	session := stack.siteAdmin(t, "editor@example.com", "illarin.editor")
	apitest.SaveProfile(t, stack.router, session, `{"displayName":"The Editor","links":[]}`)

	draft := stack.illarinDraft(t, session, "Signed and dated")
	stack.saved(t, session, draft.ID, finished(draft, nil))
	stack.published(t, session, draft.ID)

	found := stack.reader(t, draft.Slug)
	if found.Byline.DisplayName != "The Editor" {
		t.Fatalf("byline name = %q", found.Byline.DisplayName)
	}

	apitest.SaveProfile(t, stack.router, session, `{"displayName":"Someone Else","links":[]}`)
	after := stack.reader(t, draft.Slug)
	if after.Byline.DisplayName != "The Editor" {
		t.Errorf("a profile change rewrote the byline to %q", after.Byline.DisplayName)
	}
}

func TestPublishingRecordsOneEventAndOneAuditWithoutTheBody(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	session := stack.siteAdmin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "On the record")
	stack.saved(t, session, draft.ID, finished(draft, nil))
	stack.published(t, session, draft.ID)

	var announcementType string
	err := stack.pool.QueryRow(context.Background(),
		`select type from blog_announcements where post_id = $1`, draft.ID).Scan(&announcementType)
	if err != nil {
		t.Fatalf("read the announcement: %v", err)
	}
	if announcementType != "blog.post.published.v1" {
		t.Errorf("event type = %q", announcementType)
	}

	var action, credential, before, after string
	var revisionID *string
	err = stack.pool.QueryRow(context.Background(), `
		select action, credential, coalesce(before_state, ''), coalesce(after_state, ''),
		       revision_id::text
		  from blog_activity_log where post_id = $1 and action = 'post.published'
	`, draft.ID).Scan(&action, &credential, &before, &after, &revisionID)
	if err != nil {
		t.Fatalf("read the publication audit: %v", err)
	}
	if credential != "session" || before != "draft" || after != "published" {
		t.Errorf("audit = %s %s %s -> %s", action, credential, before, after)
	}
	if revisionID == nil {
		t.Error("the audit does not name the revision it captured")
	}

	var rows int
	err = stack.pool.QueryRow(context.Background(), `
		select count(*) from blog_activity_log
		 where post_id = $1 and (before_state ilike '%Illarin now keeps%'
		    or after_state ilike '%Illarin now keeps%')
	`, draft.ID).Scan(&rows)
	if err != nil {
		t.Fatalf("search the audit for the body: %v", err)
	}
	if rows != 0 {
		t.Error("the audit copied the post body")
	}
}

func TestAPostAddressCannotTakeABlogRouteOrAnotherPosts(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	session := stack.siteAdmin(t, "editor@example.com", "illarin.editor")

	first := stack.illarinDraft(t, session, "Taken already")
	stack.saved(t, session, first.ID, finished(first, nil))

	reserved := stack.illarinDraft(t, session, "Reserved words")
	if code := stack.save(t, session, reserved.ID, finished(reserved, map[string]any{
		"slug": "category",
	})).Code; code != http.StatusBadRequest {
		t.Errorf("a reserved address saved: %d", code)
	}
	if code := stack.save(t, session, reserved.ID, finished(reserved, map[string]any{
		"slug": first.Slug,
	})).Code; code != http.StatusBadRequest {
		t.Errorf("a taken address saved: %d", code)
	}
}

func TestSwitchingAWriterOffEndsPostAccessAndLeavesThePublishedPost(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	writer := stack.contributor(t, "writer@example.com", "writer.dev")
	announcement := stack.categoryBySlug(t, "announcement")
	draft := stack.started(t, writer.session, fmt.Sprintf(
		`{"categoryId":%q,"title":"Still readable"}`, announcement.ID,
	))
	stack.saved(t, writer.session, draft.ID, finished(draft, nil))
	stack.published(t, writer.session, draft.ID)

	revoke := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodDelete, "/v1/blog/writers/"+writer.writer.AccountID, "",
	), stack.admin))
	if revoke.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d: %s", revoke.Code, revoke.Body.String())
	}

	response := apitest.Send(t, stack.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/blog/posts/"+draft.ID, nil), writer.session,
	))
	if response.Code != http.StatusForbidden {
		t.Errorf("a former writer still reads the post: %d", response.Code)
	}
	if found := stack.reader(t, draft.Slug); found.Title != "Still readable" {
		t.Errorf("the switch changed the public post: %q", found.Title)
	}
}

func TestARevisionIsNotRewrittenWhenTheDraftedChangesChanges(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	session := stack.siteAdmin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "Held still")
	written := stack.saved(t, session, draft.ID, finished(draft, nil))
	stack.published(t, session, draft.ID)

	var captured string
	err := stack.pool.QueryRow(context.Background(),
		`select title from post_revisions where post_id = $1 and number = 1`, draft.ID).Scan(&captured)
	if err != nil {
		t.Fatalf("read the revision: %v", err)
	}

	stack.saved(t, session, draft.ID, finished(written, map[string]any{
		"version": written.Version,
		"title":   "Moved on",
	}))

	var after string
	err = stack.pool.QueryRow(context.Background(),
		`select title from post_revisions where post_id = $1 and number = 1`, draft.ID).Scan(&after)
	if err != nil {
		t.Fatalf("read the revision again: %v", err)
	}
	if after != captured {
		t.Errorf("the revision changed from %q to %q", captured, after)
	}
}

func TestARefusedPublicationLeavesNoRevisionEventOrByline(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	session := stack.siteAdmin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "Not ready yet")

	if code := stack.publish(t, session, draft.ID).Code; code != http.StatusBadRequest {
		t.Fatalf("an unfinished post published: %d", code)
	}

	var revisions, events, bylines int
	err := stack.pool.QueryRow(context.Background(), `
		select (select count(*) from post_revisions where post_id = $1),
		       (select count(*) from blog_announcements where post_id = $1),
		       (select count(*) from post_bylines where post_id = $1)
	`, draft.ID).Scan(&revisions, &events, &bylines)
	if err != nil {
		t.Fatalf("count what publication left behind: %v", err)
	}
	if revisions != 0 || events != 0 || bylines != 0 {
		t.Errorf("a refused publication left %d revisions, %d events and %d bylines",
			revisions, events, bylines)
	}

	current := apitest.Send(t, stack.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/blog/posts/"+draft.ID, nil), session,
	))
	if status := decodePost(t, current).Status; status != "draft" {
		t.Errorf("the post is %q after a refused publication", status)
	}
}

func TestPublishingRefusesDraftedChangesWhoseTitleWentMissing(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	session := stack.siteAdmin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "A title that goes away")
	stack.saved(t, session, draft.ID, finished(draft, nil))

	_, err := stack.pool.Exec(context.Background(),
		`update posts set title = ' ' where id = $1`, draft.ID)
	if err != nil {
		t.Fatalf("empty the title behind the editor: %v", err)
	}

	response := stack.publish(t, session, draft.ID)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("a post with no title published: %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "title") {
		t.Errorf("the refusal does not name the title: %s", response.Body.String())
	}
}

func TestAnOlderDocumentIsStoredAndPublishedAtTheCurrentVersion(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	session := stack.siteAdmin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "Written a version ago")

	written := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"body": json.RawMessage(
			`{"version":1,"content":[` +
				`{"type":"heading","level":2,"content":[{"type":"text","text":"Release notes"}]},` +
				`{"type":"paragraph","content":[{"type":"text","text":"Stored before headings had addresses."}]}]}`,
		),
	}))
	if written.Body.Version != 2 || written.BodyVersion != 2 {
		t.Fatalf("saved body = version %d, column %d", written.Body.Version, written.BodyVersion)
	}
	if written.Body.Content[0]["anchor"] != "release-notes" {
		t.Errorf("upgraded heading = %+v, want the address its words make", written.Body.Content[0])
	}

	stack.published(t, session, draft.ID)
	found := stack.reader(t, draft.Slug)
	if found.Body.Version != 2 {
		t.Errorf("published body = version %d, want the current one", found.Body.Version)
	}
	if found.Body.Content[0]["anchor"] != "release-notes" {
		t.Errorf("published heading = %+v", found.Body.Content[0])
	}
}

func TestEveryStructureSurvivesTheRoundTripThroughStorage(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	session := stack.siteAdmin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "Every structure at once")

	body := readCorpus(t, "a-dense-technical-post.json")
	written := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"body": body,
	}))
	stored, err := json.Marshal(written.Body)
	if err != nil {
		t.Fatalf("encode the stored body: %v", err)
	}
	stack.published(t, session, draft.ID)
	public, err := json.Marshal(stack.reader(t, draft.Slug).Body)
	if err != nil {
		t.Fatalf("encode the public body: %v", err)
	}
	if !bytes.Equal(stored, public) {
		t.Errorf("the published body differs from the drafted changes:\n%s\n%s", stored, public)
	}
	types := map[string]bool{}
	for _, block := range written.Body.Content {
		types[fmt.Sprint(block["type"])] = true
	}
	for _, want := range []string{
		"paragraph", "heading", "bulletList", "taskList", "quote",
		"codeBlock", "table", "callout", "divider",
	} {
		if !types[want] {
			t.Errorf("the stored body lost its %s", want)
		}
	}
}

func readCorpus(t *testing.T, name string) json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(
		"body", "testdata", "corpus", "valid", name,
	))
	if err != nil {
		t.Fatalf("read the corpus: %v", err)
	}
	var one struct {
		Body json.RawMessage `json:"document"`
	}
	if err := json.Unmarshal(raw, &one); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return one.Body
}
