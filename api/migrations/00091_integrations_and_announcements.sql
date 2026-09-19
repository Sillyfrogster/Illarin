-- +goose Up

-- A creator's update destinations and the blog's publication destinations are
-- both integrations. What the queue holds is an announcement, one attempt per
-- integration, and each try an attempt makes is recorded under it.

alter table work_update_destinations rename to work_integrations;
alter table work_update_destination_defaults rename to work_integration_defaults;
alter table work_update_events rename to work_announcements;
alter table work_update_deliveries rename to work_announcement_attempts;
alter table work_update_delivery_attempts rename to work_announcement_tries;
alter table publication_destinations rename to blog_integrations;
alter table publication_events rename to blog_announcements;
alter table publication_deliveries rename to blog_announcement_attempts;
alter table publication_delivery_attempts rename to blog_announcement_tries;
alter table post_schedule_destinations rename to post_schedule_integrations;
alter table publication_grant_destinations rename to publication_grant_integrations;
alter table publication_app_destinations rename to publication_app_integrations;

alter table work_integration_defaults rename column destination_id to integration_id;
alter table work_announcement_attempts rename column event_id to announcement_id;
alter table work_announcement_attempts rename column destination_id to integration_id;
alter table work_announcement_attempts rename column destination_name to integration_name;
alter table work_announcement_attempts rename column destination_type to integration_type;
alter table work_announcement_attempts rename column attempts to tries;
alter table work_announcement_tries rename column delivery_id to attempt_id;
alter table blog_integrations rename column events to announcements;
alter table blog_announcement_attempts rename column event_id to announcement_id;
alter table blog_announcement_attempts rename column destination_id to integration_id;
alter table blog_announcement_attempts rename column destination_name to integration_name;
alter table blog_announcement_attempts rename column attempts to tries;
alter table blog_announcement_tries rename column delivery_id to attempt_id;
alter table post_schedule_integrations rename column destination_id to integration_id;
alter table publication_grant_integrations rename column destination_id to integration_id;
alter table publication_app_integrations rename column destination_id to integration_id;
alter table publication_grants rename column destinations_overridden to integrations_overridden;
alter table publication_audits rename column destination_id to integration_id;
alter table publication_audits rename column delivery_id to attempt_id;
alter table publication_discord_repairs rename column delivery_id to attempt_id;

-- +goose StatementBegin
create function integration_word(name text) returns text language sql immutable as $$
    select regexp_replace(regexp_replace(regexp_replace(regexp_replace(regexp_replace(
           regexp_replace(regexp_replace(regexp_replace(regexp_replace(regexp_replace(
           regexp_replace(regexp_replace(regexp_replace(regexp_replace(regexp_replace(
           regexp_replace(regexp_replace(regexp_replace(name,
        '^work_update_destination_defaults_', 'work_integration_defaults_'),
        '^work_update_destinations_', 'work_integrations_'),
        '^work_update_events_', 'work_announcements_'),
        '^work_update_delivery_attempts_', 'work_announcement_tries_'),
        '^work_update_deliveries_', 'work_announcement_attempts_'),
        '^publication_destinations_', 'blog_integrations_'),
        '^publication_events_', 'blog_announcements_'),
        '^publication_delivery_attempts_', 'blog_announcement_tries_'),
        '^publication_deliveries_', 'blog_announcement_attempts_'),
        '^post_schedule_destinations_', 'post_schedule_integrations_'),
        '^publication_grant_destinations_', 'publication_grant_integrations_'),
        '^publication_app_destinations_', 'publication_app_integrations_'),
        '_destinations_overridden_', '_integrations_overridden_'),
        '_destination_(id|name|type|idx)', '_integration_\1'),
        '_delivery_id', '_attempt_id'),
        '_event_(id|idx)', '_announcement_\1'),
        '^blog_integrations_events_', 'blog_integrations_announcements_'),
        '_attempts_(check|not_null)$', '_tries_\1')
$$;
-- +goose StatementEnd

-- +goose StatementBegin
do $$
declare
    subject record;
begin
    for subject in
        select holder.oid::regclass::text as on_table, held.conname as name
          from pg_constraint held
          join pg_class holder on holder.oid = held.conrelid
          join pg_namespace space on space.oid = holder.relnamespace
         where space.nspname = 'public'
           and holder.relname in ('work_integrations', 'work_integration_defaults', 'work_announcements',
                'work_announcement_attempts', 'work_announcement_tries', 'blog_integrations',
                'blog_announcements', 'blog_announcement_attempts', 'blog_announcement_tries',
                'post_schedule_integrations', 'publication_grant_integrations',
                'publication_app_integrations', 'publication_grants', 'publication_discord_repairs')
           and integration_word(held.conname) <> held.conname
    loop
        execute format('alter table %s rename constraint %I to %I',
            subject.on_table, subject.name, integration_word(subject.name));
    end loop;

    for subject in
        select indexed.indexname as name
          from pg_indexes indexed
         where indexed.schemaname = 'public'
           and indexed.tablename in ('work_integrations', 'work_integration_defaults', 'work_announcements',
                'work_announcement_attempts', 'work_announcement_tries', 'blog_integrations',
                'blog_announcements', 'blog_announcement_attempts', 'blog_announcement_tries',
                'post_schedule_integrations', 'publication_grant_integrations',
                'publication_app_integrations')
           and integration_word(indexed.indexname) <> indexed.indexname
    loop
        execute format('alter index public.%I rename to %I', subject.name, integration_word(subject.name));
    end loop;
end $$;
-- +goose StatementEnd

drop function integration_word(text);

update publication_audits
   set action = regexp_replace(action, '^destination\.', 'integration.')
 where action like 'destination.%';
update publication_audits set action = 'attempt.replayed' where action = 'delivery.replayed';
update publication_audits set action = 'grant.integrations.set' where action = 'grant.destinations.set';
update publication_audits set action = 'app.integrations.set' where action = 'app.destinations.set';

-- +goose Down

update publication_audits set action = 'app.destinations.set' where action = 'app.integrations.set';
update publication_audits set action = 'grant.destinations.set' where action = 'grant.integrations.set';
update publication_audits set action = 'delivery.replayed' where action = 'attempt.replayed';
update publication_audits
   set action = regexp_replace(action, '^integration\.', 'destination.')
 where action like 'integration.%';

-- +goose StatementBegin
create function destination_word(name text) returns text language sql immutable as $$
    select regexp_replace(regexp_replace(regexp_replace(regexp_replace(regexp_replace(
           regexp_replace(regexp_replace(regexp_replace(regexp_replace(regexp_replace(
           regexp_replace(regexp_replace(regexp_replace(regexp_replace(regexp_replace(
           regexp_replace(regexp_replace(regexp_replace(name,
        '^blog_integrations_announcements_', 'publication_destinations_events_'),
        '^work_integration_defaults_', 'work_update_destination_defaults_'),
        '^work_integrations_', 'work_update_destinations_'),
        '^work_announcements_', 'work_update_events_'),
        '^work_announcement_tries_', 'work_update_delivery_attempts_'),
        '^work_announcement_attempts_', 'work_update_deliveries_'),
        '^blog_integrations_', 'publication_destinations_'),
        '^blog_announcements_', 'publication_events_'),
        '^blog_announcement_tries_', 'publication_delivery_attempts_'),
        '^blog_announcement_attempts_', 'publication_deliveries_'),
        '^post_schedule_integrations_', 'post_schedule_destinations_'),
        '^publication_grant_integrations_', 'publication_grant_destinations_'),
        '^publication_app_integrations_', 'publication_app_destinations_'),
        '_integrations_overridden_', '_destinations_overridden_'),
        '_integration_(id|name|type|idx)', '_destination_\1'),
        '_attempt_id', '_delivery_id'),
        '_announcement_(id|idx)', '_event_\1'),
        '_tries_(check|not_null)$', '_attempts_\1')
$$;
-- +goose StatementEnd

-- +goose StatementBegin
do $$
declare
    subject record;
begin
    for subject in
        select holder.oid::regclass::text as on_table, held.conname as name
          from pg_constraint held
          join pg_class holder on holder.oid = held.conrelid
          join pg_namespace space on space.oid = holder.relnamespace
         where space.nspname = 'public'
           and holder.relname in ('work_integrations', 'work_integration_defaults', 'work_announcements',
                'work_announcement_attempts', 'work_announcement_tries', 'blog_integrations',
                'blog_announcements', 'blog_announcement_attempts', 'blog_announcement_tries',
                'post_schedule_integrations', 'publication_grant_integrations',
                'publication_app_integrations', 'publication_grants', 'publication_discord_repairs')
           and destination_word(held.conname) <> held.conname
    loop
        execute format('alter table %s rename constraint %I to %I',
            subject.on_table, subject.name, destination_word(subject.name));
    end loop;

    for subject in
        select indexed.indexname as name
          from pg_indexes indexed
         where indexed.schemaname = 'public'
           and indexed.tablename in ('work_integrations', 'work_integration_defaults', 'work_announcements',
                'work_announcement_attempts', 'work_announcement_tries', 'blog_integrations',
                'blog_announcements', 'blog_announcement_attempts', 'blog_announcement_tries',
                'post_schedule_integrations', 'publication_grant_integrations',
                'publication_app_integrations')
           and destination_word(indexed.indexname) <> indexed.indexname
    loop
        execute format('alter index public.%I rename to %I', subject.name, destination_word(subject.name));
    end loop;
end $$;
-- +goose StatementEnd

drop function destination_word(text);

alter table publication_discord_repairs rename column attempt_id to delivery_id;
alter table publication_audits rename column attempt_id to delivery_id;
alter table publication_audits rename column integration_id to destination_id;
alter table publication_grants rename column integrations_overridden to destinations_overridden;
alter table publication_app_integrations rename column integration_id to destination_id;
alter table publication_grant_integrations rename column integration_id to destination_id;
alter table post_schedule_integrations rename column integration_id to destination_id;
alter table blog_announcement_tries rename column attempt_id to delivery_id;
alter table blog_announcement_attempts rename column tries to attempts;
alter table blog_announcement_attempts rename column integration_name to destination_name;
alter table blog_announcement_attempts rename column integration_id to destination_id;
alter table blog_announcement_attempts rename column announcement_id to event_id;
alter table blog_integrations rename column announcements to events;
alter table work_announcement_tries rename column attempt_id to delivery_id;
alter table work_announcement_attempts rename column tries to attempts;
alter table work_announcement_attempts rename column integration_type to destination_type;
alter table work_announcement_attempts rename column integration_name to destination_name;
alter table work_announcement_attempts rename column integration_id to destination_id;
alter table work_announcement_attempts rename column announcement_id to event_id;
alter table work_integration_defaults rename column integration_id to destination_id;

alter table publication_app_integrations rename to publication_app_destinations;
alter table publication_grant_integrations rename to publication_grant_destinations;
alter table post_schedule_integrations rename to post_schedule_destinations;
alter table blog_announcement_tries rename to publication_delivery_attempts;
alter table blog_announcement_attempts rename to publication_deliveries;
alter table blog_announcements rename to publication_events;
alter table blog_integrations rename to publication_destinations;
alter table work_announcement_tries rename to work_update_delivery_attempts;
alter table work_announcement_attempts rename to work_update_deliveries;
alter table work_announcements rename to work_update_events;
alter table work_integration_defaults rename to work_update_destination_defaults;
alter table work_integrations rename to work_update_destinations;
