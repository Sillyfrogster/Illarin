-- +goose Up
create table asset_update_events (
    id               uuid primary key,
    asset_id         uuid not null references assets (id) on delete cascade,
    snapshot_id      uuid not null references asset_snapshots (id) on delete cascade,
    type             text not null,
    occurred_at      timestamptz not null default now(),
    unlisted_consent boolean not null default false,
    payload          jsonb not null,
    constraint asset_update_events_type_check check (type in ('asset.update.published.v1')),
    constraint asset_update_events_snapshot_key unique (snapshot_id)
);

create index asset_update_events_asset_idx on asset_update_events (asset_id, occurred_at desc);

create table asset_update_deliveries (
    id               uuid primary key,
    event_id         uuid not null references asset_update_events (id) on delete cascade,
    destination_id   uuid references asset_update_destinations (id) on delete set null,
    destination_name text not null,
    destination_kind text not null,
    state            text not null default 'pending',
    settled_reason   text,
    message_id       text,
    run              integer not null default 1,
    attempts         integer not null default 0,
    lease_token      uuid,
    lease_expires_at timestamptz,
    due_at           timestamptz not null default now(),
    settled_at       timestamptz,
    created_at       timestamptz not null default now(),
    updated_at       timestamptz not null default now(),
    constraint asset_update_deliveries_name_check
        check (char_length(destination_name) between 1 and 48),
    constraint asset_update_deliveries_kind_check
        check (destination_kind in ('webhook', 'discord')),
    constraint asset_update_deliveries_state_check
        check (state in ('pending', 'sending', 'delivered', 'failed', 'unconfirmed')),
    constraint asset_update_deliveries_run_check check (run >= 1),
    constraint asset_update_deliveries_attempts_check check (attempts >= 0),
    constraint asset_update_deliveries_settled_check
        check ((state in ('pending', 'sending')) = (settled_at is null)),
    constraint asset_update_deliveries_reason_check
        check ((settled_reason is null) = (settled_at is null)),
    constraint asset_update_deliveries_settled_reason_check
        check (settled_reason is null or settled_reason in (
            'arrived', 'exhausted', 'refused', 'gone', 'removed', 'disabled', 'moved',
            'unconfirmed', 'withheld', 'withdrawn', 'unlisted', 'deleted'
        )),
    constraint asset_update_deliveries_lease_check
        check ((lease_token is null) = (lease_expires_at is null))
);

create index asset_update_deliveries_due_idx on asset_update_deliveries (due_at)
    where state in ('pending', 'sending');

create index asset_update_deliveries_event_idx on asset_update_deliveries (event_id);

create index asset_update_deliveries_destination_idx on asset_update_deliveries (destination_id)
    where destination_id is not null;

create unique index asset_update_deliveries_choice_idx
    on asset_update_deliveries (event_id, destination_id) where destination_id is not null;

create table asset_update_delivery_attempts (
    id           uuid primary key,
    delivery_id  uuid not null references asset_update_deliveries (id) on delete cascade,
    run          integer not null default 1,
    number       integer not null,
    outcome      text not null,
    status       integer,
    detail       text not null default '',
    took_ms      integer not null default 0,
    attempted_at timestamptz not null default now(),
    unique (delivery_id, number),
    constraint asset_update_delivery_attempts_run_check check (run >= 1),
    constraint asset_update_delivery_attempts_number_check check (number >= 1),
    constraint asset_update_delivery_attempts_outcome_check
        check (outcome in ('delivered', 'refused', 'unreachable', 'unconfirmed')),
    constraint asset_update_delivery_attempts_detail_check check (char_length(detail) <= 200)
);

create index asset_update_delivery_attempts_run_idx
    on asset_update_delivery_attempts (delivery_id, run);

-- +goose Down
drop table asset_update_delivery_attempts;
drop table asset_update_deliveries;
drop table asset_update_events;
