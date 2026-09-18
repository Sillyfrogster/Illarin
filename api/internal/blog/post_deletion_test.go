package blog_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

type postDeletion struct {
	At    time.Time `json:"at"`
	Until time.Time `json:"until"`
	By    string    `json:"by"`
}

func (s publicationStack) remove(
	t *testing.T,
	session *http.Cookie,
	id string,
	version int,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+id+"/delete",
		fmt.Sprintf(`{"version":%d}`, version),
	), session))
}

func (s publicationStack) removed(
	t *testing.T,
	session *http.Cookie,
	id string,
	version int,
) blogPost {
	t.Helper()
	response := s.remove(t, session, id, version)
	if response.Code != http.StatusOK {
		t.Fatalf("delete status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s publicationStack) recover(
	t *testing.T,
	session *http.Cookie,
	id string,
	version int,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+id+"/recover",
		fmt.Sprintf(`{"version":%d}`, version),
	), session))
}

func (s publicationStack) recovered(
	t *testing.T,
	session *http.Cookie,
	id string,
	version int,
) blogPost {
	t.Helper()
	response := s.recover(t, session, id, version)
	if response.Code != http.StatusOK {
		t.Fatalf("recover status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s publicationStack) listing(t *testing.T, session *http.Cookie, deleted bool) []blogPost {
	t.Helper()
	address := "/v1/publication/posts"
	if deleted {
		address += "?deleted=true"
	}
	response := apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, address, nil), session,
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

func (s publicationStack) clearOut(t *testing.T, at time.Time) int {
	t.Helper()
	removed, err := s.handlers.Publications.RemoveExpiredPosts(t.Context(), at)
	if err != nil {
		t.Fatalf("run the recovery worker: %v", err)
	}
	return removed
}

func blobOf(t *testing.T, stack publicationStack, mediaID string) string {
	t.Helper()
	var blob string
	err := stack.pool.QueryRow(t.Context(),
		`select blob_id::text from post_media where id = $1`, mediaID).Scan(&blob)
	if err != nil {
		t.Fatalf("read the bytes behind a picture: %v", err)
	}
	return blob
}

func markedBlob(t *testing.T, stack publicationStack, blobID string) bool {
	t.Helper()
	var sighted bool
	err := stack.pool.QueryRow(t.Context(),
		`select exists (select 1 from blob_sweep_marks where blob_id = $1)`,
		blobID).Scan(&sighted)
	if err != nil {
		t.Fatalf("read whether bytes are marked for sweeping: %v", err)
	}
	return sighted
}

func postTitles(listed []blogPost) []string {
	names := make([]string, 0, len(listed))
	for _, one := range listed {
		names = append(names, one.Title)
	}
	return names
}

func TestAContributorDeletesTheirOwnDraftAndGetsItBack(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "writer.dev")
	announcement := stack.categoryBySlug(t, "announcement")
	draft := stack.started(t, writer.session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"An abandoned draft"}`,
		writer.grant.ID, announcement.ID,
	))
	written := stack.saved(t, writer.session, draft.ID, finished(draft, nil))
	kept := stack.checkpointed(t, writer.session, written.ID, written.Version)

	gone := stack.removed(t, writer.session, written.ID, written.Version)
	if gone.Deletion == nil {
		t.Fatal("deleting a draft recorded no deletion")
	}
	window := gone.Deletion.Until.Sub(gone.Deletion.At).Hours() / 24
	if window < 29.9 || window > 30.1 {
		t.Errorf("the recovery window is %.2f days, want 30", window)
	}
	if gone.Deletion.By != "writer.dev" {
		t.Errorf("the deletion names %q", gone.Deletion.By)
	}
	if gone.Status != "draft" {
		t.Errorf("deleting changed the post to %q; deletion is not a public state", gone.Status)
	}

	if names := postTitles(stack.listing(t, writer.session, false)); slices.Contains(names, draft.Title) {
		t.Errorf("a deleted draft is still in the active listing: %v", names)
	}
	waiting := stack.listing(t, writer.session, true)
	if len(waiting) != 1 || waiting[0].ID != draft.ID {
		t.Fatalf("the deleted listing holds %v", postTitles(waiting))
	}
	if announced := stack.events(t, draft.ID); len(announced) != 0 {
		t.Errorf("deleting a post announced %v", announced)
	}

	back := stack.recovered(t, writer.session, written.ID, gone.Version)
	if back.Deletion != nil || back.Status != "draft" {
		t.Fatalf("recovery left the post %q with deletion %+v", back.Status, back.Deletion)
	}
	if back.Title != written.Title || back.Summary != written.Summary {
		t.Errorf("recovery returned %q / %q", back.Title, back.Summary)
	}
	editions := stack.revisions(t, writer.session, draft.ID)
	if len(editions) != 1 || editions[0].ID != kept.ID {
		t.Errorf("recovery left %d editions, want the one that was kept", len(editions))
	}
	if names := postTitles(stack.listing(t, writer.session, false)); !slices.Contains(names, draft.Title) {
		t.Errorf("a recovered draft is missing from the active listing: %v", names)
	}

	done, _ := stack.history(t, writer.session, draft.ID)
	for _, wanted := range []string{"post.deleted", "post.recovered"} {
		if !hasAction(done, wanted) {
			t.Errorf("the history is %v, want it to name %s", taken(done), wanted)
		}
	}
}

func TestADeletedPostIsListedToItsOwnerAndToAnAdminAndNobodyElse(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "writer.dev")
	stranger := stack.contributor(t, "stranger@example.com", "stranger.dev")
	announcement := stack.categoryBySlug(t, "announcement")
	draft := stack.started(t, writer.session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"Somebody else's draft"}`,
		writer.grant.ID, announcement.ID,
	))
	stack.removed(t, writer.session, draft.ID, draft.Version)

	session := stack.admin(t, "editor@example.com", "illarin.editor")
	if names := postTitles(stack.listing(t, session, true)); !slices.Contains(names, draft.Title) {
		t.Errorf("an admin's deleted listing is %v", names)
	}
	if listed := stack.listing(t, stranger.session, true); len(listed) != 0 {
		t.Errorf("another contributor sees %v", postTitles(listed))
	}
	if code := stack.recover(t, stranger.session, draft.ID, draft.Version).Code; code != http.StatusForbidden {
		t.Errorf("another contributor recovered the draft: %d", code)
	}
}

func TestOnlyAnAdminDeletesAPostThatHasPublishedAndOnlyOnceItIsDown(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "writer.dev")
	announcement := stack.categoryBySlug(t, "announcement")
	draft := stack.started(t, writer.session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"A post that went out"}`,
		writer.grant.ID, announcement.ID,
	))
	stack.saved(t, writer.session, draft.ID, finished(draft, nil))
	live := stack.published(t, writer.session, draft.ID)

	if code := stack.remove(t, writer.session, live.ID, live.Version).Code; code != http.StatusForbidden {
		t.Errorf("a contributor deleted a published post: %d", code)
	}

	session := stack.admin(t, "editor@example.com", "illarin.editor")
	refused := stack.remove(t, session, live.ID, live.Version)
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("an admin deleted a post readers can see: %d", refused.Code)
	}

	down := stack.withdrawn(t, session, live.ID, live.Version, "It named the wrong build.", "")
	gone := stack.removed(t, session, down.ID, down.Version)
	if gone.Deletion == nil || gone.Status != "withdrawn" {
		t.Fatalf("deleting a withdrawn post left it %q with %+v", gone.Status, gone.Deletion)
	}
	if code := stack.recover(t, writer.session, gone.ID, gone.Version).Code; code != http.StatusForbidden {
		t.Errorf("a contributor recovered a published post: %d", code)
	}
	back := stack.recovered(t, session, gone.ID, gone.Version)
	if back.Deletion != nil || back.Status != "withdrawn" {
		t.Errorf("recovery left the post %q with %+v", back.Status, back.Deletion)
	}
}

func TestADeletedPostIsNeitherWrittenNorPublished(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "A post out of the workspace")
	written := stack.saved(t, session, draft.ID, finished(draft, nil))
	gone := stack.removed(t, session, written.ID, written.Version)

	if code := stack.save(t, session, gone.ID, finished(gone, nil)).Code; code != http.StatusBadRequest {
		t.Errorf("a deleted post was saved: %d", code)
	}
	if code := stack.publish(t, session, gone.ID).Code; code != http.StatusBadRequest {
		t.Errorf("a deleted post was published: %d", code)
	}
	later := time.Now().Add(time.Hour).Truncate(time.Second)
	if code := stack.schedule(t, session, gone.ID, gone.Version, later).Code; code != http.StatusBadRequest {
		t.Errorf("a deleted post was scheduled: %d", code)
	}
	if code := stack.remove(t, session, gone.ID, gone.Version).Code; code != http.StatusBadRequest {
		t.Errorf("a deleted post was deleted again: %d", code)
	}
	if stack.read(t, gone.Slug).Code != http.StatusNotFound {
		t.Error("a deleted draft answers on its address")
	}
}

func TestDeletingAPostStopsTheEditionItHadWaiting(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	waiting, due := stack.scheduledDraft(t, session, "An edition nobody wants", "The first words.")

	gone := stack.removed(t, session, waiting.ID, waiting.Version)
	if state := scheduleState(t, stack, waiting.Schedule.ID); state != "cancelled" {
		t.Errorf("deleting left the schedule %q, want cancelled", state)
	}
	if stack.runSchedules(t, due.Add(time.Second)) != 0 {
		t.Error("the scheduler published an edition of a deleted post")
	}
	if stack.read(t, gone.Slug).Code != http.StatusNotFound {
		t.Error("a deleted post reached readers through its schedule")
	}

	back := stack.recovered(t, session, gone.ID, gone.Version)
	again := stack.schedule(t, session, back.ID, back.Version, due.Add(time.Hour))
	if again.Code != http.StatusCreated {
		t.Fatalf("a recovered post could not be scheduled again: %d: %s",
			again.Code, again.Body.String())
	}
}

func TestRecoveryEndsOnTheDeadlineAndTakesThePicturesWithIt(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "A draft nobody came back for")
	picture := stack.uploaded(t, session, draft.ID, "document", apitest.PNG(t, 800, 400))
	written := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"document": bodyWithPicture(picture.ID, "The workspace"),
	}))
	gone := stack.removed(t, session, written.ID, written.Version)
	bytes := blobOf(t, stack, picture.ID)

	if removed := stack.clearOut(t, gone.Deletion.Until.Add(-time.Minute)); removed != 0 {
		t.Fatalf("the worker removed %d posts before the deadline", removed)
	}
	if _, err := sweeper(stack.handlers.Assets).Sweep(t.Context()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if markedBlob(t, stack, bytes) {
		t.Error("a recoverable post's picture was marked for sweeping")
	}
	if len(stack.listing(t, session, true)) != 1 {
		t.Error("a recoverable post left the deleted listing early")
	}

	if removed := stack.clearOut(t, gone.Deletion.Until); removed != 1 {
		t.Fatalf("the worker removed %d posts on the deadline, want 1", removed)
	}
	if len(stack.listing(t, session, true)) != 0 {
		t.Error("a removed post is still in the deleted listing")
	}
	response := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/publication/posts/"+draft.ID, nil,
	), session))
	if response.Code != http.StatusNotFound {
		t.Errorf("a removed post still reads as %d", response.Code)
	}
	if _, err := sweeper(stack.handlers.Assets).Sweep(t.Context()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if !markedBlob(t, stack, bytes) {
		t.Error("a removed post's picture is still held out of the sweeper's reach")
	}
}

func TestCleanupLeavesBytesAnotherPostStillNeeds(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	same := apitest.PNG(t, 800, 400)

	leaving := stack.illarinDraft(t, session, "The draft that goes")
	going := stack.uploaded(t, session, leaving.ID, "document", same)
	written := stack.saved(t, session, leaving.ID, finished(leaving, map[string]any{
		"document": bodyWithPicture(going.ID, "The workspace"),
	}))

	staying := stack.illarinDraft(t, session, "The draft that stays")
	held := stack.uploaded(t, session, staying.ID, "document", same)
	stack.saved(t, session, staying.ID, finished(staying, map[string]any{
		"document": bodyWithPicture(held.ID, "The workspace"),
	}))

	bytes := blobOf(t, stack, going.ID)
	if blobOf(t, stack, held.ID) != bytes {
		t.Fatal("the two posts do not share the bytes this test is about")
	}

	gone := stack.removed(t, session, written.ID, written.Version)
	if stack.clearOut(t, gone.Deletion.Until) != 1 {
		t.Fatal("the worker left the post behind")
	}
	if _, err := sweeper(stack.handlers.Assets).Sweep(t.Context()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if markedBlob(t, stack, bytes) {
		t.Error("cleanup marked bytes another post still refers to")
	}
}

func TestAnOfficialAddressIsNeverGivenBackAfterCleanup(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	live := stack.livePost(t, session, "A post with a history")
	first := live.Slug
	moved := stack.addressCorrected(t, session, live.ID, "the-corrected-address")
	down := stack.withdrawn(t, session, moved.ID, moved.Version,
		"Legal asked for it.", "This announcement was withdrawn.")
	gone := stack.removed(t, session, down.ID, down.Version)
	if stack.clearOut(t, gone.Deletion.Until) != 1 {
		t.Fatal("the worker left the post behind")
	}

	for _, address := range []string{first, moved.Slug} {
		found, body := stack.gone(t, address)
		if found.Slug != moved.Slug {
			t.Errorf("%s answers for %q, want %q", address, found.Slug, moved.Slug)
		}
		if found.Explanation != "This announcement was withdrawn." {
			t.Errorf("%s explains itself as %q", address, found.Explanation)
		}
		if strings.Contains(body, live.Title) {
			t.Errorf("the tombstone at %s carries the article: %s", address, body)
		}
	}

	draft := stack.illarinDraft(t, session, "A post reaching for a retired address")
	for _, address := range []string{first, moved.Slug} {
		refused := stack.save(t, session, draft.ID, finished(draft, map[string]any{
			"slug": address,
		}))
		if refused.Code != http.StatusBadRequest {
			t.Errorf("a new post took the retired address %s: %d", address, refused.Code)
		}
	}
}
