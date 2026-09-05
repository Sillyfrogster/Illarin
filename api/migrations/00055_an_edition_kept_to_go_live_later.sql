-- +goose Up
alter table post_revisions drop constraint post_revisions_captured_for_check;
alter table post_revisions add constraint post_revisions_captured_for_check
    check (captured_for in ('checkpoint', 'publication', 'schedule'));

-- +goose Down
update post_revisions set captured_for = 'publication' where captured_for = 'schedule';
alter table post_revisions drop constraint post_revisions_captured_for_check;
alter table post_revisions add constraint post_revisions_captured_for_check
    check (captured_for in ('checkpoint', 'publication'));
