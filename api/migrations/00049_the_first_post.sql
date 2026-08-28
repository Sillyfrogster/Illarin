-- +goose Up
create table posts (
    id                uuid primary key,
    author_id         uuid not null references users (id) on delete restrict,
    grant_id          uuid references publication_grants (id) on delete restrict,
    category_id       uuid not null references publication_categories (id),
    status            text not null default 'draft',
    slug              text,
    title             text not null,
    summary           text not null default '',
    document          jsonb not null,
    document_version  integer not null,
    release_app_id    uuid references publication_apps (id),
    release_version   text,
    release_url       text,
    working_version   integer not null default 1,
    published_at      timestamptz,
    updated_public_at timestamptz,
    created_at        timestamptz not null default now(),
    updated_at        timestamptz not null default now(),
    constraint posts_status_check check (status in ('draft', 'published')),
    constraint posts_title_check check (char_length(title) between 1 and 160),
    constraint posts_summary_check check (char_length(summary) <= 320),
    constraint posts_slug_check
        check (slug is null or (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' and char_length(slug) <= 80)),
    constraint posts_release_version_check
        check (release_version is null or char_length(release_version) between 1 and 40),
    constraint posts_release_url_check
        check (release_url is null or (release_url like 'https://%' and char_length(release_url) <= 300)),
    constraint posts_working_version_check check (working_version >= 1),
    constraint posts_published_check
        check ((status = 'published') = (published_at is not null and slug is not null))
);

create unique index posts_slug_idx on posts (slug) where slug is not null;
create index posts_author_idx on posts (author_id, created_at desc);
create index posts_public_idx on posts (published_at desc) where status = 'published';

create table post_revisions (
    id               uuid primary key,
    post_id          uuid not null references posts (id) on delete cascade,
    number           integer not null,
    title            text not null,
    summary          text not null,
    slug             text not null,
    category_id      uuid not null references publication_categories (id),
    document         jsonb not null,
    document_version integer not null,
    release_app_id   uuid references publication_apps (id),
    release_version  text,
    release_url      text,
    captured_by      uuid references users (id) on delete set null,
    captured_at      timestamptz not null default now(),
    constraint post_revisions_number_check check (number >= 1),
    unique (post_id, number)
);

alter table posts add column public_revision_id uuid references post_revisions (id);

create table post_bylines (
    post_id       uuid primary key references posts (id) on delete cascade,
    account_id    uuid references users (id) on delete set null,
    handle        text not null,
    display_name  text not null default '',
    contact_email text not null default '',
    avatar_media_id uuid,
    positions     jsonb not null default '[]',
    distinctions  jsonb not null default '[]',
    app_id        uuid references publication_apps (id),
    app_slug      text,
    app_name      text,
    captured_at   timestamptz not null default now(),
    constraint post_bylines_handle_check check (char_length(handle) between 1 and 40),
    constraint post_bylines_app_check
        check ((app_id is null) = (app_slug is null) and (app_id is null) = (app_name is null))
);

create table publication_events (
    id          uuid primary key,
    post_id     uuid not null references posts (id) on delete cascade,
    revision_id uuid not null references post_revisions (id),
    type        text not null,
    occurred_at timestamptz not null default now(),
    constraint publication_events_type_check
        check (type in ('publication.post.published.v1', 'publication.post.updated.v1'))
);

create index publication_events_post_idx on publication_events (post_id, occurred_at desc);

alter table publication_audits add column credential text not null default 'session';
alter table publication_audits add column post_id uuid;
alter table publication_audits add column revision_id uuid;
alter table publication_audits add column before_state text;
alter table publication_audits add column after_state text;
alter table publication_audits add constraint publication_audits_credential_check
    check (credential in ('session', 'token', 'system'));

-- +goose Down
alter table publication_audits drop constraint publication_audits_credential_check;
alter table publication_audits drop column after_state;
alter table publication_audits drop column before_state;
alter table publication_audits drop column revision_id;
alter table publication_audits drop column post_id;
alter table publication_audits drop column credential;
drop table publication_events;
drop table post_bylines;
alter table posts drop column public_revision_id;
drop table post_revisions;
drop table posts;
