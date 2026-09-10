-- +goose Up
alter table publication_destinations drop constraint publication_destinations_kind_check;
alter table publication_destinations add constraint publication_destinations_kind_check
    check (kind in ('webhook', 'discord'));

alter table publication_destinations alter column signing_secret drop not null;
alter table publication_destinations add constraint publication_destinations_secret_check
    check ((kind = 'webhook') = (signing_secret is not null));

alter table publication_destinations add column guild_id text;
alter table publication_destinations add column channel_id text;
alter table publication_destinations add column webhook_name text;
alter table publication_destinations add column role_id text;
alter table publication_destinations add column role_name text;

alter table publication_destinations add constraint publication_destinations_channel_check
    check ((kind = 'discord') = (guild_id is not null and channel_id is not null));
alter table publication_destinations add constraint publication_destinations_role_check
    check ((role_id is null) = (role_name is null)
           and (role_id is null or kind = 'discord'));
alter table publication_destinations add constraint publication_destinations_role_name_check
    check (role_name is null or char_length(role_name) between 1 and 48);

alter table publication_destinations add constraint publication_destinations_announces_check
    check (kind <> 'discord'
           or events = array['publication.post.published.v1']::text[]);

alter table publication_deliveries add column mention_role boolean not null default false;
alter table publication_deliveries add column message_id text;

alter table publication_deliveries drop constraint publication_deliveries_state_check;
alter table publication_deliveries add constraint publication_deliveries_state_check
    check (state in ('pending', 'sending', 'delivered', 'failed', 'unconfirmed'));

alter table publication_deliveries drop constraint publication_deliveries_settled_reason_check;
alter table publication_deliveries add constraint publication_deliveries_settled_reason_check
    check (settled_reason is null or settled_reason in (
        'arrived', 'exhausted', 'refused', 'gone', 'removed', 'disabled', 'moved',
        'unconfirmed'
    ));

alter table publication_delivery_attempts
    drop constraint publication_delivery_attempts_outcome_check;
alter table publication_delivery_attempts
    add constraint publication_delivery_attempts_outcome_check
    check (outcome in ('delivered', 'refused', 'unreachable', 'unconfirmed'));

alter table post_schedule_destinations
    add column mention_role boolean not null default false;

-- +goose Down
alter table post_schedule_destinations drop column mention_role;
alter table publication_delivery_attempts
    drop constraint publication_delivery_attempts_outcome_check;
alter table publication_delivery_attempts
    add constraint publication_delivery_attempts_outcome_check
    check (outcome in ('delivered', 'refused', 'unreachable'));
alter table publication_deliveries drop constraint publication_deliveries_settled_reason_check;
alter table publication_deliveries add constraint publication_deliveries_settled_reason_check
    check (settled_reason is null or settled_reason in (
        'arrived', 'exhausted', 'refused', 'gone', 'removed', 'disabled', 'moved'
    ));
alter table publication_deliveries drop constraint publication_deliveries_state_check;
alter table publication_deliveries add constraint publication_deliveries_state_check
    check (state in ('pending', 'sending', 'delivered', 'failed'));
alter table publication_deliveries drop column message_id;
alter table publication_deliveries drop column mention_role;
alter table publication_destinations drop constraint publication_destinations_announces_check;
alter table publication_destinations drop constraint publication_destinations_role_name_check;
alter table publication_destinations drop constraint publication_destinations_role_check;
alter table publication_destinations drop constraint publication_destinations_channel_check;
alter table publication_destinations drop column role_name;
alter table publication_destinations drop column role_id;
alter table publication_destinations drop column webhook_name;
alter table publication_destinations drop column channel_id;
alter table publication_destinations drop column guild_id;
alter table publication_destinations drop constraint publication_destinations_secret_check;
alter table publication_destinations alter column signing_secret set not null;
alter table publication_destinations drop constraint publication_destinations_kind_check;
alter table publication_destinations add constraint publication_destinations_kind_check
    check (kind in ('webhook'));
