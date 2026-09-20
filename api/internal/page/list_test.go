package page_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func post(
	t *testing.T,
	r *gin.Engine,
	session *http.Cookie,
	works *work.Service,
	name string,
) *httptest.ResponseRecorder {
	t.Helper()
	metadata := apitest.ExampleMetadata(name)
	metadata["filename"] = name + ".lumitheme"
	return apitest.UploadAndFinish(t, r, session, works, metadata, []byte(name))
}

func TestCreateThenListRoundTrip(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewVerifiedUploadRouter(t, format.NewRegistry())

	rec := post(t, r, session, works, "Mystery")
	if rec.Code != http.StatusOK {
		t.Fatalf("poll status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}

	var created struct {
		Work *struct {
			CreatedAt time.Time `json:"createdAt"`
		} `json:"work"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created work: %v", err)
	}
	if created.Work == nil || created.Work.CreatedAt.IsZero() {
		t.Error("created work came back with no made date")
	}

	items := apitest.ListItems(t, r, "/v1/works")
	if len(items) != 1 || items[0].Name != "Mystery" {
		t.Fatalf("list returned %+v, want one work named Mystery", items)
	}
}

func TestListPagesFromWhereTheLastPageEnded(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewVerifiedUploadRouter(t, format.NewRegistry())
	for _, name := range []string{"first", "second", "third"} {
		if rec := post(t, r, session, works, name); rec.Code != http.StatusOK {
			t.Fatalf("POST %s status = %d. body: %s", name, rec.Code, rec.Body.String())
		}
	}

	page := apitest.ListPage(t, r, "/v1/works?limit=2")
	if len(page.Items) != 2 || page.Items[0].Name != "third" || page.Items[1].Name != "second" {
		t.Fatalf("first page = %+v, want third then second", page)
	}
	if page.NextCursor == nil {
		t.Fatal("first page has no next cursor")
	}

	next := apitest.ListItems(t, r, "/v1/works?limit=2&before="+
		url.QueryEscape(page.NextCursor.Before.Format(time.RFC3339Nano))+"&beforeId="+page.NextCursor.BeforeID)
	if len(next) != 1 || next[0].Name != "first" {
		t.Fatalf("second page = %+v, want only first", next)
	}
}

func TestListRefusesHalfACursor(t *testing.T) {
	t.Parallel()
	r := harness.NewRouter(t)

	for _, query := range []string{
		"/v1/works?before=2024-05-01T12:00:00Z",
		"/v1/works?beforeId=6f1e1a2c-0000-4000-8000-000000000000",
	} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, query, nil))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("GET %s status = %d, want 400: half a cursor has no fixed point", query, rec.Code)
		}
	}
}
