-- +goose Up

-- An account follows a work when it asked to, or when one of its live connected
-- apps holds the work and it has not stopped following. Nobody follows their own
-- work. The work page and update notifications both read this rule.
create view work_followers as
select follower.account_id, follower.work_id
  from (
      select account_id, work_id from work_follows where state = 'following'
      union
      select app.user_id, entry.work_id
        from app_library_entries entry
        join connected_apps app on app.id = entry.connected_app_id
       where app.revoked_at is null
         and not exists (select from work_follows stopped
                          where stopped.account_id = app.user_id and stopped.work_id = entry.work_id
                            and stopped.state = 'stopped')
  ) follower
  join works work on work.id = follower.work_id
 where work.owner_id is distinct from follower.account_id;

-- counted_follows is following as it last stood, so a follow or unfollow event is
-- written only when following really changes. The events say nothing about who.
-- History starts here, so today's followers are taken as they stand with no events.
create table counted_follows (
    account_id uuid not null references users (id) on delete cascade,
    work_id    uuid not null references works (id) on delete cascade,
    primary key (work_id, account_id)
);
insert into counted_follows select account_id, work_id from work_followers;

alter table events drop constraint events_kind_check;
alter table events add constraint events_kind_check
    check (kind in ('view', 'download', 'send', 'sign_up', 'publish', 'follow', 'unfollow'));
alter table daily_totals drop constraint daily_totals_kind_check;
alter table daily_totals add constraint daily_totals_kind_check
    check (kind in ('view', 'visit', 'download', 'send', 'sign_up', 'publish', 'follow', 'unfollow'));

-- +goose StatementBegin
create function record_follow_change() returns trigger language plpgsql as $$
declare
    changed jsonb := to_jsonb(coalesce(new, old));
    work    uuid := changed ->> 'work_id';
    account uuid := coalesce((changed ->> 'account_id')::uuid,
                             (select user_id from connected_apps where id = (changed ->> 'connected_app_id')::uuid));
begin
    if exists (select from work_followers where account_id = account and work_id = work) then
        insert into counted_follows values (account, work) on conflict do nothing;
        if found then
            insert into events (kind, work_id) values ('follow', work);
        end if;
    else
        delete from counted_follows where account_id = account and work_id = work;
        if found then
            insert into events (kind, work_id) values ('unfollow', work);
        end if;
    end if;
    return null;
end;
$$;
-- +goose StatementEnd

create trigger record_follow after insert or update of state on work_follows
    for each row execute function record_follow_change();
create trigger record_library_follow after insert or delete on app_library_entries
    for each row execute function record_follow_change();

-- +goose Down
drop trigger record_library_follow on app_library_entries;
drop trigger record_follow on work_follows;
drop function record_follow_change();
delete from daily_totals where kind in ('follow', 'unfollow');
delete from events where kind in ('follow', 'unfollow');
alter table daily_totals drop constraint daily_totals_kind_check;
alter table daily_totals add constraint daily_totals_kind_check
    check (kind in ('view', 'visit', 'download', 'send', 'sign_up', 'publish'));
alter table events drop constraint events_kind_check;
alter table events add constraint events_kind_check
    check (kind in ('view', 'download', 'send', 'sign_up', 'publish'));
drop table counted_follows;
drop view work_followers;
