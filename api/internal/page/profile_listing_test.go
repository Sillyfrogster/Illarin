package page_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

func TestCreatorProfileScopesTheBrowseListing(t *testing.T) {
	t.Parallel()
	router, _, works, pool := harness.NewVerifiedIngestRouterWithPool(t, format.NewRegistry())
	var firstID uuid.UUID
	if err := pool.QueryRow(context.Background(),
		`select id from users where username = $1`, "verified.creator").Scan(&firstID); err != nil {
		t.Fatalf("read first creator: %v", err)
	}
	secondID := uuid.New()
	if _, err := pool.Exec(context.Background(),
		`insert into users (id, username) values ($1, $2)`, secondID, "second.creator"); err != nil {
		t.Fatalf("insert second creator: %v", err)
	}
	apitest.CreateProfileWork(t, works, firstID, "First garden", false, work.VisibilityListed)
	apitest.CreateProfileWork(t, works, secondID, "Second garden", false, work.VisibilityListed)

	response := apitest.Send(t, router, httptest.NewRequest(
		http.MethodGet, "/v1/works?creator=verified.creator", nil,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("profile listing status = %d, want 200: %s", response.Code, response.Body.String())
	}

	var body apitest.ProfileListingResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode profile listing: %v", err)
	}
	if body.Total != 1 || len(body.Items) != 1 || body.Items[0].Name != "First garden" {
		t.Fatalf("profile listing = %+v, want only the requested creator's work", body)
	}
}

func TestCreatorProfileFollowsReaderAdultContentPreference(t *testing.T) {
	t.Parallel()
	router, _, works, pool := harness.NewVerifiedIngestRouterWithPool(t, format.NewRegistry())
	var creatorID uuid.UUID
	if err := pool.QueryRow(context.Background(),
		`select id from users where username = $1`, "verified.creator").Scan(&creatorID); err != nil {
		t.Fatalf("read creator: %v", err)
	}
	apitest.CreateProfileWork(t, works, creatorID, "Open garden", false, work.VisibilityListed)
	apitest.CreateProfileWork(t, works, creatorID, "Midnight garden", true, work.VisibilityListed)

	shown := readProfileListing(
		t, router, "/v1/works?creator=verified.creator&nsfw=shown", nil,
	)
	if !slices.Equal(names(shown), []string{"Midnight garden", "Open garden"}) {
		t.Fatalf("shown reader preference returned %v, want both profile works", names(shown))
	}
	hidden := readProfileListing(
		t, router, "/v1/works?creator=verified.creator&nsfw=hidden", nil,
	)
	if !slices.Equal(names(hidden), []string{"Open garden"}) || hidden.Suppressed != 1 {
		t.Fatalf("hidden reader preference = %+v", hidden)
	}
	blurred := readProfileListing(
		t, router, "/v1/works?creator=verified.creator&nsfw=blurred", nil,
	)
	if !slices.Equal(names(blurred), []string{"Midnight garden", "Open garden"}) {
		t.Fatalf("blurred reader preference returned %v, want both profile works", names(blurred))
	}
}

func TestOwnerProfileAlwaysListsActiveWorkWithoutChangingBrowse(t *testing.T) {
	t.Parallel()
	router, session, works, pool := harness.NewVerifiedIngestRouterWithPool(t, format.NewRegistry())
	var creatorID uuid.UUID
	if err := pool.QueryRow(context.Background(),
		`select id from users where username = $1`, "verified.creator").Scan(&creatorID); err != nil {
		t.Fatalf("read creator: %v", err)
	}
	apitest.CreateProfileWork(t, works, creatorID, "Public garden", false, work.VisibilityListed)
	apitest.CreateProfileWork(t, works, creatorID, "Adult garden", true, work.VisibilityListed)
	apitest.CreateProfileWork(t, works, creatorID, "Unlisted garden", false, work.VisibilityUnlisted)
	withheldID := apitest.CreateProfileWork(
		t, works, creatorID, "Withheld garden", false, work.VisibilityListed,
	)
	deletedID := apitest.CreateProfileWork(
		t, works, creatorID, "Deleted garden", false, work.VisibilityListed,
	)
	if _, err := pool.Exec(context.Background(), `
		update works
		   set withheld_at = now(), withheld_by = $2, withheld_reason = 'testing'
		 where id = $1
	`, withheldID, creatorID); err != nil {
		t.Fatalf("withhold work: %v", err)
	}
	if _, err := pool.Exec(context.Background(),
		`update works set deleted_at = now(), recoverable_until = now() + interval '30 days' where id = $1`, deletedID); err != nil {
		t.Fatalf("soft-delete work: %v", err)
	}

	saved := apitest.Send(t, router, apitest.AuthorizedJSONRequest(
		t, http.MethodPut, "/v1/account/nsfw-preference", `{"preference":"hidden"}`, session,
	))
	if saved.Code != http.StatusNoContent {
		t.Fatalf("save hidden preference status = %d: %s", saved.Code, saved.Body.String())
	}

	stranger := readProfileListing(
		t, router, "/v1/works?creator=verified.creator&nsfw=shown", nil,
	)
	if !slices.Equal(names(stranger), []string{"Adult garden", "Public garden"}) {
		t.Fatalf("stranger profile returned %v, want only the listed works", names(stranger))
	}

	owner := readProfileListing(t, router, "/v1/works?creator=verified.creator", session)
	if owner.Total != 4 || owner.Suppressed != 0 {
		t.Fatalf("owner profile counts = total %d, suppressed %d; want 4, 0",
			owner.Total, owner.Suppressed)
	}
	states := make(map[string]*string, len(owner.Items))
	for _, item := range owner.Items {
		states[item.Name] = item.OwnerState
	}
	if len(states) != 4 || states["Public garden"] != nil || states["Adult garden"] != nil ||
		states["Unlisted garden"] == nil || *states["Unlisted garden"] != "unlisted" ||
		states["Withheld garden"] == nil || *states["Withheld garden"] != "withheld" {
		t.Fatalf("owner profile states = %#v", states)
	}
	for _, item := range owner.Items {
		if item.Name == "Withheld garden" &&
			(item.Withhold == nil || item.Withhold.Reason != "testing" || item.Withhold.At.IsZero()) {
			t.Fatalf("owner profile withhold = %+v", item.Withhold)
		}
	}
	if _, exists := states["Deleted garden"]; exists {
		t.Fatalf("soft-deleted work appeared in the active owner listing")
	}

	browse := readProfileListing(t, router, "/v1/works", session)
	if !slices.Equal(names(browse), []string{"Public garden"}) || browse.Suppressed != 1 {
		t.Fatalf("ordinary browse changed for the owner: %+v", browse)
	}
}

func readProfileListing(
	t *testing.T,
	router http.Handler,
	path string,
	session *http.Cookie,
) apitest.ProfileListingResponse {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if session != nil {
		request.AddCookie(session)
	}
	response := apitest.Send(t, router, request)
	if response.Code != http.StatusOK {
		t.Fatalf("profile listing status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var body apitest.ProfileListingResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode profile listing: %v", err)
	}
	return body
}

func names(list apitest.ProfileListingResponse) []string {
	result := make([]string, len(list.Items))
	for i, item := range list.Items {
		result[i] = item.Name
	}
	return result
}
