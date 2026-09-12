-- +goose Up
alter table migration_legacy_counters rename column downloads to v1_downloads;
alter table migration_legacy_counters rename column views to v1_views;
alter table migration_legacy_counters rename column updated_at to v1_updated_at;

alter table migration_legacy_counters drop column favorites;

alter table migration_legacy_counters
    add column migrated_at timestamptz not null default now();

-- +goose StatementBegin
create function reject_legacy_counter_mutation() returns trigger
language plpgsql as $$
begin
    raise exception 'v1 legacy counters are frozen at the cutover';
end;
$$;
-- +goose StatementEnd

create trigger legacy_counters_are_immutable
before update on migration_legacy_counters
for each row execute function reject_legacy_counter_mutation();

create trigger legacy_counters_cannot_be_truncated
before truncate on migration_legacy_counters
for each statement execute function reject_legacy_counter_mutation();

-- +goose Down
drop trigger legacy_counters_cannot_be_truncated on migration_legacy_counters;
drop trigger legacy_counters_are_immutable on migration_legacy_counters;
drop function if exists reject_legacy_counter_mutation();
alter table migration_legacy_counters drop column migrated_at;
alter table migration_legacy_counters add column favorites integer not null default 0;
alter table migration_legacy_counters rename column v1_updated_at to updated_at;
alter table migration_legacy_counters rename column v1_views to views;
alter table migration_legacy_counters rename column v1_downloads to downloads;
