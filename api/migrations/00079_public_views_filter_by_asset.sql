-- +goose Up
create or replace view asset_public.asset_blocks as
select b.* from public.asset_blocks b join public.assets a on a.id = b.asset_id
where a.published_snapshot_id is null
union all
select r.id, a.id as asset_id, r.definition, r.title, r.position, r.hidden, r.layout, r.width, r.elements
from public.assets a join public.asset_snapshots s on s.id = a.published_snapshot_id
cross join lateral jsonb_array_elements(s.payload->'blocks') b
cross join lateral jsonb_populate_record(null::public.asset_blocks, b.value) r;

create or replace view asset_public.protected_content as
select p.* from public.protected_content p join public.assets a on a.id = p.asset_id
where a.published_snapshot_id is null
union all
select a.id as asset_id, r.owner_kind, r.owner_id, r.payload_type, r.payload, r.source_key, r.digest
from public.assets a join public.asset_snapshots s on s.id = a.published_snapshot_id
cross join lateral jsonb_array_elements(s.protected_payloads) p
cross join lateral jsonb_populate_record(null::public.protected_content, p.value) r;

create or replace view asset_public.asset_projections as
select p.* from public.asset_projections p join public.assets a on a.id = p.asset_id
where a.published_snapshot_id is null
union all
select a.id as asset_id, r.export, r.export_stamp, r.export_computed_at, r.facets, r.facet_stamp, r.facet_computed_at
from public.assets a join public.asset_snapshot_projections p on p.snapshot_id = a.published_snapshot_id
cross join lateral jsonb_populate_record(null::public.asset_projections, p.projection) r;

-- +goose Down
create or replace view asset_public.asset_blocks as
select b.* from public.asset_blocks b join public.assets a on a.id = b.asset_id
where a.published_snapshot_id is null
union all
select (jsonb_populate_record(null::public.asset_blocks,
    b.value || jsonb_build_object('asset_id', a.id))).*
from public.assets a join public.asset_snapshots s on s.id = a.published_snapshot_id
cross join lateral jsonb_array_elements(s.payload->'blocks') b;

create or replace view asset_public.protected_content as
select p.* from public.protected_content p join public.assets a on a.id = p.asset_id
where a.published_snapshot_id is null
union all
select (jsonb_populate_record(null::public.protected_content,
    p.value || jsonb_build_object('asset_id', a.id))).*
from public.assets a join public.asset_snapshots s on s.id = a.published_snapshot_id
cross join lateral jsonb_array_elements(s.protected_payloads) p;

create or replace view asset_public.asset_projections as
select p.* from public.asset_projections p join public.assets a on a.id = p.asset_id
where a.published_snapshot_id is null
union all
select (jsonb_populate_record(null::public.asset_projections, p.projection)).*
from public.assets a join public.asset_snapshot_projections p on p.snapshot_id = a.published_snapshot_id;
