-- +goose Up
create table post_slugs (
    slug        text primary key,
    post_id     uuid references posts (id) on delete set null,
    reserved_by uuid references users (id) on delete set null,
    reserved_at timestamptz not null default now(),
    constraint post_slugs_slug_check
        check (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' and char_length(slug) <= 80)
);

create index post_slugs_post_idx on post_slugs (post_id, reserved_at desc);

insert into post_slugs (slug, post_id, reserved_by, reserved_at)
select slug, id, author_id, published_at
  from posts
 where status = 'published' and slug is not null;

-- +goose Down
drop table post_slugs;
