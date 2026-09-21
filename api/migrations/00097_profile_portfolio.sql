-- +goose Up

-- The banner sits over the profile with the avatar; its tint is the color the page is washed with.
alter table public_profiles add column banner_media_id uuid references profile_media (id) on delete set null;
alter table public_profiles add column banner_tint text
    constraint public_profiles_banner_tint_check check (banner_tint ~ '^#[0-9a-f]{6}$');

-- Up to four of the creator's own listed works, shown first on the profile in this order.
create table profile_featured_works (
    user_id  uuid not null references users (id) on delete cascade,
    position integer not null,
    work_id  uuid not null references works (id) on delete cascade,
    primary key (user_id, position),
    unique (user_id, work_id),
    constraint profile_featured_works_position_check check (position between 0 and 3)
);

-- An account's request to hear when a creator publishes a new work.
create table creator_follows (
    account_id uuid not null references users (id) on delete cascade,
    creator_id uuid not null references users (id) on delete cascade,
    created_at timestamptz not null default now(),
    primary key (account_id, creator_id),
    constraint creator_follows_not_self_check check (account_id <> creator_id)
);

create index creator_follows_creator_id_idx on creator_follows (creator_id);

alter table notification_events drop constraint notification_events_type_check;
alter table notification_events add constraint notification_events_type_check
    check (type in ('work_taken_down', 'work_restored', 'work_updated', 'work_published', 'profile_restricted', 'profile_restored'));
alter table notifications drop constraint notifications_type_check;
alter table notifications add constraint notifications_type_check
    check (type in ('work_taken_down', 'work_restored', 'work_updated', 'work_published', 'profile_restricted', 'profile_restored'));

-- +goose Down
delete from notifications where type = 'work_published';
delete from notification_events where type = 'work_published';
alter table notifications drop constraint notifications_type_check;
alter table notifications add constraint notifications_type_check
    check (type in ('work_taken_down', 'work_restored', 'work_updated', 'profile_restricted', 'profile_restored'));
alter table notification_events drop constraint notification_events_type_check;
alter table notification_events add constraint notification_events_type_check
    check (type in ('work_taken_down', 'work_restored', 'work_updated', 'profile_restricted', 'profile_restored'));
drop table creator_follows;
drop table profile_featured_works;
alter table public_profiles drop column banner_tint;
alter table public_profiles drop column banner_media_id;
