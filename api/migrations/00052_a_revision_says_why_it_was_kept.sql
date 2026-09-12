-- +goose Up
alter table post_revisions add column captured_for text not null default 'publication';
alter table post_revisions alter column captured_for drop default;
alter table post_revisions add constraint post_revisions_captured_for_check
    check (captured_for in ('checkpoint', 'publication'));

create index post_revisions_post_idx on post_revisions (post_id, number desc);

-- +goose Down
drop index post_revisions_post_idx;
alter table post_revisions drop constraint post_revisions_captured_for_check;
alter table post_revisions drop column captured_for;
