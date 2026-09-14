-- +goose Up
create table asset_vault_pictures (
    id         uuid primary key,
    asset_id   uuid not null references assets (id) on delete cascade,
    media_id   uuid references asset_media (id) on delete cascade,
    address    text not null,
    name       text not null default '',
    block_id   uuid,
    section    text not null default '',
    position   integer not null,
    created_at timestamptz not null default now()
);

create index asset_vault_pictures_asset_idx on asset_vault_pictures (asset_id, position);

-- +goose Down
drop table asset_vault_pictures;
