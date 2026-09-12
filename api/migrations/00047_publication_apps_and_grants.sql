-- +goose Up
alter table distinction_media rename to publication_media;
alter index distinction_media_blob_id_idx rename to publication_media_blob_id_idx;

create table publication_apps (
    id            uuid primary key,
    slug          text not null unique,
    name          text not null,
    home_url      text not null,
    mark_media_id uuid references publication_media (id) on delete set null,
    position      integer not null default 0,
    retired_at    timestamptz,
    created_at    timestamptz not null default now(),
    updated_at    timestamptz not null default now(),
    constraint publication_apps_slug_check
        check (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' and char_length(slug) <= 40),
    constraint publication_apps_name_check check (char_length(name) between 1 and 48),
    constraint publication_apps_home_url_check
        check (home_url like 'https://%' and char_length(home_url) <= 300)
);

create index publication_apps_order_idx on publication_apps (position, created_at);

create table publication_categories (
    id         uuid primary key,
    slug       text not null unique,
    label      text not null,
    position   integer not null default 0,
    retired_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint publication_categories_slug_check
        check (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' and char_length(slug) <= 40),
    constraint publication_categories_label_check check (char_length(label) between 1 and 48)
);

create index publication_categories_order_idx on publication_categories (position, created_at);

create table publication_grants (
    id                  uuid primary key,
    user_id             uuid not null references users (id) on delete cascade,
    app_id              uuid not null references publication_apps (id),
    default_category_id uuid not null references publication_categories (id),
    granted_by          uuid references users (id) on delete set null,
    granted_at          timestamptz not null default now(),
    revoked_at          timestamptz,
    active              boolean not null default true,
    constraint publication_grants_state_check check (active = (revoked_at is null))
);

create unique index publication_grants_active_idx
    on publication_grants (user_id, app_id) where active;

create index publication_grants_holder_idx on publication_grants (user_id, granted_at desc);

create table publication_grant_categories (
    grant_id    uuid not null references publication_grants (id) on delete cascade,
    category_id uuid not null references publication_categories (id),
    primary key (grant_id, category_id)
);

create table publication_audits (
    id          uuid primary key,
    actor_id    uuid references users (id) on delete set null,
    action      text not null,
    app_id      uuid,
    category_id uuid,
    grant_id    uuid,
    subject_id  uuid,
    recorded_at timestamptz not null default now(),
    constraint publication_audits_action_check check (action <> '')
);

create index publication_audits_recorded_at_idx on publication_audits (recorded_at desc);

insert into publication_apps (id, slug, name, home_url, position)
values ('9d3f1c00-0000-4000-8000-000000000001', 'illarin', 'Illarin', 'https://illarin.xyz', 0);

insert into publication_categories (id, slug, label, position)
values ('9d3f1c00-0000-4000-8000-000000000011', 'announcement', 'Announcement', 0),
       ('9d3f1c00-0000-4000-8000-000000000012', 'release', 'Release', 1),
       ('9d3f1c00-0000-4000-8000-000000000013', 'article', 'Article', 2);

insert into profile_distinctions (id, form, name, explanation, position)
select '9d3f1c00-0000-4000-8000-000000000021', 'badge', 'Verified App Contributor',
       'Publishes official updates for a project on Illarin.',
       coalesce((select max(position) + 1 from profile_distinctions where form = 'badge'), 0);

-- +goose Down
delete from profile_distinctions where id = '9d3f1c00-0000-4000-8000-000000000021';
drop table publication_audits;
drop table publication_grant_categories;
drop table publication_grants;
drop table publication_categories;
drop table publication_apps;
alter index publication_media_blob_id_idx rename to distinction_media_blob_id_idx;
alter table publication_media rename to distinction_media;
