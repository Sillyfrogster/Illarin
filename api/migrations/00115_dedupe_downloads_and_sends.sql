-- +goose Up

-- A public download counts once per reader, work and UTC day by the view's daily
-- key, so a key now says which count it stands for. The download event is written
-- by the download service, which holds the key, instead of a trigger.
alter table reader_keys add column kind text not null default 'view' check (kind in ('view', 'download'));
alter table reader_keys alter column kind drop default;
alter table reader_keys drop constraint reader_keys_pkey, add primary key (day, kind, key);

drop trigger record_download on download_records;

-- A delivered send counts once per connected app, work and UTC day. The nightly
-- rollup deletes earlier days.
create table counted_sends (
    day              date not null,
    connected_app_id uuid not null,
    work_id          uuid not null,
    primary key (day, connected_app_id, work_id)
);

-- +goose StatementBegin
create or replace function record_send() returns trigger language plpgsql as $$
begin
    with counted as (
        insert into counted_sends (day, connected_app_id, work_id)
        select (new.settled_at at time zone 'utc')::date, app.id, work.id
          from works work, connected_apps app
         where work.id = new.work_id and app.id = new.connected_app_id
           and app.user_id is distinct from work.owner_id
        on conflict do nothing
        returning day, work_id
    )
    insert into events (kind, work_id, day) select 'send', work_id, day from counted;
    return null;
end;
$$;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
create or replace function record_send() returns trigger language plpgsql as $$
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

drop table counted_sends;

create trigger record_download after insert on download_records
    for each row when (new.access = 'public') execute function record_event('download', 'work_id');

delete from reader_keys where kind <> 'view';
alter table reader_keys drop constraint reader_keys_pkey, add primary key (day, key);
alter table reader_keys drop column kind;
