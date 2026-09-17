package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/gin-gonic/gin"
)

var harness = apitest.Harness{Register: register}

func register(r *gin.Engine, s apitest.Services, d api.Deadlines) error {
	handlers := NewHandlers(
		s.Assets, s.Accounts, s.Links, s.Deliveries, s.Publications,
		s.UpdateDestinations, s.Notifications, s.MaxUploadBytes,
	)
	return Register(r, handlers, d, apitest.Ready)
}

func TestFormatRegistryInvariantIsNotAnUploaderRefusal(t *testing.T) {
	t.Parallel()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	new(Handlers).refuse(ctx, format.ErrConflictingClaims)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "claim") {
		t.Fatalf("response exposed registry details: %s", recorder.Body.String())
	}
}

func post(
	t *testing.T,
	r *gin.Engine,
	session *http.Cookie,
	assets *asset.Service,
	name string,
) *httptest.ResponseRecorder {
	t.Helper()
	metadata := apitest.ExampleMetadata(name)
	metadata["filename"] = name + ".lumitheme"
	return apitest.UploadAndFinish(t, r, session, assets, metadata, []byte(name))
}

type listedAsset struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type listedPage struct {
	Items      []listedAsset `json:"items"`
	NextCursor *struct {
		Before   time.Time `json:"before"`
		BeforeID string    `json:"beforeId"`
	} `json:"nextCursor"`
}

func getPage(t *testing.T, r *gin.Engine, url string) listedPage {
	t.Helper()
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200. body: %s", url, rec.Code, rec.Body.String())
	}

	var list listedPage
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return list
}

func get(t *testing.T, r *gin.Engine, url string) []listedAsset {
	t.Helper()
	return getPage(t, r, url).Items
}

func TestCreateThenListRoundTrip(t *testing.T) {
	t.Parallel()
	r, session, assets := harness.NewVerifiedIngestRouter(t, format.NewRegistry())

	rec := post(t, r, session, assets, "Mystery")
	if rec.Code != http.StatusOK {
		t.Fatalf("poll status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}

	var created struct {
		Asset *struct {
			CreatedAt time.Time `json:"createdAt"`
		} `json:"asset"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created asset: %v", err)
	}
	if created.Asset == nil || created.Asset.CreatedAt.IsZero() {
		t.Error("created asset came back with no made date")
	}

	items := get(t, r, "/v1/assets")
	if len(items) != 1 || items[0].Name != "Mystery" {
		t.Fatalf("list returned %+v, want one asset named Mystery", items)
	}
}

func TestListPagesFromWhereTheLastPageEnded(t *testing.T) {
	t.Parallel()
	r, session, assets := harness.NewVerifiedIngestRouter(t, format.NewRegistry())
	for _, name := range []string{"first", "second", "third"} {
		if rec := post(t, r, session, assets, name); rec.Code != http.StatusOK {
			t.Fatalf("POST %s status = %d. body: %s", name, rec.Code, rec.Body.String())
		}
	}

	page := getPage(t, r, "/v1/assets?limit=2")
	if len(page.Items) != 2 || page.Items[0].Name != "third" || page.Items[1].Name != "second" {
		t.Fatalf("first page = %+v, want third then second", page)
	}
	if page.NextCursor == nil {
		t.Fatal("first page has no next cursor")
	}

	next := get(t, r, "/v1/assets?limit=2&before="+
		url.QueryEscape(page.NextCursor.Before.Format(time.RFC3339Nano))+"&beforeId="+page.NextCursor.BeforeID)
	if len(next) != 1 || next[0].Name != "first" {
		t.Fatalf("second page = %+v, want only first", next)
	}
}

func TestListRefusesHalfACursor(t *testing.T) {
	t.Parallel()
	r := harness.NewRouter(t)

	for _, query := range []string{
		"/v1/assets?before=2024-05-01T12:00:00Z",
		"/v1/assets?beforeId=6f1e1a2c-0000-4000-8000-000000000000",
	} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, query, nil))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("GET %s status = %d, want 400: half a cursor has no fixed point", query, rec.Code)
		}
	}
}
