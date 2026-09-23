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

// ListFilter holds App as a registry app id, or empty for every app
type ListFilter struct {
	Type    string
	Profile *ProfileListingScope
	App     string
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

// Takedown is what the owner of a taken-down work reads, which never names the staff member who acted
type Takedown struct {
	Reason string
	At     time.Time
}

type BrowseItem struct {
	Apps       []string
	ID         uuid.UUID
	Name       string
	Creator    string
	Type       string
	IsNSFW     *bool
	OwnerState string
	Cover      *Cover
	Takedown   *Takedown
}

type BrowsePage struct {
	Items      []BrowseItem
	Total      int
	Suppressed int
	Next       *Cursor
	EmptyState string
	Apps       []Option
	Types      []Option
	AllTypes   int
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
	app := f.App
	formats := formatsForApp(app)
	hidden := []string{}
	if app == "" && f.Profile == nil && f.Type == "" {
		hidden = s.reg.OneAppTypes()
	}
	facetKeys, facetLows, facetHighs := facetRanges(chosen)
	creatorID, ownProfile := profileListingValues(f.Profile)
	params := db.BrowseWorksParams{
		Type: f.Type, NsfwPreference: string(preference),
		CreatorID:  uuidToNullable(creatorID),
		OwnProfile: ownProfile,
		SearchText: search.Text, Author: search.Author, Tags: search.Tags,
		App: app, Formats: formats, HiddenTypes: hidden,
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
		page.Items = append(page.Items, s.browseItem(row, ownProfile, preference))
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
		App: app, Formats: formats, HiddenTypes: hidden,
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
				App:       app, Formats: formats, HiddenTypes: hidden,
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
	page.Types, page.AllTypes, err = s.countedTypes(ctx, queries, countParams, f.Type, f.Profile != nil)
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
			len(search.Tags) == 0 && app == "" && len(chosen) == 0 {
			page.EmptyState = "nothing_published"
		} else {
			page.EmptyState = "no_matches"
		}
	}
	return page, nil
}

// Featured lists the works a creator chose to show first, in their order, under the reader's adult content setting
func (s *Service) Featured(ctx context.Context, creatorID uuid.UUID, preference work.NSFWPreference) ([]BrowseItem, error) {
	rows, err := db.New(s.pool).FeaturedWorks(ctx, db.FeaturedWorksParams{
		CreatorID: uuidToPgtype(creatorID), NsfwPreference: string(preference),
	})
	if err != nil {
		return nil, fmt.Errorf("read featured works: %w", err)
	}
	items := make([]BrowseItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, s.browseItem(db.BrowseWorksRow(row), false, preference))
	}
	return items, nil
}

func (s *Service) browseItem(row db.BrowseWorksRow, ownProfile bool, preference work.NSFWPreference) BrowseItem {
	draft := work.Lifecycle(row.Lifecycle) == work.LifecycleDraft
	item := BrowseItem{
		ID: uuidFromPgtype(row.ID), Name: row.Name, Creator: row.Creator,
		Type: row.Type, IsNSFW: boolFromPgtype(row.IsNsfw),
		Apps: format.AppsReading(row.Formats),
	}
	if ownProfile {
		switch {
		case draft:
			item.OwnerState = "draft"
		case row.TakenDownAt.Valid:
			item.OwnerState = "taken_down"
			item.Takedown = &Takedown{
				Reason: row.TakenDownReason.String,
				At:     row.TakenDownAt.Time,
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
	return item
}

// countedTypes counts every type in view, and all of them together, under the search and app but without facets
func (s *Service) countedTypes(
	ctx context.Context,
	queries *db.Queries,
	base db.CountBrowseWorksParams,
	asked string,
	profile bool,
) ([]Option, int, error) {
	base.FacetKeys, base.FacetLows, base.FacetHighs = nil, nil, nil
	all := base
	all.Type = ""
	if base.App == "" && !profile {
		all.HiddenTypes = s.reg.OneAppTypes()
	}
	total, err := queries.CountBrowseWorks(ctx, all)
	if err != nil {
		return nil, 0, fmt.Errorf("count every type: %w", err)
	}
	types := s.typesInView(base.App, profile)
	if asked != "" && !slices.Contains(types, asked) && slices.Contains(block.Types(), asked) {
		types = append(types, asked)
		slices.Sort(types)
	}
	result := make([]Option, 0, len(types))
	for _, workType := range types {
		params := base
		params.Type = workType
		params.HiddenTypes = nil
		count, err := queries.CountBrowseWorks(ctx, params)
		if err != nil {
			return nil, 0, fmt.Errorf("count type %s: %w", workType, err)
		}
		result = append(result, Option{Value: workType, Count: int(count), Selected: workType == asked})
	}
	return result, int(total), nil
}

// typesInView lists the types the reader can narrow to: what their app reads, or every type but those only one app reads
func (s *Service) typesInView(app string, profile bool) []string {
	for _, known := range format.Apps() {
		if known.ID == app {
			return s.reg.TypesRead(known)
		}
	}
	types := slices.Sorted(slices.Values(block.Types()))
	if profile {
		return types
	}
	return slices.DeleteFunc(types, func(workType string) bool {
		return slices.Contains(s.reg.OneAppTypes(), workType)
	})
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
		if app.ID == chosen {
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
		params.HiddenTypes = nil
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
