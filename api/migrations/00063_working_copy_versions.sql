-- +goose Up
alter table assets add column working_copy_version bigint not null default 1 check (working_copy_version > 0);
alter table ingest_operations add column candidate_version bigint;
alter table ingest_operations drop constraint ingest_operations_failure_reason_check;
alter table ingest_operations add constraint ingest_operations_failure_reason_check
check (failure_reason is null or failure_reason in (
    'malformed_input', 'unsupported_format', 'unsupported_version',
    'safety_violation', 'wrong_kind', 'limit_exceeded', 'internal_failure',
    'working_copy_conflict', 'asset_unavailable'
));

-- +goose StatementBegin
create function invalidate_asset_candidate() returns trigger language plpgsql as $$
begin
    if row(new.owner_id, new.discovery, new.withheld_at, new.deleted_at, new.lifecycle, new.published_snapshot_id)
       is distinct from row(old.owner_id, old.discovery, old.withheld_at, old.deleted_at, old.lifecycle, old.published_snapshot_id) then
        new.working_copy_version := old.working_copy_version + 1;
    end if;
    return new;
end;
$$;
-- +goose StatementEnd
create trigger invalidate_asset_candidate before update on assets
for each row execute function invalidate_asset_candidate();

-- +goose StatementBegin
create function invalidate_protected_asset_candidate() returns trigger language plpgsql as $$
begin
    update assets set working_copy_version = working_copy_version + 1
    where id = coalesce(new.asset_id, old.asset_id);
    return null;
end;
$$;
-- +goose StatementEnd
create trigger invalidate_protected_candidate after insert or update or delete on protected_content
for each row execute function invalidate_protected_asset_candidate();
create trigger invalidate_delivery_candidate after insert or update or delete on protected_delivery_apps
for each row execute function invalidate_protected_asset_candidate();

-- +goose Down
update ingest_operations set failure_reason = 'internal_failure'
where failure_reason in ('working_copy_conflict', 'asset_unavailable');
alter table ingest_operations drop constraint ingest_operations_failure_reason_check;
alter table ingest_operations add constraint ingest_operations_failure_reason_check
check (failure_reason is null or failure_reason in (
    'malformed_input', 'unsupported_format', 'unsupported_version',
    'safety_violation', 'wrong_kind', 'limit_exceeded', 'internal_failure'
));
drop trigger invalidate_delivery_candidate on protected_delivery_apps;
drop trigger invalidate_protected_candidate on protected_content;
drop function invalidate_protected_asset_candidate();
drop trigger invalidate_asset_candidate on assets;
drop function invalidate_asset_candidate();
alter table ingest_operations drop column candidate_version;
alter table assets drop column working_copy_version;
