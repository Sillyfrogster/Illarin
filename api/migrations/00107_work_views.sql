-- +goose Up

-- A work view is an event like a download: the kind, the work and the day, never
-- who looked. The indexes let a work's page add up its lifetime counts.

alter table events drop constraint events_kind_check;
alter table events add constraint events_kind_check
    check (kind in ('view', 'download', 'send', 'sign_up', 'publish'));
alter table daily_totals drop constraint daily_totals_kind_check;
alter table daily_totals add constraint daily_totals_kind_check
    check (kind in ('view', 'visit', 'download', 'send', 'sign_up', 'publish'));

create index events_by_work on events (work_id, kind) where work_id is not null;
create index daily_totals_by_work on daily_totals (work_id, kind) where work_id is not null;
create index download_records_public_by_work on download_records (work_id) where access = 'public';

-- +goose Down
drop index download_records_public_by_work;
drop index daily_totals_by_work;
drop index events_by_work;
delete from daily_totals where kind = 'view';
delete from events where kind = 'view';
alter table daily_totals drop constraint daily_totals_kind_check;
alter table daily_totals add constraint daily_totals_kind_check
    check (kind in ('visit', 'download', 'send', 'sign_up', 'publish'));
alter table events drop constraint events_kind_check;
alter table events add constraint events_kind_check
    check (kind in ('download', 'send', 'sign_up', 'publish'));
