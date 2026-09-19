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

func (h *Handlers) ListWorks(c *gin.Context) {
	q := api.ReadQuery(c)
	aliasBrowseQuery(q)
	params := ListWorksParams{
		Type:     api.QueryText[ListWorksParamsType](q, "type"),
		App:      api.QueryText[string](q, "app"),
		Creator:  api.QueryText[string](q, "creator"),
		Q:        api.QueryText[string](q, "q"),
		Facet:    api.QueryList(q, "facet"),
		Nsfw:     api.QueryText[ListWorksParamsNsfw](q, "nsfw"),
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

	if params.Type != nil {
		f.Type = string(*params.Type)
	}
	if params.App != nil {
		f.App, f.AppSet = params.App, true
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
	preference, ok := ReaderNSFWPreference(c, h.accounts, requested)
	if !ok {
		return
	}
	found, err := h.works.Browse(c.Request.Context(), f, preference)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "could not list works")
		return
	}

	items := make([]BrowseWork, 0, len(found.Items))
	for _, item := range found.Items {
		var cover *BrowseCover
		if item.Cover != nil {
			cover = &BrowseCover{
				Url: item.Cover.URL, Width: item.Cover.Width, Height: item.Cover.Height,
			}
		}
		var ownerState *BrowseWorkOwnerState
		if item.OwnerState != "" {
			value := BrowseWorkOwnerState(item.OwnerState)
			ownerState = &value
		}
		items = append(items, BrowseWork{
			Id: item.ID, Name: item.Name, Creator: item.Creator,
			Type: BrowseWorkType(item.Type), IsNsfw: item.IsNSFW, Cover: cover,
			OwnerState: ownerState,
			Withhold:   toAPIWithhold(item.Withhold),
		})
	}
	var next *BrowseCursor
	if found.Next != nil {
		next = &BrowseCursor{Before: found.Next.MadeAt, BeforeId: found.Next.ID}
	}
	var empty *WorkListEmptyState
	if found.EmptyState != "" {
		value := WorkListEmptyState(found.EmptyState)
		empty = &value
	}
	apps := make([]BrowseOption, 0, len(found.Apps))
	for _, option := range found.Apps {
		apps = append(apps, BrowseOption{
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
	c.JSON(http.StatusOK, WorkList{
		Items: items, Total: found.Total, Suppressed: found.Suppressed,
		NSFWPreference: WorkListNSFWPreference(preference),
		NextCursor:     next, Apps: apps, Facets: facets, EmptyState: empty,
	})
}

func cursorFrom(params ListWorksParams) (*Cursor, bool) {
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
