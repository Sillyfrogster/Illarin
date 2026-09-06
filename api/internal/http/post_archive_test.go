package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type postSummary struct {
	ID             string              `json:"id"`
	Slug           string              `json:"slug"`
	OriginalSlug   string              `json:"originalSlug"`
	Title          string              `json:"title"`
	Summary        string              `json:"summary"`
	Category       publicationCategory `json:"category"`
	App            *publicationApp     `json:"app"`
	ReleaseVersion string              `json:"releaseVersion"`
	Byline         postByline          `json:"byline"`
	PublishedAt    time.Time           `json:"publishedAt"`
	UpdatedAt      *time.Time          `json:"updatedAt"`
}

type postArchive struct {
	Posts    []postSummary        `json:"posts"`
	Page     int                  `json:"page"`
	Pages    int                  `json:"pages"`
	Total    int                  `json:"total"`
	Category *publicationCategory `json:"category"`
	App      *publicationApp      `json:"app"`
}

func (s distinctionStack) browse(t *testing.T, query string) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, httptest.NewRequest(http.MethodGet, "/v1/posts"+query, nil))
}

func (s distinctionStack) readableCategories(t *testing.T) []publicationCategory {
	t.Helper()
	response := send(t, s.router,
		httptest.NewRequest(http.MethodGet, "/v1/post-categories", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("read the blog categories status = %d: %s", response.Code, response.Body.String())
	}
	var found publicationCategoryList
	if err := json.Unmarshal(response.Body.Bytes(), &found); err != nil {
		t.Fatalf("decode the blog categories: %v", err)
	}
	return found.Categories
}

func (s distinctionStack) archive(t *testing.T, query string) postArchive {
	t.Helper()
	response := s.browse(t, query)
	if response.Code != http.StatusOK {
		t.Fatalf("read the archive %q status = %d: %s", query, response.Code, response.Body.String())
	}
	var found postArchive
	if err := json.Unmarshal(response.Body.Bytes(), &found); err != nil {
		t.Fatalf("decode the archive: %v", err)
	}
	return found
}

// dated moves a published post back in time so a test can write a real chronology.
func (s distinctionStack) dated(t *testing.T, id string, at time.Time) {
	t.Helper()
	_, err := s.pool.Exec(context.Background(), `
		update posts set published_at = $2 where id = $1
	`, id, at)
	if err != nil {
		t.Fatalf("date the post: %v", err)
	}
}

// revisedOn moves a published post's last public change to the given day.
func (s distinctionStack) revisedOn(t *testing.T, id string, at time.Time) {
	t.Helper()
	_, err := s.pool.Exec(context.Background(), `
		update posts set updated_public_at = $2 where id = $1
	`, id, at)
	if err != nil {
		t.Fatalf("revise the post: %v", err)
	}
}

// publishedOn writes one finished Illarin post and puts it in public view on the given day.
func (s distinctionStack) publishedOn(
	t *testing.T,
	session *http.Cookie,
	title string,
	at time.Time,
) blogPost {
	t.Helper()
	draft := s.illarinDraft(t, session, title)
	written := s.saved(t, session, draft.ID, finished(draft, nil))
	live := s.publishedAt(t, session, written.ID, written.Version)
	s.dated(t, live.ID, at)
	return live
}

func titlesOf(found postArchive) []string {
	titles := make([]string, 0, len(found.Posts))
	for _, one := range found.Posts {
		titles = append(titles, one.Title)
	}
	return titles
}

func TestTheArchiveLeadsWithTheNewestPostAndCarriesTwelveToAPage(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	day := time.Date(2026, time.March, 1, 9, 0, 0, 0, time.UTC)

	for number := 1; number <= 14; number++ {
		stack.publishedOn(t, session,
			fmt.Sprintf("Release note %d", number), day.AddDate(0, 0, number))
	}

	first := stack.archive(t, "")
	if first.Total != 14 || first.Pages != 2 || first.Page != 1 {
		t.Fatalf("first page = %d of %d pages holding %d posts",
			first.Page, first.Pages, first.Total)
	}
	if len(first.Posts) != 12 {
		t.Fatalf("first page holds %d posts, want 12: %v", len(first.Posts), titlesOf(first))
	}
	if first.Posts[0].Title != "Release note 14" {
		t.Errorf("the lead post is %q, want the newest", first.Posts[0].Title)
	}
	for index := 1; index < len(first.Posts); index++ {
		if first.Posts[index].PublishedAt.After(first.Posts[index-1].PublishedAt) {
			t.Fatalf("the archive runs out of order: %v", titlesOf(first))
		}
	}

	second := stack.archive(t, "?page=2")
	if len(second.Posts) != 2 || second.Page != 2 {
		t.Fatalf("second page = %d holding %v", second.Page, titlesOf(second))
	}
	if second.Posts[0].Title != "Release note 2" || second.Posts[1].Title != "Release note 1" {
		t.Errorf("the second page holds %v, want the two oldest", titlesOf(second))
	}

	past := stack.archive(t, "?page=3")
	if len(past.Posts) != 0 || past.Pages != 2 {
		t.Errorf("a page past the end holds %v", titlesOf(past))
	}
}

func TestTheArchiveCarriesWhatAnEntryShowsWithoutReadingALiveProfile(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	illarin := stack.appBySlug(t, "illarin")
	release := stack.categoryBySlug(t, "release")

	draft := stack.illarinDraft(t, session, "The catalog reads faster")
	written := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"categoryId": release.ID,
		"summary":    "Browse answers in half the time it did.",
		"release":    map[string]any{"appId": illarin.ID, "version": "1.4"},
	}))
	stack.publishedAt(t, session, written.ID, written.Version)

	entry := stack.archive(t, "").Posts[0]
	if entry.Summary != "Browse answers in half the time it did." {
		t.Errorf("entry summary = %q, want the hand-written one", entry.Summary)
	}
	if entry.Byline.Handle != "illarin.editor" {
		t.Errorf("entry byline = %+v, want the stored one", entry.Byline)
	}
	if entry.Category.Slug != "release" {
		t.Errorf("entry category = %q", entry.Category.Slug)
	}
	if entry.App == nil || entry.App.Slug != "illarin" {
		t.Errorf("entry app = %+v, want the release app", entry.App)
	}
	if entry.ReleaseVersion != "1.4" {
		t.Errorf("entry version = %q", entry.ReleaseVersion)
	}
	if entry.PublishedAt.IsZero() || entry.UpdatedAt != nil {
		t.Errorf("entry dates = %v / %v", entry.PublishedAt, entry.UpdatedAt)
	}
}

func TestTheArchiveNarrowsToOneCategoryAndOneApp(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	lumiverse := stack.configureApp(t, "lumiverse", "Lumiverse", "https://lumiverse.example")
	release := stack.categoryBySlug(t, "release")
	day := time.Date(2026, time.April, 1, 9, 0, 0, 0, time.UTC)

	announcement := stack.publishedOn(t, session, "Illarin opens the blog", day)

	releaseDraft := stack.illarinDraft(t, session, "Lumiverse 2.0 is out")
	releaseWritten := stack.saved(t, session, releaseDraft.ID, finished(releaseDraft, map[string]any{
		"categoryId": release.ID,
		"release":    map[string]any{"appId": lumiverse.ID, "version": "2.0"},
	}))
	releaseLive := stack.publishedAt(t, session, releaseWritten.ID, releaseWritten.Version)
	stack.dated(t, releaseLive.ID, day.AddDate(0, 0, 1))

	announcementCategory := stack.categoryBySlug(t, "announcement")
	contributor := stack.member(t, "dev@example.com", "lumiverse.dev")
	grant := stack.approved(t, "lumiverse.dev", lumiverse.ID,
		[]string{announcementCategory.ID}, announcementCategory.ID)
	filed := stack.started(t, contributor, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"Lumiverse says hello"}`,
		grant.ID, announcementCategory.ID,
	))
	filedWritten := stack.saved(t, contributor, filed.ID, finished(filed, nil))
	filedLive := stack.publishedAt(t, contributor, filedWritten.ID, filedWritten.Version)
	stack.dated(t, filedLive.ID, day.AddDate(0, 0, -1))

	byCategory := stack.archive(t, "?category=release")
	if len(byCategory.Posts) != 1 || byCategory.Posts[0].Title != "Lumiverse 2.0 is out" {
		t.Fatalf("the release archive holds %v", titlesOf(byCategory))
	}
	if byCategory.Category == nil || byCategory.Category.Slug != "release" {
		t.Errorf("the release archive names the scope %+v", byCategory.Category)
	}
	if byCategory.App != nil {
		t.Errorf("a category archive names an app scope %+v", byCategory.App)
	}

	byApp := stack.archive(t, "?app=lumiverse")
	if len(byApp.Posts) != 2 {
		t.Fatalf("the Lumiverse archive holds %v", titlesOf(byApp))
	}
	if byApp.Posts[0].Title != "Lumiverse 2.0 is out" ||
		byApp.Posts[1].Title != "Lumiverse says hello" {
		t.Errorf("the Lumiverse archive holds %v", titlesOf(byApp))
	}
	if byApp.App == nil || byApp.App.Name != "Lumiverse" {
		t.Errorf("the app archive names the scope %+v", byApp.App)
	}

	if whole := stack.archive(t, ""); len(whole.Posts) != 3 {
		t.Errorf("the whole archive holds %v", titlesOf(whole))
	}
	if announcement.Slug == "" {
		t.Error("the announcement never took an address")
	}
}

func TestAnUnknownArchiveScopeIsNotFound(t *testing.T) {
	stack := newDistinctionStack(t)

	if got := stack.browse(t, "?category=musings"); got.Code != http.StatusNotFound {
		t.Errorf("an unknown category status = %d, want 404", got.Code)
	}
	if got := stack.browse(t, "?app=nowhere"); got.Code != http.StatusNotFound {
		t.Errorf("an unknown app status = %d, want 404", got.Code)
	}
	if got := stack.browse(t, "?page=0"); got.Code != http.StatusBadRequest {
		t.Errorf("page zero status = %d, want 400", got.Code)
	}
}

func TestTheArchiveHoldsOnlyPostsAReaderCanAlreadyOpen(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

	live := stack.publishedOn(t, session, "The one public post",
		time.Date(2026, time.May, 1, 9, 0, 0, 0, time.UTC))
	stack.illarinDraft(t, session, "A draft nobody has seen")
	stack.scheduledDraft(t, session, "An edition held for later", "It goes live next hour.")

	found := stack.archive(t, "")
	if len(found.Posts) != 1 || found.Posts[0].Slug != live.Slug {
		t.Fatalf("the archive holds %v", titlesOf(found))
	}

	stack.saved(t, session, live.ID, finished(live, map[string]any{
		"version": stack.working(t, session, live.ID).Version,
		"title":   "A title only the working copy carries",
	}))
	after := stack.archive(t, "")
	if after.Posts[0].Title != "The one public post" {
		t.Errorf("the archive shows the working copy title %q", after.Posts[0].Title)
	}
}

func TestAnArticleOffersThreeOtherPostsPreferringItsAppThenItsCategory(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	lumiverse := stack.configureApp(t, "lumiverse", "Lumiverse", "https://lumiverse.example")
	release := stack.categoryBySlug(t, "release")
	article := stack.categoryBySlug(t, "article")
	day := time.Date(2026, time.June, 1, 9, 0, 0, 0, time.UTC)

	releaseOn := func(title string, on time.Time) blogPost {
		draft := stack.illarinDraft(t, session, title)
		written := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
			"categoryId": release.ID,
			"release":    map[string]any{"appId": lumiverse.ID, "version": "3.0"},
		}))
		live := stack.publishedAt(t, session, written.ID, written.Version)
		stack.dated(t, live.ID, on)
		return live
	}
	articleOn := func(title string, on time.Time) blogPost {
		draft := stack.illarinDraft(t, session, title)
		written := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
			"categoryId": article.ID,
		}))
		live := stack.publishedAt(t, session, written.ID, written.Version)
		stack.dated(t, live.ID, on)
		return live
	}

	reading := releaseOn("Lumiverse 3.0 is out", day)
	releaseOn("Lumiverse 2.9 is out", day.AddDate(0, 0, -1))
	articleOn("Writing a lorebook that holds", day.AddDate(0, 0, -2))
	articleOn("How Illarin stores a theme", day.AddDate(0, 0, -3))
	stack.publishedOn(t, session, "Illarin opens the blog", day.AddDate(0, 0, -4))

	found := stack.reader(t, reading.Slug)
	if len(found.Related) != 3 {
		t.Fatalf("the article offers %d other posts, want 3", len(found.Related))
	}
	if found.Related[0].Title != "Lumiverse 2.9 is out" {
		t.Errorf("the first related post is %q, want the same app", found.Related[0].Title)
	}
	if found.Related[1].Title != "Writing a lorebook that holds" {
		t.Errorf("the second related post is %q", found.Related[1].Title)
	}
	for _, one := range found.Related {
		if one.ID == found.ID {
			t.Error("an article offers itself as further reading")
		}
	}

	alone := stack.reader(t, "illarin-opens-the-blog")
	for _, one := range alone.Related {
		if one.Slug == "illarin-opens-the-blog" {
			t.Error("an announcement offers itself as further reading")
		}
	}
	if len(alone.Related) != 3 {
		t.Errorf("a post with no app or category match offers %d posts", len(alone.Related))
	}
}

func TestTheBlogOffersOnlyCategoriesThatCarryWriting(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")

	if len(stack.readableCategories(t)) != 0 {
		t.Fatalf("an empty publication offers %v", stack.readableCategories(t))
	}

	stack.publishedOn(t, session, "Illarin opens the blog",
		time.Date(2026, time.July, 1, 9, 0, 0, 0, time.UTC))
	stack.illarinDraft(t, session, "A draft nobody has seen")

	offered := stack.readableCategories(t)
	if len(offered) != 1 || offered[0].Slug != "announcement" {
		t.Errorf("the blog offers %v, want announcement alone", offered)
	}
}

func TestAPostCorrectedInPublicRetakesTheLead(t *testing.T) {
	stack := newDistinctionStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	day := time.Date(2026, time.May, 1, 9, 0, 0, 0, time.UTC)

	corrected := stack.publishedOn(t, session, "The older post", day)
	stack.publishedOn(t, session, "The newer post", day.AddDate(0, 0, 3))

	if first := stack.archive(t, ""); first.Posts[0].Title != "The newer post" {
		t.Fatalf("the archive leads with %v", titlesOf(first))
	}

	stack.revisedOn(t, corrected.ID, day.AddDate(0, 0, 5))

	after := stack.archive(t, "")
	if after.Posts[0].Title != "The older post" {
		t.Errorf("a corrected post did not retake the lead: %v", titlesOf(after))
	}
	if after.Posts[0].UpdatedAt == nil {
		t.Error("the corrected post carries no updated date")
	}
	if !after.Posts[0].PublishedAt.Equal(day) {
		t.Errorf("a correction moved the published date to %v", after.Posts[0].PublishedAt)
	}
}
