package staff_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/staff"
	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestADownloadASendASignUpAndAPublishEachRecordOneEventWithNoIdentity(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.ReceivePermission})
	apitest.DeclareFormats(t, router, credentials.AccessToken, []string{"test_opaque"})
	workID := apitest.PublishedCharacter(t, router, session)

	downloaded := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, "/download/"+workID+"/test_opaque", nil))
	if downloaded.Code != http.StatusOK {
		t.Fatalf("download status = %d, want 200: %s", downloaded.Code, downloaded.Body.String())
	}
	if queued := apitest.SendToApp(t, router, session, workID, credentials.ConnectedApp.ID); queued.Code != http.StatusAccepted {
		t.Fatalf("send status = %d, want 202: %s", queued.Code, queued.Body.String())
	}
	collected := apitest.DecodeResponse[apitest.CollectedSends](t, apitest.Collect(t, router, credentials.AccessToken, nil))
	if fetched := apitest.FetchSigned(t, router, collected.Sends[0].Files[0].URL); fetched.Code != http.StatusOK {
		t.Fatalf("fetch status = %d, want 200: %s", fetched.Code, fetched.Body.String())
	}

	counts := eventCounts(t, pool)
	want := map[string]int{"sign_up": 1, "publish": 1, "download": 1, "send": 1}
	for kind, n := range want {
		if counts[kind] != n {
			t.Fatalf("events = %v, want %v", counts, want)
		}
	}
	var workEvents, columns int
	if err := pool.QueryRow(context.Background(), `
		select (select count(*) from events where work_id = $1 and kind in ('download', 'send', 'publish')),
		       (select count(*) from information_schema.columns where table_name = 'events')
	`, workID).Scan(&workEvents, &columns); err != nil {
		t.Fatalf("read the events: %v", err)
	}
	if workEvents != 3 || columns != 3 {
		t.Fatalf("%d events name the work and the table has %d columns, want 3 and 3 (kind, work_id, day)", workEvents, columns)
	}
	apitest.SetRole(t, pool, "connect.creator", "moderator")
	response := apitest.Send(t, router, apitest.Authorized(httptest.NewRequest(http.MethodGet, "/v1/staff/report", nil), session))
	if response.Code != http.StatusOK {
		t.Fatalf("report status = %d: %s", response.Code, response.Body.String())
	}
	report := apitest.DecodeResponse[staff.Report](t, response)
	today := report.Days[len(report.Days)-1]
	if today.Downloads != 1 || today.Sends != 1 || today.SignUps != 1 || today.Publishes != 1 {
		t.Fatalf("report today = %+v, want one of each event", today)
	}
	if len(report.TopWorks) != 1 || report.TopWorks[0].ID != workID || report.TopWorks[0].Downloads != 1 {
		t.Fatalf("report most downloaded works = %+v, want the downloaded work", report.TopWorks)
	}
}

func TestEveryPublishedVersionRecordsAnEvent(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewConnectRouter(t)
	workID := apitest.PublishedCharacter(t, router, session)
	page := apitest.FetchStartedWork(t, router, session, workID)
	core := apitest.BlockNamed(t, page.Blocks, "character_core")
	edited := apitest.EditableBlock(core)
	edited.Elements[0].Content = []byte(`{"text":"The west shelf moved again."}`)
	if saved := apitest.SaveBlock(t, router, session, workID, core.ID, edited); saved.Code != http.StatusOK {
		t.Fatalf("save changes = %d: %s", saved.Code, saved.Body.String())
	}
	if published := apitest.PublishWorkVersion(t, router, session, workID, `{"summary":"Revised the description"}`); published.Code != http.StatusOK {
		t.Fatalf("publish version = %d: %s", published.Code, published.Body.String())
	}
	if got := eventCounts(t, pool)["publish"]; got != 2 {
		t.Fatalf("publish events = %d, want one for each version", got)
	}
}

func TestTheNightlyRollupKeepsDailyTotalsAndDropsEventsAfterThirtyDays(t *testing.T) {
	t.Parallel()
	pool := testdb.Connect(t)
	ctx := t.Context()
	today := time.Date(2026, 9, 20, 15, 0, 0, 0, time.UTC)
	workID := uuid.New()
	if _, err := pool.Exec(ctx, `
		insert into events (kind, work_id, day) values
			('download', $1, '2026-09-19'), ('download', $1, '2026-09-19'), ('sign_up', null, '2026-09-19'),
			('download', $1, '2026-08-20'), ('send', $1, '2026-08-21'),
			('publish', $1, '2026-09-20')
	`, workID); err != nil {
		t.Fatalf("insert events: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		create schema umami;
		create table umami.website_event (visit_id uuid, event_type integer, created_at timestamptz)
	`); err != nil {
		t.Fatalf("stand in for Umami: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into umami.website_event values
			($1, 1, '2026-09-19 10:00+00'), ($1, 1, '2026-09-19 10:05+00'), ($2, 1, '2026-09-19 23:59+00'),
			($3, 2, '2026-09-19 12:00+00'), ($4, 1, '2026-09-20 01:00+00')
	`, uuid.New(), uuid.New(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("insert Umami's rows: %v", err)
	}

	service := staff.NewService(apitest.WorksOver(t, pool, format.NewRegistry()))
	for range 2 {
		if err := service.Rollup(ctx, today); err != nil {
			t.Fatalf("roll up: %v", err)
		}
	}

	totals := map[string]int{}
	rows, err := pool.Query(ctx, `select day::text || ' ' || kind || coalesce(' ' || work_id::text, ''), count from daily_totals`)
	if err != nil {
		t.Fatalf("read totals: %v", err)
	}
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			t.Fatalf("scan a total: %v", err)
		}
		totals[key] = count
	}
	want := map[string]int{
		"2026-09-19 download " + workID.String(): 2,
		"2026-09-19 sign_up":                     1,
		"2026-09-19 visit":                       2,
		"2026-08-20 download " + workID.String(): 1,
		"2026-08-21 send " + workID.String():     1,
	}
	if len(totals) != len(want) {
		t.Fatalf("totals = %v, want %v", totals, want)
	}
	for key, count := range want {
		if totals[key] != count {
			t.Fatalf("totals = %v, want %v", totals, want)
		}
	}
	left := eventCounts(t, pool)
	if left["download"] != 2 || left["send"] != 1 || left["publish"] != 1 || left["sign_up"] != 1 {
		t.Fatalf("events left = %v, want only the last 30 days and today", left)
	}
}

func TestOnlyStaffReadTheReportAndItCoversThirtyDaysThroughToday(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewConnectRouter(t)
	ctx := context.Background()
	workID := apitest.PublishedCharacter(t, router, session)
	apitest.UploadedImageID(t, router, session, workID, "avatar", apitest.PNG(t, 64, 64))
	other := uuid.New()
	if _, err := pool.Exec(ctx, `
		insert into works (id, type, name, lifecycle) values ($1, 'character', 'Quiet Shelf', 'published')
	`, other); err != nil {
		t.Fatalf("insert a second work: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into daily_totals (day, kind, work_id, count) values
			(current_date - 1, 'visit', null, 40), (current_date - 1, 'download', $2, 3),
			(current_date - 1, 'download', $1, 5), (current_date - 29, 'sign_up', null, 2),
			(current_date - 30, 'sign_up', null, 9), (current_date, 'send', $2, 1)
	`, other, workID); err != nil {
		t.Fatalf("insert totals: %v", err)
	}

	read := func() *httptest.ResponseRecorder {
		return apitest.Send(t, router, apitest.Authorized(httptest.NewRequest(http.MethodGet, "/v1/staff/report", nil), session))
	}
	if refused := read(); refused.Code != http.StatusForbidden {
		t.Fatalf("reader status = %d, want 403: %s", refused.Code, refused.Body.String())
	}
	apitest.SetRole(t, pool, "connect.creator", "moderator")
	rec := read()
	if rec.Code != http.StatusOK {
		t.Fatalf("moderator status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	report := apitest.DecodeResponse[staff.Report](t, rec)

	today := time.Now().UTC().Format(time.DateOnly)
	if len(report.Days) != 30 || report.Through != today || report.Days[29].Day != today {
		t.Fatalf("report covers %d days through %s, want 30 through %s", len(report.Days), report.Through, today)
	}
	last, first := report.Days[28], report.Days[0]
	if last.Visits != 40 || last.Downloads != 8 || last.Sends != 0 || first.SignUps != 2 || first.Day != report.From {
		t.Fatalf("days = first %+v, last %+v", first, last)
	}
	if report.Previous != (staff.ReportTotals{SignUps: 9}) {
		t.Fatalf("previous 30 days = %+v, want only the 9 sign-ups on the day before the report", report.Previous)
	}
	if len(report.TopWorks) != 2 || report.TopWorks[0].ID != other.String() || report.TopWorks[0].Downloads != 5 ||
		report.TopWorks[0].Name != "Quiet Shelf" || report.TopWorks[0].Cover != nil || report.TopWorks[1].ID != workID {
		t.Fatalf("top works = %+v", report.TopWorks)
	}
	if cover := report.TopWorks[1].Cover; cover == nil || !strings.Contains(*cover, "/media/") {
		t.Fatalf("the published character's cover = %v, want a media address", cover)
	}
}

func TestReportShowsEventsRecordedToday(t *testing.T) {
	t.Parallel()
	pool := testdb.Connect(t)
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `insert into events (kind) values ('sign_up')`); err != nil {
		t.Fatalf("record sign-up: %v", err)
	}
	if _, err := pool.Exec(ctx, `create schema umami; create table umami.website_event (visit_id uuid, event_type integer, created_at timestamptz)`); err != nil {
		t.Fatalf("set up visits: %v", err)
	}
	if _, err := pool.Exec(ctx, `insert into umami.website_event values ($1, 1, now()), ($1, 1, now())`, uuid.New()); err != nil {
		t.Fatalf("record visits: %v", err)
	}
	report, err := staff.NewService(apitest.WorksOver(t, pool, format.NewRegistry())).Report(ctx, time.Now())
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if today := report.Days[len(report.Days)-1]; today.Day != time.Now().UTC().Format(time.DateOnly) || today.SignUps != 1 || today.Visits != 1 {
		t.Fatalf("today = %+v, want one sign-up and one visit", today)
	}
}

func TestFirstUploadedVersionRecordsAnEvent(t *testing.T) {
	t.Parallel()
	pool := testdb.Connect(t)
	ctx := t.Context()
	workID := uuid.New()
	if _, err := pool.Exec(ctx, `insert into works (id, type, name, lifecycle) values ($1, 'pack', 'Loom set', 'published')`, workID); err != nil {
		t.Fatalf("insert published work: %v", err)
	}
	if _, err := pool.Exec(ctx, `select record_initial_work_version($1, false)`, workID); err != nil {
		t.Fatalf("record first version: %v", err)
	}
	if got := eventCounts(t, pool)["publish"]; got != 1 {
		t.Fatalf("publish events = %d, want 1", got)
	}
}

func eventCounts(t *testing.T, pool *pgxpool.Pool) map[string]int {
	t.Helper()
	rows, err := pool.Query(context.Background(), `select kind, count(*) from events group by kind`)
	if err != nil {
		t.Fatalf("count events: %v", err)
	}
	counts := map[string]int{}
	for rows.Next() {
		var kind string
		var count int
		if err := rows.Scan(&kind, &count); err != nil {
			t.Fatalf("scan an event count: %v", err)
		}
		counts[kind] = count
	}
	return counts
}
