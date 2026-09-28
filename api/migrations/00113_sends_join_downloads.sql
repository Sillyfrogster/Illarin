-- +goose Up

-- A delivered send counts toward a work's downloads unless it went to one of the
-- work creator's own connected apps. Sends are kept for 7 days, so only the
-- creator's own sends still on record can be taken back out of the counts.

-- +goose StatementBegin
create function record_send() returns trigger language plpgsql as $$
begin
    insert into events (kind, work_id)
    select 'send', work.id
      from works work, connected_apps app
     where work.id = new.work_id and app.id = new.connected_app_id
       and app.user_id is distinct from work.owner_id;
    return null;
end;
$$;
-- +goose StatementEnd

drop trigger record_send on sends;
create trigger record_send after update of state on sends
    for each row when (old.state <> 'delivered' and new.state = 'delivered')
    execute function record_send();

create temporary table own_sends as
select send.work_id, (send.settled_at at time zone 'utc')::date as day, count(*)::int as sends
  from sends send
  join connected_apps app on app.id = send.connected_app_id
  join works work on work.id = send.work_id
 where send.state = 'delivered' and app.user_id = work.owner_id
 group by send.work_id, day;

delete from events where ctid in (
    select numbered.ctid
      from (select ctid, work_id, day, row_number() over (partition by work_id, day) as n
              from events where kind = 'send') numbered
      join own_sends own on own.work_id = numbered.work_id and own.day = numbered.day
     where numbered.n <= own.sends
);

update daily_totals total
   set count = greatest(total.count - own.sends, 0)
  from own_sends own
 where total.kind = 'send' and total.work_id = own.work_id and total.day = own.day;

drop table own_sends;

-- +goose StatementBegin
create or replace function rank_works(through date) returns void language sql as $$
    delete from work_rankings where day = through or day < through - 7;
    insert into work_rankings (day, work_id, views, downloads)
    select through, counted.work_id,
           coalesce(sum(counted.count) filter (where counted.kind = 'view'), 0),
           coalesce(sum(counted.count) filter (where counted.kind in ('download', 'send')), 0)
      from work_day_counts counted
      join works work on work.id = counted.work_id
     where counted.kind in ('view', 'download', 'send')
       and counted.day >= through - 30 and counted.day < through
     group by counted.work_id;
$$;
-- +goose StatementEnd

select rank_works((now() at time zone 'utc')::date);

-- +goose Down

-- +goose StatementBegin
create or replace function rank_works(through date) returns void language sql as $$
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

drop trigger record_send on sends;
create trigger record_send after update of state on sends
    for each row when (old.state <> 'delivered' and new.state = 'delivered')
    execute function record_event('send', 'work_id');
drop function record_send();
