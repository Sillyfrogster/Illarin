package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

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

func (s distinctionStack) withdraw(
	t *testing.T,
	session *http.Cookie,
	id, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+id+"/withdraw", body,
	), session))
}

func (s distinctionStack) withdrawn(
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

func (s distinctionStack) republish(
	t *testing.T,
	session *http.Cookie,
	id, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+id+"/republish", body,
	), session))
}

func (s distinctionStack) republished(
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

// gone reads the tombstone one address answers with, refusing any other status.
func (s distinctionStack) gone(t *testing.T, slug string) (tombstone, string) {
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

func (s distinctionStack) events(t *testing.T, postID string) []string {
	t.Helper()
	rows, err := s.pool.Query(context.Background(), `
		select type from publication_events where post_id = $1 order by occurred_at, id
	`, postID)
	if err != nil {
		t.Fatalf("read publication events: %v", err)
	}
	defer rows.Close()
	recorded := make([]string, 0, 4)
	for rows.Next() {
		var one string
		if err := rows.Scan(&one); err != nil {
			t.Fatalf("read a publication event: %v", err)
		}
		recorded = append(recorded, one)
	}
	return recorded
}

func TestWithdrawingAPostLeavesATombstoneAndKeepsEverythingElse(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	live := stack.livePost(t, session, "A post we took down")

	gone := stack.withdrawn(t, session, live.ID, live.Version,
		"Legal asked for it while they check a claim.",
		"We are checking a claim in this article and will put it back.")

	if gone.Status != "withdrawn" {
		t.Fatalf("withdrawing left the post %q", gone.Status)
	}
	if gone.Slug != live.Slug {
		t.Errorf("withdrawing moved the post to %q", gone.Slug)
	}
	if gone.PublishedAt == nil || !gone.PublishedAt.Equal(*live.PublishedAt) {
		t.Errorf("original publication date = %v, want %v", gone.PublishedAt, live.PublishedAt)
	}
	if gone.UpdatedPublicAt != nil {
		t.Errorf("withdrawal set the public updated date to %v", gone.UpdatedPublicAt)
	}
	if gone.Byline == nil || gone.Byline.Handle != "illarin.editor" {
		t.Errorf("withdrawal changed the byline to %+v", gone.Byline)
	}
	if gone.PublicRevision != live.PublicRevision {
		t.Errorf("withdrawal changed the kept public edition to %q", gone.PublicRevision)
	}
	if len(gone.Document.Content) == 0 {
		t.Error("withdrawal emptied the working copy")
	}
	if len(stack.revisions(t, session, live.ID)) == 0 {
		t.Error("withdrawal removed the editions the post had kept")
	}

	found, body := stack.gone(t, live.Slug)
	if found.Slug != live.Slug {
		t.Errorf("tombstone address = %q, want %q", found.Slug, live.Slug)
	}
	if found.Explanation != "We are checking a claim in this article and will put it back." {
		t.Errorf("tombstone explanation = %q", found.Explanation)
	}
	for _, leaked := range []string{
		live.Title, "Illarin now keeps its own writing.", "illarin.editor",
		"Legal asked for it while they check a claim.", live.PublicRevision,
	} {
		if strings.Contains(body, leaked) {
			t.Errorf("the tombstone leaks %q: %s", leaked, body)
		}
	}
}

func TestAWithdrawnPostLeavesEveryPlaceAReaderCouldFindIt(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

	staying := stack.livePost(t, session, "The post that stays")
	going := stack.livePost(t, session, "The post that goes")

	stack.withdrawn(t, session, going.ID, going.Version, "It named the wrong version.", "")

	whole := stack.archive(t, "")
	if whole.Total != 1 || len(whole.Posts) != 1 || whole.Posts[0].ID != staying.ID {
		t.Fatalf("the archive still lists the withdrawn post: %+v", whole)
	}
	narrowed := stack.archive(t, "?category="+staying.Category.Slug)
	if narrowed.Total != 1 {
		t.Errorf("the category archive holds %d posts, want 1", narrowed.Total)
	}
	for _, further := range stack.reader(t, staying.Slug).Related {
		if further.ID == going.ID {
			t.Error("further reading offers a withdrawn post")
		}
	}
}

func TestAFormerAddressOfAWithdrawnPostReachesTheSameTombstone(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	live := stack.livePost(t, session, "A post that moved and went")
	moved := stack.addressCorrected(t, session, live.ID, "a-post-that-moved")

	stack.withdrawn(t, session, moved.ID, moved.Version, "It quoted the wrong person.", "")

	for _, address := range []string{"a-post-that-moved", live.Slug} {
		found, body := stack.gone(t, address)
		if found.Slug != "a-post-that-moved" {
			t.Errorf("%s answered with address %q", address, found.Slug)
		}
		if strings.Contains(body, live.Title) {
			t.Errorf("%s leaks the article: %s", address, body)
		}
	}
}

func TestRepublishingReturnsTheSamePostToTheSameAddress(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	live := stack.livePost(t, session, "A post that came back")

	gone := stack.withdrawn(t, session, live.ID, live.Version, "A picture needed clearing.", "")
	back := stack.republished(t, session, gone.ID, gone.Version, live.PublicRevision)

	if back.Status != "published" || back.ID != live.ID {
		t.Fatalf("republishing left the post %q with id %q", back.Status, back.ID)
	}
	if back.Slug != live.Slug {
		t.Errorf("republished address = %q, want %q", back.Slug, live.Slug)
	}
	if !back.PublishedAt.Equal(*live.PublishedAt) {
		t.Errorf("republished date = %v, want the original %v", back.PublishedAt, live.PublishedAt)
	}
	if back.UpdatedPublicAt != nil {
		t.Errorf("returning the same edition set an updated date of %v", back.UpdatedPublicAt)
	}
	if stack.reader(t, live.Slug).Title != live.Title {
		t.Error("the republished post does not answer on its address")
	}
	if whole := stack.archive(t, ""); whole.Total != 1 {
		t.Errorf("the archive holds %d posts after republication, want 1", whole.Total)
	}
	want := []string{
		"publication.post.published.v1",
		"publication.post.withdrawn.v1",
		"publication.post.published.v1",
	}
	if got := stack.events(t, live.ID); !slices.Equal(got, want) {
		t.Errorf("events = %v, want %v", got, want)
	}
}

func TestRepublishingACorrectedEditionSetsThePublicUpdatedDate(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	live := stack.livePost(t, session, "A post we fixed while it was down")

	gone := stack.withdrawn(t, session, live.ID, live.Version, "One paragraph was wrong.", "")
	corrected := stack.saved(t, session, gone.ID, finished(gone, map[string]any{
		"version":  gone.Version,
		"summary":  "What Illarin corrected this week.",
		"document": json.RawMessage(paragraph("Illarin corrected the paragraph.")),
	}))
	edition := stack.checkpointed(t, session, corrected.ID, corrected.Version)
	held := stack.working(t, session, corrected.ID)

	back := stack.republished(t, session, held.ID, held.Version, edition.ID)

	if back.UpdatedPublicAt == nil {
		t.Fatal("republishing a corrected edition set no public updated date")
	}
	if !back.PublishedAt.Equal(*live.PublishedAt) {
		t.Errorf("the original date moved to %v", back.PublishedAt)
	}
	if read := stack.reader(t, live.Slug); read.Summary != "What Illarin corrected this week." {
		t.Errorf("readers get %q, want the corrected edition", read.Summary)
	}
}

func TestWithdrawalKeepsThePrivateReasonApartFromWhatReadersSee(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	live := stack.livePost(t, session, "A post with a private reason")

	silent := stack.withdraw(t, session, live.ID,
		fmt.Sprintf(`{"version":%d,"reason":""}`, live.Version))
	if silent.Code != http.StatusBadRequest {
		t.Fatalf("a withdrawal without a reason returned %d: %s",
			silent.Code, silent.Body.String())
	}
	same := stack.withdraw(t, session, live.ID, fmt.Sprintf(
		`{"version":%d,"reason":"A claim we are checking.","explanation":"A claim we are checking."}`,
		live.Version,
	))
	if same.Code != http.StatusBadRequest {
		t.Fatalf("reusing the private reason as the public one returned %d: %s",
			same.Code, same.Body.String())
	}

	quiet := stack.withdrawn(t, session, live.ID, live.Version, "A claim we are checking.", "")
	if quiet.Withdrawal == nil || quiet.Withdrawal.Reason != "A claim we are checking." {
		t.Fatalf("the editor cannot see why the post came down: %+v", quiet.Withdrawal)
	}
	if quiet.Withdrawal.Explanation != "" || quiet.Withdrawal.By != "illarin.editor" {
		t.Errorf("withdrawal = %+v", quiet.Withdrawal)
	}
	found, body := stack.gone(t, live.Slug)
	if found.Explanation != "" {
		t.Errorf("a withdrawal with no public explanation shows %q", found.Explanation)
	}
	if strings.Contains(body, "A claim we are checking.") {
		t.Errorf("the tombstone shows the private reason: %s", body)
	}
}

func TestOnlyThePostsOwnPeopleWithdrawItAndOnlyAtTheVersionTheyHold(t *testing.T) {
	stack := newDistinctionStack(t)
	writer := stack.contributor(t, "dev@example.com", "illarin.dev")
	stranger := stack.contributor(t, "other@example.com", "other.dev")
	admin := stack.admin(t, "boss@example.com", "illarin.boss")

	draft := stack.started(t, writer.session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"A contributor's post"}`,
		writer.grant.ID, stack.categoryBySlug(t, "announcement").ID,
	))
	stack.saved(t, writer.session, draft.ID, finished(draft, nil))
	live := stack.published(t, writer.session, draft.ID)

	refused := stack.withdraw(t, stranger.session, live.ID, fmt.Sprintf(
		`{"version":%d,"reason":"I do not like it."}`, live.Version,
	))
	if refused.Code != http.StatusForbidden && refused.Code != http.StatusNotFound {
		t.Fatalf("a stranger withdrew the post: %d %s", refused.Code, refused.Body.String())
	}
	stale := stack.withdraw(t, writer.session, live.ID, fmt.Sprintf(
		`{"version":%d,"reason":"It was wrong."}`, live.Version-1,
	))
	if stale.Code != http.StatusConflict {
		t.Fatalf("a stale withdrawal returned %d: %s", stale.Code, stale.Body.String())
	}
	if stack.read(t, live.Slug).Code != http.StatusOK {
		t.Error("a refused withdrawal took the post down anyway")
	}

	byAdmin := stack.withdrawn(t, admin, live.ID, live.Version, "It named an unreleased build.", "")
	if byAdmin.Status != "withdrawn" {
		t.Errorf("an admin could not withdraw the post: %q", byAdmin.Status)
	}
}

func TestAWithdrawnPostIsPutBackByRepublishingAndNotByPublishing(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	live := stack.livePost(t, session, "A post with one way back")
	gone := stack.withdrawn(t, session, live.ID, live.Version, "It was early.", "")

	if again := stack.publishAt(t, session, gone.ID, gone.Version); again.Code != http.StatusBadRequest {
		t.Fatalf("publishing a withdrawn post returned %d: %s", again.Code, again.Body.String())
	}
	later := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	scheduled := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+gone.ID+"/schedule",
		fmt.Sprintf(`{"version":%d,"at":%q}`, gone.Version, later),
	), session))
	if scheduled.Code != http.StatusBadRequest {
		t.Fatalf("scheduling a withdrawn post returned %d: %s",
			scheduled.Code, scheduled.Body.String())
	}
	if stack.read(t, live.Slug).Code != http.StatusGone {
		t.Error("a refused publication put the post back")
	}
	if again := stack.republish(t, session, live.ID, fmt.Sprintf(
		`{"version":%d,"revisionId":%q}`, gone.Version, live.PublicRevision,
	)); again.Code != http.StatusOK {
		t.Fatalf("republish status = %d: %s", again.Code, again.Body.String())
	}
	if second := stack.republish(t, session, live.ID, fmt.Sprintf(
		`{"version":%d,"revisionId":%q}`, gone.Version, live.PublicRevision,
	)); second.Code != http.StatusBadRequest {
		t.Fatalf("republishing a public post returned %d: %s", second.Code, second.Body.String())
	}
}

func TestWithdrawalStopsAnEditionThatWasWaitingToGoLive(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	live := stack.livePost(t, session, "A post with an update on the way")
	later := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	waiting := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+live.ID+"/schedule",
		fmt.Sprintf(`{"version":%d,"at":%q}`, live.Version, later),
	), session))
	if waiting.Code != http.StatusCreated {
		t.Fatalf("schedule status = %d: %s", waiting.Code, waiting.Body.String())
	}
	held := decodePost(t, waiting)

	gone := stack.withdrawn(t, session, held.ID, held.Version, "Hold everything.", "")

	if gone.Schedule == nil || gone.Schedule.State != "cancelled" {
		t.Fatalf("the waiting edition is %+v, want a cancelled schedule", gone.Schedule)
	}
}

func TestARetriedWithdrawalThroughTheApiChangesNothingTwice(t *testing.T) {
	stack := newDistinctionStack(t)
	kit := stack.tooling(t, "robot@example.com", "release.robot")
	draft := stack.startedByTool(t, kit, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"A tooling post"}`,
		kit.who.grant.ID, stack.categoryBySlug(t, "announcement").ID,
	))
	stack.saved(t, kit.who.session, draft.ID, finished(draft, nil))
	live := stack.published(t, kit.who.session, draft.ID)

	body := fmt.Sprintf(`{"version":%d,"reason":"The build was pulled."}`, live.Version)
	first := stack.sent(t, kit.value, withKey(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+live.ID+"/withdraw", body,
	), "withdraw-once"))
	if first.Code != http.StatusOK {
		t.Fatalf("withdraw status = %d: %s", first.Code, first.Body.String())
	}
	again := stack.sent(t, kit.value, withKey(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+live.ID+"/withdraw", body,
	), "withdraw-once"))
	if again.Code != http.StatusOK || again.Body.String() != first.Body.String() {
		t.Fatalf("a retried withdrawal returned %d: %s", again.Code, again.Body.String())
	}
	var withdrawals int
	err := stack.pool.QueryRow(context.Background(), `
		select count(*) from post_withdrawals where post_id = $1
	`, live.ID).Scan(&withdrawals)
	if err != nil {
		t.Fatalf("count withdrawals: %v", err)
	}
	if withdrawals != 1 {
		t.Errorf("a retried withdrawal wrote %d records", withdrawals)
	}
	if got := stack.events(t, live.ID); len(got) != 2 {
		t.Errorf("events = %v, want the publication and one withdrawal", got)
	}
}
