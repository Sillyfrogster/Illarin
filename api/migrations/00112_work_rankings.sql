-- +goose Up

-- Download events count public handoffs only, rebuilt from the download records
drop trigger record_download on download_records;
create trigger record_download after insert on download_records
    for each row when (new.access = 'public') execute function record_event('download', 'work_id');

delete from events where kind = 'download';
insert into events (kind, work_id, day)
select 'download', work_id, (handed_off_at at time zone 'utc')::date
  from download_records
 where access = 'public'
   and (handed_off_at at time zone 'utc')::date >= (now() at time zone 'utc')::date - 30;

delete from daily_totals where kind = 'download';
insert into daily_totals (day, kind, work_id, count)
select (handed_off_at at time zone 'utc')::date, 'download', work_id, count(*)
  from download_records
 where access = 'public'
   and (handed_off_at at time zone 'utc')::date < (now() at time zone 'utc')::date
 group by 1, work_id;

-- The nightly rollup ranks each work by its views and downloads over the 30 days before that day
create table work_rankings (
    day       date not null,
    work_id   uuid not null references works (id) on delete cascade,
    views     integer not null,
    downloads integer not null,
    primary key (day, work_id)
);

-- +goose StatementBegin
create function rank_works(through date) returns void language sql as $$
    delete from work_rankings where day = through or day < through - 7;
    insert into work_rankings (day, work_id, views, downloads)
    select through, counted.work_id,
           coalesce(sum(counted.count) filter (where counted.kind = 'view'), 0),
           coalesce(sum(counted.count) filter (where counted.kind = 'download'), 0)
      from work_day_counts counted
      join works work on work.id = counted.work_id
     where counted.kind in ('view', 'download')
       and counted.day >= through - 30 and counted.day < through
     group by counted.work_id;
$$;
-- +goose StatementEnd

select rank_works((now() at time zone 'utc')::date);

-- +goose Down
drop function rank_works(date);
drop table work_rankings;
drop trigger record_download on download_records;
create trigger record_download after insert on download_records
    for each row when (new.access <> 'app') execute function record_event('download', 'work_id');
