-- +goose Up
alter table notifications add column update_count integer not null default 1;

with ranked as (
    select id,
           row_number() over (
               partition by account_id, asset_id order by created_at desc, id desc
           ) as place,
           count(*) over (partition by account_id, asset_id) as arrivals
      from notifications
     where type = 'asset_updated' and read_at is null
), latest as (
    update notifications as entry
       set update_count = ranked.arrivals
      from ranked
     where entry.id = ranked.id and ranked.place = 1
)
delete from notifications where id in (select id from ranked where place > 1);

create unique index notifications_folding on notifications (account_id, asset_id)
    where type = 'asset_updated' and read_at is null;

create view inbox_entries as
select id, account_id, type, asset_id, words, created_at, read_at, update_count
  from notifications as entry
 where entry.asset_id is null
    or exists (
        select 1
          from assets as subject
         where subject.id = entry.asset_id
           and (subject.owner_id = entry.account_id
                or (subject.deleted_at is null and subject.withheld_at is null))
    );

-- +goose Down
drop view inbox_entries;
drop index notifications_folding;
alter table notifications drop column update_count;
