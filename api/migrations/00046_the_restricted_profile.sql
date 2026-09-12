-- +goose Up
create table profile_restrictions (
    user_id       uuid primary key references users (id) on delete cascade,
    restricted_by uuid references users (id) on delete set null,
    reason        text not null,
    restricted_at timestamptz not null default now(),
    constraint profile_restrictions_reason_check
        check (char_length(btrim(reason)) between 1 and 500)
);

create table profile_restriction_audits (
    id          uuid primary key,
    actor_id    uuid references users (id) on delete set null,
    subject_id  uuid not null,
    action      text not null,
    reason      text not null default '',
    recorded_at timestamptz not null default now(),
    constraint profile_restriction_audits_action_check
        check (action in ('restrict', 'restore'))
);

create index profile_restriction_audits_recorded_at_idx
    on profile_restriction_audits (recorded_at desc);

-- +goose Down
drop table profile_restriction_audits;
drop table profile_restrictions;
