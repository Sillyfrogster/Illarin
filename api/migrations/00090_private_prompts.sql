-- +goose Up

-- Protected content is private prompts, and the apps allowed to receive them are
-- the private prompt apps. The v1 sealed blocks are preserved prompts.

drop trigger protected_content_policy_after_content_change on protected_content;
drop trigger protected_content_policy_after_policy_change on protected_delivery_apps;
drop trigger invalidate_protected_candidate on protected_content;
drop trigger invalidate_delivery_candidate on protected_delivery_apps;
drop function protected_content_requires_delivery_policy();
drop function invalidate_protected_work_candidate();

alter table protected_content rename to private_prompts;
alter table protected_delivery_apps rename to private_prompt_apps;
alter table work_versions rename column protected_payloads to private_prompts;
alter view work_public.protected_content rename to private_prompts;

-- +goose StatementBegin
do $$
declare
    subject record;
begin
    for subject in
        select held.conrelid::regclass::text as on_table, held.conname as name
          from pg_constraint held
         where held.conrelid::regclass::text in ('private_prompts', 'private_prompt_apps', 'work_versions')
           and held.conname ~ '^(protected_content|protected_delivery_apps|work_versions_protected_payloads)_'
    loop
        execute format('alter table %s rename constraint %I to %I', subject.on_table, subject.name,
            regexp_replace(regexp_replace(regexp_replace(subject.name,
                '^protected_content_', 'private_prompts_'),
                '^protected_delivery_apps_', 'private_prompt_apps_'),
                '^work_versions_protected_payloads_', 'work_versions_private_prompts_'));
    end loop;
end $$;
-- +goose StatementEnd

-- +goose StatementBegin
create function private_prompts_need_an_allowed_app()
returns trigger
language plpgsql
as $$
declare
    checked_work_id uuid;
begin
    if tg_op = 'DELETE' then
        checked_work_id := old.work_id;
    else
        checked_work_id := new.work_id;
    end if;
    if exists (
        select 1 from private_prompts where work_id = checked_work_id
    ) then
        if not exists (
            select 1 from private_prompt_apps where work_id = checked_work_id
        ) then
            raise exception 'private prompts need an allowed app';
        end if;
    elsif exists (
        select 1 from private_prompt_apps where work_id = checked_work_id
    ) then
        raise exception 'allowed apps need private prompts';
    end if;
    return null;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create function invalidate_private_prompts_candidate()
returns trigger
language plpgsql
as $$
begin
    update works set drafted_changes_version = drafted_changes_version + 1
    where id = coalesce(new.work_id, old.work_id);
    return null;
end;
$$;
-- +goose StatementEnd

create constraint trigger private_prompts_need_an_allowed_app_after_prompt_change
after insert or update or delete on private_prompts
deferrable initially deferred
for each row execute function private_prompts_need_an_allowed_app();

create constraint trigger private_prompts_need_an_allowed_app_after_app_change
after insert or update or delete on private_prompt_apps
deferrable initially deferred
for each row execute function private_prompts_need_an_allowed_app();

create trigger invalidate_private_prompts_candidate
after insert or update or delete on private_prompts
for each row execute function invalidate_private_prompts_candidate();

create trigger invalidate_private_prompt_apps_candidate
after insert or update or delete on private_prompt_apps
for each row execute function invalidate_private_prompts_candidate();

-- +goose StatementBegin
create or replace function record_work_version(target uuid, baseline boolean, given_summary text, given_notes text, given_label text)
 returns uuid
 language plpgsql
as $$
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

-- A prompt fragment kept in stored blocks marks its text private under the key
-- "private" rather than "protected".
-- +goose StatementBegin
create function rename_fragment_key(value jsonb, old_key text, new_key text) returns jsonb
    language plpgsql immutable as $$
begin
    if jsonb_typeof(value) = 'array' then
        return coalesce((select jsonb_agg(rename_fragment_key(item, old_key, new_key) order by position)
            from jsonb_array_elements(value) with ordinality listed(item, position)), '[]'::jsonb);
    end if;
    if jsonb_typeof(value) <> 'object' then
        return value;
    end if;
    return coalesce((select jsonb_object_agg(key,
        case when key = 'fragments' and jsonb_typeof(item) = 'array' then
            coalesce((select jsonb_agg(case
                    when jsonb_typeof(fragment) = 'object' and fragment ? old_key
                        then (fragment - old_key) || jsonb_build_object(new_key, fragment -> old_key)
                    else fragment end order by position)
                from jsonb_array_elements(item) with ordinality listed(fragment, position)), '[]'::jsonb)
        else rename_fragment_key(item, old_key, new_key) end)
        from jsonb_each(value) fields(key, item)), '{}'::jsonb);
end;
$$;
-- +goose StatementEnd

update work_blocks
   set elements = rename_fragment_key(elements, 'protected', 'private')
 where elements::text like '%"protected"%';

alter table work_versions disable trigger work_versions_immutable;
update work_versions
   set payload = rename_fragment_key(payload, 'protected', 'private')
 where payload::text like '%"protected"%';
alter table work_versions enable trigger work_versions_immutable;

-- A replacement waiting for its owner's answer keeps the upload it read, and the
-- names in it are field names.
update ingest_operations
   set replacement_preview = jsonb_set(jsonb_set(
           rename_fragment_key(replacement_preview, 'protected', 'private'),
           '{Prepared}', ((rename_fragment_key(replacement_preview, 'protected', 'private') -> 'Prepared') - 'Protected')
               || jsonb_build_object('PrivatePrompts', replacement_preview -> 'Prepared' -> 'Protected')),
           '{Preview}', ((replacement_preview -> 'Preview') - 'Seals')
               || jsonb_build_object('PrivatePrompts', replacement_preview -> 'Preview' -> 'Seals'))
 where replacement_preview ? 'Prepared' and replacement_preview ? 'Preview';

drop function rename_fragment_key(jsonb, text, text);

update migration_preserved_records
   set source_table = 'preserved_prompts'
 where source_table = 'preset_sealed_blocks';

-- +goose Down

update migration_preserved_records
   set source_table = 'preset_sealed_blocks'
 where source_table = 'preserved_prompts';

-- +goose StatementBegin
create function rename_fragment_key(value jsonb, old_key text, new_key text) returns jsonb
    language plpgsql immutable as $$
begin
    if jsonb_typeof(value) = 'array' then
        return coalesce((select jsonb_agg(rename_fragment_key(item, old_key, new_key) order by position)
            from jsonb_array_elements(value) with ordinality listed(item, position)), '[]'::jsonb);
    end if;
    if jsonb_typeof(value) <> 'object' then
        return value;
    end if;
    return coalesce((select jsonb_object_agg(key,
        case when key = 'fragments' and jsonb_typeof(item) = 'array' then
            coalesce((select jsonb_agg(case
                    when jsonb_typeof(fragment) = 'object' and fragment ? old_key
                        then (fragment - old_key) || jsonb_build_object(new_key, fragment -> old_key)
                    else fragment end order by position)
                from jsonb_array_elements(item) with ordinality listed(fragment, position)), '[]'::jsonb)
        else rename_fragment_key(item, old_key, new_key) end)
        from jsonb_each(value) fields(key, item)), '{}'::jsonb);
end;
$$;
-- +goose StatementEnd

update ingest_operations
   set replacement_preview = jsonb_set(jsonb_set(
           rename_fragment_key(replacement_preview, 'private', 'protected'),
           '{Prepared}', ((rename_fragment_key(replacement_preview, 'private', 'protected') -> 'Prepared') - 'PrivatePrompts')
               || jsonb_build_object('Protected', replacement_preview -> 'Prepared' -> 'PrivatePrompts')),
           '{Preview}', ((replacement_preview -> 'Preview') - 'PrivatePrompts')
               || jsonb_build_object('Seals', replacement_preview -> 'Preview' -> 'PrivatePrompts'))
 where replacement_preview ? 'Prepared' and replacement_preview ? 'Preview';

alter table work_versions disable trigger work_versions_immutable;
update work_versions
   set payload = rename_fragment_key(payload, 'private', 'protected')
 where payload::text like '%"private"%';
alter table work_versions enable trigger work_versions_immutable;

update work_blocks
   set elements = rename_fragment_key(elements, 'private', 'protected')
 where elements::text like '%"private"%';

drop function rename_fragment_key(jsonb, text, text);

drop trigger invalidate_private_prompt_apps_candidate on private_prompt_apps;
drop trigger invalidate_private_prompts_candidate on private_prompts;
drop trigger private_prompts_need_an_allowed_app_after_app_change on private_prompt_apps;
drop trigger private_prompts_need_an_allowed_app_after_prompt_change on private_prompts;
drop function invalidate_private_prompts_candidate();
drop function private_prompts_need_an_allowed_app();

alter table work_versions rename column private_prompts to protected_payloads;
alter view work_public.private_prompts rename to protected_content;
alter table private_prompt_apps rename to protected_delivery_apps;
alter table private_prompts rename to protected_content;

-- +goose StatementBegin
do $$
declare
    subject record;
begin
    for subject in
        select held.conrelid::regclass::text as on_table, held.conname as name
          from pg_constraint held
         where held.conrelid::regclass::text in ('protected_content', 'protected_delivery_apps', 'work_versions')
           and held.conname ~ '^(private_prompts|private_prompt_apps|work_versions_private_prompts)_'
    loop
        execute format('alter table %s rename constraint %I to %I', subject.on_table, subject.name,
            regexp_replace(regexp_replace(regexp_replace(subject.name,
                '^private_prompts_', 'protected_content_'),
                '^private_prompt_apps_', 'protected_delivery_apps_'),
                '^work_versions_private_prompts_', 'work_versions_protected_payloads_'));
    end loop;
end $$;
-- +goose StatementEnd

-- +goose StatementBegin
create function protected_content_requires_delivery_policy()
returns trigger
language plpgsql
as $$
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
create function invalidate_protected_work_candidate()
returns trigger
language plpgsql
as $$
begin
    update works set drafted_changes_version = drafted_changes_version + 1
    where id = coalesce(new.work_id, old.work_id);
    return null;
end;
$$;
-- +goose StatementEnd

create constraint trigger protected_content_policy_after_content_change
after insert or update or delete on protected_content
deferrable initially deferred
for each row execute function protected_content_requires_delivery_policy();

create constraint trigger protected_content_policy_after_policy_change
after insert or update or delete on protected_delivery_apps
deferrable initially deferred
for each row execute function protected_content_requires_delivery_policy();

create trigger invalidate_protected_candidate
after insert or update or delete on protected_content
for each row execute function invalidate_protected_work_candidate();

create trigger invalidate_delivery_candidate
after insert or update or delete on protected_delivery_apps
for each row execute function invalidate_protected_work_candidate();

-- +goose StatementBegin
create or replace function record_work_version(target uuid, baseline boolean, given_summary text, given_notes text, given_label text)
 returns uuid
 language plpgsql
as $$
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
