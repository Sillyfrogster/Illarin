-- +goose Up
create table extension_release_sources (
    work_id uuid primary key references works (id) on delete cascade,
    repository text not null,
    proof text not null,
    verified_at timestamptz,
    attachment text,
    include_prereleases boolean not null default false,
    next_check_at timestamptz not null default now(),
    last_error text,
    check ((attachment is null) or (length(attachment) between 1 and 255))
);

create unique index extension_release_sources_verified_repository on extension_release_sources (repository)
    where verified_at is not null;

create table extension_release_imports (
    work_id uuid not null references extension_release_sources (work_id) on delete cascade,
    release_id bigint not null,
    tag text not null,
    published_at timestamptz not null,
    asset_id bigint,
    status text not null default 'queued' check (status in ('queued', 'held', 'failed', 'published')),
    failure text,
    version_number integer,
    created_at timestamptz not null default now(),
    primary key (work_id, release_id)
);

create index extension_release_imports_queue on extension_release_imports (published_at, release_id)
    where status = 'queued';

alter table notification_events drop constraint notification_events_type_check;
alter table notification_events add constraint notification_events_type_check
    check (type in ('work_taken_down', 'work_restored', 'profile_restricted', 'profile_restored', 'work_updated', 'work_published', 'github_release_held'));
alter table notifications drop constraint notifications_type_check;
alter table notifications add constraint notifications_type_check
    check (type in ('work_taken_down', 'work_restored', 'profile_restricted', 'profile_restored', 'work_updated', 'work_published', 'github_release_held'));

-- +goose Down
alter table notification_events drop constraint notification_events_type_check;
alter table notification_events add constraint notification_events_type_check
    check (type in ('work_taken_down', 'work_restored', 'profile_restricted', 'profile_restored', 'work_updated', 'work_published'));
alter table notifications drop constraint notifications_type_check;
alter table notifications add constraint notifications_type_check
    check (type in ('work_taken_down', 'work_restored', 'profile_restricted', 'profile_restored', 'work_updated', 'work_published'));
drop table extension_release_imports;
drop table extension_release_sources;
