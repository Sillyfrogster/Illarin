-- +goose Up

-- Illarin counts what happens without recording who did it. An event is the
-- kind, the work where there is one, and the day, written by the database
-- whenever an account, a download record, a send or a published work appears.
-- The staff package rolls events into daily totals every night, keeps the
-- totals forever and deletes events older than 30 days.

create table events (
    kind    text not null check (kind in ('download', 'send', 'sign_up', 'publish')),
    work_id uuid,
    day     date not null default (now() at time zone 'utc')::date
);

create index events_by_day on events (day);

create table daily_totals (
    day     date not null,
    kind    text not null check (kind in ('visit', 'download', 'send', 'sign_up', 'publish')),
    work_id uuid,
    count   integer not null check (count >= 0),
    unique nulls not distinct (day, kind, work_id)
);

-- +goose StatementBegin
create function record_event() returns trigger language plpgsql as $$
begin
    insert into events (kind, work_id)
    values (tg_argv[0], (to_jsonb(new) ->> tg_argv[1])::uuid);
    return null;
end;
$$;
-- +goose StatementEnd

create trigger record_sign_up after insert on users
    for each row execute function record_event('sign_up');
create trigger record_download after insert on download_records
    for each row when (new.access <> 'app') execute function record_event('download', 'work_id');
create trigger record_send after insert on sends
    for each row execute function record_event('send', 'work_id');
create trigger record_publish after update of lifecycle on works
    for each row when (old.lifecycle = 'draft' and new.lifecycle = 'published')
    execute function record_event('publish', 'id');

-- +goose Down
drop trigger record_publish on works;
drop trigger record_send on sends;
drop trigger record_download on download_records;
drop trigger record_sign_up on users;
drop function record_event();
drop table daily_totals;
drop table events;
