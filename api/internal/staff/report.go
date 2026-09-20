package staff

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	// ReportDays is how many complete days the report covers
	ReportDays = 30
	// EventRetentionDays is how long a raw event stays before the nightly rollup deletes it
	EventRetentionDays  = 30
	topWorks            = 10
	rollupAfterMidnight = 20 * time.Minute
)

// RunRollup rolls up once at start and then every day shortly after midnight UTC until the context ends
func (s *Service) RunRollup(ctx context.Context, onError func(error)) {
	for {
		if err := s.Rollup(ctx, time.Now()); err != nil && ctx.Err() == nil && onError != nil {
			onError(err)
		}
		next := time.Now().UTC().Truncate(24 * time.Hour).Add(24*time.Hour + rollupAfterMidnight)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(next)):
		}
	}
}

// Rollup writes the totals for every complete day, adds Umami's visits, and deletes events past their retention
func (s *Service) Rollup(ctx context.Context, now time.Time) error {
	today := now.UTC().Truncate(24 * time.Hour)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin the rollup: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		insert into daily_totals (day, kind, work_id, count)
		select day, kind, work_id, count(*) from events where day < $1::date
		 group by day, kind, work_id
		on conflict (day, kind, work_id) do update set count = excluded.count
	`, day(today)); err != nil {
		return fmt.Errorf("roll events into daily totals: %w", err)
	}
	if _, err := tx.Exec(ctx, `delete from events where day < $1::date`,
		day(today.AddDate(0, 0, -EventRetentionDays))); err != nil {
		return fmt.Errorf("delete events past their retention: %w", err)
	}
	if err := recordVisits(ctx, tx, today); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit the rollup: %w", err)
	}
	return nil
}

// recordVisits counts Umami's visits per day where Umami runs, leaving out the oldest day its retention has already cut into
func recordVisits(ctx context.Context, tx pgx.Tx, today time.Time) error {
	var present bool
	if err := tx.QueryRow(ctx, `select to_regclass('umami.website_event') is not null`).Scan(&present); err != nil {
		return fmt.Errorf("look for Umami's tables: %w", err)
	}
	if !present {
		return nil
	}
	if _, err := tx.Exec(ctx, `
		insert into daily_totals (day, kind, work_id, count)
		select day, 'visit', null, count(distinct visit_id)
		  from (select (created_at at time zone 'utc')::date as day, visit_id
		          from umami.website_event where event_type = 1) as views
		 where day >= $1::date and day < $2::date
		 group by day
		on conflict (day, kind, work_id) do update set count = excluded.count
	`, day(today.AddDate(0, 0, -(EventRetentionDays-1))), day(today)); err != nil {
		return fmt.Errorf("record Umami's visits: %w", err)
	}
	return nil
}

// Report reads the last 30 complete days before now, every day present even when nothing happened
func (s *Service) Report(ctx context.Context, now time.Time) (Report, error) {
	through := now.UTC().Truncate(24*time.Hour).AddDate(0, 0, -1)
	from := through.AddDate(0, 0, -(ReportDays - 1))
	report := Report{From: day(from), Through: day(through), Days: make([]ReportDay, ReportDays), TopWorks: []ReportWork{}}
	for i := range report.Days {
		report.Days[i].Day = day(from.AddDate(0, 0, i))
	}
	rows, err := s.pool.Query(ctx, `
		select day, kind, sum(count)::int from daily_totals
		 where day between $1::date and $2::date group by day, kind
	`, report.From, report.Through)
	if err != nil {
		return Report{}, fmt.Errorf("read the daily totals: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var on time.Time
		var kind string
		var count int
		if err := rows.Scan(&on, &kind, &count); err != nil {
			return Report{}, fmt.Errorf("read a daily total: %w", err)
		}
		entry := &report.Days[int(on.Sub(from).Hours()/24)]
		switch kind {
		case "visit":
			entry.Visits = count
		case "download":
			entry.Downloads = count
		case "send":
			entry.Sends = count
		case "sign_up":
			entry.SignUps = count
		case "publish":
			entry.Publishes = count
		}
	}
	if err := rows.Err(); err != nil {
		return Report{}, fmt.Errorf("read the daily totals: %w", err)
	}
	works, err := s.pool.Query(ctx, `
		select total.work_id, work.name, work.type, sum(total.count)::int as downloads
		  from daily_totals total join works work on work.id = total.work_id
		 where total.kind = 'download' and total.day between $1::date and $2::date
		 group by total.work_id, work.name, work.type
		 order by downloads desc, work.name limit $3
	`, report.From, report.Through, topWorks)
	if err != nil {
		return Report{}, fmt.Errorf("read the most downloaded works: %w", err)
	}
	defer works.Close()
	for works.Next() {
		var top ReportWork
		if err := works.Scan(&top.ID, &top.Name, &top.Type, &top.Downloads); err != nil {
			return Report{}, fmt.Errorf("read a most downloaded work: %w", err)
		}
		report.TopWorks = append(report.TopWorks, top)
	}
	return report, works.Err()
}

func day(at time.Time) string {
	return at.Format(time.DateOnly)
}
