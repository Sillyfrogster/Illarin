-- +goose Up
delete from publication_media
 where id in (select mark_media_id from profile_distinctions where mark_media_id is not null);

drop table profile_distinction_audits;
drop table profile_distinction_assignments;
drop table profile_distinctions;

alter table post_bylines
    drop column distinctions,
    drop column positions;

-- +goose Down
alter table post_bylines
    add column positions jsonb not null default '[]',
    add column distinctions jsonb not null default '[]';

create table profile_distinctions (
    id            uuid primary key,
    form          text not null,
    name          text not null,
    explanation   text not null default '',
    mark_media_id uuid references publication_media (id) on delete set null,
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

insert into profile_distinctions (id, form, name, explanation, position)
values ('9d3f1c00-0000-4000-8000-000000000021', 'badge', 'Verified App Contributor',
        'Publishes official updates for a project on Illarin.', 0);
