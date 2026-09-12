-- +goose Up
alter table publication_destinations
    add column events text[] not null default '{publication.post.published.v1}';

alter table publication_destinations add constraint publication_destinations_events_check
    check (events <@ array[
        'publication.post.published.v1',
        'publication.post.updated.v1',
        'publication.post.withdrawn.v1'
    ]::text[]);

alter table publication_destinations add column previous_secret bytea;
alter table publication_destinations add column previous_secret_until timestamptz;
alter table publication_destinations
    add column signing_secret_set_at timestamptz not null default now();

alter table publication_destinations add constraint publication_destinations_overlap_check
    check ((previous_secret is null) = (previous_secret_until is null));

alter table publication_deliveries add column run integer not null default 1;
alter table publication_deliveries add column settled_reason text;
alter table publication_delivery_attempts add column run integer not null default 1;

update publication_deliveries set settled_reason = case
    when state = 'delivered' then 'arrived' else 'refused' end
 where settled_at is not null;

alter table publication_deliveries add constraint publication_deliveries_run_check
    check (run >= 1);
alter table publication_deliveries add constraint publication_deliveries_reason_check
    check ((settled_reason is null) = (settled_at is null));
alter table publication_deliveries add constraint publication_deliveries_settled_reason_check
    check (settled_reason is null or settled_reason in (
        'arrived', 'exhausted', 'refused', 'gone', 'removed', 'disabled', 'moved'
    ));
alter table publication_delivery_attempts add constraint publication_delivery_attempts_run_check
    check (run >= 1);

create index publication_delivery_attempts_run_idx
    on publication_delivery_attempts (delivery_id, run);

create index publication_deliveries_settled_idx on publication_deliveries (settled_at desc)
    where settled_at is not null;

-- +goose Down
drop index publication_deliveries_settled_idx;
drop index publication_delivery_attempts_run_idx;
alter table publication_delivery_attempts drop constraint publication_delivery_attempts_run_check;
alter table publication_deliveries drop constraint publication_deliveries_settled_reason_check;
alter table publication_deliveries drop constraint publication_deliveries_reason_check;
alter table publication_deliveries drop constraint publication_deliveries_run_check;
alter table publication_delivery_attempts drop column run;
alter table publication_deliveries drop column settled_reason;
alter table publication_deliveries drop column run;
alter table publication_destinations drop constraint publication_destinations_overlap_check;
alter table publication_destinations drop column signing_secret_set_at;
alter table publication_destinations drop column previous_secret_until;
alter table publication_destinations drop column previous_secret;
alter table publication_destinations drop constraint publication_destinations_events_check;
alter table publication_destinations drop column events;
