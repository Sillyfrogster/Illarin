package blog_test

import (
	"testing"
	"time"
)

func TestAPostKeepsTheAddressItFirstPublishedUnder(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	session := stack.siteAdmin(t, "editor@example.com", "illarin.editor")
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
