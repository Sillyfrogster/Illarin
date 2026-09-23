-- +goose Up

-- A work staff removed is taken down, a profile staff hid is restricted, a
-- handed-over download is a download record with an access of public, owner
-- or app, and the job that deletes unreferenced blobs is cleanup. Publication
-- authority goes: the account that held it is an admin.

drop view work_public.works;
alter table works rename column withheld_at to taken_down_at;
alter table works rename column withheld_by to taken_down_by;
alter table works rename column withheld_reason to taken_down_reason;
alter table works rename constraint works_withheld_check to works_taken_down_check;
alter table works rename constraint works_withheld_by_fkey to works_taken_down_by_fkey;

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

-- +goose StatementBegin
create or replace function invalidate_work_candidate() returns trigger language plpgsql as $$
begin
    if row(new.owner_id, new.visibility, new.taken_down_at, new.deleted_at, new.lifecycle, new.published_version_id)
       is distinct from row(old.owner_id, old.visibility, old.taken_down_at, old.deleted_at, old.lifecycle, old.published_version_id) then
        new.drafted_changes_version := old.drafted_changes_version + 1;
    end if;
    return new;
end;
$$;
-- +goose StatementEnd

alter table app_library_entries rename column notified_withheld_at to notified_taken_down_at;

alter table notification_events drop constraint notification_events_type_check;
alter table notifications drop constraint notifications_type_check;
update notification_events set type = 'work_taken_down' where type = 'work_withheld';
update notifications set type = 'work_taken_down' where type = 'work_withheld';
alter table notification_events add constraint notification_events_type_check
    check (type in ('work_taken_down', 'work_restored', 'work_updated', 'profile_restricted', 'profile_restored'));
alter table notifications add constraint notifications_type_check
    check (type in ('work_taken_down', 'work_restored', 'work_updated', 'profile_restricted', 'profile_restored'));

alter table work_announcement_attempts drop constraint work_announcement_attempts_settled_reason_check;
update work_announcement_attempts set settled_reason = 'taken_down' where settled_reason = 'withheld';
alter table work_announcement_attempts add constraint work_announcement_attempts_settled_reason_check
    check (settled_reason is null or settled_reason in (
        'arrived', 'exhausted', 'refused', 'gone', 'removed', 'disabled', 'moved',
        'unconfirmed', 'taken_down', 'withdrawn', 'unlisted', 'deleted'));

alter table profile_restrictions rename to restricted_profiles;
alter table restricted_profiles rename constraint profile_restrictions_pkey to restricted_profiles_pkey;
alter table restricted_profiles rename constraint profile_restrictions_reason_check to restricted_profiles_reason_check;
alter table restricted_profiles rename constraint profile_restrictions_restricted_by_fkey to restricted_profiles_restricted_by_fkey;
alter table restricted_profiles rename constraint profile_restrictions_user_id_fkey to restricted_profiles_user_id_fkey;
alter table profile_restriction_audits rename to restricted_profile_audits;
alter table restricted_profile_audits rename constraint profile_restriction_audits_pkey to restricted_profile_audits_pkey;
alter table restricted_profile_audits rename constraint profile_restriction_audits_action_check to restricted_profile_audits_action_check;
alter table restricted_profile_audits rename constraint profile_restriction_audits_actor_id_fkey to restricted_profile_audits_actor_id_fkey;
alter index profile_restriction_audits_recorded_at_idx rename to restricted_profile_audits_recorded_at_idx;

alter table download_events disable trigger download_events_are_immutable;
alter table download_events drop constraint download_events_authorization_class_check;
alter table download_events rename column authorization_class to access;
update download_events set access = 'public' where access in ('anonymous', 'signed_in');
update download_events set access = 'app' where access = 'linked_instance';
alter table download_events add constraint download_records_access_check
    check (access in ('public', 'owner', 'app'));
alter table download_events enable trigger download_events_are_immutable;
alter table download_events rename to download_records;
alter table download_records rename constraint download_events_pkey to download_records_pkey;
alter table download_records rename constraint download_events_format_check to download_records_format_check;
alter table download_records rename constraint download_events_visibility_check to download_records_visibility_check;
alter table download_records rename constraint download_events_original_file_fk to download_records_original_file_fk;
alter trigger download_events_are_immutable on download_records rename to download_records_are_immutable;
alter trigger download_events_cannot_be_truncated on download_records rename to download_records_cannot_be_truncated;
alter function reject_download_event_mutation() rename to reject_download_record_mutation;
-- +goose StatementBegin
create or replace function reject_download_record_mutation() returns trigger
language plpgsql as $$
begin
    raise exception 'download records are append-only';
end;
$$;
-- +goose StatementEnd

alter table blob_sweep_marks rename to blob_cleanup_marks;
alter table blob_cleanup_marks rename constraint blob_sweep_marks_pkey to blob_cleanup_marks_pkey;
alter table blob_cleanup_marks rename constraint blob_sweep_marks_blob_id_fkey to blob_cleanup_marks_blob_id_fkey;

update users set role = 'admin' where id in (select user_id from publication_authorities);
drop table publication_authorities;

-- +goose Down

create table publication_authorities (
    user_id     uuid primary key references users (id) on delete cascade,
    assigned_at timestamptz not null default now()
);
insert into publication_authorities (user_id) select id from users where role = 'admin';

alter table blob_cleanup_marks rename constraint blob_cleanup_marks_blob_id_fkey to blob_sweep_marks_blob_id_fkey;
alter table blob_cleanup_marks rename constraint blob_cleanup_marks_pkey to blob_sweep_marks_pkey;
alter table blob_cleanup_marks rename to blob_sweep_marks;

-- +goose StatementBegin
create or replace function reject_download_record_mutation() returns trigger
language plpgsql as $$
begin
    raise exception 'download events are append-only';
end;
$$;
-- +goose StatementEnd
alter function reject_download_record_mutation() rename to reject_download_event_mutation;
alter trigger download_records_cannot_be_truncated on download_records rename to download_events_cannot_be_truncated;
alter trigger download_records_are_immutable on download_records rename to download_events_are_immutable;
alter table download_records rename constraint download_records_original_file_fk to download_events_original_file_fk;
alter table download_records rename constraint download_records_visibility_check to download_events_visibility_check;
alter table download_records rename constraint download_records_format_check to download_events_format_check;
alter table download_records rename constraint download_records_pkey to download_events_pkey;
alter table download_records rename to download_events;
alter table download_events disable trigger download_events_are_immutable;
alter table download_events drop constraint download_records_access_check;
update download_events set access = 'anonymous' where access = 'public';
update download_events set access = 'linked_instance' where access = 'app';
alter table download_events rename column access to authorization_class;
alter table download_events add constraint download_events_authorization_class_check
    check (authorization_class in ('anonymous', 'signed_in', 'owner', 'linked_instance'));
alter table download_events enable trigger download_events_are_immutable;

alter index restricted_profile_audits_recorded_at_idx rename to profile_restriction_audits_recorded_at_idx;
alter table restricted_profile_audits rename constraint restricted_profile_audits_actor_id_fkey to profile_restriction_audits_actor_id_fkey;
alter table restricted_profile_audits rename constraint restricted_profile_audits_action_check to profile_restriction_audits_action_check;
alter table restricted_profile_audits rename constraint restricted_profile_audits_pkey to profile_restriction_audits_pkey;
alter table restricted_profile_audits rename to profile_restriction_audits;
alter table restricted_profiles rename constraint restricted_profiles_user_id_fkey to profile_restrictions_user_id_fkey;
alter table restricted_profiles rename constraint restricted_profiles_restricted_by_fkey to profile_restrictions_restricted_by_fkey;
alter table restricted_profiles rename constraint restricted_profiles_reason_check to profile_restrictions_reason_check;
alter table restricted_profiles rename constraint restricted_profiles_pkey to profile_restrictions_pkey;
alter table restricted_profiles rename to profile_restrictions;

alter table work_announcement_attempts drop constraint work_announcement_attempts_settled_reason_check;
update work_announcement_attempts set settled_reason = 'withheld' where settled_reason = 'taken_down';
alter table work_announcement_attempts add constraint work_announcement_attempts_settled_reason_check
    check (settled_reason is null or settled_reason in (
        'arrived', 'exhausted', 'refused', 'gone', 'removed', 'disabled', 'moved',
        'unconfirmed', 'withheld', 'withdrawn', 'unlisted', 'deleted'));

alter table notification_events drop constraint notification_events_type_check;
alter table notifications drop constraint notifications_type_check;
update notification_events set type = 'work_withheld' where type = 'work_taken_down';
update notifications set type = 'work_withheld' where type = 'work_taken_down';
alter table notification_events add constraint notification_events_type_check
    check (type in ('work_withheld', 'work_restored', 'work_updated', 'profile_restricted', 'profile_restored'));
alter table notifications add constraint notifications_type_check
    check (type in ('work_withheld', 'work_restored', 'work_updated', 'profile_restricted', 'profile_restored'));

alter table app_library_entries rename column notified_taken_down_at to notified_withheld_at;

drop view work_public.works;
alter table works rename constraint works_taken_down_by_fkey to works_withheld_by_fkey;
alter table works rename constraint works_taken_down_check to works_withheld_check;
alter table works rename column taken_down_reason to withheld_reason;
alter table works rename column taken_down_by to withheld_by;
alter table works rename column taken_down_at to withheld_at;

-- +goose StatementBegin
create or replace function invalidate_work_candidate() returns trigger language plpgsql as $$
begin
    if row(new.owner_id, new.visibility, new.withheld_at, new.deleted_at, new.lifecycle, new.published_version_id)
       is distinct from row(old.owner_id, old.visibility, old.withheld_at, old.deleted_at, old.lifecycle, old.published_version_id) then
        new.drafted_changes_version := old.drafted_changes_version + 1;
    end if;
    return new;
end;
$$;
-- +goose StatementEnd

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
