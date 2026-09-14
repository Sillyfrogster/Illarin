package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func (s publicationStack) correctAddress(
	t *testing.T,
	session *http.Cookie,
	id, slug string,
) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, authorized(jsonRequest(t,
		http.MethodPut, "/v1/publication/posts/"+id+"/address",
		fmt.Sprintf(`{"slug":%q}`, slug),
	), session))
}

func (s publicationStack) addressCorrected(
	t *testing.T,
	session *http.Cookie,
	id, slug string,
) blogPost {
	t.Helper()
	response := s.correctAddress(t, session, id, slug)
	if response.Code != http.StatusOK {
		t.Fatalf("correct address status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func (s publicationStack) correctByline(
	t *testing.T,
	session *http.Cookie,
	id, handle string,
) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, authorized(jsonRequest(t,
		http.MethodPut, "/v1/publication/posts/"+id+"/byline",
		fmt.Sprintf(`{"handle":%q}`, handle),
	), session))
}

func (s publicationStack) livePost(t *testing.T, session *http.Cookie, title string) blogPost {
	t.Helper()
	draft := s.illarinDraft(t, session, title)
	s.saved(t, session, draft.ID, finished(draft, nil))
	return s.published(t, session, draft.ID)
}

func TestADraftAddressIsTheAuthorsToChooseAndNormalizes(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "Choose the address")

	written := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"slug": "  What's  New — in Illarin 3!  ",
	}))
	if written.Slug != "what-s-new-in-illarin-3" {
		t.Errorf("normalized address = %q", written.Slug)
	}

	again := stack.saved(t, session, draft.ID, finished(written, map[string]any{
		"version": written.Version,
		"slug":    "WHAT_IS_NEW",
	}))
	if again.Slug != "what-is-new" {
		t.Errorf("second address = %q", again.Slug)
	}
	if len(again.FormerAddresses) != 0 {
		t.Errorf("an unpublished post left %v behind", again.FormerAddresses)
	}
}

func TestAPostAddressRefusesEveryReservedBlogRouteAndFeedName(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "Reserved words")

	for _, candidate := range []string{
		"category", "app", "page", "feed", "feed.xml", "feed.json", "rss", "sitemap", "atom",
		"media", "withdrawn",
	} {
		response := stack.save(t, session, draft.ID, finished(draft, map[string]any{
			"slug": candidate,
		}))
		if response.Code != http.StatusBadRequest {
			t.Errorf("the address %q saved: %d", candidate, response.Code)
		}
	}
}

func TestPublicationLocksTheAddressForEveryOrdinarySave(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "writer.dev")
	announcement := stack.categoryBySlug(t, "announcement")
	draft := stack.started(t, writer.session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"Address locked"}`, writer.grant.ID, announcement.ID,
	))
	written := stack.saved(t, writer.session, draft.ID, finished(draft, map[string]any{
		"slug": "address-locked",
	}))
	live := stack.published(t, writer.session, draft.ID)

	byAuthor := stack.save(t, writer.session, draft.ID, finished(written, map[string]any{
		"version": live.Version, "slug": "somewhere-else",
	}))
	if byAuthor.Code != http.StatusForbidden {
		t.Errorf("the author moved a published post: %d", byAuthor.Code)
	}

	admin := stack.admin(t, "admin@example.com", "the.admin")
	bySave := stack.save(t, admin, draft.ID, finished(written, map[string]any{
		"version": live.Version, "slug": "somewhere-else",
	}))
	if bySave.Code != http.StatusForbidden {
		t.Errorf("an admin moved a published post by saving it: %d", bySave.Code)
	}
	if still := stack.reader(t, "address-locked"); still.Slug != "address-locked" {
		t.Errorf("the published address became %q", still.Slug)
	}
}

func TestEveryCorrectedAddressReachesThePostDirectly(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	live := stack.livePost(t, session, "Illarin 3 is heer")

	once := stack.addressCorrected(t, session, live.ID, "illarin-3-is-here")
	if once.Slug != "illarin-3-is-here" {
		t.Fatalf("corrected address = %q", once.Slug)
	}
	twice := stack.addressCorrected(t, session, live.ID, "illarin-three-is-here")

	for _, address := range []string{
		"illarin-3-is-heer", "illarin-3-is-here", "illarin-three-is-here",
	} {
		found := stack.reader(t, address)
		if found.ID != live.ID {
			t.Errorf("the address %q reached %s, want %s", address, found.ID, live.ID)
		}
		if found.Slug != "illarin-three-is-here" {
			t.Errorf("the address %q answered with %q, want the current one", address, found.Slug)
		}
	}
	if len(twice.FormerAddresses) != 2 {
		t.Errorf("former addresses = %v, want both of them", twice.FormerAddresses)
	}
}

func TestAnAddressAPostHasLeftIsNeverGivenToAnother(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	live := stack.livePost(t, session, "The first name")
	stack.addressCorrected(t, session, live.ID, "the-second-name")

	other := stack.illarinDraft(t, session, "Another post entirely")
	response := stack.save(t, session, other.ID, finished(other, map[string]any{
		"slug": "the-first-name",
	}))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("a former address was reused: %d %s", response.Code, response.Body.String())
	}

	live2 := stack.livePost(t, session, "A third post")
	if code := stack.correctAddress(t, session, live2.ID, "the-first-name").Code; code != http.StatusBadRequest {
		t.Errorf("a correction took a former address: %d", code)
	}
}

func TestOnlyAnAdminCorrectsAnAddressAndOnlyOnceItIsPublished(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "writer.dev")
	announcement := stack.categoryBySlug(t, "announcement")
	draft := stack.started(t, writer.session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"Not yours to move"}`, writer.grant.ID, announcement.ID,
	))
	stack.saved(t, writer.session, draft.ID, finished(draft, nil))

	admin := stack.admin(t, "admin@example.com", "the.admin")
	if code := stack.correctAddress(t, admin, draft.ID, "too-early").Code; code != http.StatusBadRequest {
		t.Errorf("an unpublished post was corrected: %d", code)
	}

	stack.published(t, writer.session, draft.ID)
	if code := stack.correctAddress(t, writer.session, draft.ID, "mine-now").Code; code != http.StatusForbidden {
		t.Errorf("a contributor corrected an address: %d", code)
	}
	if code := stack.correctAddress(t, admin, draft.ID, "category").Code; code != http.StatusBadRequest {
		t.Errorf("a correction took a reserved address: %d", code)
	}
	stack.addressCorrected(t, admin, draft.ID, "moved-by-the-admin")

	var actions int
	err := stack.pool.QueryRow(context.Background(), `
		select count(*) from publication_audits
		 where post_id = $1 and action = 'post.address.corrected'
	`, draft.ID).Scan(&actions)
	if err != nil {
		t.Fatalf("read the correction audit: %v", err)
	}
	if actions != 1 {
		t.Errorf("the correction left %d audit rows", actions)
	}
}

func TestAnAdminCorrectsAMistakenBylineAndNothingElse(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "writer.dev")
	announcement := stack.categoryBySlug(t, "announcement")
	draft := stack.started(t, writer.session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"Signed by the wrong hand"}`,
		writer.grant.ID, announcement.ID,
	))
	stack.saved(t, writer.session, draft.ID, finished(draft, nil))
	live := stack.published(t, writer.session, draft.ID)

	colleague := stack.member(t, "colleague@example.com", "the.colleague")
	saveProfile(t, stack.router, colleague, `{"displayName":"The Colleague","links":[]}`)
	admin := stack.admin(t, "admin@example.com", "the.admin")

	if code := stack.correctByline(t, writer.session, draft.ID, "the.colleague").Code; code != http.StatusForbidden {
		t.Errorf("a contributor corrected a byline: %d", code)
	}
	if code := stack.correctByline(t, admin, draft.ID, "nobody.at.all").Code; code != http.StatusNotFound {
		t.Errorf("a byline named an account that does not exist: %d", code)
	}

	response := stack.correctByline(t, admin, draft.ID, "the.colleague")
	if response.Code != http.StatusOK {
		t.Fatalf("correct byline status = %d: %s", response.Code, response.Body.String())
	}
	corrected := decodePost(t, response)
	if corrected.Byline == nil || corrected.Byline.Handle != "the.colleague" {
		t.Fatalf("corrected byline = %+v", corrected.Byline)
	}
	if corrected.Author.Handle != "writer.dev" {
		t.Errorf("the correction changed the author to %q", corrected.Author.Handle)
	}
	if corrected.GrantID != live.GrantID {
		t.Errorf("the correction changed the grant to %q", corrected.GrantID)
	}

	found := stack.reader(t, live.Slug)
	if found.Byline.Handle != "the.colleague" || found.Byline.DisplayName != "The Colleague" {
		t.Errorf("public byline = %+v", found.Byline)
	}
	if found.Byline.App == nil || found.Byline.App.ID != live.App.ID {
		t.Errorf("the correction dropped the app attribution: %+v", found.Byline.App)
	}

	var revisions int
	err := stack.pool.QueryRow(context.Background(),
		`select count(*) from post_revisions where post_id = $1`, draft.ID).Scan(&revisions)
	if err != nil {
		t.Fatalf("count revisions: %v", err)
	}
	if revisions != 1 {
		t.Errorf("the correction left %d revisions", revisions)
	}

	var subject string
	err = stack.pool.QueryRow(context.Background(), `
		select subject_id::text from publication_audits
		 where post_id = $1 and action = 'post.byline.corrected'
	`, draft.ID).Scan(&subject)
	if err != nil {
		t.Fatalf("read the byline audit: %v", err)
	}
	if subject == "" {
		t.Error("the audit does not name who the byline now credits")
	}
}

func TestABylineIsNeverCorrectedBeforeAPostIsPublished(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	draft := stack.illarinDraft(t, session, "Still private")

	if code := stack.correctByline(t, session, draft.ID, "illarin.editor").Code; code != http.StatusBadRequest {
		t.Errorf("an unpublished post carried a byline: %d", code)
	}
	if draft.Byline != nil {
		t.Errorf("a draft carries the byline %+v", draft.Byline)
	}
}

func TestOrdinaryPublicationNeverLeavesABylineWithoutAnAccount(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	live := stack.livePost(t, session, "Somebody wrote this")

	var accounts int
	err := stack.pool.QueryRow(context.Background(),
		`select count(*) from post_bylines where post_id = $1 and account_id is not null`,
		live.ID).Scan(&accounts)
	if err != nil {
		t.Fatalf("read the byline account: %v", err)
	}
	if accounts != 1 {
		t.Errorf("publication left %d account-backed bylines", accounts)
	}
}

func TestNeitherAProfileRestrictionNorARevokedGrantRewritesAByline(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.contributor(t, "writer@example.com", "writer.dev")
	saveProfile(t, stack.router, writer.session, `{"displayName":"The Writer","links":[]}`)
	announcement := stack.categoryBySlug(t, "announcement")
	draft := stack.started(t, writer.session, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"History stands"}`, writer.grant.ID, announcement.ID,
	))
	stack.saved(t, writer.session, draft.ID, finished(draft, nil))
	live := stack.published(t, writer.session, draft.ID)

	admin := stack.admin(t, "admin@example.com", "the.admin")
	restrict := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodPut, "/v1/profiles/writer.dev/restriction", `{"reason":"checking something"}`,
	), admin))
	if restrict.Code != http.StatusOK && restrict.Code != http.StatusNoContent {
		t.Fatalf("restrict status = %d: %s", restrict.Code, restrict.Body.String())
	}
	revoke := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodDelete, "/v1/publication/grants/"+writer.grant.ID, "",
	), stack.authority))
	if revoke.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d: %s", revoke.Code, revoke.Body.String())
	}

	found := stack.reader(t, live.Slug)
	if found.Byline.DisplayName != "The Writer" {
		t.Errorf("byline name = %q after a restriction", found.Byline.DisplayName)
	}
	if found.Byline.App == nil || found.Byline.App.ID != live.App.ID {
		t.Errorf("byline app = %+v after a revocation", found.Byline.App)
	}
	if !found.PublishedAt.Equal(*live.PublishedAt) {
		t.Errorf("the publication date moved to %v", found.PublishedAt)
	}

	var revisions int
	err := stack.pool.QueryRow(context.Background(),
		`select count(*) from post_revisions where post_id = $1`, draft.ID).Scan(&revisions)
	if err != nil {
		t.Fatalf("count revisions: %v", err)
	}
	if revisions != 1 {
		t.Errorf("revocation left %d revisions", revisions)
	}
}

func TestACorrectedAddressNeverFormsARedirectChain(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	live := stack.livePost(t, session, "One hop only")
	stack.addressCorrected(t, session, live.ID, "second-address")
	stack.addressCorrected(t, session, live.ID, "third-address")

	rows, err := stack.pool.Query(context.Background(), `
		select reservation.slug, post.slug
		  from post_slugs reservation
		  join posts post on post.id = reservation.post_id
		 where reservation.post_id = $1
	`, live.ID)
	if err != nil {
		t.Fatalf("read the reservations: %v", err)
	}
	defer rows.Close()
	held := 0
	for rows.Next() {
		var reserved, current string
		if err := rows.Scan(&reserved, &current); err != nil {
			t.Fatalf("read a reservation: %v", err)
		}
		if current != "third-address" {
			t.Errorf("%q points at %q rather than the current address", reserved, current)
		}
		held++
	}
	if held != 3 {
		t.Errorf("three addresses left %d reservations", held)
	}
}

func TestACorrectedAddressIsRefusedWhenItIsAlreadyTheCurrentOne(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	live := stack.livePost(t, session, "Stay where you are")

	response := stack.correctAddress(t, session, live.ID, live.Slug)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("correcting to the same address returned %d", response.Code)
	}
	var refusal struct {
		Field string `json:"field"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &refusal); err != nil {
		t.Fatalf("decode the refusal: %v", err)
	}
	if refusal.Field != "slug" {
		t.Errorf("the refusal names %q", refusal.Field)
	}
}
