package page

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handlers) ListAssets(c *gin.Context) {
	q := api.ReadQuery(c)
	params := ListAssetsParams{
		Kind:     api.QueryText[ListAssetsParamsKind](q, "kind"),
		Platform: api.QueryText[string](q, "platform"),
		Creator:  api.QueryText[string](q, "creator"),
		Q:        api.QueryText[string](q, "q"),
		Facet:    api.QueryList(q, "facet"),
		Nsfw:     api.QueryText[ListAssetsParamsNsfw](q, "nsfw"),
		Limit:    api.QueryNumber(q, "limit"),
		Before:   api.QueryTime(q, "before"),
		BeforeId: api.QueryID(q, "beforeId"),
	}
	if q.Refused(c) {
		return
	}
	f := ListFilter{}

	if params.Creator != nil {
		creator, err := h.accounts.CreatorListing(c.Request.Context(), strings.ToLower(*params.Creator))
		if errors.Is(err, account.ErrProfileNotFound) {
			api.Refuse(c, http.StatusNotFound, "No such profile.")
			return
		}
		if err != nil {
			api.Refuse(c, http.StatusInternalServerError, "Could not read the profile.")
			return
		}
		current, err := api.Current(c)
		if err != nil {
			api.Refuse(c, http.StatusInternalServerError, "Could not read the signed-in account.")
			return
		}
		f.Profile = &ProfileListingScope{CreatorID: creator.ID}
		if current != nil {
			f.Profile.ViewerID = &current.ID
		}
	}

	if params.Kind != nil {
		f.Kind = string(*params.Kind)
	}
	if params.Platform != nil {
		f.Platform, f.PlatformSet = params.Platform, true
	}
	if params.Q != nil {
		f.Query = *params.Q
	}
	if params.Facet != nil {
		f.Facets = parseFacets(*params.Facet)
	}
	if params.Limit != nil {
		f.Limit = *params.Limit
	}

	before, ok := cursorFrom(params)
	if !ok {
		api.Refuse(c, http.StatusBadRequest, "before and beforeId belong together, send both or neither")
		return
	}
	f.Before = before

	var requested *string
	if params.Nsfw != nil {
		value := string(*params.Nsfw)
		requested = &value
	}
	visibility, ok := ReaderVisibility(c, h.accounts, requested)
	if !ok {
		return
	}
	found, err := h.works.Browse(c.Request.Context(), f, visibility)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "could not list assets")
		return
	}

	items := make([]BrowseAsset, 0, len(found.Items))
	for _, item := range found.Items {
		var cover *BrowseCover
		if item.Cover != nil {
			cover = &BrowseCover{
				Url: item.Cover.URL, Width: item.Cover.Width, Height: item.Cover.Height,
			}
		}
		var ownerState *BrowseAssetOwnerState
		if item.OwnerState != "" {
			value := BrowseAssetOwnerState(item.OwnerState)
			ownerState = &value
		}
		items = append(items, BrowseAsset{
			Id: item.ID, Name: item.Name, Creator: item.Creator,
			Kind: BrowseAssetKind(item.Kind), IsNsfw: item.IsNSFW, Cover: cover,
			OwnerState: ownerState,
			Withhold:   toAPIWithhold(item.Withhold),
		})
	}
	var next *BrowseCursor
	if found.Next != nil {
		next = &BrowseCursor{Before: found.Next.MadeAt, BeforeId: found.Next.ID}
	}
	var empty *AssetListEmptyState
	if found.EmptyState != "" {
		value := AssetListEmptyState(found.EmptyState)
		empty = &value
	}
	platforms := make([]BrowseOption, 0, len(found.Platforms))
	for _, option := range found.Platforms {
		platforms = append(platforms, BrowseOption{
			Value: option.Value, Label: option.Label, Count: option.Count, Selected: option.Selected,
		})
	}
	facets := make([]BrowseFacet, 0, len(found.Facets))
	for _, group := range found.Facets {
		options := make([]BrowseOption, 0, len(group.Options))
		for _, option := range group.Options {
			options = append(options, BrowseOption{
				Value: option.Value, Label: option.Label, Count: option.Count, Selected: option.Selected,
			})
		}
		facets = append(facets, BrowseFacet{Key: group.Key, Label: group.Label, Options: options})
	}
	c.JSON(http.StatusOK, AssetList{
		Items: items, Total: found.Total, Suppressed: found.Suppressed,
		Visibility: AssetListVisibility(visibility),
		NextCursor: next, Platforms: platforms, Facets: facets, EmptyState: empty,
	})
}

func cursorFrom(params ListAssetsParams) (*Cursor, bool) {
	switch {
	case params.Before == nil && params.BeforeId == nil:
		return nil, true
	case params.Before == nil || params.BeforeId == nil:
		return nil, false
	}
	return &Cursor{MadeAt: *params.Before, ID: uuid.UUID(*params.BeforeId)}, true
}

func parseFacets(raw []string) []FacetSelection {
	out := make([]FacetSelection, 0, len(raw))
	for _, pair := range raw {
		key, value, split := strings.Cut(pair, "=")
		if !split {
			continue
		}
		out = append(out, FacetSelection{Key: key, Value: value})
	}
	return out
}
