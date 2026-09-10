package migration

import (
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

type LegacyPath struct {
	Path    string
	AssetID uuid.UUID
}

type legacyCandidate struct {
	AssetID   uuid.UUID
	Author    string
	Handle    string
	Name      string
	CreatedAt time.Time
}

func (candidate legacyCandidate) address() string {
	head := legacySlug(candidate.Author)
	if head == "" {
		head = legacySlug(candidate.Handle)
	}
	tail := legacySlug(candidate.Name)
	if head == "" || tail == "" {
		return ""
	}
	return head + "/" + tail
}

func legacySlug(text string) string {
	var built strings.Builder
	separated := false
	for _, letter := range strings.ToLower(text) {
		if letter < unicode.MaxASCII && (unicode.IsLetter(letter) || unicode.IsDigit(letter)) {
			if separated {
				built.WriteByte('-')
			}
			separated = false
			built.WriteRune(letter)
			continue
		}
		separated = built.Len() > 0
	}
	return built.String()
}

func resolveLegacyPaths(candidates []legacyCandidate) ([]LegacyPath, []legacyCandidate) {
	ordered := append([]legacyCandidate(nil), candidates...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if !ordered[i].CreatedAt.Equal(ordered[j].CreatedAt) {
			return ordered[i].CreatedAt.Before(ordered[j].CreatedAt)
		}
		return ordered[i].AssetID.String() < ordered[j].AssetID.String()
	})
	held := make(map[string]struct{}, len(ordered))
	paths := make([]LegacyPath, 0, len(ordered))
	displaced := make([]legacyCandidate, 0)
	for _, candidate := range ordered {
		address := candidate.address()
		if address == "" {
			continue
		}
		if _, taken := held[address]; taken {
			displaced = append(displaced, candidate)
			continue
		}
		held[address] = struct{}{}
		paths = append(paths, LegacyPath{Path: address, AssetID: candidate.AssetID})
	}
	return paths, displaced
}
