package publication

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

const postSlugLimit = 80

// firstAddress reads the address a post first published under. A correction
// leaves that address still reaching the post, so it never changes.
const firstAddress = `coalesce((
	                 select earliest.slug from post_slugs earliest
	                  where earliest.post_id = post.id
	                  order by earliest.reserved_at, earliest.slug
	                  limit 1
	               ), post.slug)`

// reservedSlugs are the blog's own route prefixes and feed file names, written in the form normalization leaves them in.
var reservedSlugs = map[string]bool{
	"admin":     true,
	"api":       true,
	"app":       true,
	"apps":      true,
	"archive":   true,
	"atom":      true,
	"blog":      true,
	"category":  true,
	"feed":      true,
	"feed-json": true,
	"feed-xml":  true,
	"feeds":     true,
	"page":      true,
	"preview":   true,
	"robots":    true,
	"rss":       true,
	"sitemap":   true,
	"tag":       true,
	"tags":      true,
}

// normalizeSlug turns whatever an author typed into the address form, which is lowercase words joined by single hyphens.
func normalizeSlug(candidate string) string {
	var out strings.Builder
	previousHyphen := true
	for _, letter := range strings.ToLower(strings.TrimSpace(candidate)) {
		switch {
		case unicode.IsLetter(letter) && letter < unicode.MaxASCII, unicode.IsDigit(letter) && letter < unicode.MaxASCII:
			out.WriteRune(letter)
			previousHyphen = false
		case previousHyphen:
			continue
		default:
			out.WriteRune('-')
			previousHyphen = true
		}
		if out.Len() >= postSlugLimit {
			break
		}
	}
	return strings.Trim(out.String(), "-")
}

func checkSlug(candidate string) (string, error) {
	slug := normalizeSlug(candidate)
	if slug == "" {
		return "", FieldError{Field: "slug", Message: "Give the post an address."}
	}
	if reservedSlugs[slug] {
		return "", FieldError{Field: "slug", Message: "The blog already answers on that address."}
	}
	return slug, nil
}

// addressTaken answers whether another post carries this address or ever did, because a published address is never released.
func addressTaken(
	ctx context.Context,
	reader queryRower,
	postID uuid.UUID,
	slug string,
) (bool, error) {
	var taken bool
	err := reader.QueryRow(ctx, `
		select exists (select 1 from posts where slug = $2 and id <> $1)
		    or exists (select 1 from post_slugs where slug = $2 and post_id is distinct from $1)
	`, postID, slug).Scan(&taken)
	if err != nil {
		return false, fmt.Errorf("read who holds a post address: %w", err)
	}
	return taken, nil
}

// reserveAddress keeps an address for one post for good, so a corrected permalink still reaches the writing a reader saved it for.
func reserveAddress(
	ctx context.Context,
	writer execer,
	postID, actorID uuid.UUID,
	slug string,
) error {
	_, err := writer.Exec(ctx, `
		insert into post_slugs (slug, post_id, reserved_by) values ($1, $2, $3)
		on conflict (slug) do nothing
	`, slug, postID, actorID)
	if err != nil {
		return fmt.Errorf("reserve the post address: %w", err)
	}
	return nil
}
