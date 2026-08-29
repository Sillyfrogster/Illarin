package publication

import (
	"strings"
	"unicode"
)

const postSlugLimit = 80

// reservedSlugs are the blog's own route prefixes and file names, so no post
// can ever take an address the publication already answers on.
var reservedSlugs = map[string]bool{
	"admin":    true,
	"api":      true,
	"app":      true,
	"apps":     true,
	"blog":     true,
	"category": true,
	"feed":     true,
	"feeds":    true,
	"page":     true,
	"robots":   true,
	"sitemap":  true,
	"tag":      true,
	"tags":     true,
}

// normalizeSlug turns whatever an author typed into the address form, which is
// lowercase words joined by single hyphens.
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
