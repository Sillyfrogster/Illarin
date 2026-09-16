-- +goose Up
create table notification_events (
    id          uuid primary key,
    type        text not null,
    account_id  uuid not null references users (id) on delete cascade,
    asset_id    uuid references assets (id) on delete cascade,
    words       jsonb not null check (jsonb_typeof(words) = 'object'),
    recorded_at timestamptz not null default now(),
    constraint notification_events_type_check check (type in ('asset_withheld', 'asset_restored'))
);

create index notification_events_waiting on notification_events (recorded_at);

create table notifications (
    id         uuid primary key,
    account_id uuid not null references users (id) on delete cascade,
    type       text not null,
    asset_id   uuid references assets (id) on delete cascade,
    words      jsonb not null check (jsonb_typeof(words) = 'object'),
    created_at timestamptz not null,
    read_at    timestamptz,
    constraint notifications_type_check check (type in ('asset_withheld', 'asset_restored'))
);

create index notifications_newest on notifications (account_id, created_at desc, id desc);
create index notifications_unread on notifications (account_id) where read_at is null;
create index notifications_age on notifications (created_at);

-- +goose Down
drop table notifications;
drop table notification_events;
