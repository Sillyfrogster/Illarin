-- +goose Up

-- A work's counts by day, from the nightly totals and any events not yet rolled up
create view work_day_counts as
select day, kind, work_id, count from daily_totals where work_id is not null
union all
select day, kind, work_id, count(*)::int from events event
 where work_id is not null
   and not exists (select from daily_totals total
                    where total.day = event.day and total.kind = event.kind and total.work_id = event.work_id)
 group by day, kind, work_id;

-- +goose Down
drop view work_day_counts;
