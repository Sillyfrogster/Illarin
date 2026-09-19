package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/blog"
	"github.com/Sillyfrogster/Illarin/api/internal/integration"
	"github.com/google/uuid"
)

func (s integrationStack) eventNames(t *testing.T, postID string) []string {
	t.Helper()
	rows, err := s.pool.Query(context.Background(), `
		select type from blog_announcements where post_id = $1 order by occurred_at
	`, postID)
	if err != nil {
		t.Fatalf("read the events a post recorded: %v", err)
	}
	defer rows.Close()
	held := make([]string, 0, 4)
	for rows.Next() {
		var one string
		if err := rows.Scan(&one); err != nil {
			t.Fatalf("read one event a post recorded: %v", err)
		}
		held = append(held, one)
	}
	return held
}

func (s integrationStack) sentTo(t *testing.T, postID, name string) []string {
	t.Helper()
	held := make([]string, 0, 4)
	for _, one := range s.attempts(t, s.editor, postID).Attempts {
		if one.Integration == name {
			held = append(held, one.AnnouncementType)
		}
	}
	return held
}

func (s integrationStack) quietlyPublished(t *testing.T, ready blogPost) blogPost {
	t.Helper()
	response := s.publishTo(t, s.editor, ready.ID, ready.Version, fmt.Sprintf(
		`{"version":%d,"integrationIds":[]}`, ready.Version,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func withdrawalsAmong(held []arrived) []arrived {
	kept := make([]arrived, 0, len(held))
	for _, one := range held {
		if strings.Contains(string(one.Body), blog.PostWithdrawn) {
			kept = append(kept, one)
		}
	}
	return kept
}

func TestADestinationTakesPublishedEventsAndNothingElseByDefault(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)

	made := stack.active(t, "Release feed")

	shown := stack.integrations(t, stack.authority).Integrations[0]
	if len(shown.Announcements) != 1 || shown.Announcements[0] != blog.PostPublished {
		t.Errorf("events = %v, want the published event alone", shown.Announcements)
	}
	live := stack.publishedTo(t, stack.readyPost(t), made.Integration.ID, "")
	stack.sendQueued(t)
	stack.publishedTo(t, live, made.Integration.ID, "")

	if sent := stack.sendQueued(t); sent != 0 {
		t.Errorf("changes to a live post made %d requests, want 0", sent)
	}
}

func TestChangingALivePostReachesOnlyWhoeverAsksForUpdates(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	announcements := stack.active(t, "Announcements")
	changes := stack.activeFor(t, "Change log", []string{blog.PostUpdated})
	ready := stack.readyPost(t)

	live := stack.publishedTo(t, ready, announcements.Integration.ID, "")
	both := fmt.Sprintf(`{"version":%d,"integrationIds":[%q,%q]}`,
		live.Version, announcements.Integration.ID, changes.Integration.ID)
	response := stack.publishTo(t, stack.editor, live.ID, live.Version, both)

	if response.Code != http.StatusOK {
		t.Fatalf("publish changes status = %d: %s", response.Code, response.Body.String())
	}
	if got := stack.sentTo(t, ready.ID, "Change log"); len(got) != 1 {
		t.Fatalf("the change log was queued %v, want the update alone", got)
	} else if got[0] != blog.PostUpdated {
		t.Errorf("the change log was queued %q, want the update", got[0])
	}
	if got := stack.sentTo(t, ready.ID, "Announcements"); len(got) != 1 {
		t.Errorf("announcements was queued %v, want the publication alone", got)
	}
}

func TestWithdrawingAPostReachesOnlyWhoeverAsksForWithdrawals(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	announcements := stack.active(t, "Announcements")
	removals := stack.activeFor(t, "Takedowns", []string{blog.PostWithdrawn})
	ready := stack.readyPost(t)
	live := stack.publishedTo(t, ready, announcements.Integration.ID, "")
	stack.sendQueued(t)

	gone := stack.withdraw(t, stack.editor, live.ID, fmt.Sprintf(
		`{"version":%d,"reason":"It named the wrong version.","integrationIds":[%q,%q],`+
			`"note":"The release notes were wrong."}`,
		live.Version, announcements.Integration.ID, removals.Integration.ID,
	))

	if gone.Code != http.StatusOK {
		t.Fatalf("withdraw status = %d: %s", gone.Code, gone.Body.String())
	}
	if got := stack.sentTo(t, ready.ID, "Takedowns"); len(got) != 1 ||
		got[0] != blog.PostWithdrawn {
		t.Fatalf("takedowns was queued %v, want the withdrawal alone", got)
	}
	if got := stack.sentTo(t, ready.ID, "Announcements"); len(got) != 1 {
		t.Errorf("announcements was queued %v, want the publication alone", got)
	}
	stack.to.forget()
	stack.sendQueued(t)

	arrivals := withdrawalsAmong(stack.to.arrivals())
	if len(arrivals) != 1 {
		t.Fatalf("the receiver was sent %d requests, want the withdrawal", len(arrivals))
	}
	body := string(arrivals[0].Body)
	for _, want := range []string{
		`"type":"publication.post.withdrawn.v1"`,
		`"note":"The release notes were wrong."`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the unpublishing announcement does not carry %s: %s", want, body)
		}
	}
}

func TestAWithdrawalCanSendNothing(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	removals := stack.activeFor(t, "Takedowns", []string{blog.PostWithdrawn})
	ready := stack.readyPost(t)
	live := stack.publishedTo(t, ready, removals.Integration.ID, "")

	gone := stack.withdraw(t, stack.editor, live.ID, fmt.Sprintf(
		`{"version":%d,"reason":"It named the wrong version.","integrationIds":[]}`, live.Version,
	))

	if gone.Code != http.StatusOK {
		t.Fatalf("withdraw status = %d: %s", gone.Code, gone.Body.String())
	}
	if got := stack.sentTo(t, ready.ID, "Takedowns"); len(got) != 0 {
		t.Errorf("a quiet withdrawal queued %v", got)
	}
}

func TestARepublicationCarriesItsOwnChoice(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	live := stack.publishedTo(t, ready, made.Integration.ID, "")
	gone := stack.withdrawn(t, stack.editor, live.ID, live.Version, "It went out early.", "")

	back := stack.republish(t, stack.editor, gone.ID, fmt.Sprintf(
		`{"version":%d,"revisionId":%q,"integrationIds":[%q],"note":"It is back."}`,
		gone.Version, gone.PublicRevision, made.Integration.ID,
	))

	if back.Code != http.StatusOK {
		t.Fatalf("republish status = %d: %s", back.Code, back.Body.String())
	}
	if got := stack.sentTo(t, ready.ID, "Release feed"); len(got) != 2 {
		t.Fatalf("the release feed was queued %v, want both publications", got)
	}
	stack.to.forget()
	stack.sendQueued(t)

	arrivals := stack.to.arrivals()
	if len(arrivals) != 2 {
		t.Fatalf("the receiver was sent %d requests, want 2", len(arrivals))
	}
	if !strings.Contains(string(arrivals[1].Body), `"note":"It is back."`) {
		t.Errorf("the republication lost its note: %s", arrivals[1].Body)
	}
}

func TestTheEventNamesAreExactlyTheThreeIllarinPromises(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	ready := stack.readyPost(t)
	live := stack.quietlyPublished(t, ready)
	changed := stack.quietlyPublished(t, live)
	gone := stack.withdrawn(t, stack.editor, changed.ID, changed.Version, "It went out early.", "")
	stack.republished(t, stack.editor, gone.ID, gone.Version, gone.PublicRevision)

	held := stack.eventNames(t, ready.ID)

	want := []string{
		"publication.post.published.v1",
		"publication.post.updated.v1",
		"publication.post.withdrawn.v1",
		"publication.post.published.v1",
	}
	if len(held) != len(want) {
		t.Fatalf("the post recorded %v, want %v", held, want)
	}
	for index, name := range want {
		if held[index] != name {
			t.Errorf("event %d = %q, want %q", index, held[index], name)
		}
	}
}

func TestEditorialWorkOutsidePublicViewSendsNothing(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.activeFor(t, "Everything", []string{
		blog.PostPublished, blog.PostUpdated, blog.PostWithdrawn,
	})
	draft := stack.illarinDraft(t, stack.editor, "Illarin keeps its own writing now")
	ready := stack.saved(t, stack.editor, draft.ID, finished(draft, nil))

	kept := stack.saved(t, stack.editor, ready.ID, finished(ready, map[string]any{
		"version": ready.Version, "title": "A better title",
	}))
	removed := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+kept.ID+"/delete",
		fmt.Sprintf(`{"version":%d}`, kept.Version),
	), stack.editor))
	if removed.Code != http.StatusOK {
		t.Fatalf("delete status = %d: %s", removed.Code, removed.Body.String())
	}
	recovered := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+kept.ID+"/recover",
		fmt.Sprintf(`{"version":%d}`, kept.Version),
	), stack.editor))
	if recovered.Code != http.StatusOK {
		t.Fatalf("recover status = %d: %s", recovered.Code, recovered.Body.String())
	}

	if held := stack.eventNames(t, ready.ID); len(held) != 0 {
		t.Errorf("editorial work recorded %v, want nothing", held)
	}
	if sent := stack.sentTo(t, ready.ID, made.Integration.Name); len(sent) != 0 {
		t.Errorf("editorial work queued %v, want nothing", sent)
	}
}

func TestWhatArrivesIsWhatTheContractDocuments(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Integration.ID, "Read it in ten minutes.")
	stack.sendQueued(t)

	arrivals := stack.to.arrivals()
	if len(arrivals) != 1 {
		t.Fatalf("the receiver was sent %d requests, want 1", len(arrivals))
	}
	reader := json.NewDecoder(strings.NewReader(string(arrivals[0].Body)))
	reader.DisallowUnknownFields()
	var held integration.BlogAnnouncement
	if err := reader.Decode(&held); err != nil {
		t.Fatalf("the event does not fit the documented contract: %v", err)
	}
	if held.Type != integration.BlogAnnouncementType(blog.PostPublished) {
		t.Errorf("type = %q, want the published event", held.Type)
	}
	if held.Id == uuid.Nil || held.Post.Id == uuid.Nil || held.Post.RevisionId == uuid.Nil {
		t.Error("the event does not carry the ids a receiver deduplicates on")
	}
	if held.OccurredAt.IsZero() {
		t.Error("the event does not carry the time a receiver orders by")
	}
	if held.Note == nil || *held.Note != "Read it in ten minutes." {
		t.Errorf("note = %v, want the one the transition captured", held.Note)
	}
}

func TestASubscriptionIsRefusedForAnEventIllarinDoesNotSend(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)

	response := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/blog/integrations", fmt.Sprintf(
			`{"name":"Release feed","address":%q,"events":["publication.post.read.v1"]}`,
			stack.to.address(),
		),
	), stack.authority))

	if response.Code != http.StatusBadRequest {
		t.Errorf("add status = %d, want 400: %s", response.Code, response.Body.String())
	}
}

func TestASubscriptionCanBeNarrowedAfterwards(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")

	response := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPatch, "/v1/blog/integrations/"+made.Integration.ID,
		`{"events":["publication.post.withdrawn.v1","publication.post.published.v1"]}`,
	), stack.authority))

	if response.Code != http.StatusOK {
		t.Fatalf("update status = %d: %s", response.Code, response.Body.String())
	}
	shown := stack.integrations(t, stack.authority).Integrations[0]
	want := []string{blog.PostPublished, blog.PostWithdrawn}
	if len(shown.Announcements) != len(want) {
		t.Fatalf("events = %v, want %v", shown.Announcements, want)
	}
	for index, name := range want {
		if shown.Announcements[index] != name {
			t.Errorf("event %d = %q, want %q", index, shown.Announcements[index], name)
		}
	}
	if shown.State != "active" {
		t.Errorf("narrowing the subscription left the integration %q", shown.State)
	}
}
