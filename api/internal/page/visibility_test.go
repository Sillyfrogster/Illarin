package page_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

func TestUploadAcceptsVisibilityAndDefaultsToListed(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		visibility work.Visibility
		want       string
	}{
		{name: "omitted", want: "listed"},
		{name: "explicit unlisted", visibility: work.VisibilityUnlisted, want: "unlisted"},
	} {
		t.Run(test.name, func(t *testing.T) {
			router, session, works := harness.NewVerifiedIngestRouter(t, format.NewRegistry())
			workID := apitest.UploadVisibilityTestWork(t, router, session, works, test.visibility)

			page := apitest.FetchWorkPage(t, router, "/v1/works/"+workID)
			if page.Visibility != test.want {
				t.Fatalf("visibility = %q, want %q", page.Visibility, test.want)
			}
		})
	}
}

func TestCreatorChangesWorkVisibility(t *testing.T) {
	t.Parallel()
	router, session, works := harness.NewVerifiedIngestRouter(t, format.NewRegistry())
	workID := apitest.UploadVisibilityTestWork(t, router, session, works, work.VisibilityListed)

	changed := apitest.Send(t, router, apitest.AuthorizedJSONRequest(
		t,
		http.MethodPut,
		"/v1/works/"+workID+"/visibility",
		`{"visibility":"unlisted"}`,
		session,
	))
	if changed.Code != http.StatusNoContent {
		t.Fatalf("change visibility status = %d, want 204: %s", changed.Code, changed.Body.String())
	}

	page := apitest.FetchWorkPage(t, router, "/v1/works/"+workID)
	if page.Visibility != "unlisted" {
		t.Fatalf("visibility = %q, want unlisted", page.Visibility)
	}
}

func TestChangingVisibilityRequiresTheCreator(t *testing.T) {
	t.Parallel()
	router, session, works := harness.NewVerifiedIngestRouter(t, format.NewRegistry())
	workID := apitest.UploadVisibilityTestWork(t, router, session, works, work.VisibilityListed)

	changed := apitest.Send(t, router, httptest.NewRequest(
		http.MethodPut,
		"/v1/works/"+workID+"/visibility",
		nil,
	))
	if changed.Code != http.StatusUnauthorized {
		t.Fatalf("change visibility status = %d, want 401: %s", changed.Code, changed.Body.String())
	}
}

func TestWithheldWorkVisibilityIsFrozen(t *testing.T) {
	t.Parallel()
	router, session, works, pool := harness.NewVerifiedIngestRouterWithPool(t, format.NewRegistry())
	workID := apitest.UploadVisibilityTestWork(t, router, session, works, work.VisibilityListed)
	var ownerID uuid.UUID
	if err := pool.QueryRow(context.Background(),
		`select id from users where username = 'verified.creator'`,
	).Scan(&ownerID); err != nil {
		t.Fatalf("read creator: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		update works
		   set withheld_at = now(), withheld_by = $2, withheld_reason = 'testing'
		 where id = $1
	`, workID, ownerID); err != nil {
		t.Fatalf("withhold work: %v", err)
	}

	changed := apitest.Send(t, router, apitest.AuthorizedJSONRequest(
		t,
		http.MethodPut,
		"/v1/works/"+workID+"/visibility",
		`{"visibility":"unlisted"}`,
		session,
	))
	if changed.Code != http.StatusConflict {
		t.Fatalf("change visibility status = %d, want 409: %s", changed.Code, changed.Body.String())
	}
}
