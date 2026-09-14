-- +goose Up
alter table assets drop constraint assets_kind_check;
alter table assets add constraint assets_kind_check
    check (kind in ('character', 'lorebook', 'preset', 'theme', 'pack', 'extension'));

alter table asset_revisions add column identifier text not null default '';

-- +goose Down
alter table asset_revisions drop column identifier;

alter table assets drop constraint assets_kind_check;
alter table assets add constraint assets_kind_check
    check (kind in ('character', 'lorebook', 'preset', 'theme', 'pack'));
