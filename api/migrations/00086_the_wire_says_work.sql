-- +goose Up

-- Notification types, the name kept with a notification, upload failures, the
-- announcement type and the follow state now say work, type and follow. A queued
-- announcement keeps its old keys beside the new ones, as a new one does.

alter table notification_events drop constraint notification_events_type_check;
alter table notifications drop constraint notifications_type_check;

update notification_events set type = 'work_withheld' where type = 'asset_withheld';
update notification_events set type = 'work_restored' where type = 'asset_restored';
update notification_events set type = 'work_updated' where type = 'asset_updated';
update notifications set type = 'work_withheld' where type = 'asset_withheld';
update notifications set type = 'work_restored' where type = 'asset_restored';
update notifications set type = 'work_updated' where type = 'asset_updated';

alter table notification_events add constraint notification_events_type_check
    check (type in ('work_withheld', 'work_restored', 'work_updated', 'profile_restricted', 'profile_restored'));
alter table notifications add constraint notifications_type_check
    check (type in ('work_withheld', 'work_restored', 'work_updated', 'profile_restricted', 'profile_restored'));

update notification_events
   set words = words - 'assetName' || jsonb_build_object('workName', words -> 'assetName')
 where words ? 'assetName';
update notifications
   set words = words - 'assetName' || jsonb_build_object('workName', words -> 'assetName')
 where words ? 'assetName';

drop index notifications_folding;
create unique index notifications_folding on notifications (account_id, work_id)
    where type = 'work_updated' and read_at is null;

alter table ingest_operations drop constraint ingest_operations_failure_reason_check;
update ingest_operations set failure_reason = 'wrong_type' where failure_reason = 'wrong_kind';
update ingest_operations set failure_reason = 'work_unavailable' where failure_reason = 'asset_unavailable';
alter table ingest_operations add constraint ingest_operations_failure_reason_check
check (failure_reason is null or failure_reason in (
    'malformed_input', 'unsupported_format', 'unsupported_version',
    'safety_violation', 'wrong_type', 'limit_exceeded', 'internal_failure',
    'working_copy_conflict', 'work_unavailable'
));

alter table work_update_events drop constraint work_update_events_type_check;
update work_update_events
   set type = 'work.update.published.v1',
       payload = payload || jsonb_build_object(
           'type', 'work.update.published.v1',
           'asset', (payload -> 'asset') || jsonb_build_object('type', payload -> 'asset' -> 'kind'),
           'work', (payload -> 'asset') || jsonb_build_object('type', payload -> 'asset' -> 'kind'))
 where type = 'asset.update.published.v1';
alter table work_update_events add constraint work_update_events_type_check
    check (type in ('work.update.published.v1'));

alter table work_follows drop constraint work_follows_state_check;
update work_follows set state = 'following' where state = 'watching';
alter table work_follows add constraint work_follows_state_check
    check (state in ('following', 'stopped'));

-- +goose Down

alter table work_follows drop constraint work_follows_state_check;
update work_follows set state = 'watching' where state = 'following';
alter table work_follows add constraint work_follows_state_check
    check (state in ('watching', 'stopped'));

alter table work_update_events drop constraint work_update_events_type_check;
update work_update_events
   set type = 'asset.update.published.v1',
       payload = (payload - 'work') || jsonb_build_object(
           'type', 'asset.update.published.v1',
           'asset', (payload -> 'asset') - 'type')
 where type = 'work.update.published.v1';
alter table work_update_events add constraint work_update_events_type_check
    check (type in ('asset.update.published.v1'));

drop index notifications_folding;
create unique index notifications_folding on notifications (account_id, work_id)
    where type = 'asset_updated' and read_at is null;

update notification_events
   set words = words - 'workName' || jsonb_build_object('assetName', words -> 'workName')
 where words ? 'workName';
update notifications
   set words = words - 'workName' || jsonb_build_object('assetName', words -> 'workName')
 where words ? 'workName';

alter table ingest_operations drop constraint ingest_operations_failure_reason_check;
update ingest_operations set failure_reason = 'wrong_kind' where failure_reason = 'wrong_type';
update ingest_operations set failure_reason = 'asset_unavailable' where failure_reason = 'work_unavailable';
alter table ingest_operations add constraint ingest_operations_failure_reason_check
check (failure_reason is null or failure_reason in (
    'malformed_input', 'unsupported_format', 'unsupported_version',
    'safety_violation', 'wrong_kind', 'limit_exceeded', 'internal_failure',
    'working_copy_conflict', 'asset_unavailable'
));

alter table notification_events drop constraint notification_events_type_check;
alter table notifications drop constraint notifications_type_check;

update notification_events set type = 'asset_withheld' where type = 'work_withheld';
update notification_events set type = 'asset_restored' where type = 'work_restored';
update notification_events set type = 'asset_updated' where type = 'work_updated';
update notifications set type = 'asset_withheld' where type = 'work_withheld';
update notifications set type = 'asset_restored' where type = 'work_restored';
update notifications set type = 'asset_updated' where type = 'work_updated';

alter table notification_events add constraint notification_events_type_check
    check (type in ('asset_withheld', 'asset_restored', 'profile_restricted', 'profile_restored', 'asset_updated'));
alter table notifications add constraint notifications_type_check
    check (type in ('asset_withheld', 'asset_restored', 'profile_restricted', 'profile_restored', 'asset_updated'));
