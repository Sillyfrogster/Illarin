-- +goose Up
alter table instance_library_entries
    drop constraint instance_library_entries_address_check,
    drop column address;

-- +goose Down
alter table instance_library_entries
    add column address text,
    add constraint instance_library_entries_address_check
        check (address is null
            or (address = btrim(address)
            and char_length(address) between 1 and 512));
