-- +goose Up
create table asset_legacy_paths (
    path       text primary key,
    asset_id   uuid not null references assets (id) on delete cascade,
    created_at timestamptz not null default now(),
    constraint asset_legacy_paths_path_check check (path <> '')
);

create index asset_legacy_paths_asset_idx on asset_legacy_paths (asset_id);

create table migration_preserved_records (
    id           uuid primary key,
    source_table text  not null,
    source_id    text  not null,
    asset_id     uuid  references assets (id) on delete cascade,
    owner_id     uuid  references users (id) on delete set null,
    payload      jsonb not null,
    constraint migration_preserved_records_source_check check (source_table <> '' and source_id <> ''),
    unique (source_table, source_id)
);

create index migration_preserved_records_asset_idx
    on migration_preserved_records (asset_id)
 where asset_id is not null;

create table migration_legacy_counters (
    asset_id   uuid primary key references assets (id) on delete cascade,
    downloads  integer not null,
    views      integer not null,
    favorites  integer not null,
    updated_at timestamptz not null
);

create table migration_staged_media (
    source    text primary key,
    blob_id   uuid    not null references blobs (id) on delete cascade,
    width     integer not null,
    height    integer not null,
    staged_at timestamptz not null default now(),
    constraint migration_staged_media_source_check check (source <> ''),
    constraint migration_staged_media_size_check check (width > 0 and height > 0)
);

-- +goose Down
drop table migration_staged_media;
drop table migration_legacy_counters;
drop table migration_preserved_records;
drop table asset_legacy_paths;
