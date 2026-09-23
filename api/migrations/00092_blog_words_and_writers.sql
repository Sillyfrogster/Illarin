-- +goose Up

-- The blog's tables say blog. Access to write is one switch per account, so
-- the approvals that fixed an app, a category set and an integration policy
-- become writer rows, and the app registry and release fields go with them.

create table blog_writers (
    user_id     uuid primary key references users (id) on delete cascade,
    switched_by uuid references users (id) on delete set null,
    since       timestamptz not null default now()
);

insert into blog_writers (user_id, switched_by, since)
select distinct on (user_id) user_id, granted_by, granted_at
  from publication_grants
 where active
 order by user_id, granted_at;

alter table posts drop column grant_id;
alter table posts drop column release_app_id;
alter table posts drop column release_version;
alter table posts drop column release_url;
alter table post_revisions drop column release_app_id;
alter table post_revisions drop column release_version;
alter table post_revisions drop column release_url;
alter table post_bylines drop constraint post_bylines_app_check;
alter table post_bylines drop column app_id;
alter table post_bylines drop column app_slug;
alter table post_bylines drop column app_name;
alter table publication_audits drop column app_id;
alter table publication_audits drop column grant_id;
alter table publication_audits drop column token_id;

drop table if exists publication_idempotency;
drop table if exists publication_rate_limits;
drop table if exists publication_tokens;
drop table publication_grant_integrations;
drop table publication_grant_categories;
drop table publication_grants;
drop table publication_app_integrations;
drop table publication_apps;
drop table publication_media;

alter table publication_categories rename to blog_categories;
alter index publication_categories_order_idx rename to blog_categories_order_idx;
alter index publication_categories_slug_key rename to blog_categories_slug_key;
alter table blog_categories rename constraint publication_categories_pkey to blog_categories_pkey;
alter table blog_categories rename constraint publication_categories_slug_check to blog_categories_slug_check;
alter table blog_categories rename constraint publication_categories_label_check to blog_categories_label_check;

alter table publication_audits rename to blog_activity_log;
alter index publication_audits_recorded_at_idx rename to blog_activity_log_recorded_at_idx;
alter table blog_activity_log rename constraint publication_audits_pkey to blog_activity_log_pkey;
alter table blog_activity_log rename constraint publication_audits_credential_check to blog_activity_log_credential_check;
alter table blog_activity_log rename constraint publication_audits_state_check to blog_activity_log_state_check;
alter table blog_activity_log rename constraint publication_audits_next_state_check to blog_activity_log_next_state_check;

alter table publication_discord_repairs rename to blog_discord_repairs;
alter table blog_discord_repairs rename constraint publication_discord_repairs_pkey to blog_discord_repairs_pkey;
alter table blog_discord_repairs rename constraint publication_discord_repairs_id_fkey to blog_discord_repairs_id_fkey;

alter table post_withdrawals rename to post_unpublishings;
alter table post_unpublishings rename column withdrawn_by to unpublished_by;
alter table post_unpublishings rename column withdrawn_at to unpublished_at;
alter index post_withdrawals_post_idx rename to post_unpublishings_post_idx;
alter table post_unpublishings rename constraint post_withdrawals_pkey to post_unpublishings_pkey;
alter table post_unpublishings rename constraint post_withdrawals_explanation_check to post_unpublishings_explanation_check;
alter table post_unpublishings rename constraint post_withdrawals_reason_check to post_unpublishings_reason_check;
alter table post_unpublishings rename constraint post_withdrawals_withdrawn_by_fkey to post_unpublishings_unpublished_by_fkey;

alter table posts rename column document to body;
alter table posts rename column document_version to body_version;
alter table posts rename column social_media_id to link_card_media_id;
alter table posts rename constraint posts_social_media_id_fkey to posts_link_card_media_id_fkey;
alter table post_revisions rename column document to body;
alter table post_revisions rename column document_version to body_version;
alter table post_revisions rename column social_media_id to link_card_media_id;
alter table post_revisions rename constraint post_revisions_social_media_id_fkey to post_revisions_link_card_media_id_fkey;

alter table posts drop constraint posts_status_check;
alter table posts drop constraint posts_published_check;
update posts set status = 'unpublished' where status = 'withdrawn';
alter table posts add constraint posts_status_check
    check (status in ('draft', 'published', 'unpublished'));
alter table posts add constraint posts_published_check
    check ((published_at is not null and slug is not null) = (status in ('published', 'unpublished')));

alter table post_revisions drop constraint post_revisions_captured_for_check;
update post_revisions set captured_for = 'publish' where captured_for = 'publication';
alter table post_revisions add constraint post_revisions_captured_for_check
    check (captured_for in ('checkpoint', 'publish', 'schedule'));

alter table post_media drop constraint post_media_purpose_check;
update post_media set purpose = 'body' where purpose = 'document';
update post_media set purpose = 'link_card' where purpose = 'social';
alter table post_media add constraint post_media_purpose_check
    check (purpose in ('header', 'body', 'link_card'));

alter table blog_announcements drop constraint blog_announcements_type_check;
alter table blog_integrations drop constraint blog_integrations_announcements_check;
alter table blog_integrations drop constraint blog_integrations_announces_check;
update blog_announcements
   set type = replace(replace(type, 'publication.post.', 'blog.post.'), 'withdrawn', 'unpublished');
update blog_integrations
   set announcements = array(
       select replace(replace(one, 'publication.post.', 'blog.post.'), 'withdrawn', 'unpublished')
         from unnest(announcements) one);
alter table blog_announcements add constraint blog_announcements_type_check
    check (type in ('blog.post.published.v1', 'blog.post.updated.v1', 'blog.post.unpublished.v1'));
alter table blog_integrations add constraint blog_integrations_announcements_check
    check (announcements <@ array['blog.post.published.v1', 'blog.post.updated.v1', 'blog.post.unpublished.v1']);
alter table blog_integrations add constraint blog_integrations_announces_check
    check (type <> 'discord' or announcements = array['blog.post.published.v1']);

alter table blog_activity_log drop constraint blog_activity_log_state_check;
alter table blog_activity_log drop constraint blog_activity_log_next_state_check;
update blog_activity_log set before_state = 'unpublished' where before_state = 'withdrawn';
update blog_activity_log set after_state = 'unpublished' where after_state = 'withdrawn';
alter table blog_activity_log add constraint blog_activity_log_state_check
    check (before_state is null or before_state in ('draft', 'published', 'unpublished', 'deleted'));
alter table blog_activity_log add constraint blog_activity_log_next_state_check
    check (after_state is null or after_state in ('draft', 'published', 'unpublished', 'deleted'));
update blog_activity_log set action = 'post.unpublished' where action = 'post.withdrawn';
update blog_activity_log set action = 'writer.on' where action = 'grant.created';
update blog_activity_log set action = 'writer.off' where action = 'grant.revoked';

-- +goose Down

update blog_activity_log set action = 'grant.revoked' where action = 'writer.off';
update blog_activity_log set action = 'grant.created' where action = 'writer.on';
update blog_activity_log set action = 'post.withdrawn' where action = 'post.unpublished';
alter table blog_activity_log drop constraint blog_activity_log_next_state_check;
alter table blog_activity_log drop constraint blog_activity_log_state_check;
update blog_activity_log set after_state = 'withdrawn' where after_state = 'unpublished';
update blog_activity_log set before_state = 'withdrawn' where before_state = 'unpublished';
alter table blog_activity_log add constraint blog_activity_log_next_state_check
    check (after_state is null or after_state in ('draft', 'published', 'withdrawn', 'deleted'));
alter table blog_activity_log add constraint blog_activity_log_state_check
    check (before_state is null or before_state in ('draft', 'published', 'withdrawn', 'deleted'));

alter table blog_integrations drop constraint blog_integrations_announces_check;
alter table blog_integrations drop constraint blog_integrations_announcements_check;
alter table blog_announcements drop constraint blog_announcements_type_check;
update blog_integrations
   set announcements = array(
       select replace(replace(one, 'blog.post.', 'publication.post.'), 'unpublished', 'withdrawn')
         from unnest(announcements) one);
update blog_announcements
   set type = replace(replace(type, 'blog.post.', 'publication.post.'), 'unpublished', 'withdrawn');
alter table blog_integrations add constraint blog_integrations_announces_check
    check (type <> 'discord' or announcements = array['publication.post.published.v1']);
alter table blog_integrations add constraint blog_integrations_announcements_check
    check (announcements <@ array['publication.post.published.v1', 'publication.post.updated.v1', 'publication.post.withdrawn.v1']);
alter table blog_announcements add constraint blog_announcements_type_check
    check (type in ('publication.post.published.v1', 'publication.post.updated.v1', 'publication.post.withdrawn.v1'));

alter table post_media drop constraint post_media_purpose_check;
update post_media set purpose = 'social' where purpose = 'link_card';
update post_media set purpose = 'document' where purpose = 'body';
alter table post_media add constraint post_media_purpose_check
    check (purpose in ('header', 'document', 'social'));

alter table post_revisions drop constraint post_revisions_captured_for_check;
update post_revisions set captured_for = 'publication' where captured_for = 'publish';
alter table post_revisions add constraint post_revisions_captured_for_check
    check (captured_for in ('checkpoint', 'publication', 'schedule'));

alter table posts drop constraint posts_published_check;
alter table posts drop constraint posts_status_check;
update posts set status = 'withdrawn' where status = 'unpublished';
alter table posts add constraint posts_published_check
    check ((published_at is not null and slug is not null) = (status in ('published', 'withdrawn')));
alter table posts add constraint posts_status_check
    check (status in ('draft', 'published', 'withdrawn'));

alter table post_revisions rename constraint post_revisions_link_card_media_id_fkey to post_revisions_social_media_id_fkey;
alter table post_revisions rename column link_card_media_id to social_media_id;
alter table post_revisions rename column body_version to document_version;
alter table post_revisions rename column body to document;
alter table posts rename constraint posts_link_card_media_id_fkey to posts_social_media_id_fkey;
alter table posts rename column link_card_media_id to social_media_id;
alter table posts rename column body_version to document_version;
alter table posts rename column body to document;

alter table post_unpublishings rename constraint post_unpublishings_unpublished_by_fkey to post_withdrawals_withdrawn_by_fkey;
alter table post_unpublishings rename constraint post_unpublishings_reason_check to post_withdrawals_reason_check;
alter table post_unpublishings rename constraint post_unpublishings_explanation_check to post_withdrawals_explanation_check;
alter table post_unpublishings rename constraint post_unpublishings_pkey to post_withdrawals_pkey;
alter index post_unpublishings_post_idx rename to post_withdrawals_post_idx;
alter table post_unpublishings rename column unpublished_at to withdrawn_at;
alter table post_unpublishings rename column unpublished_by to withdrawn_by;
alter table post_unpublishings rename to post_withdrawals;

alter table blog_discord_repairs rename constraint blog_discord_repairs_id_fkey to publication_discord_repairs_id_fkey;
alter table blog_discord_repairs rename constraint blog_discord_repairs_pkey to publication_discord_repairs_pkey;
alter table blog_discord_repairs rename to publication_discord_repairs;

alter table blog_activity_log rename constraint blog_activity_log_next_state_check to publication_audits_next_state_check;
alter table blog_activity_log rename constraint blog_activity_log_state_check to publication_audits_state_check;
alter table blog_activity_log rename constraint blog_activity_log_credential_check to publication_audits_credential_check;
alter table blog_activity_log rename constraint blog_activity_log_pkey to publication_audits_pkey;
alter index blog_activity_log_recorded_at_idx rename to publication_audits_recorded_at_idx;
alter table blog_activity_log rename to publication_audits;

alter table blog_categories rename constraint blog_categories_label_check to publication_categories_label_check;
alter table blog_categories rename constraint blog_categories_slug_check to publication_categories_slug_check;
alter table blog_categories rename constraint blog_categories_pkey to publication_categories_pkey;
alter index blog_categories_slug_key rename to publication_categories_slug_key;
alter index blog_categories_order_idx rename to publication_categories_order_idx;
alter table blog_categories rename to publication_categories;

-- The dropped tables come back empty: their rows described approvals, apps
-- and tokens the code no longer has, and the writer rows carry the access on.

create table publication_media (
    id         uuid primary key,
    blob_id    uuid references blobs (id),
    width      integer not null,
    height     integer not null,
    created_at timestamptz not null default now()
);

create index publication_media_blob_id_idx on publication_media (blob_id);

create table publication_apps (
    id            uuid primary key,
    slug          text not null unique,
    name          text not null,
    home_url      text not null,
    mark_media_id uuid references publication_media (id) on delete set null,
    position      integer not null default 0,
    retired_at    timestamptz,
    created_at    timestamptz not null default now(),
    updated_at    timestamptz not null default now()
);

create table publication_grants (
    id                      uuid primary key,
    user_id                 uuid not null references users (id) on delete cascade,
    app_id                  uuid not null references publication_apps (id),
    default_category_id     uuid not null references publication_categories (id),
    granted_by              uuid references users (id) on delete set null,
    granted_at              timestamptz not null default now(),
    revoked_at              timestamptz,
    active                  boolean not null default true,
    integrations_overridden boolean not null default false
);

create table publication_grant_categories (
    grant_id    uuid not null references publication_grants (id) on delete cascade,
    category_id uuid not null references publication_categories (id),
    primary key (grant_id, category_id)
);

create table publication_grant_integrations (
    grant_id       uuid not null references publication_grants (id) on delete cascade,
    integration_id uuid not null references blog_integrations (id) on delete cascade,
    by_default     boolean not null default false,
    primary key (grant_id, integration_id)
);

create table publication_app_integrations (
    app_id         uuid not null references publication_apps (id) on delete cascade,
    integration_id uuid not null references blog_integrations (id) on delete cascade,
    by_default     boolean not null default false,
    primary key (app_id, integration_id)
);

alter table publication_audits add column app_id uuid;
alter table publication_audits add column grant_id uuid;
alter table publication_audits add column token_id uuid;
alter table post_bylines add column app_id uuid references publication_apps (id);
alter table post_bylines add column app_slug text;
alter table post_bylines add column app_name text;
alter table post_bylines add constraint post_bylines_app_check
    check ((app_id is null) = (app_slug is null) and (app_id is null) = (app_name is null));
alter table post_revisions add column release_app_id uuid references publication_apps (id);
alter table post_revisions add column release_version text;
alter table post_revisions add column release_url text;
alter table posts add column grant_id uuid references publication_grants (id) on delete restrict;
alter table posts add column release_app_id uuid references publication_apps (id);
alter table posts add column release_version text;
alter table posts add column release_url text;

drop table blog_writers;
