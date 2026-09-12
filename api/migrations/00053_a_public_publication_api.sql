-- +goose Up
create table publication_idempotency (
    token_id     uuid not null references publication_tokens (id) on delete cascade,
    operation    text not null,
    key          text not null,
    fingerprint  bytea,
    status       integer,
    response     bytea,
    claimed_at   timestamptz not null default now(),
    completed_at timestamptz,
    primary key (token_id, operation, key),
    constraint publication_idempotency_key_check
        check (key = btrim(key) and char_length(key) between 8 and 200),
    constraint publication_idempotency_operation_check
        check (char_length(operation) between 1 and 120),
    constraint publication_idempotency_fingerprint_check
        check (fingerprint is null or octet_length(fingerprint) = 32),
    constraint publication_idempotency_outcome_check check (
        (completed_at is null and fingerprint is null and status is null and response is null)
        or (completed_at is not null and fingerprint is not null and status is not null
            and response is not null)
    )
);

create index publication_idempotency_claimed_idx on publication_idempotency (claimed_at);

create table publication_rate_limits (
    token_id     uuid not null references publication_tokens (id) on delete cascade,
    operation    text not null,
    attempts     integer not null,
    window_start timestamptz not null,
    primary key (token_id, operation),
    constraint publication_rate_limits_operation_check
        check (operation = btrim(operation) and char_length(operation) between 1 and 64),
    constraint publication_rate_limits_attempts_check check (attempts > 0)
);

create index publication_rate_limits_window_start_idx on publication_rate_limits (window_start);

-- +goose Down
drop table publication_rate_limits;
drop table publication_idempotency;
