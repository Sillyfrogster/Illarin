-- +goose Up
-- work_file_fields lists the page fields each type's downloaded file carries, matching format.Registry.FileFields
create function work_file_fields(work_type text) returns text[] language sql immutable as $$
    select case work_type
        when 'character' then array['cover', 'name']
        when 'lorebook' then array['name']
        when 'pack' then array['cover', 'name']
        when 'preset' then array['blurb', 'name']
        when 'theme' then array['blurb', 'name']
        else array[]::text[]
    end
$$;

create or replace view work_public.works as
select w.id, w.type,
    case when s.id is null then w.original_file_id else s.original_file_id end as original_file_id,
    w.owner_id,
    case when s.id is null or not 'name' = any(work_file_fields(w.type)) then w.name
         else s.payload ->> 'name' end as name,
    case when s.id is null or not 'blurb' = any(work_file_fields(w.type)) then w.blurb
         else s.payload ->> 'blurb' end as blurb,
    w.tags,
    case when s.id is null or not 'cover' = any(work_file_fields(w.type)) then w.cover_media_id
         else (s.payload ->> 'cover_media_id')::uuid end as cover_media_id,
    case when s.id is null then w.is_nsfw else (s.payload ->> 'is_nsfw')::boolean end as is_nsfw,
    w.visibility, w.created_at, w.updated_at, w.indexed_at,
    w.taken_down_at, w.taken_down_by, w.taken_down_reason,
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

create or replace view work_public.work_media as
select m.id, m.work_id, m.role, m.width, m.height, m.created_at, m.blob_id, m.is_extracted,
    true as is_current
from work_media m join works w on w.id = m.work_id
where case
    when w.published_version_id is null
      or (m.role in ('avatar', 'avatar_alt') and not 'cover' = any(work_file_fields(w.type)))
    then m.is_current
    else exists (select 1 from work_version_media r
                  where r.version_id = w.published_version_id and r.media_id = m.id)
end;

-- +goose Down
create or replace view work_public.work_media as
select m.id, m.work_id, m.role, m.width, m.height, m.created_at, m.blob_id, m.is_extracted,
    true as is_current
from work_media m join works w on w.id = m.work_id
where w.published_version_id is null and m.is_current
   or exists (select 1 from work_version_media r
               where r.version_id = w.published_version_id and r.media_id = m.id);

create or replace view work_public.works as
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
    w.taken_down_at, w.taken_down_by, w.taken_down_reason,
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

drop function work_file_fields(text);
