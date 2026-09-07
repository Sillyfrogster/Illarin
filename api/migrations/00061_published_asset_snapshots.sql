-- +goose Up
create table asset_snapshots (
    id                 uuid primary key default gen_random_uuid(),
    asset_id           uuid not null references assets (id) on delete cascade,
    number             integer not null check (number > 0),
    recorded_at        timestamptz not null default now(),
    initial_recorded   boolean not null default false,
    version_label      text not null default '',
    content_generation integer not null check (content_generation > 0),
    source_revision_id uuid,
    payload            jsonb not null check (jsonb_typeof(payload) = 'object'),
    protected_payloads jsonb not null check (jsonb_typeof(protected_payloads) = 'array'),
    unique (asset_id, number),
    unique (id, asset_id),
    foreign key (source_revision_id, asset_id) references asset_revisions (id, asset_id)
        deferrable initially deferred
);

alter table assets add column published_snapshot_id uuid;
alter table assets add constraint assets_published_snapshot_fk
    foreign key (published_snapshot_id, id) references asset_snapshots (id, asset_id)
    deferrable initially deferred;

-- +goose StatementBegin
create function guard_asset_snapshot() returns trigger language plpgsql as $$
begin
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

create trigger asset_snapshots_immutable before update or delete on asset_snapshots
    for each row execute function guard_asset_snapshot();

alter table asset_media add constraint asset_media_id_asset_unique unique (id, asset_id);

create table asset_snapshot_media (
    snapshot_id uuid not null,
    asset_id uuid not null,
    media_id uuid not null,
    primary key (snapshot_id, media_id),
    foreign key (snapshot_id, asset_id) references asset_snapshots (id, asset_id) on delete cascade,
    foreign key (media_id, asset_id) references asset_media (id, asset_id) deferrable initially deferred
);

create index asset_snapshot_media_media_idx on asset_snapshot_media (media_id);

-- +goose StatementBegin
create function guard_asset_snapshot_media() returns trigger language plpgsql as $$
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

create trigger asset_snapshot_media_immutable before insert or update or delete on asset_snapshot_media
    for each row execute function guard_asset_snapshot_media();

-- +goose StatementBegin
create function guard_recorded_asset_media() returns trigger language plpgsql as $$
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

create trigger recorded_asset_media_immutable before update on asset_media
    for each row execute function guard_recorded_asset_media();

-- +goose StatementBegin
create function record_initial_asset_snapshot(target uuid, baseline boolean) returns uuid
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

select record_initial_asset_snapshot(id, true) from assets
where lifecycle = 'published' and published_snapshot_id is null;

-- +goose Down
drop function record_initial_asset_snapshot(uuid, boolean);
drop trigger recorded_asset_media_immutable on asset_media;
drop function guard_recorded_asset_media();
drop table asset_snapshot_media;
drop function guard_asset_snapshot_media();
alter table asset_media drop constraint asset_media_id_asset_unique;
alter table assets drop constraint assets_published_snapshot_fk;
alter table assets drop column published_snapshot_id;
drop table asset_snapshots;
drop function guard_asset_snapshot();
