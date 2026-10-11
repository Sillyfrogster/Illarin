-- +goose Up
create table profile_recommendations (
    user_id uuid not null references users (id) on delete cascade,
    work_id uuid not null references works (id) on delete cascade,
    created_at timestamptz not null default now(),
    primary key (user_id, work_id)
);

create index profile_recommendations_work_id_idx on profile_recommendations (work_id);

-- +goose Down
drop table profile_recommendations;
