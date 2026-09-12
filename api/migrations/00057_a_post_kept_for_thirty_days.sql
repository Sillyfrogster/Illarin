-- +goose Up
alter table posts add column deleted_at timestamptz;
alter table posts add column recoverable_until timestamptz;
alter table posts add column deleted_by uuid references users (id) on delete set null;
alter table posts add constraint posts_deleted_check
    check ((deleted_at is null) = (recoverable_until is null));

create index posts_recoverable_idx on posts (recoverable_until) where deleted_at is not null;

alter table publication_audits drop constraint publication_audits_state_check;
alter table publication_audits add constraint publication_audits_state_check
    check (before_state is null
           or before_state in ('draft', 'published', 'withdrawn', 'deleted'));
alter table publication_audits drop constraint publication_audits_next_state_check;
alter table publication_audits add constraint publication_audits_next_state_check
    check (after_state is null
           or after_state in ('draft', 'published', 'withdrawn', 'deleted'));

create table post_addresses (
    slug         text primary key references post_slugs (slug),
    current_slug text not null,
    explanation  text not null default '',
    retired_at   timestamptz not null default now(),
    constraint post_addresses_explanation_check check (char_length(explanation) <= 500)
);

create index post_addresses_current_idx on post_addresses (current_slug);

-- +goose Down
drop table post_addresses;
alter table publication_audits drop constraint publication_audits_next_state_check;
alter table publication_audits add constraint publication_audits_next_state_check
    check (after_state is null or after_state in ('draft', 'published', 'withdrawn'));
alter table publication_audits drop constraint publication_audits_state_check;
alter table publication_audits add constraint publication_audits_state_check
    check (before_state is null or before_state in ('draft', 'published', 'withdrawn'));
drop index posts_recoverable_idx;
alter table posts drop constraint posts_deleted_check;
alter table posts drop column deleted_by;
alter table posts drop column recoverable_until;
alter table posts drop column deleted_at;
