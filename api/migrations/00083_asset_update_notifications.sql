-- +goose Up
alter table notification_events alter column account_id drop not null;
alter table notification_events add constraint notification_events_hearer_check
    check (account_id is not null or asset_id is not null);

alter table notification_events drop constraint notification_events_type_check;
alter table notification_events add constraint notification_events_type_check
    check (type in ('asset_withheld', 'asset_restored', 'profile_restricted', 'profile_restored', 'asset_updated'));

alter table notifications drop constraint notifications_type_check;
alter table notifications add constraint notifications_type_check
    check (type in ('asset_withheld', 'asset_restored', 'profile_restricted', 'profile_restored', 'asset_updated'));

-- +goose Down
delete from notification_events where type = 'asset_updated';
alter table notification_events drop constraint notification_events_type_check;
alter table notification_events add constraint notification_events_type_check
    check (type in ('asset_withheld', 'asset_restored', 'profile_restricted', 'profile_restored'));
alter table notification_events drop constraint notification_events_hearer_check;
alter table notification_events alter column account_id set not null;

delete from notifications where type = 'asset_updated';
alter table notifications drop constraint notifications_type_check;
alter table notifications add constraint notifications_type_check
    check (type in ('asset_withheld', 'asset_restored', 'profile_restricted', 'profile_restored'));
