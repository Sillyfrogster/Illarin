-- name: InsertWork :one
insert into works
  (id, type, owner_id, name, blurb, tags, is_nsfw, visibility, lifecycle,
   work_version, credited_author, nickname, original_format, created_at)
values ($1, $2, $3, $4, $5, $6, sqlc.narg('is_nsfw')::boolean, $7, $8,
        $9, $10, $11, sqlc.narg('original_format')::text,
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

-- name: InsertOriginalFile :exec
insert into work_original_files
  (id, work_id, number, blob_id, media_type, format, identifier)
values ($1, $2, $3, $4, $5, $6, $7);

-- name: SetOriginalFile :exec
update works set original_file_id = $2, updated_at = now() where id = $1;

-- name: ListWorks :many
select a.id, a.type, original.format, a.original_format,
       a.work_version, a.credited_author, a.nickname, a.lifecycle,
       a.name, a.blurb, a.tags,
       coalesce(a.is_nsfw, true)::boolean as is_nsfw, a.visibility,
       a.original_file_id, a.created_at
  from works a
  left join work_original_files original on original.id = a.original_file_id
 where a.lifecycle = 'published'
   and a.visibility = 'listed'
   and a.taken_down_at is null
   and a.deleted_at is null
   and ($1 = '' or a.type = $1)
   and (not $2::boolean or original.format is not distinct from $3)
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
       a.visibility, a.taken_down_at, a.taken_down_reason
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
        and a.visibility = 'listed' and a.taken_down_at is null)
       or
       (a.owner_id = sqlc.narg('creator_id')::uuid
        and (sqlc.arg('own_profile')::boolean
             or (a.visibility = 'listed' and a.taken_down_at is null)))
   )
   and (sqlc.arg('type')::text = '' or a.type = sqlc.arg('type')::text)
   and (sqlc.arg('own_profile')::boolean
        or sqlc.arg('nsfw_preference')::text <> 'hidden' or not a.is_nsfw)
   and (sqlc.arg('app')::text = '' or exists (
        select 1
          from jsonb_array_elements(coalesce(summary.export, '[]'::jsonb)) as offered(format)
         where offered.format ->> 'format' = any(sqlc.arg('formats')::text[])
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
        and a.visibility = 'listed' and a.taken_down_at is null)
       or
       (a.owner_id = sqlc.narg('creator_id')::uuid
        and (sqlc.arg('own_profile')::boolean
             or (a.visibility = 'listed' and a.taken_down_at is null)))
   )
   and (sqlc.arg('type')::text = '' or a.type = sqlc.arg('type')::text)
   and (sqlc.arg('own_profile')::boolean
        or sqlc.arg('nsfw_preference')::text <> 'hidden' or not a.is_nsfw)
   and (sqlc.arg('app')::text = '' or exists (
        select 1
          from jsonb_array_elements(coalesce(summary.export, '[]'::jsonb)) as offered(format)
         where offered.format ->> 'format' = any(sqlc.arg('formats')::text[])
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
   and a.taken_down_at is null
   and a.deleted_at is null
   and (sqlc.narg('creator_id')::uuid is null or a.owner_id = sqlc.narg('creator_id')::uuid)
   and (sqlc.arg('type')::text = '' or a.type = sqlc.arg('type')::text)
   and (sqlc.arg('app')::text = '' or exists (
        select 1
          from jsonb_array_elements(coalesce(summary.export, '[]'::jsonb)) as offered(format)
         where offered.format ->> 'format' = any(sqlc.arg('formats')::text[])
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
       original.format as original_format, original.media_type as original_media_type,
       original.created_at as original_arrived_at,
       coalesce(original.identifier, '')::text as identifier,
       coalesce(owner.username, 'unknown') as creator,
       coalesce(a.owner_id = sqlc.narg('viewer_id')::uuid, false)::boolean as is_owner,
       a.taken_down_reason, a.taken_down_at
  from works a
  left join users owner on owner.id = a.owner_id
  left join work_original_files original on original.id = a.original_file_id
 where a.id = $1
   and a.deleted_at is null
   and (a.lifecycle = 'published' or a.owner_id = sqlc.narg('viewer_id')::uuid)
   and (a.taken_down_at is null or a.owner_id = sqlc.narg('viewer_id')::uuid);

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

-- name: OriginalFileLocation :one
select a.id as work_id, r.id as original_file_id, r.blob_id, r.media_type, a.owner_id
  from works a
  left join public.work_versions version on version.id = a.published_version_id
  join work_original_files r on r.id = case when version.id is null
      then a.original_file_id else version.original_file_id end
 where a.id = $1
   and r.blob_id is not null
   and a.lifecycle = 'published'
   and a.deleted_at is null
   and (a.taken_down_at is null or a.owner_id = sqlc.narg('viewer_id')::uuid);

-- name: WorkByID :one
select a.id, a.type, original.format, a.original_format,
       a.work_version, a.credited_author, a.nickname, a.lifecycle,
       a.name, a.blurb, a.tags,
       coalesce(a.is_nsfw, true)::boolean as is_nsfw, a.visibility,
       a.original_file_id, a.created_at
  from works a
  join work_original_files original on original.id = a.original_file_id
 where a.id = $1;

-- name: SetWorkVisibility :execrows
update works
   set visibility = $3, updated_at = now()
 where id = $1 and owner_id = $2 and lifecycle = 'published'
   and taken_down_at is null and deleted_at is null;

-- name: WorkStateForOwner :one
select taken_down_at, lifecycle
  from works
 where id = $1 and owner_id = $2 and deleted_at is null;

-- name: TakeDownWork :one
with taken_down as (
    update works as work
       set taken_down_at = now(), taken_down_by = $2, taken_down_reason = $3,
           updated_at = now()
     where work.id = $1 and work.lifecycle = 'published'
       and work.taken_down_at is null and work.deleted_at is null
    returning work.id, work.owner_id, work.name, work.published_version_id
), stopped as (
    update sends as send
       set state = 'failed', settled_at = now(), settled_reason = 'withdrawn'
     where send.work_id in (select taken_down.id from taken_down)
       and send.state = 'queued'
)
select taken_down.owner_id, coalesce(version.payload ->> 'name', taken_down.name)::text as public_name
  from taken_down
  left join work_versions version on version.id = taken_down.published_version_id;

-- name: LiftTakedown :one
with cleared as (
    update works as work
       set taken_down_at = null, taken_down_by = null, taken_down_reason = null,
           updated_at = now()
     where work.id = $1 and work.taken_down_at is not null and work.deleted_at is null
    returning work.id, work.owner_id, work.name, work.published_version_id
)
select cleared.owner_id, coalesce(version.payload ->> 'name', cleared.name)::text as public_name
  from cleared
  left join work_versions version on version.id = cleared.published_version_id;

-- name: WorkDeletionState :one
select taken_down_at, deleted_at
  from works
 where id = $1 and owner_id = $2;

-- name: SoftDeleteWork :execrows
update works
   set deleted_at = $3, recoverable_until = $4, updated_at = $3
 where id = $1 and owner_id = $2
   and taken_down_at is null and deleted_at is null;

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

-- name: DeleteExpiredConnectionRequests :execrows
with expired as (
    select device_code_hash
      from connection_requests
     where expires_at <= now()
     order by expires_at
     limit sqlc.arg('batch_size')
     for update skip locked
)
delete from connection_requests as request
 using expired
 where request.device_code_hash = expired.device_code_hash;

-- name: DeleteExpiredConnectionAuthorizations :execrows
with expired as (
    select request_hash
      from connection_authorizations
     where expires_at <= now()
     order by expires_at
     limit sqlc.arg('batch_size')
     for update skip locked
)
delete from connection_authorizations as request
 using expired
 where request.request_hash = expired.request_hash;

-- name: DeleteExpiredAppAccessTokens :execrows
with expired as (
    select token_hash
      from app_access_tokens
     where expires_at <= now()
     order by expires_at
     limit sqlc.arg('batch_size')
     for update skip locked
)
delete from app_access_tokens as token
 using expired
 where token.token_hash = expired.token_hash;

-- name: DeleteExpiredAppRefreshHistory :execrows
with expired as (
    select token_hash
      from app_refresh_history
     where detectable_until <= now()
     order by detectable_until
     limit sqlc.arg('batch_size')
     for update skip locked
)
delete from app_refresh_history as history
 using expired
 where history.token_hash = expired.token_hash;

-- name: DeleteExpiredConnectionRateLimits :execrows
with expired as (
    select key_hash, action
      from connection_rate_limits
     where connection_rate_limits.window_start <= sqlc.arg('window_cutoff')
     order by connection_rate_limits.window_start
     limit sqlc.arg('batch_size')
     for update skip locked
)
delete from connection_rate_limits as rate
 using expired
 where rate.key_hash = expired.key_hash and rate.action = expired.action;

-- name: TakeConnectionRateLimit :one
insert into connection_rate_limits as rate (key_hash, action, attempts, window_start)
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

-- name: InsertConnectionRequest :exec
insert into connection_requests (
    device_code_hash, user_code_hash,
    app_name, name, app_version, protocol_version,
    capabilities, accepted_formats, permissions, expires_at
)
values (
    sqlc.arg('device_code_hash'), sqlc.arg('user_code_hash'),
    sqlc.arg('app_name'), sqlc.arg('name'),
    sqlc.narg('app_version'), sqlc.arg('protocol_version'),
    sqlc.arg('capabilities'), sqlc.arg('accepted_formats'),
    sqlc.arg('permissions'), sqlc.arg('expires_at')
);

-- name: ReviewConnectionRequest :one
select app_name, name, app_version, protocol_version,
       capabilities, accepted_formats, permissions, expires_at
  from connection_requests
 where user_code_hash = sqlc.arg('user_code_hash')
   and expires_at > now()
   and (reviewed_by is null or reviewed_by = sqlc.arg('reviewed_by'))
   and approved_at is null
   and denied_at is null
   and redeemed_at is null;

-- name: ApproveConnectionRequest :one
update connection_requests
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
returning app_name, name, app_version, protocol_version,
          capabilities, accepted_formats, permissions, expires_at;

-- name: DenyConnectionRequest :execrows
update connection_requests
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

-- name: LockConnectionRequest :one
select approved_by, denied_at, redeemed_at, last_polled_at, poll_interval_seconds,
       app_name, name, app_version, protocol_version,
       capabilities, accepted_formats, permissions, expires_at
  from connection_requests
 where device_code_hash = sqlc.arg('device_code_hash')
 for update;

-- name: RecordConnectionPoll :one
update connection_requests
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

-- name: RedeemConnectionRequest :execrows
update connection_requests
   set redeemed_at = now()
 where device_code_hash = sqlc.arg('device_code_hash')
   and expires_at > now()
   and approved_at is not null
   and denied_at is null
   and redeemed_at is null;

-- name: InsertConnectionAuthorization :exec
insert into connection_authorizations (
    request_hash, redirect_uri, state, code_challenge,
    app_name, name, app_version, protocol_version,
    capabilities, accepted_formats, permissions, expires_at
)
values (
    sqlc.arg('request_hash'), sqlc.arg('redirect_uri'), sqlc.arg('state'),
    sqlc.arg('code_challenge'), sqlc.arg('app_name'),
    sqlc.arg('name'), sqlc.narg('app_version'),
    sqlc.arg('protocol_version'), sqlc.arg('capabilities'),
    sqlc.arg('accepted_formats'), sqlc.arg('permissions'), sqlc.arg('expires_at')
);

-- name: ReviewConnectionAuthorization :one
select redirect_uri, state, app_name, name,
       app_version, protocol_version, capabilities,
       accepted_formats, permissions, expires_at
  from connection_authorizations
 where request_hash = sqlc.arg('request_hash')
   and expires_at > now()
   and (reviewed_by is null or reviewed_by = sqlc.arg('reviewed_by'))
   and approved_at is null
   and denied_at is null
   and redeemed_at is null;

-- name: ApproveConnectionAuthorization :one
update connection_authorizations
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

-- name: DenyConnectionAuthorization :one
update connection_authorizations
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

-- name: LockConnectionAuthorization :one
select approved_by, denied_at, redeemed_at, redirect_uri, state, code_challenge,
       app_name, name, app_version, protocol_version,
       capabilities, accepted_formats, permissions, expires_at
 from connection_authorizations
 where authorization_code_hash = sqlc.arg('authorization_code_hash')
 for update;

-- name: RedeemConnectionAuthorization :execrows
update connection_authorizations
   set redeemed_at = now()
 where authorization_code_hash = sqlc.arg('authorization_code_hash')
   and expires_at > now()
   and approved_at is not null
   and denied_at is null
   and redeemed_at is null;

-- name: InsertConnectedApp :one
insert into connected_apps (
    id, user_id, app_name, name, app_version,
    protocol_version, capabilities, accepted_formats,
    refresh_token_hash, refresh_token_prefix, permissions
)
values (
    sqlc.arg('id'), sqlc.arg('user_id'), sqlc.arg('app_name'),
    sqlc.arg('name'), sqlc.narg('app_version'),
    sqlc.arg('protocol_version'), sqlc.arg('capabilities'),
    sqlc.arg('accepted_formats'), sqlc.arg('refresh_token_hash'),
    sqlc.arg('refresh_token_prefix'), sqlc.arg('permissions')
)
returning id, app_name, name, app_version,
          protocol_version, capabilities, accepted_formats,
          refresh_token_prefix, permissions, connected_at, last_seen_at, revoked_at;

-- name: InsertAppAccessToken :one
with live_app as (
    select id
      from connected_apps
     where id = sqlc.arg('connected_app_id') and revoked_at is null
     for update
)
insert into app_access_tokens (token_hash, connected_app_id, expires_at)
select sqlc.arg('token_hash'), sqlc.arg('connected_app_id'), sqlc.arg('expires_at')
  from live_app
returning token_hash, connected_app_id, expires_at;

-- name: ListConnectedApps :many
select id, app_name, name, app_version,
       protocol_version, capabilities, accepted_formats,
       refresh_token_prefix, permissions, connected_at, last_seen_at, revoked_at
  from connected_apps
 where user_id = sqlc.arg('user_id')
 order by (revoked_at is not null), coalesce(last_seen_at, connected_at) desc, connected_at desc;

-- name: TouchConnectedAppByAccessToken :one
update connected_apps as app
   set last_seen_at = now()
  from app_access_tokens as token
 where token.token_hash = sqlc.arg('token_hash')
   and token.expires_at > now()
   and app.id = token.connected_app_id
   and app.revoked_at is null
returning app.id, app.user_id,
          app.app_name, app.name,
          app.app_version, app.protocol_version,
          app.capabilities, app.accepted_formats,
          app.refresh_token_prefix, app.permissions,
          app.connected_at, app.last_seen_at;

-- name: LockConnectedAppByRefreshToken :one
select id, user_id, app_name, name, app_version,
       protocol_version, capabilities, accepted_formats,
       refresh_token_prefix, permissions, connected_at, last_seen_at
  from connected_apps
 where refresh_token_hash = sqlc.arg('refresh_token_hash') and revoked_at is null
 for update;

-- name: ConnectedAppForUsedRefreshToken :one
select history.connected_app_id, app.user_id
  from app_refresh_history as history
  join connected_apps as app on app.id = history.connected_app_id
 where history.token_hash = sqlc.arg('refresh_token_hash')
   and history.detectable_until > now();

-- name: RotateAppRefreshToken :one
with rotated as (
    update connected_apps
       set refresh_token_hash = sqlc.arg('new_refresh_token_hash'),
           refresh_token_prefix = sqlc.arg('new_refresh_token_prefix'),
           last_seen_at = now()
     where id = sqlc.arg('connected_app_id')
       and refresh_token_hash = sqlc.arg('old_refresh_token_hash')
       and revoked_at is null
    returning id
)
insert into app_refresh_history (token_hash, connected_app_id, detectable_until)
select sqlc.arg('old_refresh_token_hash'), id, sqlc.arg('detectable_until')
  from rotated
returning connected_app_id;

-- name: UpdateConnectedAppCapabilities :one
update connected_apps
   set app_version = sqlc.narg('app_version'),
       protocol_version = sqlc.arg('protocol_version'),
       capabilities = sqlc.arg('capabilities'),
       accepted_formats = sqlc.arg('accepted_formats')
 where id = sqlc.arg('connected_app_id') and revoked_at is null
returning id, app_name, name, app_version,
          protocol_version, capabilities, accepted_formats,
          refresh_token_prefix, permissions, connected_at, last_seen_at;

-- name: RevokeConnectedApp :one
with revoked as (
    update connected_apps as app
       set refresh_token_hash = null,
           app_version = null,
           library_app_version = null,
           protocol_version = null,
           capabilities = '{}',
           accepted_formats = '{}',
           revoked_at = now()
     where app.id = sqlc.arg('connected_app_id')
       and app.user_id = sqlc.arg('user_id')
       and app.revoked_at is null
    returning app.id
), deleted_access as (
    delete from app_access_tokens as token
     where token.connected_app_id in (select revoked.id from revoked)
), deleted_sends as (
    delete from sends as send
     where send.connected_app_id in (select revoked.id from revoked)
), deleted_library as (
    delete from app_library_entries as entry
     where entry.connected_app_id in (select revoked.id from revoked)
)
select exists(select 1 from revoked) as revoked;

-- name: RevokeConnectedAppByID :one
with revoked as (
    update connected_apps as app
       set refresh_token_hash = null,
           app_version = null,
           library_app_version = null,
           protocol_version = null,
           capabilities = '{}',
           accepted_formats = '{}',
           revoked_at = now()
     where app.id = sqlc.arg('connected_app_id') and app.revoked_at is null
    returning app.id
), deleted_access as (
    delete from app_access_tokens as token
     where token.connected_app_id in (select revoked.id from revoked)
), deleted_sends as (
    delete from sends as send
     where send.connected_app_id in (select revoked.id from revoked)
), deleted_library as (
    delete from app_library_entries as entry
     where entry.connected_app_id in (select revoked.id from revoked)
)
select exists(select 1 from revoked) as revoked;

-- name: UpdateDiscordProfile :exec
update users
   set display_name = $2, avatar_url = $3, banner_url = $4, updated_at = now()
 where id = $1;

-- name: LiveConnectedApps :many
select id, user_id, app_name, name, app_version,
       protocol_version, capabilities, accepted_formats,
       refresh_token_prefix, permissions, connected_at, last_seen_at
  from connected_apps
 where user_id = sqlc.arg('user_id') and revoked_at is null
 order by coalesce(last_seen_at, connected_at) desc, connected_at desc;

-- name: LiveConnectedApp :one
select id, user_id, app_name, name, app_version,
       protocol_version, capabilities, accepted_formats,
       refresh_token_prefix, permissions, connected_at, last_seen_at
  from connected_apps
 where id = sqlc.arg('connected_app_id')
   and user_id = sqlc.arg('user_id')
   and revoked_at is null;

-- name: QueueSend :one
insert into sends (id, connected_app_id, work_id, expires_at, updates_install)
select sqlc.arg('id'), sqlc.arg('connected_app_id'), sqlc.arg('work_id'),
       sqlc.arg('expires_at'),
       exists (
           select 1
             from app_library_entries as entry
            where entry.connected_app_id = sqlc.arg('connected_app_id')
              and entry.work_id = sqlc.arg('work_id')
       )
on conflict (connected_app_id, work_id) where state in ('queued', 'released') do nothing
returning id, connected_app_id, work_id, state, settled_reason, queued_at, settled_at,
          expires_at, updates_install;

-- name: LiveSendForWork :one
select id, connected_app_id, work_id, state, settled_reason, queued_at, settled_at,
       expires_at, updates_install
  from sends
 where connected_app_id = sqlc.arg('connected_app_id')
   and work_id = sqlc.arg('work_id')
   and state in ('queued', 'released');

-- name: CountLiveSends :one
select count(*)::bigint
  from sends
 where connected_app_id = sqlc.arg('connected_app_id') and state in ('queued', 'released');

-- name: AbandonExhaustedSends :execrows
update sends
   set state = 'failed', settled_at = now(), settled_reason = 'abandoned',
       lease_expires_at = null
 where connected_app_id = sqlc.arg('connected_app_id')
   and state = 'released'
   and lease_expires_at <= now()
   and attempts >= sqlc.arg('max_attempts');

-- name: ClaimSends :many
with candidates as (
    select waiting.id
      from sends as waiting
      join connected_apps as app
        on app.id = waiting.connected_app_id
       and app.revoked_at is null
       and app.permissions @> array['work:receive']
     where waiting.connected_app_id = sqlc.arg('connected_app_id')
       and waiting.expires_at > now()
       and waiting.attempts < sqlc.arg('max_attempts')
       and (waiting.state = 'queued'
            or (waiting.state = 'released' and waiting.lease_expires_at <= now()))
     order by waiting.queued_at, waiting.id
     limit sqlc.arg('batch_size')
     for update of waiting skip locked
)
update sends as send
   set state = 'released',
       attempts = send.attempts + 1,
       lease_expires_at = sqlc.arg('lease_expires_at')
  from candidates
 where send.id = candidates.id
returning send.id, send.work_id, send.queued_at,
          send.lease_expires_at;

-- name: SetSendFormat :exec
update sends
   set chosen_format = sqlc.arg('chosen_format')
 where id = sqlc.arg('id');

-- name: FailSend :exec
update sends
   set state = 'failed', settled_at = now(),
       settled_reason = sqlc.arg('settled_reason'), lease_expires_at = null
 where id = sqlc.arg('id');

-- name: AcknowledgeSends :execrows
update sends
   set state = 'delivered', settled_at = now(), lease_expires_at = null
 where connected_app_id = sqlc.arg('connected_app_id')
   and id = any(sqlc.arg('send_ids')::uuid[])
   and state = 'released';

-- name: DiscardSend :execrows
delete from sends as send
 using connected_apps as app
 where send.id = sqlc.arg('send_id')
   and send.connected_app_id = app.id
   and app.user_id = sqlc.arg('user_id');

-- name: SendForMainFile :one
select send.work_id, send.chosen_format, app.id as connected_app_id
  from sends as send
  join connected_apps as app on app.id = send.connected_app_id
 where send.id = sqlc.arg('send_id')
   and send.state = 'released'
   and send.lease_expires_at > now()
   and send.expires_at > now()
   and send.chosen_format is not null
   and app.revoked_at is null
   and app.permissions @> array['work:receive'];

-- name: DeleteExpiredSends :execrows
with expired as (
    select id
      from sends
     where expires_at <= now()
     order by expires_at
     limit sqlc.arg('batch_size')
     for update skip locked
)
delete from sends as send
 using expired
 where send.id = expired.id;

-- name: SendableWorkVersion :one
select version.number
  from works work
  join work_versions version on version.id = work.published_version_id
 where work.id = sqlc.arg('work_id')
   and deleted_at is null
   and taken_down_at is null
   and lifecycle = 'published';

-- name: WorkConnectedAppStates :many
select app.id, app.app_name, app.name,
       app.last_seen_at, app.permissions, app.capabilities,
       app.accepted_formats,
       send.id as send_id,
       coalesce(send.state, '')::text as send_state,
       send.settled_reason, send.queued_at, send.settled_at,
       send.expires_at,
       coalesce(send.updates_install, false)::boolean as updates_install,
       entry.version_number as installed_version
  from connected_apps as app
  left join lateral (
      select waiting.id, waiting.state, waiting.settled_reason,
             waiting.queued_at, waiting.settled_at, waiting.expires_at,
             waiting.updates_install
        from sends as waiting
       where waiting.connected_app_id = app.id
         and waiting.work_id = sqlc.arg('work_id')
       order by (waiting.state in ('queued', 'released')) desc,
                waiting.queued_at desc
       limit 1
  ) as send on true
  left join app_library_entries as entry
    on entry.connected_app_id = app.id and entry.work_id = sqlc.arg('work_id')
 where app.user_id = sqlc.arg('user_id') and app.revoked_at is null
 order by coalesce(app.last_seen_at, app.connected_at) desc,
          app.connected_at desc;

-- name: AppLibraryCounts :many
select entry.connected_app_id,
       count(*)::bigint as installed,
       count(*) filter (
           where version.number > entry.version_number
       )::bigint as updates_available
  from app_library_entries as entry
  join works as work on work.id = entry.work_id
  join work_versions as version on version.id = work.published_version_id
  join connected_apps as app on app.id = entry.connected_app_id
 where app.user_id = sqlc.arg('user_id')
   and app.revoked_at is null
   and work.deleted_at is null
   and work.taken_down_at is null
   and work.lifecycle = 'published'
 group by entry.connected_app_id;

-- name: ReportLibraryEntries :execrows
insert into app_library_entries
    (connected_app_id, work_id, version_number, reported_at)
select sqlc.arg('connected_app_id'), work.id,
       coalesce(nullif(reported.version_number, 0), version.number), now()
  from (
      select unnest(sqlc.arg('work_ids')::uuid[]) as work_id,
             unnest(sqlc.arg('version_numbers')::integer[]) as version_number
  ) as reported
  join works as work
    on work.id = reported.work_id
   and work.deleted_at is null
   and work.lifecycle = 'published'
  join work_versions as version on version.id = work.published_version_id
on conflict (connected_app_id, work_id) do update
   set version_number = excluded.version_number,
       reported_at = excluded.reported_at;

-- name: RemoveLibraryEntries :execrows
delete from app_library_entries
 where connected_app_id = sqlc.arg('connected_app_id')
   and work_id = any(sqlc.arg('work_ids')::uuid[]);

-- name: PruneLibraryToWhole :execrows
delete from app_library_entries
 where connected_app_id = sqlc.arg('connected_app_id')
   and not (work_id = any(sqlc.arg('work_ids')::uuid[]));

-- name: TakeTakedownNotices :many
update app_library_entries as entry
   set notified_taken_down_at = work.taken_down_at
  from work_public.works as work
 where entry.connected_app_id = sqlc.arg('connected_app_id')
   and work.id = entry.work_id
   and work.type = any(sqlc.arg('types')::text[])
   and work.taken_down_at is not null
   and work.deleted_at is null
   and entry.notified_taken_down_at is distinct from work.taken_down_at
returning entry.work_id, work.name::text as name, work.taken_down_at;

-- name: RecordLibraryAppVersion :exec
update connected_apps
   set library_app_version = nullif(sqlc.arg('app_version')::text, '')
 where id = sqlc.arg('connected_app_id');

-- name: InstalledAppVersions :many
select coalesce(app.library_app_version,
                app.app_version)::text as app_version
  from app_library_entries as entry
  join connected_apps as app
    on app.id = entry.connected_app_id
   and app.revoked_at is null
   and app.capabilities && sqlc.arg('capabilities')::text[]
 where entry.work_id = sqlc.arg('work_id')
   and coalesce(app.library_app_version, app.app_version) is not null
 group by coalesce(app.library_app_version, app.app_version)
having count(*) >= sqlc.arg('minimum_group_size')::bigint
 order by coalesce(app.library_app_version, app.app_version);
