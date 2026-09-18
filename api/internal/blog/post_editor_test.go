package blog_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/blog"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

type publicationRefusal struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Field   string `json:"field"`
	Version *int   `json:"version"`
}

func (s publicationStack) sent(
	t *testing.T,
	who contributor,
	request *http.Request,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(request, who.session))
}

func (s publicationStack) startedBy(t *testing.T, who contributor, body string) blogPost {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal([]byte(body), &fields); err != nil {
		t.Fatalf("decode start body: %v", err)
	}
	fields["grantId"] = who.grant.ID
	named, err := json.Marshal(fields)
	if err != nil {
		t.Fatalf("encode start body: %v", err)
	}
	return s.started(t, who.session, string(named))
}

func (s publicationStack) postsOf(t *testing.T, who contributor) []blogPost {
	t.Helper()
	response := s.sent(t, who, httptest.NewRequest(
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

func refusalOf(t *testing.T, response *httptest.ResponseRecorder) publicationRefusal {
	t.Helper()
	var refused publicationRefusal
	if err := json.Unmarshal(response.Body.Bytes(), &refused); err != nil {
		t.Fatalf("decode refusal %q: %v", response.Body.String(), err)
	}
	return refused
}

func TestSavingAWorkingCopyIsNeverAPublicTransition(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")

	draft := stack.startedBy(t, writer, fmt.Sprintf(
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
	saved := stack.sent(t, writer, jsonRequest(t,
		http.MethodPut, "/v1/publication/posts/"+draft.ID, string(body),
	))
	if saved.Code != http.StatusBadRequest && saved.Code != http.StatusOK {
		t.Fatalf("save status = %d: %s", saved.Code, saved.Body.String())
	}

	after := stack.sent(t, writer, httptest.NewRequest(
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

func TestAContributorReachesOnlyThePostsUnderTheirOwnGrant(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	mine := stack.contributor(t, "writer@example.com", "publication.writer")
	theirs := stack.contributor(t, "another@example.com", "publication.another")
	announcement := stack.categoryBySlug(t, "announcement")

	stack.startedBy(t, mine, fmt.Sprintf(
		`{"categoryId":%q,"title":"Mine alone"}`, announcement.ID,
	))
	hidden := stack.started(t, theirs.session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"Theirs alone"}`,
		theirs.grant.ID, announcement.ID,
	))

	listed := stack.postsOf(t, mine)
	if len(listed) != 1 || listed[0].Title != "Mine alone" {
		t.Fatalf("a contributor listed %d posts: %+v", len(listed), listed)
	}
	for _, address := range []string{
		"/v1/publication/posts/" + hidden.ID,
		"/v1/publication/posts/" + hidden.ID + "/revisions",
		"/v1/publication/posts/" + hidden.ID + "/history",
	} {
		response := stack.sent(t, mine, httptest.NewRequest(http.MethodGet, address, nil))
		if response.Code != http.StatusForbidden {
			t.Errorf("%s answered %d to another contributor: %s",
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
	writer := stack.contributor(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")

	draft := stack.startedBy(t, writer, fmt.Sprintf(
		`{"categoryId":%q,"title":"Two writers"}`, announcement.ID,
	))
	stack.saved(t, writer.session, draft.ID, finished(draft, nil))

	body, err := json.Marshal(finished(draft, map[string]any{"title": "From another tab"}))
	if err != nil {
		t.Fatalf("encode working copy: %v", err)
	}
	stale := stack.sent(t, writer, jsonRequest(t,
		http.MethodPut, "/v1/publication/posts/"+draft.ID, string(body),
	))
	if stale.Code != http.StatusConflict {
		t.Fatalf("a stale save answered %d: %s", stale.Code, stale.Body.String())
	}
	refused := refusalOf(t, stale)
	if refused.Code != string(blog.PublicationErrorCodeStaleVersion) || refused.Version == nil {
		t.Fatalf("a stale save said %+v", refused)
	}
	if *refused.Version != stack.working(t, writer.session, draft.ID).Version {
		t.Fatalf("the conflict named version %d", *refused.Version)
	}
}

func TestTheAPISpeaksInCanonicalDocumentsAndStableIdentifiers(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")

	draft := stack.startedBy(t, writer, fmt.Sprintf(
		`{"categoryId":%q,"title":"One vocabulary"}`, announcement.ID,
	))
	picture := stack.uploaded(t, writer.session, draft.ID, "document", apitest.PNG(t, 800, 400))
	sent := bodyWithPicture(picture.ID, "The workspace")
	body, err := json.Marshal(finished(draft, map[string]any{"document": sent}))
	if err != nil {
		t.Fatalf("encode working copy: %v", err)
	}
	saved := stack.sent(t, writer, jsonRequest(t,
		http.MethodPut, "/v1/publication/posts/"+draft.ID, string(body),
	))
	held := decodePost(t, saved)
	if held.DocumentVersion == 0 || len(held.Document.Content) != 2 {
		t.Fatalf("the saved document reads %+v", held.Document)
	}
	if held.Document.Content[1]["mediaId"] != picture.ID {
		t.Fatalf("the document lost the picture it placed: %+v", held.Document.Content[1])
	}

	kept := stack.sent(t, writer, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+draft.ID+"/revisions",
		fmt.Sprintf(`{"version":%d}`, held.Version),
	))
	var checkpoint postRevision
	if err := json.Unmarshal(kept.Body.Bytes(), &checkpoint); err != nil {
		t.Fatalf("decode edition: %v", err)
	}

	published := stack.sent(t, writer, jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+draft.ID+"/publish",
		fmt.Sprintf(`{"version":%d}`, held.Version),
	))
	public := decodePost(t, published)

	editions := stack.sent(t, writer, httptest.NewRequest(
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

	again := decodePost(t, stack.sent(t, writer, httptest.NewRequest(
		http.MethodGet, "/v1/publication/posts/"+draft.ID, nil,
	)))
	if again.ID != draft.ID || again.PublicRevision != public.PublicRevision ||
		len(again.Media) != 1 || again.Media[0].ID != picture.ID {
		t.Fatalf("reading the post again gave %+v", again)
	}
}

func TestAMalformedBodyRefusesWithAStableCode(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "publication.writer")
	announcement := stack.categoryBySlug(t, "announcement")
	post := stack.startedBy(t, writer, fmt.Sprintf(
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
		response := stack.sent(t, writer, jsonRequest(t, call.method, call.path, `{"version":`))
		if response.Code != http.StatusBadRequest {
			t.Errorf("%s %s answered %d: %s", call.method, call.path, response.Code, response.Body.String())
			continue
		}
		if code := refusalOf(t, response).Code; code != string(blog.PublicationErrorCodeInvalid) {
			t.Errorf("%s %s said %q, want %q", call.method, call.path, code, blog.PublicationErrorCodeInvalid)
		}
	}

	picture := stack.sent(t, writer, httptest.NewRequest(
		http.MethodPost, at+"/media", strings.NewReader("not a form"),
	))
	if picture.Code != http.StatusBadRequest || refusalOf(t, picture).Code != string(blog.PublicationErrorCodeInvalid) {
		t.Errorf("a formless upload answered %d: %s", picture.Code, picture.Body.String())
	}
}
