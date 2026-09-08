-- +goose Up
create table asset_snapshot_prompt_matches (
    snapshot_id uuid not null references asset_snapshots (id) on delete cascade,
    current_fragment_id uuid not null,
    recorded_fragment_id uuid,
    resolved_at timestamptz not null default now(),
    primary key (snapshot_id, current_fragment_id)
);

-- +goose Down
drop table asset_snapshot_prompt_matches;
