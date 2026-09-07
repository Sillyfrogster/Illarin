-- +goose Up
create schema asset_public;

create view asset_public.assets as
select a.id, a.kind,
    case when s.id is null then a.current_revision_id else s.source_revision_id end as current_revision_id,
    a.owner_id,
    case when s.id is null then a.name else s.payload->>'name' end as name,
    case when s.id is null then a.blurb else s.payload->>'blurb' end as blurb,
    case when s.id is null then a.tags else array(select jsonb_array_elements_text(s.payload->'tags')) end as tags,
    case when s.id is null then a.cover_media_id else (s.payload->>'cover_media_id')::uuid end as cover_media_id,
    case when s.id is null then a.is_nsfw else (s.payload->>'is_nsfw')::boolean end as is_nsfw,
    a.discovery, a.created_at, a.updated_at, a.indexed_at,
    a.withheld_at, a.withheld_by, a.withheld_reason, a.deleted_at, a.recoverable_until, a.lifecycle,
    case when s.id is null then a.asset_version else s.payload->>'asset_version' end as asset_version,
    case when s.id is null then a.credited_author else s.payload->>'credited_author' end as credited_author,
    case when s.id is null then a.nickname else s.payload->>'nickname' end as nickname,
    case when s.id is null then a.origin_format else s.payload->>'origin_format' end as origin_format,
    coalesce(s.content_generation, a.content_generation) as content_generation,
    a.published_snapshot_id
from public.assets a
left join public.asset_snapshots s on s.id = a.published_snapshot_id;

create view asset_public.asset_blocks as
select b.* from public.asset_blocks b join public.assets a on a.id = b.asset_id
where a.published_snapshot_id is null
union all
select (jsonb_populate_record(null::public.asset_blocks,
    b.value || jsonb_build_object('asset_id', a.id))).*
from public.assets a join public.asset_snapshots s on s.id = a.published_snapshot_id
cross join lateral jsonb_array_elements(s.payload->'blocks') b;

create view asset_public.asset_media as
select m.id, m.asset_id, m.role, m.width, m.height, m.created_at, m.blob_id,
    m.is_extracted, true as is_current
from public.asset_media m join public.assets a on a.id = m.asset_id
where (a.published_snapshot_id is null and m.is_current)
   or exists (select 1 from public.asset_snapshot_media r
              where r.snapshot_id = a.published_snapshot_id and r.media_id = m.id);

create view asset_public.asset_preserved_data as
select p.* from public.asset_preserved_data p join public.assets a on a.id = p.asset_id
where a.published_snapshot_id is null
union all
select (p.value->>'id')::uuid, a.id, p.value->>'owner_kind',
    (p.value->>'owner_id')::uuid, p.value->>'namespace', (p.value->>'payload')::json
from public.assets a join public.asset_snapshots s on s.id = a.published_snapshot_id
cross join lateral jsonb_array_elements(s.payload->'preserved_data') p;

create view asset_public.protected_content as
select p.* from public.protected_content p join public.assets a on a.id = p.asset_id
where a.published_snapshot_id is null
union all
select (jsonb_populate_record(null::public.protected_content,
    p.value || jsonb_build_object('asset_id', a.id))).*
from public.assets a join public.asset_snapshots s on s.id = a.published_snapshot_id
cross join lateral jsonb_array_elements(s.protected_payloads) p;

create table asset_snapshot_projections (
    snapshot_id uuid primary key references asset_snapshots(id) on delete cascade,
    projection jsonb not null
);
insert into asset_snapshot_projections
select s.id, to_jsonb(p) from asset_snapshots s
join asset_projections p on p.asset_id = s.asset_id;

create view asset_public.asset_projections as
select p.* from public.asset_projections p join public.assets a on a.id = p.asset_id
where a.published_snapshot_id is null
union all
select (jsonb_populate_record(null::public.asset_projections, p.projection)).*
from public.assets a join public.asset_snapshot_projections p on p.snapshot_id = a.published_snapshot_id;

-- +goose StatementBegin
create function capture_asset_snapshot_projection() returns trigger language plpgsql as $$
begin
    insert into public.asset_snapshot_projections
    select new.id, to_jsonb(p) from public.asset_projections p where p.asset_id = new.asset_id;
    return new;
end;
$$;
-- +goose StatementEnd
create trigger capture_asset_snapshot_projection after insert on asset_snapshots
for each row execute function capture_asset_snapshot_projection();

-- +goose Down
drop schema asset_public cascade;
drop trigger capture_asset_snapshot_projection on asset_snapshots;
drop function capture_asset_snapshot_projection();
drop table asset_snapshot_projections;
