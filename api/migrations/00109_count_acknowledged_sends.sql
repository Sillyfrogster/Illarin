-- +goose Up

-- A send counts once, when its connected app acknowledges delivery, rather than
-- when it is queued. The queued counts are replaced by the delivered sends still
-- on record, each counted on the day it was acknowledged.

drop trigger record_send on sends;
create trigger record_send after update of state on sends
    for each row when (old.state <> 'delivered' and new.state = 'delivered')
    execute function record_event('send', 'work_id');

delete from daily_totals where kind = 'send';
delete from events where kind = 'send';
insert into events (kind, work_id, day)
select 'send', work_id, (settled_at at time zone 'utc')::date
  from sends where state = 'delivered';

-- +goose Down
drop trigger record_send on sends;
create trigger record_send after insert on sends
    for each row execute function record_event('send', 'work_id');
