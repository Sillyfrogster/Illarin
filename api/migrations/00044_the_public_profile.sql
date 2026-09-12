-- +goose Up
create table profile_media (
    id         uuid primary key,
    user_id    uuid not null references users (id) on delete cascade,
    blob_id    uuid references blobs (id),
    width      integer not null,
    height     integer not null,
    created_at timestamptz not null default now()
);

create index profile_media_user_id_idx on profile_media (user_id);
create index profile_media_blob_id_idx on profile_media (blob_id) where blob_id is not null;

create table public_profiles (
    user_id         uuid primary key references users (id) on delete cascade,
    display_name    text not null default '',
    biography       text not null default '',
    contact_email   text not null default '',
    avatar_media_id uuid references profile_media (id) on delete set null,
    created_at      timestamptz not null default now(),
    updated_at      timestamptz not null default now(),
    constraint public_profiles_display_name_check check (char_length(display_name) <= 48),
    constraint public_profiles_biography_check check (char_length(biography) <= 400),
    constraint public_profiles_contact_email_check check (
        contact_email = lower(contact_email) and char_length(contact_email) <= 254
    )
);

create table public_profile_links (
    user_id  uuid not null references public_profiles (user_id) on delete cascade,
    position integer not null,
    label    text not null,
    url      text not null,
    primary key (user_id, position),
    constraint public_profile_links_position_check check (position between 0 and 5),
    constraint public_profile_links_label_check check (char_length(label) between 1 and 32),
    constraint public_profile_links_url_check check (
        url like 'https://%' and char_length(url) <= 300
    )
);

-- +goose Down
drop table public_profile_links;
drop table public_profiles;
drop table profile_media;
