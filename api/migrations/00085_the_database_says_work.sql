-- +goose Up

-- The tables, columns, views and functions that said asset now say work, kind
-- says type, discovery says visibility, a watch is a follow and a projection is
-- a summary. Nothing the API sends or accepts changes.

alter table assets rename to works;
alter table asset_blocks rename to work_blocks;
alter table asset_legacy_paths rename to work_legacy_paths;
alter table asset_media rename to work_media;
alter table asset_preserved_data rename to work_preserved_data;
alter table asset_projections rename to work_summaries;
alter table asset_revisions rename to work_revisions;
alter table asset_snapshot_media rename to work_snapshot_media;
alter table asset_snapshot_projections rename to work_snapshot_summaries;
alter table asset_snapshot_prompt_matches rename to work_snapshot_prompt_matches;
alter table asset_snapshots rename to work_snapshots;
alter table asset_update_deliveries rename to work_update_deliveries;
alter table asset_update_delivery_attempts rename to work_update_delivery_attempts;
alter table asset_update_destination_defaults rename to work_update_destination_defaults;
alter table asset_update_destinations rename to work_update_destinations;
alter table asset_update_events rename to work_update_events;
alter table asset_vault_pictures rename to work_vault_pictures;
alter table asset_watches rename to work_follows;

alter table works rename column kind to type;
alter table works rename column discovery to visibility;
alter table works rename column asset_version to work_version;
alter table work_blocks rename column asset_id to work_id;
alter table work_follows rename column asset_id to work_id;
alter table work_legacy_paths rename column asset_id to work_id;
alter table work_media rename column asset_id to work_id;
alter table work_preserved_data rename column asset_id to work_id;
alter table work_preserved_data rename column owner_kind to owner_type;
alter table work_revisions rename column asset_id to work_id;
alter table work_snapshot_media rename column asset_id to work_id;
alter table work_snapshot_summaries rename column projection to summary;
alter table work_snapshots rename column asset_id to work_id;
alter table work_summaries rename column asset_id to work_id;
alter table work_update_deliveries rename column destination_kind to destination_type;
alter table work_update_destination_defaults rename column asset_id to work_id;
alter table work_update_destinations rename column kind to type;
alter table work_update_events rename column asset_id to work_id;
alter table work_vault_pictures rename column asset_id to work_id;
alter table download_events rename column asset_id to work_id;
alter table download_events rename column discovery to visibility;
alter table ingest_operations rename column asset_id to work_id;
alter table ingest_operations rename column target_asset_id to target_work_id;
alter table ingest_operations rename column discovery to visibility;
alter table instance_deliveries rename column asset_id to work_id;
alter table instance_library_entries rename column asset_id to work_id;
alter table migration_exceptions rename column asset_id to work_id;
alter table migration_exceptions rename column kind to type;
alter table migration_legacy_counters rename column asset_id to work_id;
alter table migration_preserved_records rename column asset_id to work_id;
alter table notification_events rename column asset_id to work_id;
alter table notifications rename column asset_id to work_id;
alter table protected_content rename column asset_id to work_id;
alter table protected_content rename column owner_kind to owner_type;
alter table protected_delivery_apps rename column asset_id to work_id;
alter table publication_destinations rename column kind to type;
alter table users rename column nsfw_visibility to nsfw_preference;






-- A trigger holds its function by identity, so each one is renamed where it
-- stands and then given its new body.
alter function capture_asset_snapshot_projection() rename to capture_work_snapshot_summary;
alter function guard_asset_snapshot() rename to guard_work_snapshot;
alter function guard_asset_snapshot_media() rename to guard_work_snapshot_media;
alter function guard_recorded_asset_media() rename to guard_recorded_work_media;
alter function invalidate_asset_candidate() rename to invalidate_work_candidate;
alter function invalidate_protected_asset_candidate() rename to invalidate_protected_work_candidate;
alter function record_asset_snapshot(uuid, boolean, text, text, text) rename to record_work_snapshot;
alter function record_initial_asset_snapshot(uuid, boolean) rename to record_initial_work_snapshot;

-- +goose StatementBegin
create or replace function capture_work_snapshot_summary() returns trigger
    language plpgsql as $$
begin
    insert into public.work_snapshot_summaries
    select new.id, to_jsonb(s) from public.work_summaries s where s.work_id = new.work_id;
    return new;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function guard_work_snapshot() returns trigger
    language plpgsql as $$
begin
    if tg_op = 'UPDATE'
        and (to_jsonb(new) - array['summary', 'notes', 'notes_edited_at', 'withdrawn_at', 'withdrawal_explanation'])
            = (to_jsonb(old) - array['summary', 'notes', 'notes_edited_at', 'withdrawn_at', 'withdrawal_explanation'])
        and (
            (
                new.withdrawn_at is not distinct from old.withdrawn_at
                and new.withdrawal_explanation is not distinct from old.withdrawal_explanation
                and new.notes_edited_at is not null
            )
            or (
                new.summary is not distinct from old.summary
                and new.notes is not distinct from old.notes
                and new.notes_edited_at is not distinct from old.notes_edited_at
                and old.withdrawn_at is null
                and new.withdrawn_at is not null
            )
        ) then
        return new;
    end if;
    if tg_op = 'DELETE' and not exists (
        select 1 from works where id = old.work_id
        and (deleted_at is null or recoverable_until > now())
    ) then
        return old;
    end if;
    raise exception 'Published work snapshots are immutable';
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function guard_work_snapshot_media() returns trigger
    language plpgsql as $$
begin
    if tg_op = 'INSERT' then
        if exists (select 1 from work_snapshots where id = new.snapshot_id
            and payload->'media_ids' @> to_jsonb(array[new.media_id])) then
            return new;
        end if;
    elsif tg_op = 'DELETE' and not exists (
        select 1 from work_snapshots where id = old.snapshot_id
    ) then
        return old;
    end if;
    raise exception 'Published media references are immutable';
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function guard_recorded_work_media() returns trigger
    language plpgsql as $$
begin
    if not exists (select 1 from work_snapshot_media where media_id = old.id) then
        return new;
    end if;
    if (to_jsonb(new) - 'is_current') = (to_jsonb(old) - 'is_current') then
        return new;
    end if;
    if new.blob_id is null
        and (to_jsonb(new) - 'blob_id') = (to_jsonb(old) - 'blob_id')
        and (exists (select 1 from blobs b join blob_tombstones t on t.sha256 = b.sha256
                     where b.id = old.blob_id)
             or not exists (select 1 from works where id = old.work_id
                            and (deleted_at is null or recoverable_until > now()))) then
        return new;
    end if;
    raise exception 'Published media is immutable';
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function invalidate_work_candidate() returns trigger
    language plpgsql as $$
begin
    if row(new.owner_id, new.visibility, new.withheld_at, new.deleted_at, new.lifecycle, new.published_snapshot_id)
       is distinct from row(old.owner_id, old.visibility, old.withheld_at, old.deleted_at, old.lifecycle, old.published_snapshot_id) then
        new.working_copy_version := old.working_copy_version + 1;
    end if;
    return new;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function invalidate_protected_work_candidate() returns trigger
    language plpgsql as $$
begin
    update works set working_copy_version = working_copy_version + 1
    where id = coalesce(new.work_id, old.work_id);
    return null;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function protected_content_requires_delivery_policy() returns trigger
    language plpgsql as $$
declare
    protected_work_id uuid;
begin
    if tg_op = 'DELETE' then
        protected_work_id := old.work_id;
    else
        protected_work_id := new.work_id;
    end if;
    if exists (
        select 1 from protected_content where work_id = protected_work_id
    ) then
        if not exists (
            select 1 from protected_delivery_apps where work_id = protected_work_id
        ) then
            raise exception 'protected content requires an allowed delivery app';
        end if;
    elsif exists (
        select 1 from protected_delivery_apps where work_id = protected_work_id
    ) then
        raise exception 'allowed delivery apps require protected content';
    end if;
    return null;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function record_work_snapshot(
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

    insert into work_snapshots
        (work_id, number, initial_recorded, version_label, content_generation,
         source_revision_id, payload, protected_payloads, summary, notes)
    values (target,
        (select coalesce(max(number), 0) + 1 from work_snapshots where work_id = target),
        baseline, coalesce(given_label, owned.work_version), owned.content_generation,
        owned.current_revision_id,
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
            from protected_content p where p.work_id = target), '[]'::jsonb),
        coalesce(given_summary, ''), coalesce(given_notes, ''))
    returning id into recorded;

    insert into work_snapshot_media (snapshot_id, work_id, media_id)
    select recorded, target, unnest(pictures);

    update works set published_snapshot_id = recorded where id = target;
    return recorded;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function record_initial_work_snapshot(target uuid, baseline boolean)
    returns uuid language plpgsql as $$
declare
    existing uuid;
begin
    select published_snapshot_id into existing from works where id = target for update;
    if existing is not null then
        return existing;
    end if;
    return record_work_snapshot(target, baseline, '', '', null);
end;
$$;
-- +goose StatementEnd

alter trigger asset_snapshot_media_immutable on work_snapshot_media rename to work_snapshot_media_immutable;
alter trigger asset_snapshots_immutable on work_snapshots rename to work_snapshots_immutable;
alter trigger capture_asset_snapshot_projection on work_snapshots rename to capture_work_snapshot_summary;
alter trigger invalidate_asset_candidate on works rename to invalidate_work_candidate;
alter trigger recorded_asset_media_immutable on work_media rename to recorded_work_media_immutable;

alter table work_snapshots disable trigger work_snapshots_immutable;

-- A recorded snapshot keeps a copy of the work's details, so the keys in it are
-- column names and they move with the columns.
update work_snapshots
   set payload = payload - 'kind' - 'asset_version'
       || jsonb_build_object('type', payload -> 'kind', 'work_version', payload -> 'asset_version')
       || jsonb_build_object('preserved_data', coalesce((
              select jsonb_agg(entry - 'owner_kind' || jsonb_build_object('owner_type', entry -> 'owner_kind'))
                from jsonb_array_elements(payload -> 'preserved_data') entry
          ), '[]'::jsonb)),
       protected_payloads = coalesce((
           select jsonb_agg(entry - 'owner_kind' || jsonb_build_object('owner_type', entry -> 'owner_kind'))
             from jsonb_array_elements(protected_payloads) entry
       ), '[]'::jsonb);

update work_snapshot_summaries
   set summary = summary - 'asset_id' || jsonb_build_object('work_id', summary -> 'asset_id')
 where summary ? 'asset_id';

alter table work_snapshots enable trigger work_snapshots_immutable;

-- Every constraint and index named for the old table names follows them. The
-- words replaced here all contain "asset", which no other name does, so the
-- replacement cannot reach a name it was not meant to.
-- +goose StatementBegin
create function work_word(name text) returns text language sql immutable as $$
    select replace(replace(replace(replace(replace(
        name,
        'asset_watches', 'work_follows'),
        'asset_snapshot_projections', 'work_snapshot_summaries'),
        'asset_projections', 'work_summaries'),
        'assets', 'works'),
        'asset', 'work')
$$;
-- +goose StatementEnd

-- +goose StatementBegin
do $$
declare
    subject record;
begin
    for subject in
        select holder.oid::regclass::text as on_table, held.conname as name
          from pg_constraint held
          join pg_class holder on holder.oid = held.conrelid
          join pg_namespace space on space.oid = holder.relnamespace
         where space.nspname = 'public'
           and work_word(held.conname) <> held.conname
    loop
        execute format('alter table %s rename constraint %I to %I',
            subject.on_table, subject.name, work_word(subject.name));
    end loop;

    for subject in
        select indexed.indexname as name
          from pg_indexes indexed
         where indexed.schemaname = 'public'
           and work_word(indexed.indexname) <> indexed.indexname
    loop
        execute format('alter index public.%I rename to %I', subject.name, work_word(subject.name));
    end loop;
end $$;
-- +goose StatementEnd

drop function work_word(text);

-- The rest are named one at a time, because kind, type, summary and visibility
-- are all words this schema already uses for other things.
alter table work_revisions rename constraint work_sources_work_id_not_null
    to work_revisions_work_id_not_null;
alter table work_revisions rename constraint work_sources_created_at_not_null
    to work_revisions_created_at_not_null;
alter table work_revisions rename constraint work_sources_id_not_null
    to work_revisions_id_not_null;
alter table work_revisions rename constraint work_sources_media_type_not_null
    to work_revisions_media_type_not_null;
alter table work_revisions rename constraint work_sources_revision_not_null
    to work_revisions_revision_not_null;
alter table works rename constraint works_kind_check to works_type_check;
alter table works rename constraint works_kind_not_null to works_type_not_null;
alter table works rename constraint works_discovery_check to works_visibility_check;
alter table works rename constraint works_publication_not_null to works_visibility_not_null;
alter table work_snapshot_summaries rename constraint work_snapshot_summaries_projection_not_null
    to work_snapshot_summaries_summary_not_null;
alter table work_preserved_data rename constraint work_preserved_data_owner_kind_check
    to work_preserved_data_owner_type_check;
alter table work_preserved_data rename constraint work_preserved_data_owner_kind_not_null
    to work_preserved_data_owner_type_not_null;
alter table work_preserved_data rename constraint work_preserved_data_work_id_owner_kind_owner_id_namespace_key
    to work_preserved_data_work_id_owner_type_owner_id_namespace_key;
alter table protected_content rename constraint protected_content_owner_kind_not_null
    to protected_content_owner_type_not_null;
alter table work_update_deliveries rename constraint work_update_deliveries_kind_check
    to work_update_deliveries_type_check;
alter table work_update_deliveries rename constraint work_update_deliveries_destination_kind_not_null
    to work_update_deliveries_destination_type_not_null;
alter table work_update_destinations rename constraint work_update_destinations_kind_check
    to work_update_destinations_type_check;
alter table work_update_destinations rename constraint work_update_destinations_kind_not_null
    to work_update_destinations_type_not_null;
alter table migration_exceptions rename constraint migration_exceptions_kind_check
    to migration_exceptions_type_check;
alter table migration_exceptions rename constraint migration_exceptions_kind_not_null
    to migration_exceptions_type_not_null;
alter index public.migration_exceptions_kind_idx rename to migration_exceptions_type_idx;
alter table publication_destinations rename constraint publication_destinations_kind_check
    to publication_destinations_type_check;
alter table publication_destinations rename constraint publication_destinations_kind_not_null
    to publication_destinations_type_not_null;
alter table download_events rename constraint download_events_discovery_check
    to download_events_visibility_check;
alter table download_events rename constraint download_events_discovery_not_null
    to download_events_visibility_not_null;
alter table ingest_operations rename constraint ingest_operations_discovery_check
    to ingest_operations_visibility_check;
alter table ingest_operations rename constraint ingest_operations_discovery_not_null
    to ingest_operations_visibility_not_null;
-- The ingest queue's index said work for a job, which is now the word for the
-- thing being uploaded.
alter index public.ingest_operations_work_idx rename to ingest_operations_queue_idx;
alter table users rename constraint users_nsfw_visibility_check to users_nsfw_preference_check;
alter table users rename constraint users_nsfw_visibility_not_null to users_nsfw_preference_not_null;

drop view public.inbox_entries;

create schema work_public;

create view work_public.works as
select w.id, w.type,
    case when s.id is null then w.current_revision_id else s.source_revision_id end as current_revision_id,
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
    coalesce(s.content_generation, w.content_generation) as content_generation,
    w.published_snapshot_id
  from works w
  left join work_snapshots s on s.id = w.published_snapshot_id;

create view work_public.work_blocks as
select b.* from public.work_blocks b join public.works w on w.id = b.work_id
where w.published_snapshot_id is null
union all
select r.id, w.id as work_id, r.definition, r.title, r.position, r.hidden, r.layout, r.width, r.elements
from public.works w join public.work_snapshots s on s.id = w.published_snapshot_id
cross join lateral jsonb_array_elements(s.payload->'blocks') b
cross join lateral jsonb_populate_record(null::public.work_blocks, b.value) r;

create view work_public.work_media as
select m.id, m.work_id, m.role, m.width, m.height, m.created_at, m.blob_id, m.is_extracted,
    true as is_current
from work_media m join works w on w.id = m.work_id
where w.published_snapshot_id is null and m.is_current
   or exists (select 1 from work_snapshot_media r
               where r.snapshot_id = w.published_snapshot_id and r.media_id = m.id);

create view work_public.work_preserved_data as
select p.id, p.work_id, p.owner_type, p.owner_id, p.namespace, p.payload
from work_preserved_data p join works w on w.id = p.work_id
where w.published_snapshot_id is null
union all
select (p.value ->> 'id')::uuid as id, w.id as work_id,
    p.value ->> 'owner_type' as owner_type,
    (p.value ->> 'owner_id')::uuid as owner_id,
    p.value ->> 'namespace' as namespace,
    (p.value ->> 'payload')::json as payload
from works w join work_snapshots s on s.id = w.published_snapshot_id
cross join lateral jsonb_array_elements(s.payload -> 'preserved_data') p;

create view work_public.work_summaries as
select s.work_id, s.export, s.export_stamp, s.export_computed_at,
    s.facets, s.facet_stamp, s.facet_computed_at
from work_summaries s join works w on w.id = s.work_id
where w.published_snapshot_id is null
union all
select w.id as work_id, r.export, r.export_stamp, r.export_computed_at,
    r.facets, r.facet_stamp, r.facet_computed_at
from works w join work_snapshot_summaries s on s.snapshot_id = w.published_snapshot_id
cross join lateral jsonb_populate_record(null::public.work_summaries, s.summary) r;

create view work_public.protected_content as
select p.* from public.protected_content p join public.works w on w.id = p.work_id
where w.published_snapshot_id is null
union all
select w.id as work_id, r.owner_type, r.owner_id, r.payload_type, r.payload, r.source_key, r.digest
from public.works w join public.work_snapshots s on s.id = w.published_snapshot_id
cross join lateral jsonb_array_elements(s.protected_payloads) p
cross join lateral jsonb_populate_record(null::public.protected_content, p.value) r;

create view public.inbox_entries as
select entry.id, entry.account_id, entry.type, entry.work_id, entry.words,
    entry.created_at, entry.read_at, entry.update_count
  from notifications entry
 where entry.work_id is null
    or exists (
        select 1
          from works as subject
         where subject.id = entry.work_id
           and (subject.owner_id = entry.account_id
                or (subject.deleted_at is null and subject.withheld_at is null))
    );

drop schema asset_public cascade;

-- +goose Down

drop view public.inbox_entries;
drop schema work_public cascade;

alter table work_snapshots disable trigger work_snapshots_immutable;

update work_snapshots
   set payload = payload - 'type' - 'work_version'
       || jsonb_build_object('kind', payload -> 'type', 'asset_version', payload -> 'work_version')
       || jsonb_build_object('preserved_data', coalesce((
              select jsonb_agg(entry - 'owner_type' || jsonb_build_object('owner_kind', entry -> 'owner_type'))
                from jsonb_array_elements(payload -> 'preserved_data') entry
          ), '[]'::jsonb)),
       protected_payloads = coalesce((
           select jsonb_agg(entry - 'owner_type' || jsonb_build_object('owner_kind', entry -> 'owner_type'))
             from jsonb_array_elements(protected_payloads) entry
       ), '[]'::jsonb);

update work_snapshot_summaries
   set summary = summary - 'work_id' || jsonb_build_object('asset_id', summary -> 'work_id')
 where summary ? 'work_id';

alter table work_snapshots enable trigger work_snapshots_immutable;

alter table users rename constraint users_nsfw_preference_not_null to users_nsfw_visibility_not_null;
alter table users rename constraint users_nsfw_preference_check to users_nsfw_visibility_check;
alter table ingest_operations rename constraint ingest_operations_visibility_not_null
    to ingest_operations_discovery_not_null;
alter table ingest_operations rename constraint ingest_operations_visibility_check
    to ingest_operations_discovery_check;
alter table download_events rename constraint download_events_visibility_not_null
    to download_events_discovery_not_null;
alter table download_events rename constraint download_events_visibility_check
    to download_events_discovery_check;
alter table publication_destinations rename constraint publication_destinations_type_not_null
    to publication_destinations_kind_not_null;
alter table publication_destinations rename constraint publication_destinations_type_check
    to publication_destinations_kind_check;
alter index public.migration_exceptions_type_idx rename to migration_exceptions_kind_idx;
alter table migration_exceptions rename constraint migration_exceptions_type_not_null
    to migration_exceptions_kind_not_null;
alter table migration_exceptions rename constraint migration_exceptions_type_check
    to migration_exceptions_kind_check;
alter table work_update_destinations rename constraint work_update_destinations_type_not_null
    to work_update_destinations_kind_not_null;
alter table work_update_destinations rename constraint work_update_destinations_type_check
    to work_update_destinations_kind_check;
alter table work_update_deliveries rename constraint work_update_deliveries_destination_type_not_null
    to work_update_deliveries_destination_kind_not_null;
alter table work_update_deliveries rename constraint work_update_deliveries_type_check
    to work_update_deliveries_kind_check;
alter table protected_content rename constraint protected_content_owner_type_not_null
    to protected_content_owner_kind_not_null;
alter table work_preserved_data rename constraint work_preserved_data_work_id_owner_type_owner_id_namespace_key
    to work_preserved_data_work_id_owner_kind_owner_id_namespace_key;
alter table work_preserved_data rename constraint work_preserved_data_owner_type_not_null
    to work_preserved_data_owner_kind_not_null;
alter table work_preserved_data rename constraint work_preserved_data_owner_type_check
    to work_preserved_data_owner_kind_check;
alter table work_snapshot_summaries rename constraint work_snapshot_summaries_summary_not_null
    to work_snapshot_summaries_projection_not_null;
alter table works rename constraint works_visibility_not_null to works_publication_not_null;
alter table works rename constraint works_visibility_check to works_discovery_check;
alter table works rename constraint works_type_not_null to works_kind_not_null;
alter table works rename constraint works_type_check to works_kind_check;
alter table work_revisions rename constraint work_revisions_revision_not_null
    to work_sources_revision_not_null;
alter table work_revisions rename constraint work_revisions_media_type_not_null
    to work_sources_media_type_not_null;
alter table work_revisions rename constraint work_revisions_id_not_null
    to work_sources_id_not_null;
alter table work_revisions rename constraint work_revisions_created_at_not_null
    to work_sources_created_at_not_null;
alter table work_revisions rename constraint work_revisions_work_id_not_null
    to work_sources_work_id_not_null;

-- The replaced words all start with "work", and the underscore keeps the rule
-- off working_copy_version and every other word that merely begins the same way.
-- +goose StatementBegin
create function asset_word(name text) returns text language sql immutable as $$
    select replace(replace(replace(replace(replace(
        name,
        'work_follows', 'asset_watches'),
        'work_snapshot_summaries', 'asset_snapshot_projections'),
        'work_summaries', 'asset_projections'),
        'works_', 'assets_'),
        'work_', 'asset_')
$$;
-- +goose StatementEnd

-- +goose StatementBegin
do $$
declare
    subject record;
begin
    for subject in
        select holder.oid::regclass::text as on_table, held.conname as name
          from pg_constraint held
          join pg_class holder on holder.oid = held.conrelid
          join pg_namespace space on space.oid = holder.relnamespace
         where space.nspname = 'public'
           and asset_word(held.conname) <> held.conname
    loop
        execute format('alter table %s rename constraint %I to %I',
            subject.on_table, subject.name, asset_word(subject.name));
    end loop;

    for subject in
        select indexed.indexname as name
          from pg_indexes indexed
         where indexed.schemaname = 'public'
           and asset_word(indexed.indexname) <> indexed.indexname
    loop
        execute format('alter index public.%I rename to %I', subject.name, asset_word(subject.name));
    end loop;
end $$;
-- +goose StatementEnd

drop function asset_word(text);

alter index public.ingest_operations_queue_idx rename to ingest_operations_work_idx;

alter table users rename column nsfw_preference to nsfw_visibility;
alter table publication_destinations rename column type to kind;
alter table protected_delivery_apps rename column work_id to asset_id;
alter table protected_content rename column owner_type to owner_kind;
alter table protected_content rename column work_id to asset_id;
alter table notifications rename column work_id to asset_id;
alter table notification_events rename column work_id to asset_id;
alter table migration_preserved_records rename column work_id to asset_id;
alter table migration_legacy_counters rename column work_id to asset_id;
alter table migration_exceptions rename column type to kind;
alter table migration_exceptions rename column work_id to asset_id;
alter table instance_library_entries rename column work_id to asset_id;
alter table instance_deliveries rename column work_id to asset_id;
alter table ingest_operations rename column visibility to discovery;
alter table ingest_operations rename column target_work_id to target_asset_id;
alter table ingest_operations rename column work_id to asset_id;
alter table download_events rename column visibility to discovery;
alter table download_events rename column work_id to asset_id;
alter table work_vault_pictures rename column work_id to asset_id;
alter table work_update_events rename column work_id to asset_id;
alter table work_update_destinations rename column type to kind;
alter table work_update_destination_defaults rename column work_id to asset_id;
alter table work_update_deliveries rename column destination_type to destination_kind;
alter table work_summaries rename column work_id to asset_id;
alter table work_snapshots rename column work_id to asset_id;
alter table work_snapshot_summaries rename column summary to projection;
alter table work_snapshot_media rename column work_id to asset_id;
alter table work_revisions rename column work_id to asset_id;
alter table work_preserved_data rename column owner_type to owner_kind;
alter table work_preserved_data rename column work_id to asset_id;
alter table work_media rename column work_id to asset_id;
alter table work_legacy_paths rename column work_id to asset_id;
alter table work_follows rename column work_id to asset_id;
alter table work_blocks rename column work_id to asset_id;
alter table works rename column work_version to asset_version;
alter table works rename column visibility to discovery;
alter table works rename column type to kind;

alter table work_follows rename to asset_watches;
alter table work_vault_pictures rename to asset_vault_pictures;
alter table work_update_events rename to asset_update_events;
alter table work_update_destinations rename to asset_update_destinations;
alter table work_update_destination_defaults rename to asset_update_destination_defaults;
alter table work_update_delivery_attempts rename to asset_update_delivery_attempts;
alter table work_update_deliveries rename to asset_update_deliveries;
alter table work_snapshots rename to asset_snapshots;
alter table work_snapshot_prompt_matches rename to asset_snapshot_prompt_matches;
alter table work_snapshot_summaries rename to asset_snapshot_projections;
alter table work_snapshot_media rename to asset_snapshot_media;
alter table work_revisions rename to asset_revisions;
alter table work_summaries rename to asset_projections;
alter table work_preserved_data rename to asset_preserved_data;
alter table work_media rename to asset_media;
alter table work_legacy_paths rename to asset_legacy_paths;
alter table work_blocks rename to asset_blocks;
alter table works rename to assets;

alter trigger work_snapshot_media_immutable on asset_snapshot_media rename to asset_snapshot_media_immutable;
alter trigger work_snapshots_immutable on asset_snapshots rename to asset_snapshots_immutable;
alter trigger capture_work_snapshot_summary on asset_snapshots rename to capture_asset_snapshot_projection;
alter trigger invalidate_work_candidate on assets rename to invalidate_asset_candidate;
alter trigger recorded_work_media_immutable on asset_media rename to recorded_asset_media_immutable;

alter function capture_work_snapshot_summary() rename to capture_asset_snapshot_projection;
alter function guard_work_snapshot() rename to guard_asset_snapshot;
alter function guard_work_snapshot_media() rename to guard_asset_snapshot_media;
alter function guard_recorded_work_media() rename to guard_recorded_asset_media;
alter function invalidate_work_candidate() rename to invalidate_asset_candidate;
alter function invalidate_protected_work_candidate() rename to invalidate_protected_asset_candidate;
alter function record_work_snapshot(uuid, boolean, text, text, text) rename to record_asset_snapshot;
alter function record_initial_work_snapshot(uuid, boolean) rename to record_initial_asset_snapshot;

-- +goose StatementBegin
create or replace function capture_asset_snapshot_projection() returns trigger
    language plpgsql as $$
begin
    insert into public.asset_snapshot_projections
    select new.id, to_jsonb(p) from public.asset_projections p where p.asset_id = new.asset_id;
    return new;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function guard_asset_snapshot() returns trigger
    language plpgsql as $$
begin
    if tg_op = 'UPDATE'
        and (to_jsonb(new) - array['summary', 'notes', 'notes_edited_at', 'withdrawn_at', 'withdrawal_explanation'])
            = (to_jsonb(old) - array['summary', 'notes', 'notes_edited_at', 'withdrawn_at', 'withdrawal_explanation'])
        and (
            (
                new.withdrawn_at is not distinct from old.withdrawn_at
                and new.withdrawal_explanation is not distinct from old.withdrawal_explanation
                and new.notes_edited_at is not null
            )
            or (
                new.summary is not distinct from old.summary
                and new.notes is not distinct from old.notes
                and new.notes_edited_at is not distinct from old.notes_edited_at
                and old.withdrawn_at is null
                and new.withdrawn_at is not null
            )
        ) then
        return new;
    end if;
    if tg_op = 'DELETE' and not exists (
        select 1 from assets where id = old.asset_id
        and (deleted_at is null or recoverable_until > now())
    ) then
        return old;
    end if;
    raise exception 'Published asset snapshots are immutable';
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function guard_asset_snapshot_media() returns trigger
    language plpgsql as $$
begin
    if tg_op = 'INSERT' then
        if exists (select 1 from asset_snapshots where id = new.snapshot_id
            and payload->'media_ids' @> to_jsonb(array[new.media_id])) then
            return new;
        end if;
    elsif tg_op = 'DELETE' and not exists (
        select 1 from asset_snapshots where id = old.snapshot_id
    ) then
        return old;
    end if;
    raise exception 'Published media references are immutable';
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function guard_recorded_asset_media() returns trigger
    language plpgsql as $$
begin
    if not exists (select 1 from asset_snapshot_media where media_id = old.id) then
        return new;
    end if;
    if (to_jsonb(new) - 'is_current') = (to_jsonb(old) - 'is_current') then
        return new;
    end if;
    if new.blob_id is null
        and (to_jsonb(new) - 'blob_id') = (to_jsonb(old) - 'blob_id')
        and (exists (select 1 from blobs b join blob_tombstones t on t.sha256 = b.sha256
                     where b.id = old.blob_id)
             or not exists (select 1 from assets where id = old.asset_id
                            and (deleted_at is null or recoverable_until > now()))) then
        return new;
    end if;
    raise exception 'Published media is immutable';
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function invalidate_asset_candidate() returns trigger
    language plpgsql as $$
begin
    if row(new.owner_id, new.discovery, new.withheld_at, new.deleted_at, new.lifecycle, new.published_snapshot_id)
       is distinct from row(old.owner_id, old.discovery, old.withheld_at, old.deleted_at, old.lifecycle, old.published_snapshot_id) then
        new.working_copy_version := old.working_copy_version + 1;
    end if;
    return new;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function invalidate_protected_asset_candidate() returns trigger
    language plpgsql as $$
begin
    update assets set working_copy_version = working_copy_version + 1
    where id = coalesce(new.asset_id, old.asset_id);
    return null;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function protected_content_requires_delivery_policy() returns trigger
    language plpgsql as $$
declare
    protected_asset_id uuid;
begin
    if tg_op = 'DELETE' then
        protected_asset_id := old.asset_id;
    else
        protected_asset_id := new.asset_id;
    end if;
    if exists (
        select 1 from protected_content where asset_id = protected_asset_id
    ) then
        if not exists (
            select 1 from protected_delivery_apps where asset_id = protected_asset_id
        ) then
            raise exception 'protected content requires an allowed delivery app';
        end if;
    elsif exists (
        select 1 from protected_delivery_apps where asset_id = protected_asset_id
    ) then
        raise exception 'allowed delivery apps require protected content';
    end if;
    return null;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function record_asset_snapshot(
    target uuid, baseline boolean, given_summary text, given_notes text, given_label text
) returns uuid language plpgsql as $$
declare
    owned assets%rowtype;
    recorded uuid;
    pictures uuid[];
begin
    select * into owned from assets where id = target for update;
    if not found or owned.lifecycle <> 'published'
        or (owned.deleted_at is not null and owned.recoverable_until <= now()) then
        return null;
    end if;

    select coalesce(array_agg(distinct media_id), '{}'::uuid[]) into pictures from (
        select id as media_id from asset_media where asset_id = target and is_current
        union select owned.cover_media_id where owned.cover_media_id is not null
        union select (value #>> '{}')::uuid from asset_blocks,
            lateral jsonb_path_query(elements, '$[*] ? (@.type == "image_set").content.images[*].mediaId') value
            where asset_id = target
    ) referenced;
    perform 1 from asset_media where id = any(pictures) for share;

    insert into asset_snapshots
        (asset_id, number, initial_recorded, version_label, content_generation,
         source_revision_id, payload, protected_payloads, summary, notes)
    values (target,
        (select coalesce(max(number), 0) + 1 from asset_snapshots where asset_id = target),
        baseline, coalesce(given_label, owned.asset_version), owned.content_generation,
        owned.current_revision_id,
        jsonb_build_object(
            'schema_version', 1,
            'kind', owned.kind, 'name', owned.name, 'blurb', owned.blurb,
            'tags', owned.tags, 'is_nsfw', owned.is_nsfw,
            'asset_version', owned.asset_version, 'credited_author', owned.credited_author,
            'nickname', owned.nickname, 'origin_format', owned.origin_format,
            'cover_media_id', owned.cover_media_id,
            'media_ids', pictures,
            'blocks', coalesce((select jsonb_agg(to_jsonb(b) - 'asset_id' order by b.position)
                from asset_blocks b where b.asset_id = target), '[]'::jsonb),
            'preserved_data', coalesce((select jsonb_agg(jsonb_build_object(
                'id', p.id, 'owner_kind', p.owner_kind, 'owner_id', p.owner_id,
                'namespace', p.namespace, 'payload', p.payload::text) order by p.id)
                from asset_preserved_data p where p.asset_id = target), '[]'::jsonb)),
        coalesce((select jsonb_agg(to_jsonb(p) - 'asset_id' order by p.owner_kind, p.owner_id)
            from protected_content p where p.asset_id = target), '[]'::jsonb),
        coalesce(given_summary, ''), coalesce(given_notes, ''))
    returning id into recorded;

    insert into asset_snapshot_media (snapshot_id, asset_id, media_id)
    select recorded, target, unnest(pictures);

    update assets set published_snapshot_id = recorded where id = target;
    return recorded;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function record_initial_asset_snapshot(target uuid, baseline boolean)
    returns uuid language plpgsql as $$
declare
    existing uuid;
begin
    select published_snapshot_id into existing from assets where id = target for update;
    if existing is not null then
        return existing;
    end if;
    return record_asset_snapshot(target, baseline, '', '', null);
end;
$$;
-- +goose StatementEnd

create schema asset_public;

create view asset_public.assets as
select a.id, a.kind,
    case when s.id is null then a.current_revision_id else s.source_revision_id end as current_revision_id,
    a.owner_id,
    case when s.id is null then a.name else s.payload ->> 'name' end as name,
    case when s.id is null then a.blurb else s.payload ->> 'blurb' end as blurb,
    case when s.id is null then a.tags
         else array(select jsonb_array_elements_text(s.payload -> 'tags')) end as tags,
    case when s.id is null then a.cover_media_id
         else (s.payload ->> 'cover_media_id')::uuid end as cover_media_id,
    case when s.id is null then a.is_nsfw else (s.payload ->> 'is_nsfw')::boolean end as is_nsfw,
    a.discovery, a.created_at, a.updated_at, a.indexed_at,
    a.withheld_at, a.withheld_by, a.withheld_reason,
    a.deleted_at, a.recoverable_until, a.lifecycle,
    case when s.id is null then a.asset_version else s.payload ->> 'asset_version' end as asset_version,
    case when s.id is null then a.credited_author
         else s.payload ->> 'credited_author' end as credited_author,
    case when s.id is null then a.nickname else s.payload ->> 'nickname' end as nickname,
    case when s.id is null then a.origin_format else s.payload ->> 'origin_format' end as origin_format,
    coalesce(s.content_generation, a.content_generation) as content_generation,
    a.published_snapshot_id
  from assets a
  left join asset_snapshots s on s.id = a.published_snapshot_id;

create view asset_public.asset_blocks as
select b.* from public.asset_blocks b join public.assets a on a.id = b.asset_id
where a.published_snapshot_id is null
union all
select r.id, a.id as asset_id, r.definition, r.title, r.position, r.hidden, r.layout, r.width, r.elements
from public.assets a join public.asset_snapshots s on s.id = a.published_snapshot_id
cross join lateral jsonb_array_elements(s.payload->'blocks') b
cross join lateral jsonb_populate_record(null::public.asset_blocks, b.value) r;

create view asset_public.asset_media as
select m.id, m.asset_id, m.role, m.width, m.height, m.created_at, m.blob_id, m.is_extracted,
    true as is_current
from asset_media m join assets a on a.id = m.asset_id
where a.published_snapshot_id is null and m.is_current
   or exists (select 1 from asset_snapshot_media r
               where r.snapshot_id = a.published_snapshot_id and r.media_id = m.id);

create view asset_public.asset_preserved_data as
select p.id, p.asset_id, p.owner_kind, p.owner_id, p.namespace, p.payload
from asset_preserved_data p join assets a on a.id = p.asset_id
where a.published_snapshot_id is null
union all
select (p.value ->> 'id')::uuid as id, a.id as asset_id,
    p.value ->> 'owner_kind' as owner_kind,
    (p.value ->> 'owner_id')::uuid as owner_id,
    p.value ->> 'namespace' as namespace,
    (p.value ->> 'payload')::json as payload
from assets a join asset_snapshots s on s.id = a.published_snapshot_id
cross join lateral jsonb_array_elements(s.payload -> 'preserved_data') p;

create view asset_public.asset_projections as
select p.asset_id, p.export, p.export_stamp, p.export_computed_at,
    p.facets, p.facet_stamp, p.facet_computed_at
from asset_projections p join assets a on a.id = p.asset_id
where a.published_snapshot_id is null
union all
select a.id as asset_id, r.export, r.export_stamp, r.export_computed_at,
    r.facets, r.facet_stamp, r.facet_computed_at
from assets a join asset_snapshot_projections p on p.snapshot_id = a.published_snapshot_id
cross join lateral jsonb_populate_record(null::public.asset_projections, p.projection) r;

create view asset_public.protected_content as
select p.* from public.protected_content p join public.assets a on a.id = p.asset_id
where a.published_snapshot_id is null
union all
select a.id as asset_id, r.owner_kind, r.owner_id, r.payload_type, r.payload, r.source_key, r.digest
from public.assets a join public.asset_snapshots s on s.id = a.published_snapshot_id
cross join lateral jsonb_array_elements(s.protected_payloads) p
cross join lateral jsonb_populate_record(null::public.protected_content, p.value) r;

create view public.inbox_entries as
select entry.id, entry.account_id, entry.type, entry.asset_id, entry.words,
    entry.created_at, entry.read_at, entry.update_count
  from notifications entry
 where entry.asset_id is null
    or exists (
        select 1
          from assets as subject
         where subject.id = entry.asset_id
           and (subject.owner_id = entry.account_id
                or (subject.deleted_at is null and subject.withheld_at is null))
    );
