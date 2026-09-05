-- +goose Up
-- A schedule names an edition and an instant. It never holds the edition itself,
-- so editing the working copy afterwards cannot reach what will go live.
create table post_schedules (
    id               uuid primary key,
    post_id          uuid not null references posts (id) on delete cascade,
    revision_id      uuid not null references post_revisions (id) on delete restrict,
    due_at           timestamptz not null,
    state            text not null default 'pending',
    created_by       uuid references users (id) on delete set null,
    attempts         integer not null default 0,
    lease_token      uuid,
    lease_expires_at timestamptz,
    stopped_because  text,
    created_at       timestamptz not null default now(),
    updated_at       timestamptz not null default now(),
    settled_at       timestamptz,
    constraint post_schedules_state_check
        check (state in ('pending', 'publishing', 'published', 'cancelled', 'stopped')),
    constraint post_schedules_attempts_check check (attempts >= 0),
    constraint post_schedules_settled_check
        check ((state in ('pending', 'publishing')) = (settled_at is null)),
    constraint post_schedules_stopped_check
        check (stopped_because is null or state = 'stopped'),
    constraint post_schedules_lease_check
        check ((lease_token is null) = (lease_expires_at is null))
);

create unique index post_schedules_waiting_idx on post_schedules (post_id)
    where state in ('pending', 'publishing');

create index post_schedules_due_idx on post_schedules (due_at)
    where state in ('pending', 'publishing');

create index post_schedules_post_idx on post_schedules (post_id, created_at desc);

alter table publication_audits add column schedule_id uuid;

-- +goose Down
alter table publication_audits drop column schedule_id;
drop table post_schedules;
