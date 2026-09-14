-- +goose Up
alter table instance_deliveries
    drop constraint instance_deliveries_state_check,
    drop constraint instance_deliveries_settled_check,
    add constraint instance_deliveries_state_check
        check (state in ('queued', 'released', 'delivered', 'failed')),
    add constraint instance_deliveries_settled_check
        check ((state in ('delivered', 'failed')) = (settled_at is not null)
           and (state = 'failed') = (settled_reason is not null)),
    add column updates_install boolean not null default false;

alter table instance_library_entries
    add column address text,
    add constraint instance_library_entries_address_check
        check (address is null
            or (address = btrim(address)
            and char_length(address) between 1 and 512));

-- +goose Down
delete from instance_deliveries where state = 'delivered';

alter table instance_library_entries
    drop constraint instance_library_entries_address_check,
    drop column address;

alter table instance_deliveries
    drop column updates_install,
    drop constraint instance_deliveries_settled_check,
    drop constraint instance_deliveries_state_check,
    add constraint instance_deliveries_state_check
        check (state in ('queued', 'released', 'failed')),
    add constraint instance_deliveries_settled_check
        check ((state = 'failed') = (settled_at is not null)
           and (state = 'failed') = (settled_reason is not null));
