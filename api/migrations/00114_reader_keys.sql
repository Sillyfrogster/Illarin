-- +goose Up

-- A reader key lets a view count once per reader, work and UTC day without keeping who
-- read. Each day has its own random secret, and deleting a day's secret deletes its keys.
create table reader_secrets (
    day    date primary key,
    secret bytea not null
);

create table reader_keys (
    day date not null references reader_secrets (day) on delete cascade,
    key bytea not null,
    primary key (day, key)
);

-- +goose Down
drop table reader_keys;
drop table reader_secrets;
