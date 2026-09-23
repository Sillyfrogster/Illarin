-- +goose Up

-- Reading a file in is an upload, the pictures a README showed wait as found
-- images, and the format a work was born as is its original format. The
-- passthrough column goes too: no migration ever created it here and nothing
-- reads it, so it only exists on databases that drifted.

alter table ingest_operations rename to upload_operations;
alter table upload_operations rename constraint ingest_operations_pkey to upload_operations_pkey;
alter table upload_operations rename constraint ingest_operations_blob_id_fkey to upload_operations_blob_id_fkey;
alter table upload_operations rename constraint ingest_operations_owner_id_fkey to upload_operations_owner_id_fkey;
alter table upload_operations rename constraint ingest_operations_target_work_id_fkey to upload_operations_target_work_id_fkey;
alter table upload_operations rename constraint ingest_operations_work_id_fkey to upload_operations_work_id_fkey;
alter table upload_operations rename constraint ingest_operations_failure_reason_check to upload_operations_failure_reason_check;
alter table upload_operations rename constraint ingest_operations_lease_check to upload_operations_lease_check;
alter table upload_operations rename constraint ingest_operations_result_check to upload_operations_result_check;
alter table upload_operations rename constraint ingest_operations_status_check to upload_operations_status_check;
alter table upload_operations rename constraint ingest_operations_visibility_check to upload_operations_visibility_check;
alter index ingest_operations_created_work_idx rename to upload_operations_created_work_idx;
alter index ingest_operations_queue_idx rename to upload_operations_queue_idx;
alter table upload_operations drop column if exists passthrough_platform;

alter table work_vault_pictures rename to work_found_images;
alter table work_found_images rename constraint work_vault_pictures_pkey to work_found_images_pkey;
alter table work_found_images rename constraint work_vault_pictures_media_id_fkey to work_found_images_media_id_fkey;
alter table work_found_images rename constraint work_vault_pictures_work_id_fkey to work_found_images_work_id_fkey;
alter index work_vault_pictures_work_idx rename to work_found_images_work_idx;

drop view work_public.works;
alter table works rename column origin_format to original_format;

create view work_public.works as
select w.id, w.type,
    case when s.id is null then w.original_file_id else s.original_file_id end as original_file_id,
    w.owner_id,
    case when s.id is null then w.name else s.payload ->> 'name' end as name,
    case when s.id is null then w.blurb else s.payload ->> 'blurb' end as blurb,
    case when s.id is null then w.tags
         else array(select jsonb_array_elements_text(s.payload -> 'tags')) end as tags,
    case when s.id is null then w.cover_media_id
         else (s.payload ->> 'cover_media_id')::uuid end as cover_media_id,
    case when s.id is null then w.is_nsfw else (s.payload ->> 'is_nsfw')::boolean end as is_nsfw,
    w.visibility, w.created_at, w.updated_at, w.indexed_at,
    w.withheld_at, w.withheld_by, w.withheld_reason,
    w.deleted_at, w.recoverable_until, w.lifecycle,
    case when s.id is null then w.work_version else s.payload ->> 'work_version' end as work_version,
    case when s.id is null then w.credited_author
         else s.payload ->> 'credited_author' end as credited_author,
    case when s.id is null then w.nickname else s.payload ->> 'nickname' end as nickname,
    case when s.id is null then w.original_format
         else s.payload ->> 'original_format' end as original_format,
    s.number as version_number,
    w.published_version_id
  from works w
  left join work_versions s on s.id = w.published_version_id;

alter table work_versions disable trigger work_versions_immutable;
update work_versions
   set payload = (payload - 'origin_format')
       || jsonb_build_object('original_format', payload -> 'origin_format')
 where payload ? 'origin_format';
alter table work_versions enable trigger work_versions_immutable;

-- +goose StatementBegin
create or replace function record_work_version(
    target uuid, baseline boolean, given_summary text, given_notes text, given_label text
) returns uuid language plpgsql as $$
declare
    owned works%rowtype;
    recorded uuid;
    pictures uuid[];
begin
    select * into owned from works where id = target for update;
    if not found or owned.lifecycle <> 'published'
        or (owned.deleted_at is not null and owned.recoverable_until <= now()) then
        return null;
    end if;

    select coalesce(array_agg(distinct media_id), '{}'::uuid[]) into pictures from (
        select id as media_id from work_media where work_id = target and is_current
        union select owned.cover_media_id where owned.cover_media_id is not null
        union select (value #>> '{}')::uuid from work_blocks,
            lateral jsonb_path_query(elements, '$[*] ? (@.type == "image_set").content.images[*].mediaId') value
            where work_id = target
    ) referenced;
    perform 1 from work_media where id = any(pictures) for share;

    insert into work_versions
        (work_id, number, initial_recorded, version_label,
         original_file_id, payload, private_prompts, summary, notes)
    values (target,
        (select coalesce(max(number), 0) + 1 from work_versions where work_id = target),
        baseline, coalesce(given_label, owned.work_version),
        owned.original_file_id,
        jsonb_build_object(
            'schema_version', 1,
            'type', owned.type, 'name', owned.name, 'blurb', owned.blurb,
            'tags', owned.tags, 'is_nsfw', owned.is_nsfw,
            'work_version', owned.work_version, 'credited_author', owned.credited_author,
            'nickname', owned.nickname, 'original_format', owned.original_format,
            'cover_media_id', owned.cover_media_id,
            'media_ids', pictures,
            'blocks', coalesce((select jsonb_agg(to_jsonb(b) - 'work_id' order by b.position)
                from work_blocks b where b.work_id = target), '[]'::jsonb),
            'preserved_data', coalesce((select jsonb_agg(jsonb_build_object(
                'id', p.id, 'owner_type', p.owner_type, 'owner_id', p.owner_id,
                'namespace', p.namespace, 'payload', p.payload::text) order by p.id)
                from work_preserved_data p where p.work_id = target), '[]'::jsonb)),
        coalesce((select jsonb_agg(to_jsonb(p) - 'work_id' order by p.owner_type, p.owner_id)
            from private_prompts p where p.work_id = target), '[]'::jsonb),
        coalesce(given_summary, ''), coalesce(given_notes, ''))
    returning id into recorded;

    insert into work_version_media (version_id, work_id, media_id)
    select recorded, target, unnest(pictures);

    update works set published_version_id = recorded where id = target;
    return recorded;
end;
$$;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
create or replace function record_work_version(
    target uuid, baseline boolean, given_summary text, given_notes text, given_label text
) returns uuid language plpgsql as $$
declare
    owned works%rowtype;
    recorded uuid;
    pictures uuid[];
begin
    select * into owned from works where id = target for update;
    if not found or owned.lifecycle <> 'published'
        or (owned.deleted_at is not null and owned.recoverable_until <= now()) then
        return null;
    end if;

    select coalesce(array_agg(distinct media_id), '{}'::uuid[]) into pictures from (
        select id as media_id from work_media where work_id = target and is_current
        union select owned.cover_media_id where owned.cover_media_id is not null
        union select (value #>> '{}')::uuid from work_blocks,
            lateral jsonb_path_query(elements, '$[*] ? (@.type == "image_set").content.images[*].mediaId') value
            where work_id = target
    ) referenced;
    perform 1 from work_media where id = any(pictures) for share;

    insert into work_versions
        (work_id, number, initial_recorded, version_label,
         original_file_id, payload, private_prompts, summary, notes)
    values (target,
        (select coalesce(max(number), 0) + 1 from work_versions where work_id = target),
        baseline, coalesce(given_label, owned.work_version),
        owned.original_file_id,
        jsonb_build_object(
            'schema_version', 1,
            'type', owned.type, 'name', owned.name, 'blurb', owned.blurb,
            'tags', owned.tags, 'is_nsfw', owned.is_nsfw,
            'work_version', owned.work_version, 'credited_author', owned.credited_author,
            'nickname', owned.nickname, 'origin_format', owned.origin_format,
            'cover_media_id', owned.cover_media_id,
            'media_ids', pictures,
            'blocks', coalesce((select jsonb_agg(to_jsonb(b) - 'work_id' order by b.position)
                from work_blocks b where b.work_id = target), '[]'::jsonb),
            'preserved_data', coalesce((select jsonb_agg(jsonb_build_object(
                'id', p.id, 'owner_type', p.owner_type, 'owner_id', p.owner_id,
                'namespace', p.namespace, 'payload', p.payload::text) order by p.id)
                from work_preserved_data p where p.work_id = target), '[]'::jsonb)),
        coalesce((select jsonb_agg(to_jsonb(p) - 'work_id' order by p.owner_type, p.owner_id)
            from private_prompts p where p.work_id = target), '[]'::jsonb),
        coalesce(given_summary, ''), coalesce(given_notes, ''))
    returning id into recorded;

    insert into work_version_media (version_id, work_id, media_id)
    select recorded, target, unnest(pictures);

    update works set published_version_id = recorded where id = target;
    return recorded;
end;
$$;
-- +goose StatementEnd

alter table work_versions disable trigger work_versions_immutable;
update work_versions
   set payload = (payload - 'original_format')
       || jsonb_build_object('origin_format', payload -> 'original_format')
 where payload ? 'original_format';
alter table work_versions enable trigger work_versions_immutable;

drop view work_public.works;
alter table works rename column original_format to origin_format;

create view work_public.works as
select w.id, w.type,
    case when s.id is null then w.original_file_id else s.original_file_id end as original_file_id,
    w.owner_id,
    case when s.id is null then w.name else s.payload ->> 'name' end as name,
    case when s.id is null then w.blurb else s.payload ->> 'blurb' end as blurb,
    case when s.id is null then w.tags
         else array(select jsonb_array_elements_text(s.payload -> 'tags')) end as tags,
    case when s.id is null then w.cover_media_id
         else (s.payload ->> 'cover_media_id')::uuid end as cover_media_id,
    case when s.id is null then w.is_nsfw else (s.payload ->> 'is_nsfw')::boolean end as is_nsfw,
    w.visibility, w.created_at, w.updated_at, w.indexed_at,
    w.withheld_at, w.withheld_by, w.withheld_reason,
    w.deleted_at, w.recoverable_until, w.lifecycle,
    case when s.id is null then w.work_version else s.payload ->> 'work_version' end as work_version,
    case when s.id is null then w.credited_author
         else s.payload ->> 'credited_author' end as credited_author,
    case when s.id is null then w.nickname else s.payload ->> 'nickname' end as nickname,
    case when s.id is null then w.origin_format else s.payload ->> 'origin_format' end as origin_format,
    s.number as version_number,
    w.published_version_id
  from works w
  left join work_versions s on s.id = w.published_version_id;

alter index work_found_images_work_idx rename to work_vault_pictures_work_idx;
alter table work_found_images rename constraint work_found_images_work_id_fkey to work_vault_pictures_work_id_fkey;
alter table work_found_images rename constraint work_found_images_media_id_fkey to work_vault_pictures_media_id_fkey;
alter table work_found_images rename constraint work_found_images_pkey to work_vault_pictures_pkey;
alter table work_found_images rename to work_vault_pictures;

alter index upload_operations_queue_idx rename to ingest_operations_queue_idx;
alter index upload_operations_created_work_idx rename to ingest_operations_created_work_idx;
alter table upload_operations rename constraint upload_operations_visibility_check to ingest_operations_visibility_check;
alter table upload_operations rename constraint upload_operations_status_check to ingest_operations_status_check;
alter table upload_operations rename constraint upload_operations_result_check to ingest_operations_result_check;
alter table upload_operations rename constraint upload_operations_lease_check to ingest_operations_lease_check;
alter table upload_operations rename constraint upload_operations_failure_reason_check to ingest_operations_failure_reason_check;
alter table upload_operations rename constraint upload_operations_work_id_fkey to ingest_operations_work_id_fkey;
alter table upload_operations rename constraint upload_operations_target_work_id_fkey to ingest_operations_target_work_id_fkey;
alter table upload_operations rename constraint upload_operations_owner_id_fkey to ingest_operations_owner_id_fkey;
alter table upload_operations rename constraint upload_operations_blob_id_fkey to ingest_operations_blob_id_fkey;
alter table upload_operations rename constraint upload_operations_pkey to ingest_operations_pkey;
alter table upload_operations rename to ingest_operations;
