-- +goose Up
-- A browser authorization now carries the code its app shows, so the ones already open cannot be approved and are dropped.
delete from connection_authorizations;
alter table connection_authorizations add column user_code_hash bytea not null
    constraint connection_authorizations_user_code_hash_check check (octet_length(user_code_hash) = 32);

-- +goose Down
alter table connection_authorizations drop column user_code_hash;
