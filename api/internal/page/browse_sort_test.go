package page_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestRecentOrdersWorksByFirstPublication(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	drafted := apitest.StartCharacter(t, router, session)
	apitest.WriteCharacterFloor(t, router, session, drafted)
	older := apitest.PublishedCharacter(t, router, session)
	if published := apitest.PublishWork(t, router, session, drafted.ID); published.Code != http.StatusOK {
		t.Fatalf("publish the draft status = %d, want 200: %s", published.Code, published.Body.String())
	}
	started := apitest.FetchStartedWork(t, router, session, older)
	coreBlock := apitest.BlockNamed(t, started.Blocks, "character_core")
	core := apitest.EditableBlock(coreBlock)
	core.Elements[0].Content = json.RawMessage(`{"text":"A second telling."}`)
	if saved := apitest.SaveBlock(t, router, session, older, coreBlock.ID, core); saved.Code != http.StatusOK {
		t.Fatalf("save status = %d, want 200: %s", saved.Code, saved.Body.String())
	}
	if updated := apitest.PublishWorkVersion(t, router, session, older, `{"summary":"A second telling"}`); updated.Code != http.StatusOK {
		t.Fatalf("publish version 2 status = %d, want 200: %s", updated.Code, updated.Body.String())
	}

	assertOrder(t, apitest.ListItems(t, router, "/v1/works?sort=recent"), drafted.ID, older)
}

func TestActivitySortsCountTheLast30DaysOfPublicWorksAndPageThroughTies(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewConnectRouter(t)
	first := apitest.PublishedCharacter(t, router, session)
	second := apitest.PublishedCharacter(t, router, session)
	third := apitest.PublishedCharacter(t, router, session)
	fourth := apitest.PublishedCharacter(t, router, session)
	unlisted := apitest.PublishedCharacter(t, router, session)
	if got := apitest.Send(t, router, apitest.AuthorizedJSONRequest(
		t, http.MethodPut, "/v1/works/"+unlisted+"/visibility", `{"visibility":"unlisted"}`, session)); got.Code != http.StatusNoContent {
		t.Fatalf("unlist status = %d, want 204: %s", got.Code, got.Body.String())
	}

	for _, workID := range []string{first, first, second, third, unlisted, unlisted, unlisted} {
		viewWork(t, router, workID, browserAgent, nil)
	}
	download := func(workID string, session *http.Cookie) {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/download/"+workID+"/test_opaque", nil)
		if session != nil {
			request = apitest.Authorized(request, session)
		}
		if got := apitest.Send(t, router, request); got.Code != http.StatusOK {
			t.Fatalf("download status = %d, want 200: %s", got.Code, got.Body.String())
		}
	}
	download(fourth, nil)
	download(fourth, nil)
	download(second, nil)
	download(first, session)
	download(unlisted, nil)
	download(unlisted, nil)
	download(unlisted, nil)
	for _, backdated := range []string{
		`insert into daily_totals (day, kind, work_id, count)
		 values ((now() at time zone 'utc')::date - 40, 'view', $1, 5)`,
		`insert into download_records (work_id, original_file_id, format, handed_off_at, access, visibility)
		 select work_id, original_file_id, format, now() - interval '40 days', access, visibility
		   from download_records, generate_series(1, 3) where work_id = $1`,
	} {
		if _, err := pool.Exec(t.Context(), backdated, second); err != nil {
			t.Fatalf("insert activity from 40 days ago: %v", err)
		}
	}

	viewed := apitest.ListPage(t, router, "/v1/works?sort=views&limit=2")
	assertOrder(t, viewed.Items, first, third)
	if viewed.NextCursor == nil {
		t.Fatal("first page by views has no next cursor")
	}
	rest := apitest.ListItems(t, router, "/v1/works?sort=views&limit=2&before="+
		url.QueryEscape(viewed.NextCursor.Before.Format(time.RFC3339Nano))+
		"&beforeId="+viewed.NextCursor.BeforeID+"&beforeCount="+strconv.Itoa(viewed.NextCursor.BeforeCount))
	assertOrder(t, rest, second, fourth)
	if card := rest[0]; card.ViewCount != 6 || card.DownloadCount != 4 {
		t.Errorf("card counts = %d views and %d downloads, want the lifetime 6 and 4", card.ViewCount, card.DownloadCount)
	}
	if card := rest[1]; card.ViewCount != 0 || card.DownloadCount != 2 {
		t.Errorf("card counts = %d views and %d downloads, want 0 and 2", card.ViewCount, card.DownloadCount)
	}

	assertOrder(t, apitest.ListItems(t, router, "/v1/works?sort=downloads"), fourth, second, third, first)
}

func assertOrder(t *testing.T, items []apitest.ListedWork, want ...string) {
	t.Helper()
	got := make([]string, 0, len(items))
	for _, item := range items {
		got = append(got, item.ID)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("listed %v, want %v", got, want)
	}
}
