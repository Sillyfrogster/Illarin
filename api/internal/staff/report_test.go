package staff_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
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

	service := staff.NewService(pool)
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

func TestOnlyStaffReadTheReportAndItCoversThirtyCompleteDays(t *testing.T) {
	t.Parallel()
	router, session, pool := harness.NewConnectRouter(t)
	ctx := context.Background()
	workID := apitest.PublishedCharacter(t, router, session)
	other := uuid.New()
	if _, err := pool.Exec(ctx, `
		insert into works (id, type, name, lifecycle) values ($1, 'character', 'Quiet Shelf', 'published')
	`, other); err != nil {
		t.Fatalf("insert a second work: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into daily_totals (day, kind, work_id, count) values
			(current_date - 1, 'visit', null, 40), (current_date - 1, 'download', $2, 3),
			(current_date - 1, 'download', $1, 5), (current_date - 30, 'sign_up', null, 2),
			(current_date - 31, 'sign_up', null, 9), (current_date, 'send', $2, 1)
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

	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format(time.DateOnly)
	if len(report.Days) != 30 || report.Through != yesterday || report.Days[29].Day != yesterday {
		t.Fatalf("report covers %d days through %s, want 30 through %s", len(report.Days), report.Through, yesterday)
	}
	last, first := report.Days[29], report.Days[0]
	if last.Visits != 40 || last.Downloads != 8 || last.Sends != 0 || first.SignUps != 2 || first.Day != report.From {
		t.Fatalf("days = first %+v, last %+v", first, last)
	}
	if len(report.TopWorks) != 2 || report.TopWorks[0].ID != other.String() || report.TopWorks[0].Downloads != 5 ||
		report.TopWorks[0].Name != "Quiet Shelf" || report.TopWorks[1].ID != workID {
		t.Fatalf("top works = %+v", report.TopWorks)
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
