-- +goose Up

-- Found images become the shelf, which holds Markdown sections beside pictures and remembers placements for undo.

create table work_shelf_imports (
    id         uuid primary key,
    work_id    uuid not null references works (id) on delete cascade,
    source     text not null check (source in ('readme', 'pasted')),
    title      text not null default '',
    created_at timestamptz not null default now()
);

create index work_shelf_imports_work_idx on work_shelf_imports (work_id, created_at);

alter table work_found_images rename to work_shelf_pieces;
alter table work_shelf_pieces rename constraint work_found_images_pkey to work_shelf_pieces_pkey;
alter table work_shelf_pieces rename constraint work_found_images_media_id_fkey to work_shelf_pieces_media_id_fkey;
alter table work_shelf_pieces rename constraint work_found_images_work_id_fkey to work_shelf_pieces_work_id_fkey;
alter index work_found_images_work_idx rename to work_shelf_pieces_work_idx;

alter table work_shelf_pieces
    add column import_id uuid references work_shelf_imports (id) on delete cascade,
    add column kind text not null default 'picture' check (kind in ('section', 'picture')),
    add column text text not null default '',
    add column placed_at timestamptz,
    add column placed_block_id uuid,
    add column placed_before jsonb,
    add column placed_after jsonb,
    alter column address set default '';

insert into work_shelf_imports (id, work_id, source, created_at)
select gen_random_uuid(), work_id, 'readme', min(created_at)
  from work_shelf_pieces
 group by work_id;

update work_shelf_pieces piece
   set import_id = shelf_import.id
  from work_shelf_imports shelf_import
 where shelf_import.work_id = piece.work_id;

alter table work_shelf_pieces
    alter column import_id set not null,
    alter column kind drop default;

create index work_shelf_pieces_import_idx on work_shelf_pieces (import_id);

-- +goose Down

delete from work_shelf_pieces where kind = 'section' or placed_at is not null;

drop index work_shelf_pieces_import_idx;
alter table work_shelf_pieces
    drop column import_id,
    drop column kind,
    drop column text,
    drop column placed_at,
    drop column placed_block_id,
    drop column placed_before,
    drop column placed_after,
    alter column address drop default;

alter index work_shelf_pieces_work_idx rename to work_found_images_work_idx;
alter table work_shelf_pieces rename constraint work_shelf_pieces_work_id_fkey to work_found_images_work_id_fkey;
alter table work_shelf_pieces rename constraint work_shelf_pieces_media_id_fkey to work_found_images_media_id_fkey;
alter table work_shelf_pieces rename constraint work_shelf_pieces_pkey to work_found_images_pkey;
alter table work_shelf_pieces rename to work_found_images;

drop table work_shelf_imports;
