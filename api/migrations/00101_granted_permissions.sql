-- +goose Up
-- A permission set may now be empty, since an app may ask for nothing and an owner may grant nothing.
-- +goose StatementBegin
create or replace function is_app_permission_set(permissions text[]) returns boolean
    language sql immutable as $$
    select permissions <@ array['work:receive', 'library:sync']
       and cardinality(permissions) = (select count(distinct permission) from unnest(permissions) as permission);
$$;
-- +goose StatementEnd

-- A request's permissions are what the app asked for; the owner's choice is stored apart and set only on approval.
alter table connection_requests add column granted_permissions text[]
    constraint connection_requests_granted_permissions_check
    check (granted_permissions is null or is_app_permission_set(granted_permissions));
alter table connection_authorizations add column granted_permissions text[]
    constraint connection_authorizations_granted_permissions_check
    check (granted_permissions is null or is_app_permission_set(granted_permissions));

-- +goose Down
alter table connection_authorizations drop column granted_permissions;
alter table connection_requests drop column granted_permissions;

-- +goose StatementBegin
create or replace function is_app_permission_set(permissions text[]) returns boolean
    language sql immutable as $$
    select permissions <@ array['work:receive', 'library:sync']
       and cardinality(permissions) > 0
       and cardinality(permissions) = (select count(distinct permission) from unnest(permissions) as permission);
$$;
-- +goose StatementEnd
