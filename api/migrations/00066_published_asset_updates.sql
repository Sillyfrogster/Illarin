-- +goose Up
alter table asset_snapshots
    add column summary text not null default '',
    add column notes text not null default '';

-- +goose StatementBegin
create function record_asset_snapshot(
    target uuid,
    baseline boolean,
    given_summary text,
    given_notes text,
    given_label text
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
create or replace function record_initial_asset_snapshot(target uuid, baseline boolean) returns uuid
language plpgsql as $$
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

-- +goose Down
-- +goose StatementBegin
create or replace function record_initial_asset_snapshot(target uuid, baseline boolean) returns uuid
language plpgsql as $$
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
    if owned.published_snapshot_id is not null then
        return owned.published_snapshot_id;
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
        (asset_id, number, initial_recorded, version_label, content_generation, source_revision_id, payload, protected_payloads)
    values (target, 1, baseline, owned.asset_version, owned.content_generation, owned.current_revision_id,
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
            from protected_content p where p.asset_id = target), '[]'::jsonb))
    returning id into recorded;

    insert into asset_snapshot_media (snapshot_id, asset_id, media_id)
    select recorded, target, unnest(pictures);

    update assets set published_snapshot_id = recorded where id = target;
    return recorded;
end;
$$;
-- +goose StatementEnd

drop function record_asset_snapshot(uuid, boolean, text, text, text);
alter table asset_snapshots drop column notes;
alter table asset_snapshots drop column summary;
