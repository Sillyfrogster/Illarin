package http

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
)

type postAuthor struct {
	Handle string `json:"handle"`
}

type postRelease struct {
	App     publicationApp `json:"app"`
	Version string         `json:"version"`
	Address string         `json:"address"`
}

type postByline struct {
	Handle       string           `json:"handle"`
	DisplayName  string           `json:"displayName"`
	ContactEmail string           `json:"contactEmail"`
	Avatar       *distinctionMark `json:"avatar"`
	Positions    []string         `json:"positions"`
	Distinctions []string         `json:"distinctions"`
	App          *publicationApp  `json:"app"`
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
	ID          string              `json:"id"`
	Slug        string              `json:"slug"`
	Title       string              `json:"title"`
	Summary     string              `json:"summary"`
	Category    publicationCategory `json:"category"`
	Document    postDocument        `json:"document"`
	Release     *postRelease        `json:"release"`
	Header      *postHeader         `json:"header"`
	SocialImage *postPicture        `json:"socialImage"`
	Media       []postPicture       `json:"media"`
	Byline      postByline          `json:"byline"`
	PublishedAt time.Time           `json:"publishedAt"`
	UpdatedAt   *time.Time          `json:"updatedAt"`
}

// paragraph is a one-sentence post body in the document vocabulary.
func paragraph(words string) string {
	return fmt.Sprintf(
		`{"version":2,"content":[{"type":"paragraph","content":[{"type":"text","text":%q}]}]}`,
		words,
	)
}

func (s distinctionStack) start(
	t *testing.T,
	session *http.Cookie,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts", body,
	), session))
}

func (s distinctionStack) started(t *testing.T, session *http.Cookie, body string) blogPost {
	t.Helper()
	response := s.start(t, session, body)
	if response.Code != http.StatusCreated {
		t.Fatalf("start post status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s distinctionStack) save(
	t *testing.T,
	session *http.Cookie,
	id string,
	working map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(working)
	if err != nil {
		t.Fatalf("encode working copy: %v", err)
	}
	return send(t, s.router, authorized(jsonRequest(t,
		http.MethodPut, "/v1/publication/posts/"+id, string(body),
	), session))
}

// finished is the working copy of a post that has everything publication asks for.
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

func (s distinctionStack) saved(
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

func (s distinctionStack) publish(
	t *testing.T,
	session *http.Cookie,
	id string,
) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+id+"/publish", "",
	), session))
}

func (s distinctionStack) published(t *testing.T, session *http.Cookie, id string) blogPost {
	t.Helper()
	response := s.publish(t, session, id)
	if response.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s distinctionStack) read(t *testing.T, slug string) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, httptest.NewRequest(http.MethodGet, "/v1/posts/"+slug, nil))
}

func (s distinctionStack) reader(t *testing.T, slug string) publicPost {
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

// admin is a verified account carrying the admin role.
func (s distinctionStack) admin(t *testing.T, email, handle string) *http.Cookie {
	t.Helper()
	session := s.member(t, email, handle)
	setRole(t, s.pool, handle, "admin")
	return session
}

func (s distinctionStack) illarinDraft(t *testing.T, session *http.Cookie, title string) blogPost {
	t.Helper()
	announcement := s.categoryBySlug(t, "announcement")
	return s.started(t, session, fmt.Sprintf(
		`{"categoryId":%q,"title":%q}`, announcement.ID, title,
	))
}

func TestAnAdminWritesAndPublishesTheFirstPost(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

	draft := stack.illarinDraft(t, session, "Illarin has a blog again")
	if draft.Status != "draft" {
		t.Errorf("a new post is %q, want draft", draft.Status)
	}
	if draft.Slug != "illarin-has-a-blog-again" {
		t.Errorf("candidate address = %q", draft.Slug)
	}
	if draft.Version != 1 {
		t.Errorf("a new working copy is version %d, want 1", draft.Version)
	}

	written := stack.saved(t, session, draft.ID, finished(draft, nil))
	if written.Version != 2 {
		t.Errorf("saving left the working copy at version %d, want 2", written.Version)
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
	if len(found.Document.Content) != 1 || found.Document.Version != 2 {
		t.Errorf("public body = %+v", found.Document)
	}
	if found.Byline.Handle != "illarin.editor" || found.Byline.App != nil {
		t.Errorf("byline = %+v, want an Illarin byline", found.Byline)
	}
}

func TestAContributorPublishesUnderTheirGrantAndNobodyElses(t *testing.T) {
	stack := newDistinctionStack(t)
	lumiverse := stack.configureApp(t, "lumiverse", "Lumiverse", "https://lumiverse.example")
	announcement := stack.categoryBySlug(t, "announcement")
	article := stack.categoryBySlug(t, "article")
	session := stack.member(t, "dev@example.com", "lumiverse.dev")
	grant := stack.approved(t, "lumiverse.dev", lumiverse.ID, []string{announcement.ID}, announcement.ID)

	refused := stack.start(t, session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"An article"}`, grant.ID, article.ID,
	))
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("a category outside the grant returned %d: %s", refused.Code, refused.Body.String())
	}

	alone := stack.start(t, session, fmt.Sprintf(
		`{"categoryId":%q,"title":"No grant named"}`, announcement.ID,
	))
	if alone.Code != http.StatusForbidden {
		t.Fatalf("writing with no grant returned %d", alone.Code)
	}

	draft := stack.started(t, session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"Lumiverse 3 is out"}`, grant.ID, announcement.ID,
	))
	stack.saved(t, session, draft.ID, finished(draft, nil))
	stack.published(t, session, draft.ID)

	found := stack.reader(t, draft.Slug)
	if found.Byline.App == nil || found.Byline.App.Slug != "lumiverse" {
		t.Fatalf("byline app = %+v, want Lumiverse", found.Byline.App)
	}
	if found.Byline.Handle != "lumiverse.dev" {
		t.Errorf("byline handle = %q", found.Byline.Handle)
	}
}

func TestOneContributorNeverReachesAnothersPost(t *testing.T) {
	stack := newDistinctionStack(t)
	first := stack.contributor(t, "first@example.com", "first.dev")
	sillytavern := stack.configureApp(t, "sillytavern", "SillyTavern", "https://sillytavern.example")
	announcement := stack.categoryBySlug(t, "announcement")
	second := stack.member(t, "second@example.com", "second.dev")
	stack.approved(t, "second.dev", sillytavern.ID, []string{announcement.ID}, announcement.ID)

	draft := stack.started(t, first.session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"First post"}`, first.grant.ID, announcement.ID,
	))

	reading := send(t, stack.router, authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/posts/"+draft.ID, nil), second,
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

	listed := send(t, stack.router, authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/posts", nil), second,
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
	stack := newDistinctionStack(t)
	writer := stack.contributor(t, "writer@example.com", "writer.dev")
	announcement := stack.categoryBySlug(t, "announcement")
	draft := stack.started(t, writer.session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"Quiet news"}`, writer.grant.ID, announcement.ID,
	))

	moderator := stack.member(t, "mod@example.com", "the.moderator")
	setRole(t, stack.pool, "the.moderator", "moderator")
	ordinary := stack.member(t, "reader@example.com", "just.reading")

	for name, session := range map[string]*http.Cookie{
		"moderator": moderator, "ordinary account": ordinary,
	} {
		response := send(t, stack.router, authorized(
			httptest.NewRequest(http.MethodGet, "/v1/publication/posts/"+draft.ID, nil), session,
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

func TestAStaleSaveIsRefusedAndLeavesTheNewerWorkingCopy(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
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

	current := send(t, stack.router, authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/posts/"+draft.ID, nil), session,
	))
	if title := decodePost(t, current).Title; title != "The newer title" {
		t.Errorf("the stale save overwrote the working copy: %q", title)
	}
}

func TestPublicationRefusesAPostThatIsNotFinished(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

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

func TestAReleaseNeedsTheProjectAndVersionItAnnounces(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	release := stack.categoryBySlug(t, "release")
	illarin := stack.appBySlug(t, "illarin")

	draft := stack.started(t, session, fmt.Sprintf(
		`{"categoryId":%q,"title":"Illarin 2.0"}`, release.ID,
	))
	bare := stack.save(t, session, draft.ID, finished(draft, nil))
	if bare.Code != http.StatusBadRequest {
		t.Fatalf("a release with no version saved: %d", bare.Code)
	}

	stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"release": map[string]any{
			"appId": illarin.ID, "version": "2.0", "address": "https://illarin.xyz/releases/2-0",
		},
	}))
	stack.published(t, session, draft.ID)

	found := stack.reader(t, draft.Slug)
	if found.Release == nil || found.Release.Version != "2.0" {
		t.Fatalf("public release = %+v", found.Release)
	}
	if found.Release.App.Slug != "illarin" {
		t.Errorf("release app = %q", found.Release.App.Slug)
	}
}

func TestARefusedReleaseAddressNamesTheField(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	release := stack.categoryBySlug(t, "release")
	illarin := stack.appBySlug(t, "illarin")

	draft := stack.started(t, session, fmt.Sprintf(
		`{"categoryId":%q,"title":"Illarin 2.1"}`, release.ID,
	))
	response := stack.save(t, session, draft.ID, finished(draft, map[string]any{
		"release": map[string]any{
			"appId": illarin.ID, "version": "2.1", "address": "http://illarin.xyz/releases",
		},
	}))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("a plain http release address saved: %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "release.address") {
		t.Errorf("the refusal does not name the field: %s", response.Body.String())
	}
}

func TestARefusedDocumentNamesWhereItWentWrong(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "Nothing dangerous here")

	response := stack.save(t, session, draft.ID, finished(draft, map[string]any{
		"document": json.RawMessage(
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
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "First edition")

	written := stack.saved(t, session, draft.ID, finished(draft, nil))
	stack.published(t, session, draft.ID)

	edited := stack.saved(t, session, draft.ID, finished(written, map[string]any{
		"version": written.Version,
		"title":   "Second edition",
		"summary": "Now with more detail.",
	}))
	if edited.Title != "Second edition" {
		t.Fatalf("the working copy did not change: %q", edited.Title)
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
		t.Errorf("two publications left %d revisions", revisions)
	}
}

func TestABylineIsCopiedOnceAndSurvivesAProfileChange(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	saveProfile(t, stack.router, session, `{"displayName":"The Editor","links":[]}`)
	founder := stack.defined(t, "position", "Founder", "Runs Illarin.")
	stack.assigned(t, "illarin.editor", founder.ID)

	draft := stack.illarinDraft(t, session, "Signed and dated")
	stack.saved(t, session, draft.ID, finished(draft, nil))
	stack.published(t, session, draft.ID)

	found := stack.reader(t, draft.Slug)
	if found.Byline.DisplayName != "The Editor" {
		t.Fatalf("byline name = %q", found.Byline.DisplayName)
	}
	if len(found.Byline.Positions) != 1 || found.Byline.Positions[0] != "Founder" {
		t.Fatalf("byline positions = %v", found.Byline.Positions)
	}

	saveProfile(t, stack.router, session, `{"displayName":"Someone Else","links":[]}`)
	after := stack.reader(t, draft.Slug)
	if after.Byline.DisplayName != "The Editor" {
		t.Errorf("a profile change rewrote the byline to %q", after.Byline.DisplayName)
	}
}

func TestPublishingRecordsOneEventAndOneAuditWithoutTheBody(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "On the record")
	stack.saved(t, session, draft.ID, finished(draft, nil))
	stack.published(t, session, draft.ID)

	var eventType string
	err := stack.pool.QueryRow(context.Background(),
		`select type from publication_events where post_id = $1`, draft.ID).Scan(&eventType)
	if err != nil {
		t.Fatalf("read the publication event: %v", err)
	}
	if eventType != "publication.post.published.v1" {
		t.Errorf("event type = %q", eventType)
	}

	var action, credential, before, after string
	var revisionID *string
	err = stack.pool.QueryRow(context.Background(), `
		select action, credential, coalesce(before_state, ''), coalesce(after_state, ''),
		       revision_id::text
		  from publication_audits where post_id = $1 and action = 'post.published'
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
		select count(*) from publication_audits
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
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

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

func TestRevokingAGrantEndsPostAccessAndLeavesThePublishedPost(t *testing.T) {
	stack := newDistinctionStack(t)
	writer := stack.contributor(t, "writer@example.com", "writer.dev")
	announcement := stack.categoryBySlug(t, "announcement")
	draft := stack.started(t, writer.session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"Still readable"}`, writer.grant.ID, announcement.ID,
	))
	stack.saved(t, writer.session, draft.ID, finished(draft, nil))
	stack.published(t, writer.session, draft.ID)

	revoke := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodDelete, "/v1/publication/grants/"+writer.grant.ID, "",
	), stack.authority))
	if revoke.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d: %s", revoke.Code, revoke.Body.String())
	}

	response := send(t, stack.router, authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/posts/"+draft.ID, nil), writer.session,
	))
	if response.Code != http.StatusForbidden {
		t.Errorf("a revoked contributor still reads the post: %d", response.Code)
	}
	if found := stack.reader(t, draft.Slug); found.Title != "Still readable" {
		t.Errorf("revocation changed the public post: %q", found.Title)
	}
}

func TestARevisionIsNotRewrittenWhenTheWorkingCopyChanges(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
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
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "Not ready yet")

	if code := stack.publish(t, session, draft.ID).Code; code != http.StatusBadRequest {
		t.Fatalf("an unfinished post published: %d", code)
	}

	var revisions, events, bylines int
	err := stack.pool.QueryRow(context.Background(), `
		select (select count(*) from post_revisions where post_id = $1),
		       (select count(*) from publication_events where post_id = $1),
		       (select count(*) from post_bylines where post_id = $1)
	`, draft.ID).Scan(&revisions, &events, &bylines)
	if err != nil {
		t.Fatalf("count what publication left behind: %v", err)
	}
	if revisions != 0 || events != 0 || bylines != 0 {
		t.Errorf("a refused publication left %d revisions, %d events and %d bylines",
			revisions, events, bylines)
	}

	current := send(t, stack.router, authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/posts/"+draft.ID, nil), session,
	))
	if status := decodePost(t, current).Status; status != "draft" {
		t.Errorf("the post is %q after a refused publication", status)
	}
}

func TestPublishingRefusesAWorkingCopyWhoseTitleWentMissing(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
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
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "Written a version ago")

	written := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"document": json.RawMessage(
			`{"version":1,"content":[` +
				`{"type":"heading","level":2,"content":[{"type":"text","text":"Release notes"}]},` +
				`{"type":"paragraph","content":[{"type":"text","text":"Stored before headings had addresses."}]}]}`,
		),
	}))
	if written.Document.Version != 2 || written.DocumentVersion != 2 {
		t.Fatalf("saved body = version %d, column %d", written.Document.Version, written.DocumentVersion)
	}
	if written.Document.Content[0]["anchor"] != "release-notes" {
		t.Errorf("upgraded heading = %+v, want the address its words make", written.Document.Content[0])
	}

	stack.published(t, session, draft.ID)
	found := stack.reader(t, draft.Slug)
	if found.Document.Version != 2 {
		t.Errorf("published body = version %d, want the current one", found.Document.Version)
	}
	if found.Document.Content[0]["anchor"] != "release-notes" {
		t.Errorf("published heading = %+v", found.Document.Content[0])
	}
}

func TestEveryStructureSurvivesTheRoundTripThroughStorage(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "Every structure at once")

	body := readCorpus(t, "a-dense-technical-post.json")
	written := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"document": body,
	}))
	stored, err := json.Marshal(written.Document)
	if err != nil {
		t.Fatalf("encode the stored body: %v", err)
	}
	stack.published(t, session, draft.ID)
	public, err := json.Marshal(stack.reader(t, draft.Slug).Document)
	if err != nil {
		t.Fatalf("encode the public body: %v", err)
	}
	if !bytes.Equal(stored, public) {
		t.Errorf("the published body differs from the working copy:\n%s\n%s", stored, public)
	}
	kinds := map[string]bool{}
	for _, block := range written.Document.Content {
		kinds[fmt.Sprint(block["type"])] = true
	}
	for _, want := range []string{
		"paragraph", "heading", "bulletList", "taskList", "quote",
		"codeBlock", "table", "callout", "divider",
	} {
		if !kinds[want] {
			t.Errorf("the stored body lost its %s", want)
		}
	}
}

// readCorpus reads one document from the corpus Go validation and the site share.
func readCorpus(t *testing.T, name string) json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(
		"..", "postdoc", "testdata", "corpus", "valid", name,
	))
	if err != nil {
		t.Fatalf("read the corpus: %v", err)
	}
	var one struct {
		Document json.RawMessage `json:"document"`
	}
	if err := json.Unmarshal(raw, &one); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return one.Document
}
