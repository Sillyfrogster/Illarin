-- +goose Up
alter table notification_events drop constraint notification_events_type_check;
alter table notification_events add constraint notification_events_type_check
    check (type in ('asset_withheld', 'asset_restored', 'profile_restricted', 'profile_restored'));

alter table notifications drop constraint notifications_type_check;
alter table notifications add constraint notifications_type_check
    check (type in ('asset_withheld', 'asset_restored', 'profile_restricted', 'profile_restored'));

-- +goose Down
delete from notification_events where type in ('profile_restricted', 'profile_restored');
alter table notification_events drop constraint notification_events_type_check;
alter table notification_events add constraint notification_events_type_check
    check (type in ('asset_withheld', 'asset_restored'));

delete from notifications where type in ('profile_restricted', 'profile_restored');
alter table notifications drop constraint notifications_type_check;
alter table notifications add constraint notifications_type_check
    check (type in ('asset_withheld', 'asset_restored'));
