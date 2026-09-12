-- +goose Up
create table publication_discord_repairs (
    id uuid primary key references publication_audits (id),
    actor_id uuid not null,
    delivery_id uuid not null,
    target_message_id text not null,
    fingerprint text not null,
    result jsonb not null,
    created_at timestamptz not null default now()
);

-- +goose Down
drop table publication_discord_repairs;
