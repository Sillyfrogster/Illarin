-- +goose Up
create table publication_destinations (
    id             uuid primary key,
    kind           text not null,
    name           text not null,
    host           text not null,
    address        bytea not null,
    signing_secret bytea not null,
    state          text not null default 'unverified',
    verified_at    timestamptz,
    disabled_at    timestamptz,
    created_by     uuid references users (id) on delete set null,
    created_at     timestamptz not null default now(),
    updated_at     timestamptz not null default now(),
    constraint publication_destinations_kind_check check (kind in ('webhook')),
    constraint publication_destinations_name_check check (char_length(name) between 1 and 48),
    constraint publication_destinations_host_check check (char_length(host) between 1 and 253),
    constraint publication_destinations_state_check
        check (state in ('unverified', 'active', 'disabled')),
    constraint publication_destinations_verified_check
        check (state <> 'active' or verified_at is not null),
    constraint publication_destinations_disabled_check
        check ((state = 'disabled') = (disabled_at is not null))
);

create index publication_destinations_order_idx on publication_destinations (name, created_at);

-- An app's baseline reaches every grant that names it. A grant that overrides
-- carries its own rows instead, which is how one contributor is narrowed
-- without narrowing the app.
create table publication_app_destinations (
    app_id         uuid not null references publication_apps (id) on delete cascade,
    destination_id uuid not null references publication_destinations (id) on delete cascade,
    by_default     boolean not null default false,
    primary key (app_id, destination_id)
);

create table publication_grant_destinations (
    grant_id       uuid not null references publication_grants (id) on delete cascade,
    destination_id uuid not null references publication_destinations (id) on delete cascade,
    by_default     boolean not null default false,
    primary key (grant_id, destination_id)
);

alter table publication_grants
    add column destinations_overridden boolean not null default false;

alter table publication_events add column note text not null default '';
alter table publication_events add constraint publication_events_note_check
    check (char_length(note) <= 500);

-- A delivery is the captured choice as much as it is the work, so it holds the
-- name it was sent under and survives a destination the authority later removes.
create table publication_deliveries (
    id               uuid primary key,
    event_id         uuid not null references publication_events (id) on delete cascade,
    destination_id   uuid references publication_destinations (id) on delete set null,
    destination_name text not null,
    state            text not null default 'pending',
    attempts         integer not null default 0,
    lease_token      uuid,
    lease_expires_at timestamptz,
    due_at           timestamptz not null default now(),
    settled_at       timestamptz,
    created_at       timestamptz not null default now(),
    updated_at       timestamptz not null default now(),
    constraint publication_deliveries_name_check
        check (char_length(destination_name) between 1 and 48),
    constraint publication_deliveries_state_check
        check (state in ('pending', 'sending', 'delivered', 'failed')),
    constraint publication_deliveries_attempts_check check (attempts >= 0),
    constraint publication_deliveries_settled_check
        check ((state in ('pending', 'sending')) = (settled_at is null)),
    constraint publication_deliveries_lease_check
        check ((lease_token is null) = (lease_expires_at is null))
);

create index publication_deliveries_due_idx on publication_deliveries (due_at)
    where state in ('pending', 'sending');

create index publication_deliveries_event_idx on publication_deliveries (event_id);

create unique index publication_deliveries_choice_idx
    on publication_deliveries (event_id, destination_id) where destination_id is not null;

alter table post_schedules add column note text not null default '';
alter table post_schedules add constraint post_schedules_note_check
    check (char_length(note) <= 500);

create table post_schedule_destinations (
    schedule_id    uuid not null references post_schedules (id) on delete cascade,
    destination_id uuid not null references publication_destinations (id) on delete cascade,
    primary key (schedule_id, destination_id)
);

create table publication_delivery_attempts (
    id           uuid primary key,
    delivery_id  uuid not null references publication_deliveries (id) on delete cascade,
    number       integer not null,
    outcome      text not null,
    status       integer,
    detail       text not null default '',
    took_ms      integer not null default 0,
    attempted_at timestamptz not null default now(),
    unique (delivery_id, number),
    constraint publication_delivery_attempts_number_check check (number >= 1),
    constraint publication_delivery_attempts_outcome_check
        check (outcome in ('delivered', 'refused', 'unreachable')),
    constraint publication_delivery_attempts_detail_check check (char_length(detail) <= 200)
);

alter table publication_audits add column destination_id uuid;
alter table publication_audits add column delivery_id uuid;

-- +goose Down
alter table publication_audits drop column delivery_id;
alter table publication_audits drop column destination_id;
drop table post_schedule_destinations;
alter table post_schedules drop constraint post_schedules_note_check;
alter table post_schedules drop column note;
drop table publication_delivery_attempts;
drop table publication_deliveries;
alter table publication_events drop constraint publication_events_note_check;
alter table publication_events drop column note;
alter table publication_grants drop column destinations_overridden;
drop table publication_grant_destinations;
drop table publication_app_destinations;
drop table publication_destinations;
