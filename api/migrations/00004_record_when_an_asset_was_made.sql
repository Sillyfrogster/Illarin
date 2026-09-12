-- +goose Up
alter table assets add column indexed_at timestamptz not null default now();

drop index assets_browse_idx;
create index assets_browse_idx
    on assets (kind, created_at desc, id desc)
    where publication = 'public';

-- +goose Down
drop index assets_browse_idx;
create index assets_browse_idx on assets (kind, created_at desc) where publication = 'public';
alter table assets drop column indexed_at;
