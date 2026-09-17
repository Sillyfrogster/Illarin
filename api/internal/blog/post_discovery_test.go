package blog_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func (s publicationStack) readableApps(t *testing.T) []publicationApp {
	t.Helper()
	response := apitest.Send(t, s.router,
		httptest.NewRequest(http.MethodGet, "/v1/post-apps", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("read the blog apps status = %d: %s", response.Code, response.Body.String())
	}
	var found publicationAppList
	if err := json.Unmarshal(response.Body.Bytes(), &found); err != nil {
		t.Fatalf("decode the blog apps: %v", err)
	}
	return found.Apps
}

func appSlugsOf(found []publicationApp) []string {
	slugs := make([]string, 0, len(found))
	for _, one := range found {
		slugs = append(slugs, one.Slug)
	}
	return slugs
}

func TestOnlyAnAppWithPublishedWritingIsReadable(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	lumiverse := stack.configureApp(t, "lumiverse", "Lumiverse", "https://lumiverse.example")
	stack.configureApp(t, "sillytavern", "SillyTavern", "https://sillytavern.example")
	release := stack.categoryBySlug(t, "release")

	if slugs := appSlugsOf(stack.readableApps(t)); len(slugs) != 0 {
		t.Fatalf("an empty publication offers the apps %v", slugs)
	}

	draft := stack.illarinDraft(t, session, "Lumiverse 2.0 is out")
	written := stack.saved(t, session, draft.ID, finished(draft, map[string]any{
		"categoryId": release.ID,
		"release":    map[string]any{"appId": lumiverse.ID, "version": "2.0"},
	}))
	stack.publishedAt(t, session, written.ID, written.Version)

	slugs := appSlugsOf(stack.readableApps(t))
	if len(slugs) != 1 || slugs[0] != "lumiverse" {
		t.Errorf("the readable apps are %v, want only the one that has published", slugs)
	}
}

func TestAPostKeepsTheAddressItFirstPublishedUnder(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	session := stack.admin(t, "editor@example.com", "illarin.editor")
	day := time.Date(2026, time.May, 4, 9, 0, 0, 0, time.UTC)
	live := stack.publishedOn(t, session, "The first address", day)

	stack.addressCorrected(t, session, live.ID, "a-corrected-address")
	stack.addressCorrected(t, session, live.ID, "a-second-correction")

	post := stack.reader(t, "a-second-correction")
	if post.Slug != "a-second-correction" {
		t.Errorf("the post answers at %q, want its current address", post.Slug)
	}
	if post.OriginalSlug != live.Slug {
		t.Errorf("the post's first address is %q, want %q", post.OriginalSlug, live.Slug)
	}

	entry := stack.archive(t, "").Posts[0]
	if entry.OriginalSlug != live.Slug {
		t.Errorf("the archive entry's first address is %q, want %q",
			entry.OriginalSlug, live.Slug)
	}
}
