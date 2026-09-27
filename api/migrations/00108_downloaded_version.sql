-- +goose Up
-- Older download records keep a null version rather than one guessed from their time.
alter table download_records
    add column version_number integer check (version_number > 0);

-- +goose Down
alter table download_records drop column version_number;
