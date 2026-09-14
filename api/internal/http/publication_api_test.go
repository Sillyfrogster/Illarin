package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
)

type publicationRefusal struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Field   string `json:"field"`
	Version *int   `json:"version"`
}

type tooling struct {
	who   contributor
	value string
}

func (s publicationStack) tooling(t *testing.T, email, handle string) tooling {
	t.Helper()
	who := s.contributor(t, email, handle)
	return tooling{who: who, value: s.issued(t, who, "Release robot").Value}
}

func (s publicationStack) sent(
	t *testing.T,
	value string,
	request *http.Request,
) *httptest.ResponseRecorder {
	t.Helper()
	request.Header.Set("Authorization", "Bearer "+value)
	return send(t, s.router, request)
}

func withKey(request *http.Request, key string) *http.Request {
	request.Header.Set(idempotencyKeyHeader, key)
	return request
}

func (s publicationStack) startedByTool(
	t *testing.T,
	kit tooling,
	body string,
) blogPost {
	t.Helper()
	response := s.sent(t, kit.value, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts", body,
	))
	if response.Code != http.StatusCreated {
		t.Fatalf("start post status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s publicationStack) toolPosts(t *testing.T, value string) []blogPost {
	t.Helper()
	response := s.sent(t, value, httptest.NewRequest(
		http.MethodGet, "/v1/publication/posts", nil,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("list posts status = %d: %s", response.Code, response.Body.String())
	}
	var listed postList
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode posts: %v", err)
	}
	return listed.Posts
}

func (s publicationStack) uploadedByTool(
	t *testing.T,
	kit tooling,
	postID, purpose string,
	file []byte,
) postPicture {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	writeMetadataPart(t, form, map[string]any{"purpose": purpose})
	writeFilePartNamed(t, form, "picture.png", file)
	if err := form.Close(); err != nil {
		t.Fatalf("close picture form: %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPost, "/v1/publication/posts/"+postID+"/media", &body,
	)
	request.Header.Set("Content-Type", form.FormDataContentType())
	response := s.sent(t, kit.value, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload status = %d: %s", response.Code, response.Body.String())
	}
	var picture postPicture
	if err := json.Unmarshal(response.Body.Bytes(), &picture); err != nil {
		t.Fatalf("decode picture: %v", err)
	}
	return picture
}

func refusalOf(t *testing.T, response *httptest.ResponseRecorder) publicationRefusal {
	t.Helper()
	var refused publicationRefusal
	if err := json.Unmarshal(response.Body.Bytes(), &refused); err != nil {
		t.Fatalf("decode refusal %q: %v", response.Body.String(), err)
	}
	return refused
}

func newPacedPublicationStack(t *testing.T, rates publication.Rates) publicationStack {
	t.Helper()
	pool := testdb.Connect(t)
	outbox := &verificationOutbox{}
	handlers := newTestHandlersWithDelivery(
		t, pool, 1<<20, outbox, testDeliverySettings(), rates, nil,
	)
	router := registerTestRouter(t, handlers, DefaultDeadlines())
	session := verifiedSignUp(t, router, outbox, "authority@example.com", "publication.authority")
	holdsAuthority(t, pool, "publication.authority")
	return publicationStack{router: router, pool: pool, outbox: outbox, authority: session}
}

func TestATokenRunsTheWholeContributorWorkflow(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	kit := stack.tooling(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")

	if opening := stack.toolPosts(t, kit.value); len(opening) != 0 {
		t.Fatalf("a new grant already has %d posts", len(opening))
	}

	draft := stack.startedByTool(t, kit, fmt.Sprintf(
		`{"categoryId":%q,"title":"Lumiverse 3 is out"}`, announcement.ID,
	))
	if draft.Status != "draft" || draft.GrantID != kit.who.grant.ID {
		t.Fatalf("a token started %+v", draft)
	}

	picture := stack.uploadedByTool(t, kit, draft.ID, "document", httpTestPNG(t, 900, 500))
	working := finished(draft, map[string]any{
		"document": bodyWithPicture(picture.ID, "The workspace"),
	})
	body, err := json.Marshal(working)
	if err != nil {
		t.Fatalf("encode working copy: %v", err)
	}
	saved := stack.sent(t, kit.value, jsonRequest(t,
		http.MethodPut, "/v1/publication/posts/"+draft.ID, string(body),
	))
	if saved.Code != http.StatusOK {
		t.Fatalf("save status = %d: %s", saved.Code, saved.Body.String())
	}
	held := decodePost(t, saved)
	if held.Status != "draft" {
		t.Fatalf("saving a working copy made the post %q", held.Status)
	}

	kept := stack.sent(t, kit.value, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+draft.ID+"/revisions",
		fmt.Sprintf(`{"version":%d}`, held.Version),
	))
	if kept.Code != http.StatusCreated {
		t.Fatalf("checkpoint status = %d: %s", kept.Code, kept.Body.String())
	}

	published := stack.sent(t, kit.value, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+draft.ID+"/publish",
		fmt.Sprintf(`{"version":%d}`, held.Version),
	))
	if published.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", published.Code, published.Body.String())
	}
	public := decodePost(t, published)
	if public.Status != "published" || public.PublicRevision == "" {
		t.Fatalf("a token published %+v", public)
	}

	reader := stack.reader(t, public.Slug)
	if reader.Byline.Handle != "publication.writer" || reader.Byline.App == nil {
		t.Fatalf("the published post reads as %+v", reader.Byline)
	}
}

func TestSavingAWorkingCopyIsNeverAPublicTransition(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	kit := stack.tooling(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")

	draft := stack.startedByTool(t, kit, fmt.Sprintf(
		`{"categoryId":%q,"title":"Quiet by default"}`, announcement.ID,
	))
	working := finished(draft, map[string]any{
		"status":         "published",
		"publishedAt":    time.Now(),
		"publish":        true,
		"publicRevision": draft.ID,
	})
	body, err := json.Marshal(working)
	if err != nil {
		t.Fatalf("encode working copy: %v", err)
	}
	saved := stack.sent(t, kit.value, jsonRequest(t,
		http.MethodPut, "/v1/publication/posts/"+draft.ID, string(body),
	))
	if saved.Code != http.StatusBadRequest && saved.Code != http.StatusOK {
		t.Fatalf("save status = %d: %s", saved.Code, saved.Body.String())
	}

	after := stack.sent(t, kit.value, httptest.NewRequest(
		http.MethodGet, "/v1/publication/posts/"+draft.ID, nil,
	))
	held := decodePost(t, after)
	if held.Status != "draft" || held.PublishedAt != nil {
		t.Fatalf("a save published the post: %+v", held)
	}
	if reading := stack.read(t, draft.Slug); reading.Code != http.StatusNotFound {
		t.Fatalf("a saved draft answered readers with %d", reading.Code)
	}
}

func TestATokenWritesUnderItsOwnGrantAndNoOther(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	kit := stack.tooling(t, "writer@example.com", "publication.writer")
	other := stack.contributor(t, "another@example.com", "publication.another")
	announcement := stack.categoryBySlug(t, "announcement")
	article := stack.categoryBySlug(t, "article")

	own := stack.startedByTool(t, kit, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"Named its own grant"}`,
		kit.who.grant.ID, announcement.ID,
	))
	if own.GrantID != kit.who.grant.ID {
		t.Fatalf("naming its own grant produced %q", own.GrantID)
	}

	borrowed := stack.sent(t, kit.value, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts", fmt.Sprintf(
			`{"grantId":%q,"categoryId":%q,"title":"Not its grant"}`,
			other.grant.ID, announcement.ID,
		),
	))
	if borrowed.Code != http.StatusForbidden {
		t.Fatalf("naming another grant answered %d: %s", borrowed.Code, borrowed.Body.String())
	}

	outside := stack.sent(t, kit.value, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts", fmt.Sprintf(
			`{"categoryId":%q,"title":"Outside the approval"}`, article.ID,
		),
	))
	if outside.Code != http.StatusBadRequest ||
		refusalOf(t, outside).Code != string(CodeCategoryRefused) {
		t.Fatalf("a category outside the grant answered %d: %s",
			outside.Code, outside.Body.String())
	}
}

func TestATokenReachesOnlyThePostsUnderItsOwnGrant(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	mine := stack.tooling(t, "writer@example.com", "publication.writer")
	theirs := stack.contributor(t, "another@example.com", "publication.another")
	announcement := stack.categoryBySlug(t, "announcement")

	stack.startedByTool(t, mine, fmt.Sprintf(
		`{"categoryId":%q,"title":"Mine alone"}`, announcement.ID,
	))
	hidden := stack.started(t, theirs.session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"Theirs alone"}`,
		theirs.grant.ID, announcement.ID,
	))

	listed := stack.toolPosts(t, mine.value)
	if len(listed) != 1 || listed[0].Title != "Mine alone" {
		t.Fatalf("a token listed %d posts: %+v", len(listed), listed)
	}
	for _, address := range []string{
		"/v1/publication/posts/" + hidden.ID,
		"/v1/publication/posts/" + hidden.ID + "/revisions",
		"/v1/publication/posts/" + hidden.ID + "/history",
	} {
		response := stack.sent(t, mine.value, httptest.NewRequest(http.MethodGet, address, nil))
		if response.Code != http.StatusForbidden {
			t.Errorf("%s answered %d to another grant's token: %s",
				address, response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "publication.another") {
			t.Errorf("%s named the other contributor: %s", address, response.Body.String())
		}
	}
}

func TestAStaleWorkingCopyConflictsWithEnoughToReload(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	kit := stack.tooling(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")

	draft := stack.startedByTool(t, kit, fmt.Sprintf(
		`{"categoryId":%q,"title":"Two writers"}`, announcement.ID,
	))
	stack.saved(t, kit.who.session, draft.ID, finished(draft, nil))

	body, err := json.Marshal(finished(draft, map[string]any{"title": "From the tool"}))
	if err != nil {
		t.Fatalf("encode working copy: %v", err)
	}
	stale := stack.sent(t, kit.value, jsonRequest(t,
		http.MethodPut, "/v1/publication/posts/"+draft.ID, string(body),
	))
	if stale.Code != http.StatusConflict {
		t.Fatalf("a stale save answered %d: %s", stale.Code, stale.Body.String())
	}
	refused := refusalOf(t, stale)
	if refused.Code != string(CodeStaleVersion) || refused.Version == nil {
		t.Fatalf("a stale save said %+v", refused)
	}
	if *refused.Version != stack.working(t, kit.who.session, draft.ID).Version {
		t.Fatalf("the conflict named version %d", *refused.Version)
	}
}

func TestARetriedMutationReturnsItsFirstOutcome(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	kit := stack.tooling(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")
	body := fmt.Sprintf(`{"categoryId":%q,"title":"Sent twice"}`, announcement.ID)

	first := stack.sent(t, kit.value, withKey(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts", body,
	), "release-2026-09-04"))
	second := stack.sent(t, kit.value, withKey(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts", body,
	), "release-2026-09-04"))

	if first.Code != http.StatusCreated || second.Code != first.Code {
		t.Fatalf("the retry answered %d after %d", second.Code, first.Code)
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("the retry answered differently:\n%s\n%s", first.Body, second.Body)
	}
	if listed := stack.toolPosts(t, kit.value); len(listed) != 1 {
		t.Fatalf("a retried create left %d posts", len(listed))
	}
}

func TestAReusedKeyWithADifferentRequestConflicts(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	kit := stack.tooling(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")

	stack.sent(t, kit.value, withKey(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts",
		fmt.Sprintf(`{"categoryId":%q,"title":"The first one"}`, announcement.ID),
	), "one-key-only"))
	clashing := stack.sent(t, kit.value, withKey(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts",
		fmt.Sprintf(`{"categoryId":%q,"title":"A different one"}`, announcement.ID),
	), "one-key-only"))

	if clashing.Code != http.StatusConflict ||
		refusalOf(t, clashing).Code != string(CodeIdempotencyMismatch) {
		t.Fatalf("a reused key answered %d: %s", clashing.Code, clashing.Body.String())
	}
	if listed := stack.toolPosts(t, kit.value); len(listed) != 1 {
		t.Fatalf("a refused retry left %d posts", len(listed))
	}
}

func TestAnIdempotencyKeyBelongsToOneCredentialAndOperation(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	mine := stack.tooling(t, "writer@example.com", "publication.writer")
	theirs := stack.tooling(t, "another@example.com", "publication.another")
	announcement := stack.categoryBySlug(t, "announcement")
	body := fmt.Sprintf(`{"categoryId":%q,"title":"Same key"}`, announcement.ID)

	first := stack.sent(t, mine.value, withKey(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts", body,
	), "shared-key-value"))
	second := stack.sent(t, theirs.value, withKey(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts", body,
	), "shared-key-value"))

	if first.Code != http.StatusCreated || second.Code != http.StatusCreated {
		t.Fatalf("two credentials sharing a key answered %d and %d", first.Code, second.Code)
	}
	if decodePost(t, first).ID == decodePost(t, second).ID {
		t.Fatal("one credential was given another's outcome")
	}
	if len(stack.toolPosts(t, mine.value)) != 1 || len(stack.toolPosts(t, theirs.value)) != 1 {
		t.Fatal("a shared key crossed between grants")
	}
}

func TestThePaceHoldsATokenAndLeavesTheEditorAlone(t *testing.T) {
	t.Parallel()
	rates := publication.Rates{
		Read:   publication.Rate{Attempts: 2, Window: time.Minute},
		Write:  publication.Rate{Attempts: 1, Window: time.Minute},
		Upload: publication.Rate{Attempts: 1, Window: time.Minute},
	}
	stack := newPacedPublicationStack(t, rates)
	kit := stack.tooling(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")

	for attempt := range 2 {
		allowed := stack.sent(t, kit.value, httptest.NewRequest(
			http.MethodGet, "/v1/publication/posts", nil,
		))
		if allowed.Code != http.StatusOK {
			t.Fatalf("read %d answered %d", attempt+1, allowed.Code)
		}
	}
	limited := stack.sent(t, kit.value, httptest.NewRequest(
		http.MethodGet, "/v1/publication/posts", nil,
	))
	if limited.Code != http.StatusTooManyRequests ||
		refusalOf(t, limited).Code != string(CodeRateLimited) {
		t.Fatalf("a third read answered %d: %s", limited.Code, limited.Body.String())
	}
	if limited.Header().Get("Retry-After") == "" {
		t.Fatal("a paced refusal did not say when to come back")
	}

	for range 3 {
		editing := send(t, stack.router, authorized(httptest.NewRequest(
			http.MethodGet, "/v1/publication/posts", nil,
		), kit.who.session))
		if editing.Code != http.StatusOK {
			t.Fatalf("the editor was paced too: %d", editing.Code)
		}
	}

	started := stack.sent(t, kit.value, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts",
		fmt.Sprintf(`{"categoryId":%q,"title":"One write"}`, announcement.ID),
	))
	if started.Code != http.StatusCreated {
		t.Fatalf("the first write answered %d: %s", started.Code, started.Body.String())
	}
	again := stack.sent(t, kit.value, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts",
		fmt.Sprintf(`{"categoryId":%q,"title":"Two writes"}`, announcement.ID),
	))
	if again.Code != http.StatusTooManyRequests {
		t.Fatalf("a second write answered %d", again.Code)
	}
}

func TestAPublicationTokenIsRefusedFromABrowser(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	kit := stack.tooling(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")

	for _, origin := range []string{testBrowserOrigin, "https://blog.illarin.xyz"} {
		request := jsonRequest(t, http.MethodPost, "/v1/publication/posts", fmt.Sprintf(
			`{"categoryId":%q,"title":"From a page"}`, announcement.ID,
		))
		request.Header.Set("Origin", origin)
		response := stack.sent(t, kit.value, request)
		if response.Code != http.StatusUnauthorized {
			t.Errorf("a token from %s answered %d: %s",
				origin, response.Code, response.Body.String())
		}
	}
	if listed := stack.toolPosts(t, kit.value); len(listed) != 0 {
		t.Fatalf("a browser request wrote %d posts", len(listed))
	}
}

func TestEveryPublicationRefusalNamesItselfAndNobodyElse(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	kit := stack.tooling(t, "writer@example.com", "publication.writer")
	expiring := stack.issued(t, kit.who, "Short life")
	revoked := stack.issued(t, kit.who, "Gone")

	if response := stack.revokeToken(t, kit.who.session, revoked.Token.ID); response.Code !=
		http.StatusNoContent {
		t.Fatalf("revoke token status = %d", response.Code)
	}
	expireToken(t, stack, expiring.Token.ID)

	refusals := []struct {
		name  string
		value string
		want  PublicationErrorCode
	}{
		{"an unknown token", kit.value[:len(kit.value)-1] + "x", CodeUnauthenticated},
		{"an expired token", expiring.Value, CodeTokenExpired},
		{"a revoked token", revoked.Value, CodeTokenRevoked},
	}
	for _, refusal := range refusals {
		response := stack.sent(t, refusal.value, httptest.NewRequest(
			http.MethodGet, "/v1/publication/posts", nil,
		))
		if response.Code != http.StatusUnauthorized {
			t.Errorf("%s answered %d", refusal.name, response.Code)
		}
		if code := refusalOf(t, response).Code; code != string(refusal.want) {
			t.Errorf("%s said %q, want %q", refusal.name, code, refusal.want)
		}
		if strings.Contains(response.Body.String(), "publication.writer") {
			t.Errorf("%s named the contributor: %s", refusal.name, response.Body.String())
		}
	}

	dropGrant(t, stack, kit.who.grant.ID)
	gone := stack.sent(t, kit.value, httptest.NewRequest(
		http.MethodGet, "/v1/publication/posts", nil,
	))
	if gone.Code != http.StatusUnauthorized ||
		refusalOf(t, gone).Code != string(CodeGrantRevoked) {
		t.Fatalf("a revoked grant answered %d: %s", gone.Code, gone.Body.String())
	}
}

func expireToken(t *testing.T, stack publicationStack, id string) {
	t.Helper()
	_, err := stack.pool.Exec(context.Background(), `
		update publication_tokens
		   set created_at = now() - interval '2 hours', expires_at = now() - interval '1 hour'
		 where id = $1
	`, id)
	if err != nil {
		t.Fatalf("expire token: %v", err)
	}
}

func dropGrant(t *testing.T, stack publicationStack, id string) {
	t.Helper()
	response := send(t, stack.router, authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/publication/grants/"+id, nil,
	), stack.authority))
	if response.Code != http.StatusNoContent {
		t.Fatalf("revoke grant status = %d: %s", response.Code, response.Body.String())
	}
}

func TestAKeyStillRunningRefusesRatherThanRepeatingTheWork(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	kit := stack.tooling(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")
	claimOnly(t, stack, kit, "POST /v1/publication/posts", "still-running")

	answered := stack.sent(t, kit.value, withKey(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts",
		fmt.Sprintf(`{"categoryId":%q,"title":"Held up"}`, announcement.ID),
	), "still-running"))

	if answered.Code != http.StatusConflict ||
		refusalOf(t, answered).Code != string(CodeIdempotencyInProgress) {
		t.Fatalf("a running key answered %d: %s", answered.Code, answered.Body.String())
	}
	if listed := stack.toolPosts(t, kit.value); len(listed) != 0 {
		t.Fatalf("a refused retry wrote %d posts", len(listed))
	}
}

func TestAnIdempotencyKeyStopsBeingKeptAfterItsWindow(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	kit := stack.tooling(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")
	body := fmt.Sprintf(`{"categoryId":%q,"title":"Kept for a day"}`, announcement.ID)

	first := stack.sent(t, kit.value, withKey(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts", body,
	), "kept-for-a-day"))
	if first.Code != http.StatusCreated {
		t.Fatalf("the first request answered %d: %s", first.Code, first.Body.String())
	}

	ageEveryKey(t, stack)
	sweeper := publication.NewService(
		stack.pool, nil, publication.DefaultRates(), testPublishing(nil),
	)
	swept, err := sweeper.SweepAttempts(context.Background())
	if err != nil {
		t.Fatalf("sweep keys: %v", err)
	}
	if swept != 1 {
		t.Fatalf("the sweep dropped %d keys, want 1", swept)
	}

	second := stack.sent(t, kit.value, withKey(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts", body,
	), "kept-for-a-day"))
	if second.Code != http.StatusCreated ||
		decodePost(t, first).ID == decodePost(t, second).ID {
		t.Fatalf("a swept key replayed: %d %s", second.Code, second.Body.String())
	}
}

func claimOnly(t *testing.T, stack publicationStack, kit tooling, operation, key string) {
	t.Helper()
	_, err := stack.pool.Exec(context.Background(), `
		insert into publication_idempotency (token_id, operation, key)
		select token.id, $2, $3
		  from publication_tokens token
		 where token.grant_id = $1
		 order by token.created_at desc
		 limit 1
	`, kit.who.grant.ID, operation, key)
	if err != nil {
		t.Fatalf("claim a key: %v", err)
	}
}

func ageEveryKey(t *testing.T, stack publicationStack) {
	t.Helper()
	_, err := stack.pool.Exec(context.Background(), `
		update publication_idempotency set claimed_at = now() - interval '2 days'
	`)
	if err != nil {
		t.Fatalf("age the stored keys: %v", err)
	}
}

func TestATokenReadsItsContextAndSubmitsNoIdentityOfItsOwn(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	kit := stack.tooling(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")

	reading := stack.sent(t, kit.value, httptest.NewRequest(
		http.MethodGet, "/v1/publication/token", nil,
	))
	if reading.Code != http.StatusOK {
		t.Fatalf("reading the credential answered %d: %s", reading.Code, reading.Body.String())
	}
	var held publicationCredential
	if err := json.Unmarshal(reading.Body.Bytes(), &held); err != nil {
		t.Fatalf("decode credential: %v", err)
	}
	if held.Grant.App.Slug != "illarin" || held.Grant.DefaultCategory.Slug != "announcement" ||
		len(held.Grant.Categories) != 1 {
		t.Fatalf("the credential carries %+v", held.Grant)
	}

	draft := stack.startedByTool(t, kit, fmt.Sprintf(
		`{"categoryId":%q,"title":"Signed by the grant"}`, announcement.ID,
	))
	stack.saved(t, kit.who.session, draft.ID, finished(draft, nil))
	published := stack.published(t, kit.who.session, draft.ID)
	if published.Byline == nil || published.Byline.Handle != "publication.writer" ||
		published.Byline.App == nil || published.Byline.App.Slug != "illarin" {
		t.Fatalf("the byline reads %+v", published.Byline)
	}

	for _, correction := range []struct {
		address string
		body    string
	}{
		{"/v1/publication/posts/" + draft.ID + "/address", `{"slug":"moved-by-a-tool"}`},
		{"/v1/publication/posts/" + draft.ID + "/byline", `{"handle":"publication.authority"}`},
	} {
		refused := stack.sent(t, kit.value, jsonRequest(t,
			http.MethodPut, correction.address, correction.body,
		))
		if refused.Code != http.StatusForbidden {
			t.Errorf("%s answered a token %d: %s",
				correction.address, refused.Code, refused.Body.String())
		}
	}

	empty := stack.sent(t, kit.value, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts",
		fmt.Sprintf(`{"categoryId":%q,"title":"   "}`, announcement.ID),
	))
	if empty.Code != http.StatusBadRequest || refusalOf(t, empty).Code != string(CodeInvalid) {
		t.Fatalf("an empty title answered %d: %s", empty.Code, empty.Body.String())
	}
	if refusalOf(t, empty).Field != "title" {
		t.Fatalf("an empty title named %q", refusalOf(t, empty).Field)
	}
}

func TestTheAPISpeaksInCanonicalDocumentsAndStableIdentifiers(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	kit := stack.tooling(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")

	draft := stack.startedByTool(t, kit, fmt.Sprintf(
		`{"categoryId":%q,"title":"One vocabulary"}`, announcement.ID,
	))
	picture := stack.uploadedByTool(t, kit, draft.ID, "document", httpTestPNG(t, 800, 400))
	sent := bodyWithPicture(picture.ID, "The workspace")
	body, err := json.Marshal(finished(draft, map[string]any{"document": sent}))
	if err != nil {
		t.Fatalf("encode working copy: %v", err)
	}
	saved := stack.sent(t, kit.value, jsonRequest(t,
		http.MethodPut, "/v1/publication/posts/"+draft.ID, string(body),
	))
	held := decodePost(t, saved)
	if held.DocumentVersion == 0 || len(held.Document.Content) != 2 {
		t.Fatalf("the saved document reads %+v", held.Document)
	}
	if held.Document.Content[1]["mediaId"] != picture.ID {
		t.Fatalf("the document lost the picture it placed: %+v", held.Document.Content[1])
	}

	kept := stack.sent(t, kit.value, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+draft.ID+"/revisions",
		fmt.Sprintf(`{"version":%d}`, held.Version),
	))
	var checkpoint postRevision
	if err := json.Unmarshal(kept.Body.Bytes(), &checkpoint); err != nil {
		t.Fatalf("decode edition: %v", err)
	}

	published := stack.sent(t, kit.value, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+draft.ID+"/publish",
		fmt.Sprintf(`{"version":%d}`, held.Version),
	))
	public := decodePost(t, published)

	editions := stack.sent(t, kit.value, httptest.NewRequest(
		http.MethodGet, "/v1/publication/posts/"+draft.ID+"/revisions", nil,
	))
	var listed struct {
		Revisions []postRevision `json:"revisions"`
	}
	if err := json.Unmarshal(editions.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode editions: %v", err)
	}
	if len(listed.Revisions) != 2 {
		t.Fatalf("the post kept %d editions", len(listed.Revisions))
	}
	names := map[string]bool{}
	for _, edition := range listed.Revisions {
		names[edition.ID] = true
	}
	if !names[checkpoint.ID] || !names[public.PublicRevision] {
		t.Fatalf("the editions %v do not name %q and %q",
			names, checkpoint.ID, public.PublicRevision)
	}

	again := decodePost(t, stack.sent(t, kit.value, httptest.NewRequest(
		http.MethodGet, "/v1/publication/posts/"+draft.ID, nil,
	)))
	if again.ID != draft.ID || again.PublicRevision != public.PublicRevision ||
		len(again.Media) != 1 || again.Media[0].ID != picture.ID {
		t.Fatalf("reading the post again gave %+v", again)
	}
}

func TestAnIdempotencyKeyMustBeLongEnoughToMeanSomething(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	kit := stack.tooling(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")
	body := fmt.Sprintf(`{"categoryId":%q,"title":"Short key"}`, announcement.ID)

	for _, key := range []string{"short", " padded-key ", strings.Repeat("x", 201)} {
		refused := stack.sent(t, kit.value, withKey(jsonRequest(t,
			http.MethodPost, "/v1/publication/posts", body,
		), key))
		if refused.Code != http.StatusBadRequest ||
			refusalOf(t, refused).Field != idempotencyKeyHeader {
			t.Errorf("the key %q answered %d: %s", key, refused.Code, refused.Body.String())
		}
	}
	if listed := stack.toolPosts(t, kit.value); len(listed) != 0 {
		t.Fatalf("a refused key wrote %d posts", len(listed))
	}
}

func TestAMalformedBodyRefusesWithAStableCode(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	kit := stack.tooling(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")
	post := stack.startedByTool(t, kit, fmt.Sprintf(
		`{"categoryId":%q,"title":"Notes"}`, announcement.ID,
	))
	at := "/v1/publication/posts/" + post.ID

	malformed := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/v1/publication/posts"},
		{http.MethodPut, at},
		{http.MethodPost, at + "/revisions"},
		{http.MethodPost, at + "/publish"},
		{http.MethodPost, at + "/withdraw"},
		{http.MethodPost, at + "/republish"},
		{http.MethodPost, at + "/delete"},
		{http.MethodPost, at + "/recover"},
		{http.MethodPost, at + "/schedule"},
		{http.MethodPut, at + "/schedule"},
		{http.MethodPost, at + "/import"},
	}
	for _, call := range malformed {
		response := stack.sent(t, kit.value, jsonRequest(t, call.method, call.path, `{"version":`))
		if response.Code != http.StatusBadRequest {
			t.Errorf("%s %s answered %d: %s", call.method, call.path, response.Code, response.Body.String())
			continue
		}
		if code := refusalOf(t, response).Code; code != string(CodeInvalid) {
			t.Errorf("%s %s said %q, want %q", call.method, call.path, code, CodeInvalid)
		}
	}

	picture := stack.sent(t, kit.value, httptest.NewRequest(
		http.MethodPost, at+"/media", strings.NewReader("not a form"),
	))
	if picture.Code != http.StatusBadRequest || refusalOf(t, picture).Code != string(CodeInvalid) {
		t.Errorf("a formless upload answered %d: %s", picture.Code, picture.Body.String())
	}
}
