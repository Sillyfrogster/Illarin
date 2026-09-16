-- +goose Up
create table asset_watches (
    account_id uuid not null references users (id) on delete cascade,
    asset_id   uuid not null references assets (id) on delete cascade,
    state      text not null,
    set_at     timestamptz not null default now(),
    primary key (account_id, asset_id),
    constraint asset_watches_state_check check (state in ('watching', 'stopped'))
);

create index asset_watches_asset_id_idx on asset_watches (asset_id);

-- +goose Down
drop table asset_watches;
