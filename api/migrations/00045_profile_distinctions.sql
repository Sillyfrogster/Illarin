-- +goose Up
create table publication_authorities (
    user_id     uuid primary key references users (id) on delete cascade,
    assigned_at timestamptz not null default now()
);

create table distinction_media (
    id         uuid primary key,
    blob_id    uuid references blobs (id),
    width      integer not null,
    height     integer not null,
    created_at timestamptz not null default now()
);

create index distinction_media_blob_id_idx on distinction_media (blob_id) where blob_id is not null;

create table profile_distinctions (
    id            uuid primary key,
    form          text not null,
    name          text not null,
    explanation   text not null default '',
    mark_media_id uuid references distinction_media (id) on delete set null,
    position      integer not null default 0,
    retired_at    timestamptz,
    created_at    timestamptz not null default now(),
    updated_at    timestamptz not null default now(),
    constraint profile_distinctions_form_check check (form in ('position', 'title', 'badge')),
    constraint profile_distinctions_name_check check (char_length(name) between 1 and 48),
    constraint profile_distinctions_explanation_check check (char_length(explanation) <= 200),
    constraint profile_distinctions_mark_check check (mark_media_id is null or form = 'badge')
);

create index profile_distinctions_order_idx on profile_distinctions (form, position, created_at);

create table profile_distinction_assignments (
    id             uuid primary key,
    user_id        uuid not null references users (id) on delete cascade,
    distinction_id uuid not null references profile_distinctions (id) on delete cascade,
    issued_by      uuid references users (id) on delete set null,
    source         text not null,
    position       integer not null default 0,
    assigned_at    timestamptz not null default now(),
    active         boolean not null default true,
    deactivated_at timestamptz,
    constraint profile_distinction_assignments_source_check
        check (source in ('manual', 'publication-grant')),
    constraint profile_distinction_assignments_state_check
        check (active = (deactivated_at is null))
);

create unique index profile_distinction_assignments_active_idx
    on profile_distinction_assignments (user_id, distinction_id) where active;

create index profile_distinction_assignments_user_idx
    on profile_distinction_assignments (user_id, position, assigned_at);

create table profile_distinction_audits (
    id             uuid primary key,
    actor_id       uuid references users (id) on delete set null,
    action         text not null,
    distinction_id uuid,
    assignment_id  uuid,
    subject_id     uuid,
    recorded_at    timestamptz not null default now(),
    constraint profile_distinction_audits_action_check check (action <> '')
);

create index profile_distinction_audits_recorded_at_idx
    on profile_distinction_audits (recorded_at desc);

-- +goose Down
drop table profile_distinction_audits;
drop table profile_distinction_assignments;
drop table profile_distinctions;
drop table distinction_media;
drop table publication_authorities;
