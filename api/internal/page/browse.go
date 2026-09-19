package page

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

type ListFilter struct {
	Type    string
	Profile *ProfileListingScope
	App     *string
	AppSet  bool
	Tags    []string
	Facets  []FacetSelection
	Query   string
	Limit   int
	Before  *Cursor
}

type ProfileListingScope struct {
	CreatorID uuid.UUID
	ViewerID  *uuid.UUID
}

type Cursor struct {
	MadeAt time.Time
	ID     uuid.UUID
}

type Cover struct {
	URL    string
	Width  int
	Height int
}

// Withhold is what the owner of a withheld work reads, which never names the staff member who acted
type Withhold struct {
	Reason string
	At     time.Time
}

type BrowseItem struct {
	ID         uuid.UUID
	Name       string
	Creator    string
	Type       string
	IsNSFW     *bool
	OwnerState string
	Cover      *Cover
	Withhold   *Withhold
}

type BrowsePage struct {
	Items      []BrowseItem
	Total      int
	Suppressed int
	Next       *Cursor
	EmptyState string
	Apps       []Option
	Facets     []Filter
}

type Option struct {
	Value    string
	Label    string
	Count    int
	Selected bool
}

type Filter struct {
	Key     string
	Label   string
	Options []Option
}

type FacetSelection struct {
	Key   string
	Value string
}

func (s *Service) Browse(
	ctx context.Context,
	f ListFilter,
	preference work.NSFWPreference,
) (BrowsePage, error) {
	if f.Limit <= 0 || f.Limit > 24 {
		f.Limit = 24
	}
	if preference != work.NSFWHidden && preference != work.NSFWShown {
		preference = work.NSFWBlurred
	}
	tx, err := s.works.BeginReadSnapshot(ctx)
	if err != nil {
		return BrowsePage{}, err
	}
	defer tx.Rollback(ctx)
	queries := db.New(tx)
	search := parseBrowseQuery(f.Query)
	facetDefinitions := block.Facets(f.Type)
	chosen := declaredFacetSelections(f.Type, f.Facets)
	app := ""
	if f.App != nil {
		app = normalizeBrowseText(*f.App)
	}
	formats := formatsForApp(app)
	facetKeys, facetLows, facetHighs := facetRanges(chosen)
	creatorID, ownProfile := profileListingValues(f.Profile)
	params := db.BrowseWorksParams{
		Type: f.Type, NsfwPreference: string(preference),
		CreatorID:  uuidToNullable(creatorID),
		OwnProfile: ownProfile,
		SearchText: search.Text, Author: search.Author, Tags: search.Tags,
		App: app, Formats: formats,
		FacetKeys: facetKeys, FacetLows: facetLows, FacetHighs: facetHighs,
		PageSize: int32(f.Limit + 1),
	}
	if f.Before != nil {
		params.Before = timeToNullable(&f.Before.MadeAt)
		params.BeforeID = uuidToPgtype(f.Before.ID)
	}
	rows, err := queries.BrowseWorks(ctx, params)
	if err != nil {
		return BrowsePage{}, fmt.Errorf("browse works: %w", err)
	}

	page := BrowsePage{Items: make([]BrowseItem, 0, min(len(rows), f.Limit))}
	for _, row := range rows[:min(len(rows), f.Limit)] {
		draft := work.Lifecycle(row.Lifecycle) == work.LifecycleDraft
		item := BrowseItem{
			ID: uuidFromPgtype(row.ID), Name: row.Name, Creator: row.Creator,
			Type: row.Type, IsNSFW: boolFromPgtype(row.IsNsfw),
		}
		if ownProfile {
			switch {
			case draft:
				item.OwnerState = "draft"
			case row.WithheldAt.Valid:
				item.OwnerState = "withheld"
				item.Withhold = &Withhold{
					Reason: row.WithheldReason.String,
					At:     row.WithheldAt.Time,
				}
			case row.Visibility == "unlisted":
				item.OwnerState = "unlisted"
			}
		}
		if row.CoverID.Valid && row.CoverWidth.Valid && row.CoverHeight.Valid {
			flagged := item.IsNSFW != nil && *item.IsNSFW
			item.Cover = &Cover{
				URL: s.works.ImageAddress(
					uuidFromPgtype(row.CoverID), "grid",
					preference != work.NSFWShown && flagged, draft,
				),
				Width: int(row.CoverWidth.Int32), Height: int(row.CoverHeight.Int32),
			}
		}
		page.Items = append(page.Items, item)
	}
	if len(rows) > f.Limit && len(page.Items) > 0 {
		last := rows[f.Limit-1]
		page.Next = &Cursor{MadeAt: timeFromPgtype(last.CreatedAt), ID: uuidFromPgtype(last.ID)}
	}

	countParams := db.CountBrowseWorksParams{
		Type: f.Type, NsfwPreference: string(preference),
		CreatorID:  uuidToNullable(creatorID),
		OwnProfile: ownProfile,
		SearchText: search.Text, Author: search.Author, Tags: search.Tags,
		App: app, Formats: formats,
		FacetKeys: facetKeys, FacetLows: facetLows, FacetHighs: facetHighs,
	}
	count, err := queries.CountBrowseWorks(ctx, countParams)
	if err != nil {
		return BrowsePage{}, fmt.Errorf("count browse works: %w", err)
	}
	page.Total = int(count)
	if preference == work.NSFWHidden && !ownProfile {
		suppressed, err := queries.CountSuppressedBrowseWorks(
			ctx, db.CountSuppressedBrowseWorksParams{
				Type: f.Type, SearchText: search.Text, Author: search.Author, Tags: search.Tags,
				CreatorID: uuidToNullable(creatorID),
				App:       app, Formats: formats,
				FacetKeys: facetKeys, FacetLows: facetLows, FacetHighs: facetHighs,
			},
		)
		if err != nil {
			return BrowsePage{}, fmt.Errorf("count suppressed browse works: %w", err)
		}
		page.Suppressed = int(suppressed)
	}
	page.Apps, err = countedApps(ctx, queries, countParams, app)
	if err != nil {
		return BrowsePage{}, err
	}
	page.Facets, err = countedFacets(ctx, queries, countParams, chosen, facetDefinitions)
	if err != nil {
		return BrowsePage{}, err
	}
	if page.Total == 0 {
		if page.Suppressed > 0 {
			page.EmptyState = "suppressed"
		} else if f.Type == "" && search.Text == "" && search.Author == "" &&
			len(search.Tags) == 0 && f.App == nil && len(chosen) == 0 {
			page.EmptyState = "nothing_published"
		} else {
			page.EmptyState = "no_matches"
		}
	}
	return page, nil
}

func profileListingValues(scope *ProfileListingScope) (*uuid.UUID, bool) {
	if scope == nil {
		return nil, false
	}
	ownedByViewer := scope.ViewerID != nil && *scope.ViewerID == scope.CreatorID
	return &scope.CreatorID, ownedByViewer
}

func formatsForApp(chosen string) []string {
	for _, app := range format.Apps() {
		if normalizeBrowseText(app.ID) == chosen {
			return app.Reads
		}
	}
	return nil
}

type chosenFacet struct {
	FacetSelection
	bucket block.Bucket
}

func declaredFacetSelections(workType string, requested []FacetSelection) []chosenFacet {
	chosen := make([]chosenFacet, 0, len(requested))
	for _, request := range requested {
		facet, known := block.FacetByKey(workType, request.Key)
		if !known {
			continue
		}
		bucket, offered := facet.Bucket(request.Value)
		if !offered {
			continue
		}
		if slices.ContainsFunc(chosen, func(already chosenFacet) bool {
			return already.FacetSelection == request
		}) {
			continue
		}
		chosen = append(chosen, chosenFacet{FacetSelection: request, bucket: bucket})
	}
	return chosen
}

func facetRanges(chosen []chosenFacet) (keys []string, lows, highs []int32) {
	keys = make([]string, 0, len(chosen))
	lows = make([]int32, 0, len(chosen))
	highs = make([]int32, 0, len(chosen))
	for _, one := range chosen {
		keys = append(keys, one.Key)
		lows = append(lows, int32(one.bucket.Min))
		highs = append(highs, int32(one.bucket.Max))
	}
	return keys, lows, highs
}

func countedApps(
	ctx context.Context,
	queries *db.Queries,
	base db.CountBrowseWorksParams,
	selected string,
) ([]Option, error) {
	apps := format.Apps()
	result := make([]Option, 0, len(apps))
	for _, app := range apps {
		params := base
		params.App = app.ID
		params.Formats = app.Reads
		count, err := queries.CountBrowseWorks(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("count app %s: %w", app.ID, err)
		}
		result = append(result, Option{
			Value: app.ID, Label: app.Label, Count: int(count),
			Selected: app.ID == selected,
		})
	}
	return result, nil
}

func countedFacets(
	ctx context.Context,
	queries *db.Queries,
	base db.CountBrowseWorksParams,
	chosen []chosenFacet,
	definitions []block.Facet,
) ([]Filter, error) {
	result := make([]Filter, 0, len(definitions))
	for _, definition := range definitions {
		group := Filter{Key: string(definition.Key), Label: definition.Label}
		for _, bucket := range definition.Buckets {
			choice := chosenFacet{
				FacetSelection: FacetSelection{
					Key: string(definition.Key), Value: bucket.Value,
				},
				bucket: bucket,
			}
			picked := slices.Contains(chosen, choice)
			candidate := slices.Clone(chosen)
			if !picked {
				candidate = append(candidate, choice)
			}
			keys, lows, highs := facetRanges(candidate)
			params := base
			params.FacetKeys, params.FacetLows, params.FacetHighs = keys, lows, highs
			count, err := queries.CountBrowseWorks(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("count facet %s=%s: %w", definition.Key, bucket.Value, err)
			}
			group.Options = append(group.Options, Option{
				Value: bucket.Value, Label: bucket.Label, Count: int(count),
				Selected: picked,
			})
		}
		result = append(result, group)
	}
	return result, nil
}

type browseQuery struct {
	Text   string
	Author string
	Tags   []string
}

func parseBrowseQuery(raw string) browseQuery {
	var parsed browseQuery
	var text []string
	for _, token := range browseTokens(raw) {
		lower := normalizeBrowseText(token)
		switch {
		case strings.HasPrefix(lower, "tag:") && len(lower) > len("tag:"):
			parsed.Tags = append(parsed.Tags, normalizeBrowseText(token[len("tag:"):]))
		case strings.HasPrefix(lower, "author:") && len(lower) > len("author:"):
			if parsed.Author == "" {
				parsed.Author = normalizeBrowseText(token[len("author:"):])
			}
		default:
			text = append(text, lower)
		}
	}
	parsed.Text = strings.Join(text, " ")
	return parsed
}

func browseTokens(raw string) []string {
	var tokens []string
	var token strings.Builder
	quoted := false
	for _, char := range raw {
		switch {
		case char == '"':
			quoted = !quoted
		case !quoted && (char == ' ' || char == '\t' || char == '\n' || char == '\r'):
			if token.Len() > 0 {
				tokens = append(tokens, token.String())
				token.Reset()
			}
		default:
			token.WriteRune(char)
		}
	}
	if token.Len() > 0 {
		tokens = append(tokens, token.String())
	}
	return tokens
}

func normalizeBrowseText(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
