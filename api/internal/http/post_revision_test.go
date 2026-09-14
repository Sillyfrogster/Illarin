package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type postRevision struct {
	ID          string              `json:"id"`
	Number      int                 `json:"number"`
	Title       string              `json:"title"`
	Summary     string              `json:"summary"`
	Slug        string              `json:"slug"`
	Category    publicationCategory `json:"category"`
	CapturedFor string              `json:"capturedFor"`
	CapturedBy  string              `json:"capturedBy"`
	CapturedAt  time.Time           `json:"capturedAt"`
	Public      bool                `json:"public"`
}

type postAction struct {
	ID         string    `json:"id"`
	Actor      string    `json:"actor"`
	Credential string    `json:"credential"`
	Action     string    `json:"action"`
	Revision   *int      `json:"revision"`
	Before     string    `json:"before"`
	After      string    `json:"after"`
	At         time.Time `json:"at"`
}

func (s publicationStack) checkpoint(
	t *testing.T,
	session *http.Cookie,
	id string,
	version int,
) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+id+"/revisions",
		fmt.Sprintf(`{"version":%d}`, version),
	), session))
}

func (s publicationStack) checkpointed(
	t *testing.T,
	session *http.Cookie,
	id string,
	version int,
) postRevision {
	t.Helper()
	response := s.checkpoint(t, session, id, version)
	if response.Code != http.StatusCreated {
		t.Fatalf("checkpoint status = %d: %s", response.Code, response.Body.String())
	}
	var kept postRevision
	if err := json.Unmarshal(response.Body.Bytes(), &kept); err != nil {
		t.Fatalf("decode kept edition: %v", err)
	}
	return kept
}

func (s publicationStack) restore(
	t *testing.T,
	session *http.Cookie,
	id, revisionID string,
	version int,
) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, authorized(jsonRequest(t,
		http.MethodPost,
		"/v1/publication/posts/"+id+"/revisions/"+revisionID+"/restore",
		fmt.Sprintf(`{"version":%d}`, version),
	), session))
}

func (s publicationStack) restored(
	t *testing.T,
	session *http.Cookie,
	id, revisionID string,
	version int,
) blogPost {
	t.Helper()
	response := s.restore(t, session, id, revisionID, version)
	if response.Code != http.StatusOK {
		t.Fatalf("restore status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s publicationStack) revisions(
	t *testing.T,
	session *http.Cookie,
	id string,
) []postRevision {
	t.Helper()
	response := send(t, s.router, authorized(httptest.NewRequest(
		http.MethodGet, "/v1/publication/posts/"+id+"/revisions", nil,
	), session))
	if response.Code != http.StatusOK {
		t.Fatalf("revisions status = %d: %s", response.Code, response.Body.String())
	}
	var listed struct {
		Revisions []postRevision `json:"revisions"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode kept editions: %v", err)
	}
	return listed.Revisions
}

func (s publicationStack) history(
	t *testing.T,
	session *http.Cookie,
	id string,
) ([]postAction, string) {
	t.Helper()
	response := send(t, s.router, authorized(httptest.NewRequest(
		http.MethodGet, "/v1/publication/posts/"+id+"/history", nil,
	), session))
	if response.Code != http.StatusOK {
		t.Fatalf("history status = %d: %s", response.Code, response.Body.String())
	}
	var listed struct {
		Actions []postAction `json:"actions"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode post history: %v", err)
	}
	return listed.Actions, response.Body.String()
}

func taken(done []postAction) []string {
	names := make([]string, 0, len(done))
	for index := len(done) - 1; index >= 0; index-- {
		names = append(names, done[index].Action)
	}
	return names
}

func TestACheckpointKeepsTheEditionAndPublishesNothing(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "keeper@example.com", "illarin.keeper")
	draft := stack.illarinDraft(t, session, "The week in Illarin")
	written := stack.saved(t, session, draft.ID, finished(draft, nil))

	kept := stack.checkpointed(t, session, draft.ID, written.Version)
	if kept.Number != 1 || kept.CapturedFor != "checkpoint" || kept.Public {
		t.Fatalf("kept edition = %+v, want the first checkpoint and not public", kept)
	}
	if kept.Title != written.Title || kept.CapturedBy != "illarin.keeper" {
		t.Errorf("kept edition names %q by %q", kept.Title, kept.CapturedBy)
	}

	after := stack.working(t, session, draft.ID)
	if after.Version != written.Version {
		t.Errorf("a checkpoint moved the working copy to version %d", after.Version)
	}
	if after.Status != "draft" {
		t.Errorf("a checkpoint left the post %q", after.Status)
	}
	if stack.read(t, draft.Slug).Code != http.StatusNotFound {
		t.Error("a checkpointed draft answers on its address")
	}
}

func TestEditingAPublishedPostLeavesReadersOnTheEditionTheyHave(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.quiet")
	draft := stack.illarinDraft(t, session, "Illarin ships weekly")
	stack.saved(t, session, draft.ID, finished(draft, nil))
	live := stack.published(t, session, draft.ID)

	edited := stack.saved(t, session, draft.ID, finished(live, map[string]any{
		"version":  live.Version,
		"title":    "Illarin ships weekly, and here is the rest of it",
		"document": json.RawMessage(paragraph("An unfinished thought.")),
	}))
	reading := stack.reader(t, draft.Slug)
	if reading.Title != "Illarin ships weekly" {
		t.Fatalf("readers were given %q while the working copy was edited", reading.Title)
	}
	if strings.Contains(string(mustEncode(t, reading.Document)), "unfinished") {
		t.Error("an unpublished edit reached a reader")
	}

	updated := stack.publishedAt(t, session, draft.ID, edited.Version)
	if updated.PublishedAt == nil || !updated.PublishedAt.Equal(*live.PublishedAt) {
		t.Errorf("an update moved the publication date to %v", updated.PublishedAt)
	}
	if updated.UpdatedPublicAt == nil {
		t.Fatal("an update recorded no updated date")
	}
	if now := stack.reader(t, draft.Slug); now.Title != edited.Title {
		t.Errorf("after publishing the update readers see %q", now.Title)
	}
}

func TestRestoringAnEditionCopiesItForwardAndLeavesHistoryAlone(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "restore@example.com", "illarin.restore")
	draft := stack.illarinDraft(t, session, "The first shape of it")
	first := stack.saved(t, session, draft.ID, finished(draft, nil))
	live := stack.published(t, session, draft.ID)

	second := stack.saved(t, session, draft.ID, finished(live, map[string]any{
		"version":  live.Version,
		"title":    "A shape I liked less",
		"summary":  "A summary I liked less.",
		"document": json.RawMessage(paragraph("A sentence I liked less.")),
	}))
	before := stack.revisions(t, session, draft.ID)
	if len(before) != 1 || before[0].CapturedFor != "publication" || !before[0].Public {
		t.Fatalf("kept editions = %+v, want one published edition", before)
	}

	if live.PublicRevision != before[0].ID {
		t.Errorf("the post names %q as its public edition", live.PublicRevision)
	}

	back := stack.restored(t, session, draft.ID, before[0].ID, second.Version)
	if back.Title != first.Title || back.Summary != first.Summary {
		t.Errorf("the restored working copy reads %q / %q", back.Title, back.Summary)
	}
	if back.Version <= second.Version {
		t.Errorf("restoring left the working copy at version %d", back.Version)
	}
	if back.Status != "published" || back.Slug != live.Slug {
		t.Errorf("restoring changed the post to %q at %q", back.Status, back.Slug)
	}

	after := stack.revisions(t, session, draft.ID)
	if len(after) != 1 || after[0].ID != before[0].ID || !after[0].Public {
		t.Fatalf("restoring changed the kept editions to %+v", after)
	}
	if reading := stack.reader(t, draft.Slug); reading.Title != first.Title {
		t.Errorf("readers were given %q during a restore", reading.Title)
	}
}

func TestARestoredEditionBecomesTheNextPublishedOne(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "forward@example.com", "illarin.forward")
	draft := stack.illarinDraft(t, session, "Take two")
	stack.saved(t, session, draft.ID, finished(draft, nil))
	first := stack.published(t, session, draft.ID)

	second := stack.saved(t, session, draft.ID, finished(first, map[string]any{
		"version": first.Version,
		"title":   "Take three",
	}))
	live := stack.publishedAt(t, session, draft.ID, second.Version)
	kept := stack.revisions(t, session, draft.ID)
	if len(kept) != 2 {
		t.Fatalf("kept editions = %d, want two publications", len(kept))
	}

	back := stack.restored(t, session, draft.ID, kept[1].ID, live.Version)
	republished := stack.publishedAt(t, session, draft.ID, back.Version)
	if republished.Title != "Take two" {
		t.Errorf("the republished post reads %q", republished.Title)
	}
	if now := stack.revisions(t, session, draft.ID); len(now) != 3 || !now[0].Public {
		t.Fatalf("kept editions = %+v, want a third one on public view", now)
	}
	if reading := stack.reader(t, draft.Slug); reading.Title != "Take two" {
		t.Errorf("readers were given %q after republishing take two", reading.Title)
	}
}

func TestStaleCheckpointRestoreAndPublishChangeNothing(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "stale@example.com", "illarin.stale")
	draft := stack.illarinDraft(t, session, "Two people, one post")
	stack.saved(t, session, draft.ID, finished(draft, nil))
	live := stack.published(t, session, draft.ID)
	kept := stack.revisions(t, session, draft.ID)

	moved := stack.saved(t, session, draft.ID, finished(live, map[string]any{
		"version": live.Version,
		"title":   "Someone else got here first",
	}))
	stale := live.Version

	if code := stack.checkpoint(t, session, draft.ID, stale).Code; code != http.StatusConflict {
		t.Errorf("a stale checkpoint returned %d, want 409", code)
	}
	if code := stack.restore(t, session, draft.ID, kept[0].ID, stale).Code; code != http.StatusConflict {
		t.Errorf("a stale restore returned %d, want 409", code)
	}
	if code := stack.publishAt(t, session, draft.ID, stale).Code; code != http.StatusConflict {
		t.Errorf("a stale publish returned %d, want 409", code)
	}

	after := stack.working(t, session, draft.ID)
	if after.Version != moved.Version || after.Title != moved.Title {
		t.Errorf("a stale request changed the working copy to %q at %d", after.Title, after.Version)
	}
	if now := stack.revisions(t, session, draft.ID); len(now) != len(kept) {
		t.Errorf("a stale request kept %d editions, want %d", len(now), len(kept))
	}
	if reading := stack.reader(t, draft.Slug); reading.Title != live.Title {
		t.Errorf("a stale request gave readers %q", reading.Title)
	}
}

func TestTheHistoryNamesWhatHappenedAndNeverTheWriting(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "history@example.com", "illarin.history")
	draft := stack.illarinDraft(t, session, "A post with a past")
	written := stack.saved(t, session, draft.ID, finished(draft, nil))
	stack.checkpointed(t, session, draft.ID, written.Version)
	live := stack.published(t, session, draft.ID)
	kept := stack.revisions(t, session, draft.ID)
	stack.restored(t, session, draft.ID, kept[0].ID, live.Version)

	done, raw := stack.history(t, session, draft.ID)
	want := []string{
		"post.created", "post.checkpointed", "post.published", "post.revision.restored",
	}
	if got := taken(done); len(got) != len(want) {
		t.Fatalf("history = %v, want %v", got, want)
	} else {
		for index, action := range want {
			if got[index] != action {
				t.Fatalf("history = %v, want %v", got, want)
			}
		}
	}
	for _, one := range done {
		if one.Actor != "illarin.history" || one.Credential != "session" {
			t.Errorf("action %q was recorded by %q over %q", one.Action, one.Actor, one.Credential)
		}
	}
	if strings.Contains(raw, "A post with a past") || strings.Contains(raw, "Illarin now keeps") {
		t.Error("the history carries the words of the post")
	}
	if published := done[1]; published.Revision == nil || *published.Revision != 2 {
		t.Errorf("publication named edition %v, want the second", published.Revision)
	}
}

func TestOneContributorNeverReachesAnothersEditions(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	mine := stack.contributor(t, "mine@example.com", "mine.dev")
	sillytavern := stack.configureApp(t, "sillytavern", "SillyTavern", "https://sillytavern.example")
	announcement := stack.categoryBySlug(t, "announcement")
	theirs := stack.member(t, "theirs@example.com", "theirs.dev")
	stack.approved(t, "theirs.dev", sillytavern.ID, []string{announcement.ID}, announcement.ID)

	draft := stack.started(t, mine.session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"Mine alone"}`, mine.grant.ID, announcement.ID,
	))
	written := stack.saved(t, mine.session, draft.ID, finished(draft, nil))
	kept := stack.checkpointed(t, mine.session, draft.ID, written.Version)

	listing := send(t, stack.router, authorized(httptest.NewRequest(
		http.MethodGet, "/v1/publication/posts/"+draft.ID+"/revisions", nil,
	), theirs))
	if listing.Code != http.StatusForbidden {
		t.Errorf("another contributor listed the editions: %d", listing.Code)
	}
	reading := send(t, stack.router, authorized(httptest.NewRequest(
		http.MethodGet, "/v1/publication/posts/"+draft.ID+"/history", nil,
	), theirs))
	if reading.Code != http.StatusForbidden {
		t.Errorf("another contributor read the history: %d", reading.Code)
	}
	if code := stack.checkpoint(t, theirs, draft.ID, written.Version).Code; code != http.StatusForbidden {
		t.Errorf("another contributor checkpointed the post: %d", code)
	}
	code := stack.restore(t, theirs, draft.ID, kept.ID, written.Version).Code
	if code != http.StatusForbidden {
		t.Errorf("another contributor restored an edition: %d", code)
	}

	admin := stack.admin(t, "over@example.com", "illarin.over")
	if len(stack.revisions(t, admin, draft.ID)) != 1 {
		t.Error("an admin could not read the editions of a contributor's post")
	}
}

func TestAnEditionOfAnotherPostIsNotRestorable(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "elsewhere@example.com", "illarin.elsewhere")
	first := stack.illarinDraft(t, session, "The post with the edition")
	firstWritten := stack.saved(t, session, first.ID, finished(first, nil))
	kept := stack.checkpointed(t, session, first.ID, firstWritten.Version)

	second := stack.illarinDraft(t, session, "The post reaching for it")
	written := stack.saved(t, session, second.ID, finished(second, nil))

	response := stack.restore(t, session, second.ID, kept.ID, written.Version)
	if response.Code != http.StatusNotFound {
		t.Fatalf("restoring another post's edition returned %d", response.Code)
	}
	if after := stack.working(t, session, second.ID); after.Title != written.Title {
		t.Errorf("the working copy became %q", after.Title)
	}
}

func TestACheckpointedPictureStaysBehindItsSignature(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "signed@example.com", "illarin.signed")
	draft := stack.illarinDraft(t, session, "A draft with a picture in it")
	picture := stack.uploaded(t, session, draft.ID, "document", httpTestPNG(t, 800, 400))
	written := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"document": bodyWithPicture(picture.ID, "A picture nobody has seen"),
	}))
	stack.checkpointed(t, session, draft.ID, written.Version)

	plain := strings.SplitN(picture.URL, "?", 2)[0]
	if answered := stack.fetch(t, plain); answered.Code != http.StatusNotFound {
		t.Errorf("a checkpointed picture answers %d without a signature", answered.Code)
	}
}

func TestRestoringBringsBackThePicturesTheEditionUsed(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "pictured@example.com", "illarin.pictured")
	draft := stack.illarinDraft(t, session, "The post that had a picture")
	picture := stack.uploaded(t, session, draft.ID, "document", httpTestPNG(t, 900, 500))
	written := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"document": bodyWithPicture(picture.ID, "The workspace as it stood"),
	}))
	kept := stack.checkpointed(t, session, draft.ID, written.Version)

	bare := stack.saved(t, session, draft.ID, finished(written, map[string]any{
		"version": written.Version,
	}))
	if len(bare.Media) != 0 {
		t.Fatalf("the working copy still refers to %+v", bare.Media)
	}

	back := stack.restored(t, session, draft.ID, kept.ID, bare.Version)
	if len(back.Media) != 1 || back.Media[0].ID != picture.ID {
		t.Fatalf("the restored working copy refers to %+v", back.Media)
	}
}

func mustEncode(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	return encoded
}
