-- +goose Up

-- The version number is the one number that tells a connected app a work has
-- something new, so the content generation goes. A recorded version is a
-- version, not a snapshot; the working copy is the drafted changes; and an
-- uploaded file is an original file, not a revision.

drop view work_public.works;

alter table works drop column content_generation;
alter table work_snapshots drop column content_generation;
alter table instance_library_entries rename column content_generation to version_number;

alter table works rename column working_copy_version to drafted_changes_version;

alter table work_revisions rename to work_original_files;
alter table work_original_files rename column revision to number;
alter table works rename column current_revision_id to original_file_id;
alter table work_snapshots rename column source_revision_id to original_file_id;
alter table download_events rename column revision_id to original_file_id;

alter table work_snapshots rename to work_versions;
alter table work_snapshot_media rename to work_version_media;
alter table work_snapshot_prompt_matches rename to work_version_prompt_matches;
alter table work_snapshot_summaries rename to work_version_summaries;
alter table work_version_media rename column snapshot_id to version_id;
alter table work_version_prompt_matches rename column snapshot_id to version_id;
alter table work_version_summaries rename column snapshot_id to version_id;
alter table work_update_events rename column snapshot_id to version_id;
alter table works rename column published_snapshot_id to published_version_id;

-- An upload refused because the drafted changes moved says so under the new word.
alter table ingest_operations drop constraint ingest_operations_failure_reason_check;
update ingest_operations set failure_reason = 'drafted_changes_conflict'
 where failure_reason = 'working_copy_conflict';
alter table ingest_operations add constraint ingest_operations_failure_reason_check check (
    failure_reason is null or failure_reason in (
        'malformed_input', 'unsupported_format', 'unsupported_version', 'safety_violation',
        'wrong_type', 'limit_exceeded', 'internal_failure', 'drafted_changes_conflict',
        'work_unavailable'));

-- A trigger holds its function by identity, so each one is renamed where it
-- stands and then given its new body.
alter function capture_work_snapshot_summary() rename to capture_work_version_summary;
alter function guard_work_snapshot() rename to guard_work_version;
alter function guard_work_snapshot_media() rename to guard_work_version_media;
alter function record_work_snapshot(uuid, boolean, text, text, text) rename to record_work_version;
alter function record_initial_work_snapshot(uuid, boolean) rename to record_initial_work_version;

-- +goose StatementBegin
create or replace function capture_work_version_summary() returns trigger
    language plpgsql as $$
begin
    insert into public.work_version_summaries
    select new.id, to_jsonb(s) from public.work_summaries s where s.work_id = new.work_id;
    return new;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function guard_work_version() returns trigger
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
    raise exception 'Published versions are immutable';
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function guard_work_version_media() returns trigger
    language plpgsql as $$
begin
    if tg_op = 'INSERT' then
        if exists (select 1 from work_versions where id = new.version_id
            and payload->'media_ids' @> to_jsonb(array[new.media_id])) then
            return new;
        end if;
    elsif tg_op = 'DELETE' and not exists (
        select 1 from work_versions where id = old.version_id
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
    if not exists (select 1 from work_version_media where media_id = old.id) then
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
    if row(new.owner_id, new.visibility, new.withheld_at, new.deleted_at, new.lifecycle, new.published_version_id)
       is distinct from row(old.owner_id, old.visibility, old.withheld_at, old.deleted_at, old.lifecycle, old.published_version_id) then
        new.drafted_changes_version := old.drafted_changes_version + 1;
    end if;
    return new;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function invalidate_protected_work_candidate() returns trigger
    language plpgsql as $$
begin
    update works set drafted_changes_version = drafted_changes_version + 1
    where id = coalesce(new.work_id, old.work_id);
    return null;
end;
$$;
-- +goose StatementEnd

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
         original_file_id, payload, protected_payloads, summary, notes)
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
            from protected_content p where p.work_id = target), '[]'::jsonb),
        coalesce(given_summary, ''), coalesce(given_notes, ''))
    returning id into recorded;

    insert into work_version_media (version_id, work_id, media_id)
    select recorded, target, unnest(pictures);

    update works set published_version_id = recorded where id = target;
    return recorded;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function record_initial_work_version(target uuid, baseline boolean)
    returns uuid language plpgsql as $$
declare
    existing uuid;
begin
    select published_version_id into existing from works where id = target for update;
    if existing is not null then
        return existing;
    end if;
    return record_work_version(target, baseline, '', '', null);
end;
$$;
-- +goose StatementEnd

alter trigger capture_work_snapshot_summary on work_versions rename to capture_work_version_summary;
alter trigger work_snapshot_media_immutable on work_version_media rename to work_version_media_immutable;
alter trigger work_snapshots_immutable on work_versions rename to work_versions_immutable;

-- Every constraint and index named for the old words follows them. The blog's
-- own revisions keep their name, so its tables are left out.
-- +goose StatementBegin
create function version_word(name text) returns text language sql immutable as $$
    select replace(replace(replace(replace(replace(replace(replace(replace(replace(
        name,
        'snapshot', 'version'),
        'work_revisions_work_id_revision_key', 'work_original_files_work_id_number_key'),
        'work_revisions_revision_not_null', 'work_original_files_number_not_null'),
        'work_revisions', 'work_original_files'),
        'source_revision_id', 'original_file_id'),
        'current_revision', 'original_file'),
        'revision', 'original_file'),
        'working_copy_version', 'drafted_changes_version'),
        'instance_library_entries_generation_check', 'instance_library_entries_version_number_check')
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
           and holder.relname not like 'post%'
           and holder.relname not like 'publication%'
           and version_word(held.conname) <> held.conname
    loop
        execute format('alter table %s rename constraint %I to %I',
            subject.on_table, subject.name, version_word(subject.name));
    end loop;

    for subject in
        select indexed.indexname as name
          from pg_indexes indexed
         where indexed.schemaname = 'public'
           and indexed.tablename not like 'post%'
           and indexed.tablename not like 'publication%'
           and version_word(indexed.indexname) <> indexed.indexname
    loop
        execute format('alter index public.%I rename to %I', subject.name, version_word(subject.name));
    end loop;
end $$;
-- +goose StatementEnd

drop function version_word(text);

alter table instance_library_entries rename constraint instance_library_entries_content_generation_not_null
    to instance_library_entries_version_number_not_null;

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

-- +goose Down

drop view work_public.works;

-- +goose StatementBegin
create function snapshot_word(name text) returns text language sql immutable as $$
    select replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(
        name,
        'instance_library_entries_version_number', 'instance_library_entries_generation'),
        'drafted_changes_version', 'working_copy_version'),
        'work_original_files_work_id_number_key', 'work_revisions_work_id_revision_key'),
        'work_original_files_number_not_null', 'work_revisions_revision_not_null'),
        'work_original_files', 'work_revisions'),
        'work_versions_original_file_id', 'work_snapshots_source_revision_id'),
        'works_original_file_fk', 'works_current_revision_fk'),
        'download_events_original_file_fk', 'download_events_revision_fk'),
        'work_version_withdrawal_check', 'work_snapshot_withdrawal_check'),
        'work_version_media', 'work_snapshot_media'),
        'work_version_summaries', 'work_snapshot_summaries'),
        'work_version_prompt_matches', 'work_snapshot_prompt_matches'),
        'work_versions', 'work_snapshots'),
        'published_version', 'published_snapshot'),
        '_version_id', '_snapshot_id'),
        'work_update_events_version_key', 'work_update_events_snapshot_key')
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
           and holder.relname in ('works', 'work_original_files', 'download_events',
                'instance_library_entries', 'work_versions', 'work_version_media',
                'work_version_summaries', 'work_version_prompt_matches', 'work_update_events')
           and snapshot_word(held.conname) <> held.conname
    loop
        execute format('alter table %s rename constraint %I to %I',
            subject.on_table, subject.name, snapshot_word(subject.name));
    end loop;

    for subject in
        select indexed.indexname as name
          from pg_indexes indexed
         where indexed.schemaname = 'public'
           and indexed.tablename in ('works', 'work_original_files', 'download_events',
                'instance_library_entries', 'work_versions', 'work_version_media',
                'work_version_summaries', 'work_version_prompt_matches', 'work_update_events')
           and snapshot_word(indexed.indexname) <> indexed.indexname
    loop
        execute format('alter index public.%I rename to %I', subject.name, snapshot_word(subject.name));
    end loop;
end $$;
-- +goose StatementEnd

drop function snapshot_word(text);

alter table instance_library_entries rename constraint instance_library_entries_generation_not_null
    to instance_library_entries_content_generation_not_null;

alter trigger capture_work_version_summary on work_versions rename to capture_work_snapshot_summary;
alter trigger work_version_media_immutable on work_version_media rename to work_snapshot_media_immutable;
alter trigger work_versions_immutable on work_versions rename to work_snapshots_immutable;

alter function capture_work_version_summary() rename to capture_work_snapshot_summary;
alter function guard_work_version() rename to guard_work_snapshot;
alter function guard_work_version_media() rename to guard_work_snapshot_media;
alter function record_work_version(uuid, boolean, text, text, text) rename to record_work_snapshot;
alter function record_initial_work_version(uuid, boolean) rename to record_initial_work_snapshot;

alter table works rename column published_version_id to published_snapshot_id;
alter table work_update_events rename column version_id to snapshot_id;
alter table work_version_summaries rename column version_id to snapshot_id;
alter table work_version_prompt_matches rename column version_id to snapshot_id;
alter table work_version_media rename column version_id to snapshot_id;
alter table work_version_summaries rename to work_snapshot_summaries;
alter table work_version_prompt_matches rename to work_snapshot_prompt_matches;
alter table work_version_media rename to work_snapshot_media;
alter table work_versions rename to work_snapshots;

alter table download_events rename column original_file_id to revision_id;
alter table work_snapshots rename column original_file_id to source_revision_id;
alter table works rename column original_file_id to current_revision_id;
alter table work_original_files rename column number to revision;
alter table work_original_files rename to work_revisions;

alter table works rename column drafted_changes_version to working_copy_version;

alter table ingest_operations drop constraint ingest_operations_failure_reason_check;
update ingest_operations set failure_reason = 'working_copy_conflict'
 where failure_reason = 'drafted_changes_conflict';
alter table ingest_operations add constraint ingest_operations_failure_reason_check check (
    failure_reason is null or failure_reason in (
        'malformed_input', 'unsupported_format', 'unsupported_version', 'safety_violation',
        'wrong_type', 'limit_exceeded', 'internal_failure', 'working_copy_conflict',
        'work_unavailable'));

-- The content generation cannot be recovered once dropped; the version number
-- stands in for it, which is what the amended ADR-0023 says it means.
alter table instance_library_entries rename column version_number to content_generation;
alter table work_snapshots add column content_generation integer;
update work_snapshots set content_generation = number;
alter table work_snapshots alter column content_generation set not null;
alter table work_snapshots add constraint work_snapshots_content_generation_check check (content_generation > 0);
alter table works add column content_generation integer not null default 1;
update works work set content_generation = snapshot.number
  from work_snapshots snapshot where snapshot.id = work.published_snapshot_id;
alter table works add constraint works_content_generation_check check (content_generation > 0);

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
