-- +goose Up
alter table asset_snapshots
    add column notes_edited_at timestamptz,
    add column withdrawn_at timestamptz,
    add column withdrawal_explanation text,
    add constraint asset_snapshot_withdrawal_check check (
        (withdrawn_at is null and withdrawal_explanation is null)
        or (withdrawn_at is not null and char_length(btrim(withdrawal_explanation)) between 1 and 1000)
    );

-- +goose StatementBegin
create or replace function guard_asset_snapshot() returns trigger language plpgsql as $$
begin
    if tg_op = 'UPDATE'
        and (to_jsonb(new) - array['summary', 'notes', 'notes_edited_at', 'withdrawn_at', 'withdrawal_explanation'])
            = (to_jsonb(old) - array['summary', 'notes', 'notes_edited_at', 'withdrawn_at', 'withdrawal_explanation'])
        and (
            (
                new.withdrawn_at is not distinct from old.withdrawn_at
                and new.withdrawal_explanation is not distinct from old.withdrawal_explanation
                and new.notes_edited_at is not null
            )
            or (
                new.summary is not distinct from old.summary
                and new.notes is not distinct from old.notes
                and new.notes_edited_at is not distinct from old.notes_edited_at
                and old.withdrawn_at is null
                and new.withdrawn_at is not null
            )
        ) then
        return new;
    end if;
    if tg_op = 'DELETE' and not exists (
        select 1 from assets where id = old.asset_id
        and (deleted_at is null or recoverable_until > now())
    ) then
        return old;
    end if;
    raise exception 'Published asset snapshots are immutable';
end;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
create or replace function guard_asset_snapshot() returns trigger language plpgsql as $$
begin
    if tg_op = 'DELETE' and not exists (
        select 1 from assets where id = old.asset_id
        and (deleted_at is null or recoverable_until > now())
    ) then
        return old;
    end if;
    raise exception 'Published asset snapshots are immutable';
end;
$$;
-- +goose StatementEnd

alter table asset_snapshots drop constraint asset_snapshot_withdrawal_check;
alter table asset_snapshots
    drop column withdrawal_explanation,
    drop column withdrawn_at,
    drop column notes_edited_at;
