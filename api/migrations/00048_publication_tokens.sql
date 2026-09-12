-- +goose Up
create table publication_tokens (
    id           uuid primary key,
    grant_id     uuid not null references publication_grants (id) on delete cascade,
    name         text not null,
    prefix       text not null unique,
    token_hash   bytea not null unique,
    created_at   timestamptz not null default now(),
    expires_at   timestamptz,
    last_used_at timestamptz,
    revoked_at   timestamptz,
    constraint publication_tokens_name_check check (char_length(name) between 1 and 48),
    constraint publication_tokens_prefix_check check (prefix ~ '^[BCDFGHJKLMNPQRSTVWXZ23456789]{8}$'),
    constraint publication_tokens_hash_check check (octet_length(token_hash) = 32),
    constraint publication_tokens_expiry_check check (expires_at is null or expires_at > created_at)
);

create index publication_tokens_grant_idx on publication_tokens (grant_id, created_at desc);

alter table publication_audits add column token_id uuid;

-- +goose Down
alter table publication_audits drop column token_id;
drop table publication_tokens;
