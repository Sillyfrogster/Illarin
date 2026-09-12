-- +goose Up
create table post_media (
    id         uuid primary key,
    post_id    uuid not null references posts (id) on delete cascade,
    blob_id    uuid references blobs (id),
    purpose    text not null,
    width      integer not null,
    height     integer not null,
    created_at timestamptz not null default now(),
    constraint post_media_purpose_check check (purpose in ('header', 'document', 'social'))
);

create index post_media_post_idx on post_media (post_id, created_at desc);
create index post_media_blob_idx on post_media (blob_id) where blob_id is not null;

create table post_media_uses (
    media_id    uuid not null references post_media (id) on delete cascade,
    post_id     uuid not null references posts (id) on delete cascade,
    revision_id uuid references post_revisions (id) on delete cascade
);

create unique index post_media_uses_working_idx
    on post_media_uses (media_id, post_id) where revision_id is null;
create unique index post_media_uses_revision_idx
    on post_media_uses (media_id, revision_id) where revision_id is not null;
create index post_media_uses_media_idx on post_media_uses (media_id);
create index post_media_uses_post_idx on post_media_uses (post_id);

alter table posts add column header_media_id uuid references post_media (id);
alter table posts add column header_alt text;
alter table posts add column header_caption text;
alter table posts add column social_media_id uuid references post_media (id);
alter table posts add constraint posts_header_alt_check
    check (header_media_id is null or char_length(coalesce(header_alt, '')) between 1 and 300);
alter table posts add constraint posts_header_caption_check
    check (header_caption is null or char_length(header_caption) <= 300);

alter table post_revisions add column header_media_id uuid references post_media (id);
alter table post_revisions add column header_alt text;
alter table post_revisions add column header_caption text;
alter table post_revisions add column social_media_id uuid references post_media (id);
alter table post_revisions add constraint post_revisions_header_alt_check
    check (header_media_id is null or char_length(coalesce(header_alt, '')) between 1 and 300);
alter table post_revisions add constraint post_revisions_header_caption_check
    check (header_caption is null or char_length(header_caption) <= 300);

-- +goose Down
alter table post_revisions drop constraint post_revisions_header_caption_check;
alter table post_revisions drop constraint post_revisions_header_alt_check;
alter table post_revisions drop column social_media_id;
alter table post_revisions drop column header_caption;
alter table post_revisions drop column header_alt;
alter table post_revisions drop column header_media_id;
alter table posts drop constraint posts_header_caption_check;
alter table posts drop constraint posts_header_alt_check;
alter table posts drop column social_media_id;
alter table posts drop column header_caption;
alter table posts drop column header_alt;
alter table posts drop column header_media_id;
drop table post_media_uses;
drop table post_media;
