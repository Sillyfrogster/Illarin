-- +goose Up

-- A linked instance is a connected app, a link request is a connection
-- request, a scope is a permission, an instance credential is app credentials,
-- a delivery is a send, and the installed library is the app's library. An app
-- is an app rather than an application, and the formats it accepts are formats
-- rather than targets.

alter table link_requests rename to connection_requests;
alter table link_authorizations rename to connection_authorizations;
alter table link_rate_limits rename to connection_rate_limits;
alter table linked_instances rename to connected_apps;
alter table instance_access_tokens rename to app_access_tokens;
alter table instance_refresh_history rename to app_refresh_history;
alter table instance_deliveries rename to sends;
alter table instance_library_entries rename to app_library_entries;

alter table connection_requests rename column application_name to app_name;
alter table connection_requests rename column instance_name to name;
alter table connection_requests rename column application_version to app_version;
alter table connection_requests rename column accepted_targets to accepted_formats;
alter table connection_requests rename column scopes to permissions;

alter table connection_authorizations rename column application_name to app_name;
alter table connection_authorizations rename column instance_name to name;
alter table connection_authorizations rename column application_version to app_version;
alter table connection_authorizations rename column accepted_targets to accepted_formats;
alter table connection_authorizations rename column scopes to permissions;

alter table connected_apps rename column application_name to app_name;
alter table connected_apps rename column instance_name to name;
alter table connected_apps rename column application_version to app_version;
alter table connected_apps rename column library_application_version to library_app_version;
alter table connected_apps rename column accepted_targets to accepted_formats;
alter table connected_apps rename column scopes to permissions;
alter table connected_apps rename column linked_at to connected_at;

alter table app_access_tokens rename column instance_id to connected_app_id;
alter table app_refresh_history rename column instance_id to connected_app_id;
alter table sends rename column instance_id to connected_app_id;
alter table sends rename column chosen_target to chosen_format;
alter table app_library_entries rename column instance_id to connected_app_id;

-- The permission to receive works says work, so the stored sets move with the
-- check that guards them.
alter table connection_requests drop constraint link_requests_scopes_check;
alter table connection_authorizations drop constraint link_authorizations_scopes_check;
alter table connected_apps drop constraint linked_instances_scopes_check;
drop function is_instance_scope_set(text[]);

update connection_requests set permissions = array_replace(permissions, 'asset:receive', 'work:receive');
update connection_authorizations set permissions = array_replace(permissions, 'asset:receive', 'work:receive');
update connected_apps set permissions = array_replace(permissions, 'asset:receive', 'work:receive');

-- +goose StatementBegin
create function is_app_permission_set(permissions text[]) returns boolean
    language sql immutable as $$
    select permissions <@ array['work:receive', 'library:sync']
       and cardinality(permissions) > 0
       and cardinality(permissions) = (select count(distinct permission) from unnest(permissions) as permission);
$$;
-- +goose StatementEnd

alter table connection_requests add constraint connection_requests_permissions_check
    check (is_app_permission_set(permissions));
alter table connection_authorizations add constraint connection_authorizations_permissions_check
    check (is_app_permission_set(permissions));
alter table connected_apps add constraint connected_apps_permissions_check
    check (is_app_permission_set(permissions));

-- Which apps may receive private prompts is checked against the app registry
-- in Go, so the database no longer names an app.
alter table protected_delivery_apps drop constraint protected_delivery_apps_known_check;

-- +goose StatementBegin
create function connect_word(old text) returns text language plpgsql immutable as $$
declare
    renamed text := old;
begin
    renamed := regexp_replace(renamed, '^link_requests_', 'connection_requests_');
    renamed := regexp_replace(renamed, '^link_authorizations_', 'connection_authorizations_');
    renamed := regexp_replace(renamed, '^link_rate_limits_', 'connection_rate_limits_');
    renamed := regexp_replace(renamed, '^linked_instances_', 'connected_apps_');
    renamed := regexp_replace(renamed, '^instance_access_tokens_', 'app_access_tokens_');
    renamed := regexp_replace(renamed, '^instance_refresh_history_', 'app_refresh_history_');
    renamed := regexp_replace(renamed, '^instance_deliveries_', 'sends_');
    renamed := regexp_replace(renamed, '^instance_library_entries_', 'app_library_entries_');
    renamed := replace(renamed, 'application_name', 'app_name');
    renamed := replace(renamed, 'application_version', 'app_version');
    renamed := replace(renamed, 'instance_name', 'name');
    renamed := replace(renamed, 'accepted_targets', 'accepted_formats');
    renamed := replace(renamed, 'scopes', 'permissions');
    renamed := replace(renamed, 'instance_id', 'connected_app_id');
    renamed := replace(renamed, 'linked_at', 'connected_at');
    renamed := replace(renamed, 'declaration_state', 'capabilities_state');
    renamed := regexp_replace(renamed, '^sends_target_check$', 'sends_format_check');
    return renamed;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
do $$
declare
    subject record;
begin
    for subject in
        select held.conrelid::regclass::text as on_table, held.conname as name
          from pg_constraint held
         where held.conrelid::regclass::text in (
                   'connection_requests', 'connection_authorizations', 'connection_rate_limits',
                   'connected_apps', 'app_access_tokens', 'app_refresh_history', 'sends',
                   'app_library_entries')
           and connect_word(held.conname) <> held.conname
    loop
        execute format('alter table %s rename constraint %I to %I',
            subject.on_table, subject.name, connect_word(subject.name));
    end loop;

    for subject in
        select indexed.indexname as name
          from pg_indexes indexed
         where indexed.schemaname = 'public'
           and indexed.tablename in (
                   'connection_requests', 'connection_authorizations', 'connection_rate_limits',
                   'connected_apps', 'app_access_tokens', 'app_refresh_history', 'sends',
                   'app_library_entries')
           and connect_word(indexed.indexname) <> indexed.indexname
    loop
        execute format('alter index public.%I rename to %I', subject.name, connect_word(subject.name));
    end loop;
end $$;
-- +goose StatementEnd

drop function connect_word(text);

alter table download_events rename column export_target to format;
alter table download_events rename constraint download_events_export_target_check to download_events_format_check;
alter table download_events rename constraint download_events_export_target_not_null to download_events_format_not_null;

-- +goose Down

alter table download_events rename constraint download_events_format_not_null to download_events_export_target_not_null;
alter table download_events rename constraint download_events_format_check to download_events_export_target_check;
alter table download_events rename column format to export_target;

alter table protected_delivery_apps add constraint protected_delivery_apps_known_check
    check (app = 'lumiverse');

-- +goose StatementBegin
create function link_word(new_name text) returns text language plpgsql immutable as $$
declare
    renamed text := new_name;
begin
    renamed := regexp_replace(renamed, '^sends_format_check$', 'sends_target_check');
    renamed := replace(renamed, 'capabilities_state', 'declaration_state');
    renamed := replace(renamed, 'connected_at', 'linked_at');
    renamed := replace(renamed, 'connected_app_id', 'instance_id');
    renamed := replace(renamed, 'accepted_formats', 'accepted_targets');
    renamed := replace(renamed, 'permissions', 'scopes');
    renamed := regexp_replace(renamed, '^(connection_requests|connection_authorizations)_name_', '\1_instance_name_');
    renamed := regexp_replace(renamed, '^connected_apps_name_(check|length_check)$', 'connected_apps_instance_name_\1');
    renamed := replace(renamed, 'app_version', 'application_version');
    renamed := replace(renamed, 'app_name', 'application_name');
    renamed := regexp_replace(renamed, '^connection_requests_', 'link_requests_');
    renamed := regexp_replace(renamed, '^connection_authorizations_', 'link_authorizations_');
    renamed := regexp_replace(renamed, '^connection_rate_limits_', 'link_rate_limits_');
    renamed := regexp_replace(renamed, '^connected_apps_', 'linked_instances_');
    renamed := regexp_replace(renamed, '^app_access_tokens_', 'instance_access_tokens_');
    renamed := regexp_replace(renamed, '^app_refresh_history_', 'instance_refresh_history_');
    renamed := regexp_replace(renamed, '^sends_', 'instance_deliveries_');
    renamed := regexp_replace(renamed, '^app_library_entries_', 'instance_library_entries_');
    return renamed;
end;
$$;
-- +goose StatementEnd

alter table connection_requests drop constraint connection_requests_permissions_check;
alter table connection_authorizations drop constraint connection_authorizations_permissions_check;
alter table connected_apps drop constraint connected_apps_permissions_check;
drop function is_app_permission_set(text[]);

update connection_requests set permissions = array_replace(permissions, 'work:receive', 'asset:receive');
update connection_authorizations set permissions = array_replace(permissions, 'work:receive', 'asset:receive');
update connected_apps set permissions = array_replace(permissions, 'work:receive', 'asset:receive');

-- +goose StatementBegin
create function is_instance_scope_set(scopes text[]) returns boolean
    language sql immutable as $$
    select scopes <@ array['asset:receive', 'library:sync']
       and cardinality(scopes) > 0
       and cardinality(scopes) = (select count(distinct scope) from unnest(scopes) as scope);
$$;
-- +goose StatementEnd

-- +goose StatementBegin
do $$
declare
    subject record;
begin
    for subject in
        select held.conrelid::regclass::text as on_table, held.conname as name
          from pg_constraint held
         where held.conrelid::regclass::text in (
                   'connection_requests', 'connection_authorizations', 'connection_rate_limits',
                   'connected_apps', 'app_access_tokens', 'app_refresh_history', 'sends',
                   'app_library_entries')
           and link_word(held.conname) <> held.conname
    loop
        execute format('alter table %s rename constraint %I to %I',
            subject.on_table, subject.name, link_word(subject.name));
    end loop;

    for subject in
        select indexed.indexname as name
          from pg_indexes indexed
         where indexed.schemaname = 'public'
           and indexed.tablename in (
                   'connection_requests', 'connection_authorizations', 'connection_rate_limits',
                   'connected_apps', 'app_access_tokens', 'app_refresh_history', 'sends',
                   'app_library_entries')
           and link_word(indexed.indexname) <> indexed.indexname
    loop
        execute format('alter index public.%I rename to %I', subject.name, link_word(subject.name));
    end loop;
end $$;
-- +goose StatementEnd

drop function link_word(text);

alter table app_library_entries rename column connected_app_id to instance_id;
alter table sends rename column chosen_format to chosen_target;
alter table sends rename column connected_app_id to instance_id;
alter table app_refresh_history rename column connected_app_id to instance_id;
alter table app_access_tokens rename column connected_app_id to instance_id;

alter table connected_apps rename column connected_at to linked_at;
alter table connected_apps rename column permissions to scopes;
alter table connected_apps rename column accepted_formats to accepted_targets;
alter table connected_apps rename column library_app_version to library_application_version;
alter table connected_apps rename column app_version to application_version;
alter table connected_apps rename column name to instance_name;
alter table connected_apps rename column app_name to application_name;

alter table connection_authorizations rename column permissions to scopes;
alter table connection_authorizations rename column accepted_formats to accepted_targets;
alter table connection_authorizations rename column app_version to application_version;
alter table connection_authorizations rename column name to instance_name;
alter table connection_authorizations rename column app_name to application_name;

alter table connection_requests rename column permissions to scopes;
alter table connection_requests rename column accepted_formats to accepted_targets;
alter table connection_requests rename column app_version to application_version;
alter table connection_requests rename column name to instance_name;
alter table connection_requests rename column app_name to application_name;

alter table connection_requests add constraint link_requests_scopes_check
    check (is_instance_scope_set(scopes));
alter table connection_authorizations add constraint link_authorizations_scopes_check
    check (is_instance_scope_set(scopes));
alter table connected_apps add constraint linked_instances_scopes_check
    check (is_instance_scope_set(scopes));

alter table app_library_entries rename to instance_library_entries;
alter table sends rename to instance_deliveries;
alter table app_refresh_history rename to instance_refresh_history;
alter table app_access_tokens rename to instance_access_tokens;
alter table connected_apps rename to linked_instances;
alter table connection_rate_limits rename to link_rate_limits;
alter table connection_authorizations rename to link_authorizations;
alter table connection_requests rename to link_requests;
