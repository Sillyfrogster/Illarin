package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

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

func (s publicationStack) schedule(
	t *testing.T,
	session *http.Cookie,
	id string,
	version int,
	at time.Time,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+id+"/schedule",
		fmt.Sprintf(`{"version":%d,"at":%q}`, version, at.Format(time.RFC3339Nano)),
	), session))
}

func (s publicationStack) scheduled(
	t *testing.T,
	session *http.Cookie,
	id string,
	version int,
	at time.Time,
) blogPost {
	t.Helper()
	response := s.schedule(t, session, id, version, at)
	if response.Code != http.StatusCreated {
		t.Fatalf("schedule status = %d, want 201: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s publicationStack) replaceSchedule(
	t *testing.T,
	session *http.Cookie,
	id, revisionID string,
	at time.Time,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPut, "/v1/publication/posts/"+id+"/schedule",
		fmt.Sprintf(`{"revisionId":%q,"at":%q}`, revisionID, at.Format(time.RFC3339Nano)),
	), session))
}

func (s publicationStack) cancelSchedule(
	t *testing.T,
	session *http.Cookie,
	id string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/publication/posts/"+id+"/schedule", nil,
	), session))
}

func (s publicationStack) runSchedules(t *testing.T, at time.Time) int {
	t.Helper()
	settled, err := s.handlers.Publications.PublishDueSchedules(t.Context(), at)
	if err != nil {
		t.Fatalf("run the scheduler: %v", err)
	}
	return settled
}

func (s publicationStack) scheduledDraft(
	t *testing.T,
	session *http.Cookie,
	title, text string,
) (blogPost, time.Time) {
	t.Helper()
	draft := s.illarinDraft(t, session, title)
	written := s.saved(t, session, draft.ID, finished(draft, map[string]any{
		"document": json.RawMessage(paragraph(text)),
	}))
	due := time.Now().Add(time.Hour).Truncate(time.Second)
	return s.scheduled(t, session, written.ID, written.Version, due), due
}

func TestASchedulePublishesTheEditionItNamedAndNotALaterOne(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

	waiting, due := stack.scheduledDraft(t, session, "Illarin ships on Tuesday", "The first words.")
	if waiting.Status != "draft" || waiting.Schedule == nil {
		t.Fatalf("scheduling left the post %q with schedule %+v", waiting.Status, waiting.Schedule)
	}
	if waiting.Schedule.State != "pending" || waiting.Schedule.RevisionNumber != 1 {
		t.Fatalf("schedule = %+v", waiting.Schedule)
	}
	if !waiting.Schedule.At.Equal(due.UTC()) {
		t.Errorf("schedule is due at %s, want %s", waiting.Schedule.At, due.UTC())
	}
	if stack.read(t, waiting.Slug).Code != http.StatusNotFound {
		t.Fatal("a scheduled post is already public")
	}
	if settled := stack.runSchedules(t, due.Add(-time.Minute)); settled != 0 {
		t.Fatalf("the worker published %d editions before they were due", settled)
	}

	edited := stack.saved(t, session, waiting.ID, finished(waiting, map[string]any{
		"version":  waiting.Version,
		"document": json.RawMessage(paragraph("Words written after the schedule was made.")),
	}))
	stack.checkpointed(t, session, edited.ID, edited.Version)

	if settled := stack.runSchedules(t, due.Add(time.Second)); settled != 1 {
		t.Fatalf("the worker settled %d editions, want 1", settled)
	}
	found := stack.reader(t, waiting.Slug)
	if !strings.Contains(firstWords(found.Document), "The first words.") {
		t.Fatalf("readers got %s, want the edition that was scheduled", found.Document.Content[0])
	}
	after := stack.working(t, session, waiting.ID)
	if after.Status != "published" || after.PublicRevision != waiting.Schedule.RevisionID {
		t.Fatalf("public revision = %q, want the scheduled one", after.PublicRevision)
	}
	if after.Schedule == nil || after.Schedule.State != "published" {
		t.Fatalf("schedule after publishing = %+v", after.Schedule)
	}
}

func TestAPublishedPostKeepsItsPublicEditionWhileAnotherIsScheduled(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

	draft := stack.illarinDraft(t, session, "Release notes")
	written := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"document": json.RawMessage(paragraph("What readers can see today.")),
	}))
	live := stack.publishedAt(t, session, written.ID, written.Version)

	updated := stack.saved(t, session, live.ID, finished(live, map[string]any{
		"version":  live.Version,
		"document": json.RawMessage(paragraph("What readers get on Tuesday.")),
	}))
	due := time.Now().Add(time.Hour).Truncate(time.Second)
	waiting := stack.scheduled(t, session, updated.ID, updated.Version, due)
	if waiting.Status != "published" || waiting.Schedule == nil {
		t.Fatalf("scheduling an update left the post %q", waiting.Status)
	}
	if waiting.PublicRevision != live.PublicRevision {
		t.Fatal("scheduling an update changed what readers see")
	}
	before := stack.reader(t, live.Slug)
	if !strings.Contains(firstWords(before.Document), "today") {
		t.Fatalf("readers already got the scheduled edition: %s", before.Document.Content[0])
	}
	if before.UpdatedAt != nil {
		t.Error("scheduling recorded a public update date")
	}

	stack.runSchedules(t, due.Add(time.Second))

	found := stack.reader(t, live.Slug)
	if !strings.Contains(firstWords(found.Document), "Tuesday") {
		t.Fatalf("readers got %s after the schedule ran", found.Document.Content[0])
	}
	if found.UpdatedAt == nil || !found.PublishedAt.Equal(before.PublishedAt) {
		t.Errorf("dates after a scheduled update = %v / %v", found.PublishedAt, found.UpdatedAt)
	}
}

func TestReplacingAScheduleNamesAnotherKeptEditionAndLeavesTheOldOneWhole(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

	waiting, due := stack.scheduledDraft(t, session, "A correction is coming", "The first attempt.")
	first := waiting.Schedule.RevisionID

	corrected := stack.saved(t, session, waiting.ID, finished(waiting, map[string]any{
		"version":  waiting.Version,
		"document": json.RawMessage(paragraph("The corrected words.")),
	}))
	second := stack.checkpointed(t, session, corrected.ID, corrected.Version)

	later := due.Add(2 * time.Hour)
	response := stack.replaceSchedule(t, session, waiting.ID, second.ID, later)
	if response.Code != http.StatusOK {
		t.Fatalf("replace status = %d: %s", response.Code, response.Body.String())
	}
	replaced := decodePost(t, response)
	if replaced.Schedule.RevisionID != second.ID || !replaced.Schedule.At.Equal(later.UTC()) {
		t.Fatalf("replaced schedule = %+v", replaced.Schedule)
	}
	if replaced.Schedule.ID != waiting.Schedule.ID {
		t.Error("replacing a schedule started a second one")
	}

	kept := stack.revisions(t, session, waiting.ID)
	for _, one := range kept {
		if one.ID == first && one.Title != waiting.Title {
			t.Fatalf("the edition the schedule left was rewritten: %+v", one)
		}
	}
	if settled := stack.runSchedules(t, due.Add(time.Second)); settled != 0 {
		t.Fatalf("the worker published %d editions at the instant it no longer owed", settled)
	}
	stack.runSchedules(t, later.Add(time.Second))
	found := stack.reader(t, waiting.Slug)
	if !strings.Contains(firstWords(found.Document), "corrected") {
		t.Fatalf("readers got %s, want the replacement", found.Document.Content[0])
	}

	done, _ := stack.history(t, session, waiting.ID)
	if !hasAction(done, "post.schedule.replaced") {
		t.Errorf("replacing a schedule recorded no action: %+v", done)
	}
}

func TestCancellingAScheduleStopsItAndAskingTwiceIsHarmless(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

	waiting, due := stack.scheduledDraft(t, session, "Not this week after all", "Held back.")

	response := stack.cancelSchedule(t, session, waiting.ID)
	if response.Code != http.StatusOK {
		t.Fatalf("cancel status = %d: %s", response.Code, response.Body.String())
	}
	cancelled := decodePost(t, response)
	if cancelled.Schedule == nil || cancelled.Schedule.State != "cancelled" {
		t.Fatalf("schedule after cancelling = %+v", cancelled.Schedule)
	}

	again := stack.cancelSchedule(t, session, waiting.ID)
	if again.Code != http.StatusOK {
		t.Fatalf("cancelling twice returned %d: %s", again.Code, again.Body.String())
	}

	if settled := stack.runSchedules(t, due.Add(time.Second)); settled != 0 {
		t.Fatalf("the worker published %d cancelled editions", settled)
	}
	if stack.read(t, waiting.Slug).Code != http.StatusNotFound {
		t.Error("a cancelled schedule still put the post in public view")
	}
	done, _ := stack.history(t, session, waiting.ID)
	if !hasAction(done, "post.schedule.cancelled") {
		t.Errorf("cancelling recorded no action: %+v", done)
	}
}

func TestASecondScheduleIsRefusedRatherThanQueuedBehindTheFirst(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

	waiting, due := stack.scheduledDraft(t, session, "Only one at a time", "The words.")

	response := stack.schedule(t, session, waiting.ID, waiting.Version, due.Add(time.Hour))
	if response.Code != http.StatusConflict {
		t.Fatalf("a second schedule returned %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "already_scheduled") {
		t.Errorf("refusal = %s", response.Body.String())
	}
}

func TestAScheduleWillNotPointAtAnInstantThatHasPassed(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

	draft := stack.illarinDraft(t, session, "Yesterday")
	written := stack.saved(t, session, draft.ID, finished(draft, nil))

	past := stack.schedule(t, session, written.ID, written.Version, time.Now().Add(-time.Minute))
	if past.Code != http.StatusBadRequest {
		t.Fatalf("an instant in the past returned %d: %s", past.Code, past.Body.String())
	}

	local := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+written.ID+"/schedule",
		fmt.Sprintf(`{"version":%d,"at":"2030-01-01T09:00:00"}`, written.Version),
	), session))
	if local.Code != http.StatusBadRequest {
		t.Fatalf("a time with no offset returned %d: %s", local.Code, local.Body.String())
	}
}

func TestRevokedApprovalCannotPublishThroughAScheduleItLeftBehind(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "writer.dev")
	announcement := stack.categoryBySlug(t, "announcement")

	draft := stack.started(t, writer.session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"Written under approval"}`,
		writer.grant.ID, announcement.ID,
	))
	written := stack.saved(t, writer.session, draft.ID, finished(draft, nil))
	due := time.Now().Add(time.Hour).Truncate(time.Second)
	waiting := stack.scheduled(t, writer.session, written.ID, written.Version, due)

	revoked := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/publication/grants/"+writer.grant.ID, nil,
	), stack.authority))
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d: %s", revoked.Code, revoked.Body.String())
	}

	if settled := stack.runSchedules(t, due.Add(time.Second)); settled != 0 {
		t.Fatalf("a revoked approval published %d editions", settled)
	}
	if stack.read(t, waiting.Slug).Code != http.StatusNotFound {
		t.Fatal("a revoked approval put its post in public view")
	}
	if state := scheduleState(t, stack, waiting.Schedule.ID); state != "stopped" {
		t.Errorf("schedule under a revoked approval is %q, want stopped", state)
	}
}

func TestAnAdminOwnedScheduleStillPublishesWhileOtherApprovalsAreRevoked(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	writer := stack.contributor(t, "writer@example.com", "writer.dev")

	waiting, due := stack.scheduledDraft(t, session, "Illarin speaks for itself", "Still going.")

	revoked := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/publication/grants/"+writer.grant.ID, nil,
	), stack.authority))
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d", revoked.Code)
	}

	if settled := stack.runSchedules(t, due.Add(time.Second)); settled != 1 {
		t.Fatalf("an admin-owned schedule settled %d editions, want 1", settled)
	}
	if stack.read(t, waiting.Slug).Code != http.StatusOK {
		t.Error("an admin-owned schedule did not publish")
	}
}

func TestAnInterruptedWorkerLeavesTheEditionRecoverableAndNotHalfPublic(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

	waiting, due := stack.scheduledDraft(t, session, "Survives a restart", "The words that go live.")

	_, err := stack.pool.Exec(t.Context(), `
		update post_schedules
		   set state = 'publishing', attempts = 1,
		       lease_token = gen_random_uuid(), lease_expires_at = $2
		 where id = $1
	`, waiting.Schedule.ID, due.Add(time.Second))
	if err != nil {
		t.Fatalf("leave a lease behind: %v", err)
	}
	if stack.read(t, waiting.Slug).Code != http.StatusNotFound {
		t.Fatal("an interrupted attempt left the post partly public")
	}

	if settled := stack.runSchedules(t, due.Add(time.Minute)); settled != 1 {
		t.Fatalf("the worker recovered %d abandoned editions, want 1", settled)
	}
	found := stack.reader(t, waiting.Slug)
	if !strings.Contains(firstWords(found.Document), "go live") {
		t.Fatalf("the recovered edition reads %s", found.Document.Content[0])
	}
}

func TestAWorkerThatKeepsFailingStopsTheScheduleInsteadOfRetryingForever(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

	waiting, due := stack.scheduledDraft(t, session, "Never quite made it", "The words.")

	_, err := stack.pool.Exec(t.Context(), `
		update post_schedules set attempts = $2 where id = $1
	`, waiting.Schedule.ID, 5)
	if err != nil {
		t.Fatalf("wear the attempts down: %v", err)
	}

	stack.runSchedules(t, due.Add(time.Second))

	if stack.read(t, waiting.Slug).Code != http.StatusNotFound {
		t.Error("an exhausted schedule published anyway")
	}
	if state := scheduleState(t, stack, waiting.Schedule.ID); state != "stopped" {
		t.Errorf("an exhausted schedule is %q, want stopped", state)
	}
	after := stack.working(t, session, waiting.ID)
	if after.Schedule == nil || after.Schedule.StoppedBecause == "" {
		t.Errorf("a stopped schedule says nothing about why: %+v", after.Schedule)
	}
}

func TestAScheduledEditionKeepsItsPicturesAfterTheWorkingCopyDropsThem(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

	draft := stack.illarinDraft(t, session, "An article with a picture")
	picture := stack.uploaded(t, session, draft.ID, "document", apitest.PNG(t, 800, 400))
	written := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"document": bodyWithPicture(picture.ID, "The workspace"),
	}))
	due := time.Now().Add(time.Hour).Truncate(time.Second)
	waiting := stack.scheduled(t, session, written.ID, written.Version, due)

	stack.saved(t, session, waiting.ID, finished(waiting, map[string]any{
		"version":  waiting.Version,
		"document": json.RawMessage(paragraph("No picture any more.")),
	}))

	var held int
	err := stack.pool.QueryRow(t.Context(), `
		select count(*) from post_media_uses where media_id = $1 and revision_id = $2
	`, picture.ID, waiting.Schedule.RevisionID).Scan(&held)
	if err != nil {
		t.Fatalf("count what the scheduled edition refers to: %v", err)
	}
	if held != 1 {
		t.Fatalf("the scheduled edition refers to %d pictures, want 1", held)
	}
	if _, err := stack.handlers.Assets.Sweep(t.Context()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	var blob *string
	err = stack.pool.QueryRow(t.Context(),
		`select blob_id::text from post_media where id = $1`, picture.ID).Scan(&blob)
	if err != nil {
		t.Fatalf("read the picture after sweeping: %v", err)
	}
	if blob == nil {
		t.Fatal("sweeping took the picture a scheduled edition still needs")
	}
}

func TestSchedulingRecordsWhatHappenedWithoutCopyingTheArticle(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

	secret := "Words that belong only in the article."
	waiting, due := stack.scheduledDraft(t, session, "Accountable", secret)
	stack.runSchedules(t, due.Add(time.Second))

	done, raw := stack.history(t, session, waiting.ID)
	if !hasAction(done, "post.scheduled") {
		t.Errorf("scheduling recorded no action: %+v", done)
	}
	published := actionOf(done, "post.published")
	if published == nil {
		t.Fatalf("a scheduled run recorded no publication: %+v", done)
	}
	if published.Credential != "system" {
		t.Errorf("a scheduled run acted as %q, want system", published.Credential)
	}
	if published.Actor != "illarin.editor" {
		t.Errorf("a scheduled run is credited to %q", published.Actor)
	}
	if strings.Contains(raw, secret) {
		t.Error("the private record copied the article body")
	}
}

func hasAction(done []postAction, named string) bool {
	return actionOf(done, named) != nil
}

func actionOf(done []postAction, named string) *postAction {
	for index := range done {
		if done[index].Action == named {
			return &done[index]
		}
	}
	return nil
}

func scheduleState(t *testing.T, stack publicationStack, id string) string {
	t.Helper()
	var state string
	err := stack.pool.QueryRow(t.Context(),
		`select state from post_schedules where id = $1`, id).Scan(&state)
	if err != nil {
		t.Fatalf("read the schedule state: %v", err)
	}
	return state
}

func firstWords(document postDocument) string {
	if len(document.Content) == 0 {
		return ""
	}
	written, err := json.Marshal(document.Content[0])
	if err != nil {
		return ""
	}
	return string(written)
}

func TestSchedulingRefusesAWorkingCopySomeoneElseHasMovedPast(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

	draft := stack.illarinDraft(t, session, "Two people at one desk")
	written := stack.saved(t, session, draft.ID, finished(draft, nil))
	stack.saved(t, session, written.ID, finished(written, map[string]any{
		"version":  written.Version,
		"document": json.RawMessage(paragraph("Someone else got here first.")),
	}))

	stale := stack.schedule(t, session, written.ID, written.Version,
		time.Now().Add(time.Hour).Truncate(time.Second))
	if stale.Code != http.StatusConflict {
		t.Fatalf("a stale schedule returned %d: %s", stale.Code, stale.Body.String())
	}
	if !strings.Contains(stale.Body.String(), "stale_version") {
		t.Errorf("refusal = %s", stale.Body.String())
	}
	if held := scheduleCount(t, stack, written.ID); held != 0 {
		t.Errorf("a refused schedule left %d rows behind", held)
	}
}

func scheduleCount(t *testing.T, stack publicationStack, postID string) int {
	t.Helper()
	var held int
	err := stack.pool.QueryRow(t.Context(),
		`select count(*) from post_schedules where post_id = $1`, postID).Scan(&held)
	if err != nil {
		t.Fatalf("count schedules: %v", err)
	}
	return held
}

func TestPublishingNowStopsTheScheduleItOvertook(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

	waiting, due := stack.scheduledDraft(t, session, "Out early", "The scheduled words.")

	edited := stack.saved(t, session, waiting.ID, finished(waiting, map[string]any{
		"version":  waiting.Version,
		"document": json.RawMessage(paragraph("The words the author sent out early.")),
	}))
	live := stack.publishedAt(t, session, edited.ID, edited.Version)
	if live.Schedule == nil || live.Schedule.State != "cancelled" {
		t.Fatalf("publishing left the schedule %+v", live.Schedule)
	}

	if settled := stack.runSchedules(t, due.Add(time.Second)); settled != 0 {
		t.Fatalf("an overtaken schedule published %d editions", settled)
	}
	found := stack.reader(t, waiting.Slug)
	if !strings.Contains(firstWords(found.Document), "early") {
		t.Fatalf("readers were sent back to %s", firstWords(found.Document))
	}
	done, _ := stack.history(t, session, waiting.ID)
	if !hasAction(done, "post.schedule.cancelled") {
		t.Errorf("overtaking a schedule recorded no action: %+v", done)
	}
}

func TestTheSchedulerStopsWithTheProcessItRunsIn(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	ctx, stop := context.WithCancel(t.Context())
	stopped := make(chan struct{})

	go func() {
		defer close(stopped)
		stack.handlers.Publications.RunScheduler(ctx, func(error) {})
	}()
	stop()

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("the scheduler kept running after its context was cancelled")
	}
}
