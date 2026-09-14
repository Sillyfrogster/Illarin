-- +goose Up
alter table instance_library_entries
    add column notified_withheld_at timestamptz;

-- +goose Down
alter table instance_library_entries
    drop column notified_withheld_at;
