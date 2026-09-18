-- name: InsertWork :one
insert into works
  (id, type, owner_id, name, blurb, tags, is_nsfw, visibility, lifecycle,
   work_version, credited_author, nickname, origin_format, created_at)
values ($1, $2, $3, $4, $5, $6, sqlc.narg('is_nsfw')::boolean, $7, $8,
        $9, $10, $11, sqlc.narg('origin_format')::text,
        coalesce(sqlc.narg('created_at')::timestamptz, now()))
returning created_at;

-- name: InsertWorkBlock :exec
insert into work_blocks
  (id, work_id, definition, title, position, hidden, layout, width, elements)
values ($1, $2, $3, sqlc.narg('title')::text, $4, $5, $6, $7, $8);

-- name: WorkBlocks :many
select id, definition, title, position, hidden, layout, width, elements
  from work_blocks
 where work_id = $1
 order by position;

-- name: InsertRevision :exec
insert into work_revisions
  (id, work_id, revision, blob_id, media_type, format, identifier)
values ($1, $2, $3, $4, $5, $6, $7);

-- name: SetCurrentRevision :exec
update works set current_revision_id = $2, updated_at = now() where id = $1;

-- name: ListWorks :many
select a.id, a.type, revision.format, a.origin_format,
       a.work_version, a.credited_author, a.nickname, a.lifecycle,
       a.name, a.blurb, a.tags,
       coalesce(a.is_nsfw, true)::boolean as is_nsfw, a.visibility,
       a.current_revision_id, a.created_at
  from works a
  left join work_revisions revision on revision.id = a.current_revision_id
 where a.lifecycle = 'published'
   and a.visibility = 'listed'
   and a.withheld_at is null
   and a.deleted_at is null
   and ($1 = '' or a.type = $1)
   and (not $2::boolean or revision.format is not distinct from $3)
   and ($4::text[] is null or a.tags @> $4)
   and (sqlc.narg('before')::timestamptz is null
        or (a.created_at, a.id)
           < (sqlc.narg('before')::timestamptz, sqlc.narg('before_id')::uuid))
 order by a.created_at desc, a.id desc
 limit $5;

-- name: BrowseWorks :many
select a.id, a.name, coalesce(owner.username, 'unknown') as creator,
       a.type, a.is_nsfw, a.created_at, a.lifecycle,
       cover.id as cover_id, cover.width as cover_width, cover.height as cover_height,
       a.visibility, a.withheld_at, a.withheld_reason
  from works a
  left join work_summaries summary on summary.work_id = a.id
  left join users owner on owner.id = a.owner_id
  left join work_media cover
    on cover.id = a.cover_media_id and cover.work_id = a.id
   and cover.is_current
   and cover.width is not null and cover.height is not null
   and cover.blob_id is not null
 where (a.lifecycle = 'published'
        or (sqlc.arg('own_profile')::boolean
            and a.owner_id = sqlc.narg('creator_id')::uuid))
   and a.deleted_at is null
   and (
       (sqlc.narg('creator_id')::uuid is null
        and a.visibility = 'listed' and a.withheld_at is null)
       or
       (a.owner_id = sqlc.narg('creator_id')::uuid
        and (sqlc.arg('own_profile')::boolean
             or (a.visibility = 'listed' and a.withheld_at is null)))
   )
   and (sqlc.arg('type')::text = '' or a.type = sqlc.arg('type')::text)
   and (sqlc.arg('own_profile')::boolean
        or sqlc.arg('nsfw_preference')::text <> 'hidden' or not a.is_nsfw)
   and (sqlc.arg('platform')::text = '' or exists (
        select 1
          from jsonb_array_elements(coalesce(summary.export, '[]'::jsonb)) as offered(target)
         where offered.target ->> 'format' = any(sqlc.arg('formats')::text[])
   ))
   and (cardinality(sqlc.arg('facet_keys')::text[]) = 0 or not exists (
        select 1
          from unnest(sqlc.arg('facet_keys')::text[]) with ordinality as chosen(key, at)
          join unnest(sqlc.arg('facet_lows')::int[]) with ordinality as lows(low, at)
            on lows.at = chosen.at
          join unnest(sqlc.arg('facet_highs')::int[]) with ordinality as highs(high, at)
            on highs.at = chosen.at
         group by chosen.key
        having not bool_or(
                 coalesce((summary.facets ->> chosen.key)::int, 0) >= lows.low
                 and (highs.high < 0
                      or coalesce((summary.facets ->> chosen.key)::int, 0) <= highs.high))
   ))
   and (sqlc.arg('search_text')::text = ''
        or position(sqlc.arg('search_text')::text in lower(a.name)) > 0
        or position(sqlc.arg('search_text')::text in lower(a.blurb)) > 0
        or position(sqlc.arg('search_text')::text in lower(coalesce(owner.username, ''))) > 0)
   and (sqlc.arg('author')::text = '' or lower(coalesce(owner.username, '')) = sqlc.arg('author')::text)
   and (cardinality(sqlc.arg('tags')::text[]) = 0 or not exists (
        select 1 from unnest(sqlc.arg('tags')::text[]) wanted(tag)
         where not exists (
             select 1 from unnest(a.tags) stored(tag)
              where lower(btrim(stored.tag)) = wanted.tag
         )
   ))
   and (sqlc.narg('before')::timestamptz is null
        or (a.created_at, a.id)
           < (sqlc.narg('before')::timestamptz, sqlc.narg('before_id')::uuid))
 order by a.created_at desc, a.id desc
 limit sqlc.arg('page_size');

-- name: CountBrowseWorks :one
select count(*)
  from works a
  left join work_summaries summary on summary.work_id = a.id
  left join users owner on owner.id = a.owner_id
 where (a.lifecycle = 'published'
        or (sqlc.arg('own_profile')::boolean
            and a.owner_id = sqlc.narg('creator_id')::uuid))
   and a.deleted_at is null
   and (
       (sqlc.narg('creator_id')::uuid is null
        and a.visibility = 'listed' and a.withheld_at is null)
       or
       (a.owner_id = sqlc.narg('creator_id')::uuid
        and (sqlc.arg('own_profile')::boolean
             or (a.visibility = 'listed' and a.withheld_at is null)))
   )
   and (sqlc.arg('type')::text = '' or a.type = sqlc.arg('type')::text)
   and (sqlc.arg('own_profile')::boolean
        or sqlc.arg('nsfw_preference')::text <> 'hidden' or not a.is_nsfw)
   and (sqlc.arg('platform')::text = '' or exists (
        select 1
          from jsonb_array_elements(coalesce(summary.export, '[]'::jsonb)) as offered(target)
         where offered.target ->> 'format' = any(sqlc.arg('formats')::text[])
   ))
   and (cardinality(sqlc.arg('facet_keys')::text[]) = 0 or not exists (
        select 1
          from unnest(sqlc.arg('facet_keys')::text[]) with ordinality as chosen(key, at)
          join unnest(sqlc.arg('facet_lows')::int[]) with ordinality as lows(low, at)
            on lows.at = chosen.at
          join unnest(sqlc.arg('facet_highs')::int[]) with ordinality as highs(high, at)
            on highs.at = chosen.at
         group by chosen.key
        having not bool_or(
                 coalesce((summary.facets ->> chosen.key)::int, 0) >= lows.low
                 and (highs.high < 0
                      or coalesce((summary.facets ->> chosen.key)::int, 0) <= highs.high))
   ))
   and (sqlc.arg('search_text')::text = ''
        or position(sqlc.arg('search_text')::text in lower(a.name)) > 0
        or position(sqlc.arg('search_text')::text in lower(a.blurb)) > 0
        or position(sqlc.arg('search_text')::text in lower(coalesce(owner.username, ''))) > 0)
   and (sqlc.arg('author')::text = '' or lower(coalesce(owner.username, '')) = sqlc.arg('author')::text)
   and (cardinality(sqlc.arg('tags')::text[]) = 0 or not exists (
        select 1 from unnest(sqlc.arg('tags')::text[]) wanted(tag)
         where not exists (
             select 1 from unnest(a.tags) stored(tag)
              where lower(btrim(stored.tag)) = wanted.tag
         )
   ));

-- name: CountSuppressedBrowseWorks :one
select count(*)
  from works a
  left join work_summaries summary on summary.work_id = a.id
  left join users owner on owner.id = a.owner_id
 where a.lifecycle = 'published'
   and a.visibility = 'listed'
   and a.withheld_at is null
   and a.deleted_at is null
   and (sqlc.narg('creator_id')::uuid is null or a.owner_id = sqlc.narg('creator_id')::uuid)
   and (sqlc.arg('type')::text = '' or a.type = sqlc.arg('type')::text)
   and (sqlc.arg('platform')::text = '' or exists (
        select 1
          from jsonb_array_elements(coalesce(summary.export, '[]'::jsonb)) as offered(target)
         where offered.target ->> 'format' = any(sqlc.arg('formats')::text[])
   ))
   and (cardinality(sqlc.arg('facet_keys')::text[]) = 0 or not exists (
        select 1
          from unnest(sqlc.arg('facet_keys')::text[]) with ordinality as chosen(key, at)
          join unnest(sqlc.arg('facet_lows')::int[]) with ordinality as lows(low, at)
            on lows.at = chosen.at
          join unnest(sqlc.arg('facet_highs')::int[]) with ordinality as highs(high, at)
            on highs.at = chosen.at
         group by chosen.key
        having not bool_or(
                 coalesce((summary.facets ->> chosen.key)::int, 0) >= lows.low
                 and (highs.high < 0
                      or coalesce((summary.facets ->> chosen.key)::int, 0) <= highs.high))
   ))
   and (sqlc.arg('search_text')::text = ''
        or position(sqlc.arg('search_text')::text in lower(a.name)) > 0
        or position(sqlc.arg('search_text')::text in lower(a.blurb)) > 0
        or position(sqlc.arg('search_text')::text in lower(coalesce(owner.username, ''))) > 0)
   and (sqlc.arg('author')::text = '' or lower(coalesce(owner.username, '')) = sqlc.arg('author')::text)
   and (cardinality(sqlc.arg('tags')::text[]) = 0 or not exists (
        select 1 from unnest(sqlc.arg('tags')::text[]) wanted(tag)
         where not exists (
             select 1 from unnest(a.tags) stored(tag)
              where lower(btrim(stored.tag)) = wanted.tag
         )
   ))
   and a.is_nsfw;

-- name: WorkPage :one
select a.id, a.type, a.name, a.blurb, a.tags, a.is_nsfw, a.visibility,
       a.lifecycle, a.created_at,
       revision.format as original_format, revision.media_type as original_media_type,
       revision.created_at as original_arrived_at,
       coalesce(revision.identifier, '')::text as identifier,
       coalesce(owner.username, 'unknown') as creator,
       coalesce(a.owner_id = sqlc.narg('viewer_id')::uuid, false)::boolean as is_owner,
       a.withheld_reason, a.withheld_at
  from works a
  left join users owner on owner.id = a.owner_id
  left join work_revisions revision on revision.id = a.current_revision_id
 where a.id = $1
   and a.deleted_at is null
   and (a.lifecycle = 'published' or a.owner_id = sqlc.narg('viewer_id')::uuid)
   and (a.withheld_at is null or a.owner_id = sqlc.narg('viewer_id')::uuid);

-- name: WorkPageMedia :many
select media.id, media.role, media.width, media.height, blob.byte_size,
       coalesce(media.id = a.cover_media_id, false)::boolean as is_cover
  from works a
  join work_media media on media.work_id = a.id
  join blobs blob on blob.id = media.blob_id
 where a.id = $1
   and media.is_current
   and media.width is not null
   and media.height is not null
   and media.blob_id is not null
 order by (media.id = a.cover_media_id) desc,
		  case media.role
		    when 'avatar' then 1
		    when 'avatar_alt' then 2
		    when 'gallery' then 3
		    when 'expression' then 4
		    when 'pack_item' then 5
		    else 6
		  end,
          media.created_at desc, media.id desc;

-- name: CurrentRevisionLocation :one
select a.id as work_id, r.id as revision_id, r.blob_id, r.media_type, a.owner_id
  from works a
  left join public.work_snapshots snapshot on snapshot.id = a.published_snapshot_id
  join work_revisions r on r.id = case when snapshot.id is null
      then a.current_revision_id else snapshot.source_revision_id end
 where a.id = $1
   and r.blob_id is not null
   and a.lifecycle = 'published'
   and a.deleted_at is null
   and (a.withheld_at is null or a.owner_id = sqlc.narg('viewer_id')::uuid);

-- name: WorkByID :one
select a.id, a.type, revision.format, a.origin_format,
       a.work_version, a.credited_author, a.nickname, a.lifecycle,
       a.name, a.blurb, a.tags,
       coalesce(a.is_nsfw, true)::boolean as is_nsfw, a.visibility,
       a.current_revision_id, a.created_at
  from works a
  join work_revisions revision on revision.id = a.current_revision_id
 where a.id = $1;

-- name: SetWorkVisibility :execrows
update works
   set visibility = $3, updated_at = now()
 where id = $1 and owner_id = $2 and lifecycle = 'published'
   and withheld_at is null and deleted_at is null;

-- name: WorkStateForOwner :one
select withheld_at, lifecycle
  from works
 where id = $1 and owner_id = $2 and deleted_at is null;

-- name: WithholdWork :one
with withheld as (
    update works as work
       set withheld_at = now(), withheld_by = $2, withheld_reason = $3,
           updated_at = now()
     where work.id = $1 and work.lifecycle = 'published'
       and work.withheld_at is null and work.deleted_at is null
    returning work.id, work.owner_id, work.name, work.published_snapshot_id
), stopped as (
    update instance_deliveries as delivery
       set state = 'failed', settled_at = now(), settled_reason = 'withdrawn'
     where delivery.work_id in (select withheld.id from withheld)
       and delivery.state = 'queued'
)
select withheld.owner_id, coalesce(snapshot.payload ->> 'name', withheld.name)::text as public_name
  from withheld
  left join work_snapshots snapshot on snapshot.id = withheld.published_snapshot_id;

-- name: ClearWorkWithhold :one
with cleared as (
    update works as work
       set withheld_at = null, withheld_by = null, withheld_reason = null,
           updated_at = now()
     where work.id = $1 and work.withheld_at is not null and work.deleted_at is null
    returning work.id, work.owner_id, work.name, work.published_snapshot_id
)
select cleared.owner_id, coalesce(snapshot.payload ->> 'name', cleared.name)::text as public_name
  from cleared
  left join work_snapshots snapshot on snapshot.id = cleared.published_snapshot_id;

-- name: WorkDeletionState :one
select withheld_at, deleted_at
  from works
 where id = $1 and owner_id = $2;

-- name: SoftDeleteWork :execrows
update works
   set deleted_at = $3, recoverable_until = $4, updated_at = $3
 where id = $1 and owner_id = $2
   and withheld_at is null and deleted_at is null;

-- name: RestoreWork :execrows
update works
   set deleted_at = null, recoverable_until = null, updated_at = $3
 where id = $1 and owner_id = $2
   and deleted_at is not null and recoverable_until > $3;

-- name: ListDeletedWorks :many
select work.id, work.name, work.type, work.deleted_at, work.recoverable_until
  from works work
  join users owner on owner.id = work.owner_id
 where work.owner_id = $1 and owner.username = $2
   and work.deleted_at is not null and work.recoverable_until > $3
 order by work.deleted_at desc, work.id desc;

-- name: UpsertBlob :one
insert into blobs (id, sha256, byte_size, storage_key)
values ($1, $2, $3, $4)
on conflict (sha256) do update set sha256 = excluded.sha256
returning id, sha256, byte_size, storage_key;

-- name: BlobLocation :one
select storage_key, byte_size from blobs where id = $1;

-- name: LockHandle :one
select 1 from pg_advisory_xact_lock(
    hashtextextended('illarin-handle:' || sqlc.arg('handle')::text, 0)
);

-- name: LockEmail :one
select 1 from pg_advisory_xact_lock(
    hashtextextended('illarin-email:' || sqlc.arg('email')::text, 0)
);

-- name: HandleUnavailable :one
select exists (
    select 1 from users where username = $1
    union all
    select 1 from retired_handles where handle = $1
);

-- name: VerifiedEmailExists :one
select exists (
    select 1 from users where email = $1 and email_verified_at is not null
);

-- name: VerifiedEmailBelongsToDiscordAccount :one
select exists (
    select 1
      from users u
      join oauth_identities oi on oi.user_id = u.id and oi.provider = 'discord'
     where u.email = $1 and u.email_verified_at is not null
);

-- name: InsertUser :one
insert into users (id, username, email, password_hash, email_source)
values ($1, $2, $3, $4, 'creator')
returning id, username, email, email_verified_at;

-- name: InsertEmailVerificationToken :exec
insert into email_verification_tokens (token_hash, user_id, email, expires_at)
values ($1, $2, $3, $4);

-- name: InsertSession :exec
insert into sessions (token_hash, user_id, expires_at)
values ($1, $2, $3);

-- name: UserBySessionHash :one
select u.id, u.username, u.email, u.email_verified_at,
       u.role,
       case when u.password_hash is null then false else true end as has_password,
       exists (
           select 1 from oauth_identities oi
            where oi.user_id = u.id and oi.provider = 'discord'
       ) as discord_linked
  from sessions s
  join users u on u.id = s.user_id
 where s.token_hash = $1 and s.expires_at > now();

-- name: NSFWPreferenceBySessionHash :one
select u.nsfw_preference
  from sessions session
  join users u on u.id = session.user_id
 where session.token_hash = $1 and session.expires_at > now();

-- name: SetNSFWPreferenceBySessionHash :execrows
update users u
   set nsfw_preference = $1, updated_at = now()
  from sessions session
 where session.user_id = u.id and session.token_hash = $2
   and session.expires_at > now();

-- name: VerificationByHash :one
select user_id, email
  from email_verification_tokens
 where token_hash = $1 and expires_at > now()
 for update;

-- name: VerificationEmailByHash :one
select email
  from email_verification_tokens
 where token_hash = $1 and expires_at > now();

-- name: VerifyUserEmail :one
with verified as (
    update users
       set email_verified_at = now(), updated_at = now()
     where users.id = $1 and users.email = $2 and users.email_verified_at is null
    returning users.id, users.username, users.email,
              users.email_verified_at, users.password_hash
)
select v.id, v.username, v.email, v.email_verified_at,
       case when v.password_hash is null then false else true end as has_password,
       exists (
           select 1 from oauth_identities oi
            where oi.user_id = v.id and oi.provider = 'discord'
       ) as discord_linked
  from verified v;

-- name: ClearPendingEmailCopies :exec
update users
   set email = null, email_source = null, updated_at = now()
 where email = $1 and id <> $2 and email_verified_at is null;

-- name: DeleteVerificationTokensForEmail :exec
delete from email_verification_tokens where email = $1;

-- name: DeleteVerificationTokensForUser :exec
delete from email_verification_tokens where user_id = $1;

-- name: UsersForSignIn :many
select u.id, u.username, u.email, u.email_verified_at, u.password_hash,
       u.role,
       exists (
           select 1 from oauth_identities oi
            where oi.user_id = u.id and oi.provider = 'discord'
       ) as discord_linked
  from users u
 where u.email = $1 and u.password_hash is not null;

-- name: DeleteSession :exec
delete from sessions where token_hash = $1;

-- name: UserHandleForUpdate :one
select username from users where id = $1 for update;

-- name: InsertRetiredHandle :exec
insert into retired_handles (handle) values ($1);

-- name: UpdateUserHandle :one
update users set username = $2, updated_at = now() where id = $1
returning id, username, email, email_verified_at;

-- name: ProfileByHandle :one
select id, username, show_nsfw_contributions_on_profile
  from users where username = $1;

-- name: UpdateUnverifiedEmail :one
update users
   set email = $2, email_source = 'creator', updated_at = now()
 where id = $1 and email_verified_at is null
returning id, username, email, email_verified_at;

-- name: InsertOAuthState :exec
insert into oauth_states (token_hash, intent, user_id, expires_at)
values ($1, $2, $3, $4);

-- name: TakeOAuthState :one
delete from oauth_states
 where token_hash = $1 and expires_at > now()
returning intent, user_id;

-- name: LockOAuthIdentity :one
select 1 from pg_advisory_xact_lock(
    hashtextextended('illarin-oauth:' || sqlc.arg('provider')::text || ':' ||
                     sqlc.arg('subject')::text, 0)
);

-- name: LockOAuthUser :one
select 1 from pg_advisory_xact_lock(
    hashtextextended('illarin-oauth-user:' || sqlc.arg('user_id')::uuid::text, 0)
);

-- name: UserByOAuthIdentity :one
select u.id, u.username, u.email, u.email_verified_at, u.email_source,
       u.role,
       case when u.password_hash is null then false else true end as has_password
  from oauth_identities oi
  join users u on u.id = oi.user_id
 where oi.provider = $1 and oi.subject = $2;

-- name: InsertDiscordUser :one
insert into users
  (id, username, email, email_verified_at, email_source, display_name, avatar_url, banner_url)
values ($1, $2, $3, $4, case when $3::text is null then null else 'discord' end, $5, $6, $7)
returning id, username, email, email_verified_at;

-- name: InsertOAuthIdentity :exec
insert into oauth_identities (user_id, provider, subject, provider_email)
values ($1, $2, $3, $4);

-- name: UpdateDiscordEmail :one
update users
   set email = $2, email_verified_at = now(), email_source = 'discord', updated_at = now()
 where id = $1
   and (email is null or email_source = 'discord')
returning id, username, email, email_verified_at, email_source;

-- name: UpdateOAuthIdentityEmail :exec
update oauth_identities
   set provider_email = $3, updated_at = now()
 where provider = $1 and subject = $2;

-- name: UserForDiscordAttach :one
select id, username, email, email_verified_at, email_source,
       role,
       case when password_hash is null then false else true end as has_password
  from users
 where id = $1
 for update;

-- name: SetFirstPassword :one
update users
   set password_hash = $2, updated_at = now()
 where id = $1 and password_hash is null
returning id, username, email, email_verified_at;

-- name: ReplacePassword :exec
update users
   set password_hash = $2, updated_at = now()
 where id = $1;

-- name: DiscordSubjectsForUser :many
select subject
  from oauth_identities
 where user_id = $1 and provider = 'discord'
 order by subject;

-- name: DeleteOAuthIdentitiesForUser :exec
delete from oauth_identities
 where user_id = $1 and provider = 'discord';

-- name: VerifiedUserIDByEmail :one
select id from users where email = $1 and email_verified_at is not null;

-- name: DeletePasswordResetForUser :exec
delete from password_reset_tokens where user_id = $1;

-- name: InsertPasswordReset :exec
insert into password_reset_tokens (token_hash, user_id, expires_at)
values ($1, $2, $3);

-- name: TakePasswordReset :one
delete from password_reset_tokens
 where token_hash = $1 and expires_at > now()
returning user_id;

-- name: DeleteExpiredDeviceLinkRequests :execrows
with expired as (
    select device_code_hash
      from link_requests
     where expires_at <= now()
     order by expires_at
     limit sqlc.arg('batch_size')
     for update skip locked
)
delete from link_requests as request
 using expired
 where request.device_code_hash = expired.device_code_hash;

-- name: DeleteExpiredLinkAuthorizations :execrows
with expired as (
    select request_hash
      from link_authorizations
     where expires_at <= now()
     order by expires_at
     limit sqlc.arg('batch_size')
     for update skip locked
)
delete from link_authorizations as link_auth
 using expired
 where link_auth.request_hash = expired.request_hash;

-- name: DeleteExpiredInstanceAccessTokens :execrows
with expired as (
    select token_hash
      from instance_access_tokens
     where expires_at <= now()
     order by expires_at
     limit sqlc.arg('batch_size')
     for update skip locked
)
delete from instance_access_tokens as token
 using expired
 where token.token_hash = expired.token_hash;

-- name: DeleteExpiredInstanceRefreshHistory :execrows
with expired as (
    select token_hash
      from instance_refresh_history
     where detectable_until <= now()
     order by detectable_until
     limit sqlc.arg('batch_size')
     for update skip locked
)
delete from instance_refresh_history as history
 using expired
 where history.token_hash = expired.token_hash;

-- name: DeleteExpiredLinkRateLimits :execrows
with expired as (
    select key_hash, action
      from link_rate_limits
     where link_rate_limits.window_start <= sqlc.arg('window_cutoff')
     order by link_rate_limits.window_start
     limit sqlc.arg('batch_size')
     for update skip locked
)
delete from link_rate_limits as rate
 using expired
 where rate.key_hash = expired.key_hash and rate.action = expired.action;

-- name: TakeLinkRateLimit :one
insert into link_rate_limits as rate (key_hash, action, attempts, window_start)
values (sqlc.arg('key_hash'), sqlc.arg('action'), 1, now())
on conflict (key_hash, action) do update
   set attempts = case
           when rate.window_start > sqlc.arg('window_cutoff')
               then rate.attempts + 1
           else 1
       end,
       window_start = case
           when rate.window_start > sqlc.arg('window_cutoff')
               then rate.window_start
           else now()
       end
returning rate.attempts, rate.window_start;

-- name: InsertDeviceLinkRequest :exec
insert into link_requests (
    device_code_hash, user_code_hash,
    application_name, instance_name, application_version, protocol_version,
    capabilities, accepted_targets, scopes, expires_at
)
values (
    sqlc.arg('device_code_hash'), sqlc.arg('user_code_hash'),
    sqlc.arg('application_name'), sqlc.arg('instance_name'),
    sqlc.narg('application_version'), sqlc.arg('protocol_version'),
    sqlc.arg('capabilities'), sqlc.arg('accepted_targets'),
    sqlc.arg('scopes'), sqlc.arg('expires_at')
);

-- name: ReviewDeviceLinkRequest :one
select application_name, instance_name, application_version, protocol_version,
       capabilities, accepted_targets, scopes, expires_at
  from link_requests
 where user_code_hash = sqlc.arg('user_code_hash')
   and expires_at > now()
   and (reviewed_by is null or reviewed_by = sqlc.arg('reviewed_by'))
   and approved_at is null
   and denied_at is null
   and redeemed_at is null;

-- name: ApproveDeviceLinkRequest :one
update link_requests
   set review_token_hash = sqlc.arg('review_token_hash'),
       reviewed_by = sqlc.arg('reviewed_by'),
       approved_by = sqlc.arg('reviewed_by'),
       approved_at = now()
 where user_code_hash = sqlc.arg('user_code_hash')
   and (reviewed_by is null or reviewed_by = sqlc.arg('reviewed_by'))
   and expires_at > now()
   and approved_at is null
   and denied_at is null
   and redeemed_at is null
returning application_name, instance_name, application_version, protocol_version,
          capabilities, accepted_targets, scopes, expires_at;

-- name: DenyDeviceLinkRequest :execrows
update link_requests
   set review_token_hash = sqlc.arg('review_token_hash'),
       reviewed_by = sqlc.arg('reviewed_by'),
       denied_by = sqlc.arg('reviewed_by'),
       denied_at = now()
 where user_code_hash = sqlc.arg('user_code_hash')
   and (reviewed_by is null or reviewed_by = sqlc.arg('reviewed_by'))
   and expires_at > now()
   and approved_at is null
   and denied_at is null
   and redeemed_at is null;

-- name: LockDeviceLinkRequest :one
select approved_by, denied_at, redeemed_at, last_polled_at, poll_interval_seconds,
       application_name, instance_name, application_version, protocol_version,
       capabilities, accepted_targets, scopes, expires_at
  from link_requests
 where device_code_hash = sqlc.arg('device_code_hash')
 for update;

-- name: RecordDeviceLinkPoll :one
update link_requests
   set last_polled_at = now(),
       poll_interval_seconds = case
           when sqlc.arg('slow_down')::boolean
               then least(poll_interval_seconds + 5, 60)
           else poll_interval_seconds
       end
 where device_code_hash = sqlc.arg('device_code_hash')
   and expires_at > now()
   and redeemed_at is null
returning poll_interval_seconds;

-- name: RedeemDeviceLinkRequest :execrows
update link_requests
   set redeemed_at = now()
 where device_code_hash = sqlc.arg('device_code_hash')
   and expires_at > now()
   and approved_at is not null
   and denied_at is null
   and redeemed_at is null;

-- name: InsertLinkAuthorization :exec
insert into link_authorizations (
    request_hash, redirect_uri, state, code_challenge,
    application_name, instance_name, application_version, protocol_version,
    capabilities, accepted_targets, scopes, expires_at
)
values (
    sqlc.arg('request_hash'), sqlc.arg('redirect_uri'), sqlc.arg('state'),
    sqlc.arg('code_challenge'), sqlc.arg('application_name'),
    sqlc.arg('instance_name'), sqlc.narg('application_version'),
    sqlc.arg('protocol_version'), sqlc.arg('capabilities'),
    sqlc.arg('accepted_targets'), sqlc.arg('scopes'), sqlc.arg('expires_at')
);

-- name: ReviewLinkAuthorization :one
select redirect_uri, state, application_name, instance_name,
       application_version, protocol_version, capabilities,
       accepted_targets, scopes, expires_at
  from link_authorizations
 where request_hash = sqlc.arg('request_hash')
   and expires_at > now()
   and (reviewed_by is null or reviewed_by = sqlc.arg('reviewed_by'))
   and approved_at is null
   and denied_at is null
   and redeemed_at is null;

-- name: ApproveLinkAuthorization :one
update link_authorizations
   set reviewed_by = sqlc.arg('reviewed_by'),
       authorization_code_hash = sqlc.arg('authorization_code_hash'),
       approved_by = sqlc.arg('reviewed_by'),
       approved_at = now()
 where request_hash = sqlc.arg('request_hash')
   and (reviewed_by is null or reviewed_by = sqlc.arg('reviewed_by'))
   and expires_at > now()
   and approved_at is null
   and denied_at is null
   and redeemed_at is null
returning redirect_uri, state, expires_at;

-- name: DenyLinkAuthorization :one
update link_authorizations
   set reviewed_by = sqlc.arg('reviewed_by'),
       denied_by = sqlc.arg('reviewed_by'),
       denied_at = now()
 where request_hash = sqlc.arg('request_hash')
   and (reviewed_by is null or reviewed_by = sqlc.arg('reviewed_by'))
   and expires_at > now()
   and approved_at is null
   and denied_at is null
   and redeemed_at is null
returning redirect_uri, state;

-- name: LockLinkAuthorization :one
select approved_by, denied_at, redeemed_at, redirect_uri, state, code_challenge,
       application_name, instance_name, application_version, protocol_version,
       capabilities, accepted_targets, scopes, expires_at
 from link_authorizations
 where authorization_code_hash = sqlc.arg('authorization_code_hash')
 for update;

-- name: RedeemLinkAuthorization :execrows
update link_authorizations
   set redeemed_at = now()
 where authorization_code_hash = sqlc.arg('authorization_code_hash')
   and expires_at > now()
   and approved_at is not null
   and denied_at is null
   and redeemed_at is null;

-- name: InsertLinkedInstance :one
insert into linked_instances (
    id, user_id, application_name, instance_name, application_version,
    protocol_version, capabilities, accepted_targets,
    refresh_token_hash, refresh_token_prefix, scopes
)
values (
    sqlc.arg('id'), sqlc.arg('user_id'), sqlc.arg('application_name'),
    sqlc.arg('instance_name'), sqlc.narg('application_version'),
    sqlc.arg('protocol_version'), sqlc.arg('capabilities'),
    sqlc.arg('accepted_targets'), sqlc.arg('refresh_token_hash'),
    sqlc.arg('refresh_token_prefix'), sqlc.arg('scopes')
)
returning id, application_name, instance_name, application_version,
          protocol_version, capabilities, accepted_targets,
          refresh_token_prefix, scopes, linked_at, last_seen_at, revoked_at;

-- name: InsertInstanceAccessToken :one
with live_instance as (
    select id
      from linked_instances
     where id = sqlc.arg('instance_id') and revoked_at is null
     for update
)
insert into instance_access_tokens (token_hash, instance_id, expires_at)
select sqlc.arg('token_hash'), sqlc.arg('instance_id'), sqlc.arg('expires_at')
  from live_instance
returning token_hash, instance_id, expires_at;

-- name: ListLinkedInstances :many
select id, application_name, instance_name, application_version,
       protocol_version, capabilities, accepted_targets,
       refresh_token_prefix, scopes, linked_at, last_seen_at, revoked_at
  from linked_instances
 where user_id = sqlc.arg('user_id')
 order by (revoked_at is not null), coalesce(last_seen_at, linked_at) desc, linked_at desc;

-- name: TouchLinkedInstanceByAccessToken :one
update linked_instances as instance
   set last_seen_at = now()
  from instance_access_tokens as token
 where token.token_hash = sqlc.arg('token_hash')
   and token.expires_at > now()
   and instance.id = token.instance_id
   and instance.revoked_at is null
returning instance.id, instance.user_id,
          instance.application_name, instance.instance_name,
          instance.application_version, instance.protocol_version,
          instance.capabilities, instance.accepted_targets,
          instance.refresh_token_prefix, instance.scopes,
          instance.linked_at, instance.last_seen_at;

-- name: LockLinkedInstanceByRefreshToken :one
select id, user_id, application_name, instance_name, application_version,
       protocol_version, capabilities, accepted_targets,
       refresh_token_prefix, scopes, linked_at, last_seen_at
  from linked_instances
 where refresh_token_hash = sqlc.arg('refresh_token_hash') and revoked_at is null
 for update;

-- name: InstanceForUsedRefreshToken :one
select history.instance_id, instance.user_id
  from instance_refresh_history as history
  join linked_instances as instance on instance.id = history.instance_id
 where history.token_hash = sqlc.arg('refresh_token_hash')
   and history.detectable_until > now();

-- name: RotateInstanceRefreshToken :one
with rotated as (
    update linked_instances
       set refresh_token_hash = sqlc.arg('new_refresh_token_hash'),
           refresh_token_prefix = sqlc.arg('new_refresh_token_prefix'),
           last_seen_at = now()
     where id = sqlc.arg('instance_id')
       and refresh_token_hash = sqlc.arg('old_refresh_token_hash')
       and revoked_at is null
    returning id
)
insert into instance_refresh_history (token_hash, instance_id, detectable_until)
select sqlc.arg('old_refresh_token_hash'), id, sqlc.arg('detectable_until')
  from rotated
returning instance_id;

-- name: UpdateLinkedInstanceDeclaration :one
update linked_instances
   set application_version = sqlc.narg('application_version'),
       protocol_version = sqlc.arg('protocol_version'),
       capabilities = sqlc.arg('capabilities'),
       accepted_targets = sqlc.arg('accepted_targets')
 where id = sqlc.arg('instance_id') and revoked_at is null
returning id, application_name, instance_name, application_version,
          protocol_version, capabilities, accepted_targets,
          refresh_token_prefix, scopes, linked_at, last_seen_at;

-- name: RevokeLinkedInstance :one
with revoked as (
    update linked_instances as instance
       set refresh_token_hash = null,
           application_version = null,
           library_application_version = null,
           protocol_version = null,
           capabilities = '{}',
           accepted_targets = '{}',
           revoked_at = now()
     where instance.id = sqlc.arg('instance_id')
       and instance.user_id = sqlc.arg('user_id')
       and instance.revoked_at is null
    returning instance.id
), deleted_access as (
    delete from instance_access_tokens as token
     where token.instance_id in (select revoked.id from revoked)
), deleted_deliveries as (
    delete from instance_deliveries as delivery
     where delivery.instance_id in (select revoked.id from revoked)
), deleted_library as (
    delete from instance_library_entries as entry
     where entry.instance_id in (select revoked.id from revoked)
)
select exists(select 1 from revoked) as revoked;

-- name: RevokeLinkedInstanceByID :one
with revoked as (
    update linked_instances as instance
       set refresh_token_hash = null,
           application_version = null,
           library_application_version = null,
           protocol_version = null,
           capabilities = '{}',
           accepted_targets = '{}',
           revoked_at = now()
     where instance.id = sqlc.arg('instance_id') and instance.revoked_at is null
    returning instance.id
), deleted_access as (
    delete from instance_access_tokens as token
     where token.instance_id in (select revoked.id from revoked)
), deleted_deliveries as (
    delete from instance_deliveries as delivery
     where delivery.instance_id in (select revoked.id from revoked)
), deleted_library as (
    delete from instance_library_entries as entry
     where entry.instance_id in (select revoked.id from revoked)
)
select exists(select 1 from revoked) as revoked;

-- name: UpdateDiscordProfile :exec
update users
   set display_name = $2, avatar_url = $3, banner_url = $4, updated_at = now()
 where id = $1;

-- name: LiveLinkedInstances :many
select id, user_id, application_name, instance_name, application_version,
       protocol_version, capabilities, accepted_targets,
       refresh_token_prefix, scopes, linked_at, last_seen_at
  from linked_instances
 where user_id = sqlc.arg('user_id') and revoked_at is null
 order by coalesce(last_seen_at, linked_at) desc, linked_at desc;

-- name: LiveLinkedInstance :one
select id, user_id, application_name, instance_name, application_version,
       protocol_version, capabilities, accepted_targets,
       refresh_token_prefix, scopes, linked_at, last_seen_at
  from linked_instances
 where id = sqlc.arg('instance_id')
   and user_id = sqlc.arg('user_id')
   and revoked_at is null;

-- name: QueueDelivery :one
insert into instance_deliveries (id, instance_id, work_id, expires_at, updates_install)
select sqlc.arg('id'), sqlc.arg('instance_id'), sqlc.arg('work_id'),
       sqlc.arg('expires_at'),
       exists (
           select 1
             from instance_library_entries as entry
            where entry.instance_id = sqlc.arg('instance_id')
              and entry.work_id = sqlc.arg('work_id')
       )
on conflict (instance_id, work_id) where state in ('queued', 'released') do nothing
returning id, instance_id, work_id, state, settled_reason, queued_at, settled_at,
          expires_at, updates_install;

-- name: LiveDeliveryForWork :one
select id, instance_id, work_id, state, settled_reason, queued_at, settled_at,
       expires_at, updates_install
  from instance_deliveries
 where instance_id = sqlc.arg('instance_id')
   and work_id = sqlc.arg('work_id')
   and state in ('queued', 'released');

-- name: CountLiveDeliveries :one
select count(*)::bigint
  from instance_deliveries
 where instance_id = sqlc.arg('instance_id') and state in ('queued', 'released');

-- name: AbandonExhaustedDeliveries :execrows
update instance_deliveries
   set state = 'failed', settled_at = now(), settled_reason = 'abandoned',
       lease_expires_at = null
 where instance_id = sqlc.arg('instance_id')
   and state = 'released'
   and lease_expires_at <= now()
   and attempts >= sqlc.arg('max_attempts');

-- name: ClaimDeliveries :many
with candidates as (
    select waiting.id
      from instance_deliveries as waiting
      join linked_instances as instance
        on instance.id = waiting.instance_id
       and instance.revoked_at is null
       and instance.scopes @> array['asset:receive']
     where waiting.instance_id = sqlc.arg('instance_id')
       and waiting.expires_at > now()
       and waiting.attempts < sqlc.arg('max_attempts')
       and (waiting.state = 'queued'
            or (waiting.state = 'released' and waiting.lease_expires_at <= now()))
     order by waiting.queued_at, waiting.id
     limit sqlc.arg('batch_size')
     for update of waiting skip locked
)
update instance_deliveries as delivery
   set state = 'released',
       attempts = delivery.attempts + 1,
       lease_expires_at = sqlc.arg('lease_expires_at')
  from candidates
 where delivery.id = candidates.id
returning delivery.id, delivery.work_id, delivery.queued_at,
          delivery.lease_expires_at;

-- name: SetDeliveryTarget :exec
update instance_deliveries
   set chosen_target = sqlc.arg('chosen_target')
 where id = sqlc.arg('id');

-- name: FailDelivery :exec
update instance_deliveries
   set state = 'failed', settled_at = now(),
       settled_reason = sqlc.arg('settled_reason'), lease_expires_at = null
 where id = sqlc.arg('id');

-- name: AcknowledgeDeliveries :execrows
update instance_deliveries
   set state = 'delivered', settled_at = now(), lease_expires_at = null
 where instance_id = sqlc.arg('instance_id')
   and id = any(sqlc.arg('delivery_ids')::uuid[])
   and state = 'released';

-- name: DiscardDelivery :execrows
delete from instance_deliveries as delivery
 using linked_instances as instance
 where delivery.id = sqlc.arg('delivery_id')
   and delivery.instance_id = instance.id
   and instance.user_id = sqlc.arg('user_id');

-- name: DeliveryForArtifact :one
select delivery.work_id, delivery.chosen_target, instance.id as instance_id
  from instance_deliveries as delivery
  join linked_instances as instance on instance.id = delivery.instance_id
 where delivery.id = sqlc.arg('delivery_id')
   and delivery.state = 'released'
   and delivery.lease_expires_at > now()
   and delivery.expires_at > now()
   and delivery.chosen_target is not null
   and instance.revoked_at is null
   and instance.scopes @> array['asset:receive'];

-- name: DeleteExpiredDeliveries :execrows
with expired as (
    select id
      from instance_deliveries
     where expires_at <= now()
     order by expires_at
     limit sqlc.arg('batch_size')
     for update skip locked
)
delete from instance_deliveries as delivery
 using expired
 where delivery.id = expired.id;

-- name: SendableWorkGeneration :one
select content_generation
  from works
 where id = sqlc.arg('work_id')
   and deleted_at is null
   and withheld_at is null
   and lifecycle = 'published';

-- name: WorkInstanceStates :many
select instance.id, instance.application_name, instance.instance_name,
       instance.last_seen_at, instance.scopes, instance.capabilities,
       instance.accepted_targets,
       delivery.id as delivery_id,
       coalesce(delivery.state, '')::text as delivery_state,
       delivery.settled_reason, delivery.queued_at, delivery.settled_at,
       delivery.expires_at,
       coalesce(delivery.updates_install, false)::boolean as updates_install,
       entry.content_generation as installed_generation
  from linked_instances as instance
  left join lateral (
      select waiting.id, waiting.state, waiting.settled_reason,
             waiting.queued_at, waiting.settled_at, waiting.expires_at,
             waiting.updates_install
        from instance_deliveries as waiting
       where waiting.instance_id = instance.id
         and waiting.work_id = sqlc.arg('work_id')
       order by (waiting.state in ('queued', 'released')) desc,
                waiting.queued_at desc
       limit 1
  ) as delivery on true
  left join instance_library_entries as entry
    on entry.instance_id = instance.id and entry.work_id = sqlc.arg('work_id')
 where instance.user_id = sqlc.arg('user_id') and instance.revoked_at is null
 order by coalesce(instance.last_seen_at, instance.linked_at) desc,
          instance.linked_at desc;

-- name: InstanceLibraryCounts :many
select entry.instance_id,
       count(*)::bigint as installed,
       count(*) filter (
           where work.content_generation > entry.content_generation
       )::bigint as updates_available
  from instance_library_entries as entry
  join works as work on work.id = entry.work_id
  join linked_instances as instance on instance.id = entry.instance_id
 where instance.user_id = sqlc.arg('user_id')
   and instance.revoked_at is null
   and work.deleted_at is null
   and work.withheld_at is null
   and work.lifecycle = 'published'
 group by entry.instance_id;

-- name: ReportLibraryEntries :execrows
insert into instance_library_entries
    (instance_id, work_id, content_generation, reported_at)
select sqlc.arg('instance_id'), work.id,
       coalesce(nullif(reported.generation, 0), work.content_generation), now()
  from (
      select unnest(sqlc.arg('work_ids')::uuid[]) as work_id,
             unnest(sqlc.arg('generations')::integer[]) as generation
  ) as reported
  join works as work
    on work.id = reported.work_id
   and work.deleted_at is null
   and work.lifecycle = 'published'
on conflict (instance_id, work_id) do update
   set content_generation = excluded.content_generation,
       reported_at = excluded.reported_at;

-- name: RemoveLibraryEntries :execrows
delete from instance_library_entries
 where instance_id = sqlc.arg('instance_id')
   and work_id = any(sqlc.arg('work_ids')::uuid[]);

-- name: PruneLibraryToSnapshot :execrows
delete from instance_library_entries
 where instance_id = sqlc.arg('instance_id')
   and not (work_id = any(sqlc.arg('work_ids')::uuid[]));

-- name: TakeWithheldNotices :many
update instance_library_entries as entry
   set notified_withheld_at = work.withheld_at
  from work_public.works as work
 where entry.instance_id = sqlc.arg('instance_id')
   and work.id = entry.work_id
   and work.type = any(sqlc.arg('types')::text[])
   and work.withheld_at is not null
   and work.deleted_at is null
   and entry.notified_withheld_at is distinct from work.withheld_at
returning entry.work_id, work.name::text as name, work.withheld_at;

-- name: RecordLibraryApplicationVersion :exec
update linked_instances
   set library_application_version = nullif(sqlc.arg('application_version')::text, '')
 where id = sqlc.arg('instance_id');

-- name: InstalledApplicationVersions :many
select coalesce(instance.library_application_version,
                instance.application_version)::text as application_version
  from instance_library_entries as entry
  join linked_instances as instance
    on instance.id = entry.instance_id
   and instance.revoked_at is null
   and instance.capabilities && sqlc.arg('capabilities')::text[]
 where entry.work_id = sqlc.arg('work_id')
   and coalesce(instance.library_application_version, instance.application_version) is not null
 group by coalesce(instance.library_application_version, instance.application_version)
having count(*) >= sqlc.arg('minimum_group_size')::bigint
 order by coalesce(instance.library_application_version, instance.application_version);
