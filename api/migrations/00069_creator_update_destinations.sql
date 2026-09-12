-- +goose Up
create table asset_update_destinations (
    id uuid primary key,
    owner_id uuid not null references users (id) on delete cascade,
    kind text not null check (kind in ('webhook', 'discord')),
    name text not null check (char_length(name) between 1 and 48),
    host text not null,
    address bytea not null,
    signing_secret bytea,
    signing_secret_set_at timestamptz,
    previous_secret bytea,
    previous_secret_until timestamptz,
    guild_id text,
    channel_id text,
    state text not null default 'unverified' check (state in ('unverified', 'active', 'disabled')),
    verified_at timestamptz,
    disabled_at timestamptz,
    version bigint not null default 1,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    check ((kind = 'webhook') = (signing_secret is not null)),
    check ((kind = 'webhook') = (signing_secret_set_at is not null)),
    check ((kind = 'discord') = (guild_id is not null and channel_id is not null)),
    check ((previous_secret is null) = (previous_secret_until is null)),
    check (state <> 'active' or verified_at is not null),
    check ((state = 'disabled') = (disabled_at is not null))
);

create index asset_update_destinations_owner_idx on asset_update_destinations (owner_id, name, id);

create table asset_update_destination_defaults (
    asset_id uuid not null references assets (id) on delete cascade,
    destination_id uuid not null references asset_update_destinations (id) on delete cascade,
    primary key (asset_id, destination_id)
);

-- +goose Down
drop table asset_update_destination_defaults;
drop table asset_update_destinations;
