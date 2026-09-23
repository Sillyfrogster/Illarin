-- +goose Up

-- An app id from the registry or 'any'; null means the reader has not said yet.
alter table users add column app_preference text;

-- Preferences picked on the sign-up page, applied to the account Discord creates.
alter table oauth_states add column app_preference text;
alter table oauth_states add column nsfw_preference text
    constraint oauth_states_nsfw_preference_check
    check (nsfw_preference in ('hidden', 'blurred', 'shown'));

-- +goose Down
alter table oauth_states drop column nsfw_preference;
alter table oauth_states drop column app_preference;
alter table users drop column app_preference;
