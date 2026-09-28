-- +goose Up

-- A site visit counts once per reader per UTC day by the same daily key as a view,
-- made without a work id, so the staff report no longer reads Umami.
alter table reader_keys drop constraint reader_keys_kind_check;
alter table reader_keys add constraint reader_keys_kind_check check (kind in ('view', 'download', 'visit'));
alter table events drop constraint events_kind_check;
alter table events add constraint events_kind_check
    check (kind in ('view', 'visit', 'download', 'send', 'sign_up', 'publish', 'follow', 'unfollow'));

-- +goose Down
delete from events where kind = 'visit';
alter table events drop constraint events_kind_check;
alter table events add constraint events_kind_check
    check (kind in ('view', 'download', 'send', 'sign_up', 'publish', 'follow', 'unfollow'));
delete from reader_keys where kind = 'visit';
alter table reader_keys drop constraint reader_keys_kind_check;
alter table reader_keys add constraint reader_keys_kind_check check (kind in ('view', 'download'));
