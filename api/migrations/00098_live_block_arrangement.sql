-- +goose Up
create or replace view work_public.work_blocks as
select b.* from public.work_blocks b join public.works w on w.id = b.work_id
where w.published_version_id is null
union all
select r.id, w.id as work_id, r.definition,
    case when live.id is null then r.title else live.title end as title,
    coalesce(live.position, r.position) as position,
    coalesce(live.hidden, r.hidden) as hidden,
    r.layout, coalesce(live.width, r.width) as width, r.elements
from public.works w join public.work_versions s on s.id = w.published_version_id
cross join lateral jsonb_array_elements(s.payload->'blocks') b
cross join lateral jsonb_populate_record(null::public.work_blocks, b.value) r
left join public.work_blocks live on live.id = r.id and live.work_id = w.id;

-- +goose Down
create or replace view work_public.work_blocks as
select b.* from public.work_blocks b join public.works w on w.id = b.work_id
where w.published_version_id is null
union all
select r.id, w.id as work_id, r.definition, r.title, r.position, r.hidden, r.layout, r.width, r.elements
from public.works w join public.work_versions s on s.id = w.published_version_id
cross join lateral jsonb_array_elements(s.payload->'blocks') b
cross join lateral jsonb_populate_record(null::public.work_blocks, b.value) r;
