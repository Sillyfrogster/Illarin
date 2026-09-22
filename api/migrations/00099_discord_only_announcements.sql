-- +goose Up
create table discord_webhooks (
    id         uuid primary key default gen_random_uuid(),
    owner_id   uuid unique references users (id) on delete cascade,
    address    bytea not null,
    updated_at timestamptz not null default now()
);

-- The blog's own channel is the one row without an owner
create unique index discord_webhooks_blog on discord_webhooks ((owner_id is null)) where owner_id is null;

create table discord_posts (
    id         uuid primary key default gen_random_uuid(),
    webhook_id uuid not null references discord_webhooks (id) on delete cascade,
    body       jsonb not null,
    tries      integer not null default 0 check (tries >= 0),
    due_at     timestamptz not null default now(),
    sent_at    timestamptz,
    failed_at  timestamptz,
    created_at timestamptz not null default now()
);

create index discord_posts_due on discord_posts (due_at) where sent_at is null and failed_at is null;

insert into discord_webhooks (owner_id, address)
select distinct on (owner_id) owner_id, address
  from work_integrations
 where type = 'discord' and state = 'active'
 order by owner_id, created_at;

insert into discord_webhooks (owner_id, address)
select null, address
  from blog_integrations
 where type = 'discord' and state = 'active'
 order by created_at
 limit 1;

alter table post_schedules add column post_to_discord boolean not null default false;

update post_schedules schedule
   set post_to_discord = true
 where exists (select 1 from post_schedule_integrations chosen where chosen.schedule_id = schedule.id);

drop table post_schedule_integrations;
drop table blog_announcement_tries;
drop table blog_announcement_attempts;
drop table blog_announcements;
drop table blog_integrations;
drop table work_announcement_tries;
drop table work_announcement_attempts;
drop table work_announcements;
drop table work_integration_defaults;
drop table work_integrations;

-- +goose Down
-- The old tables come back empty; the carried-over Discord addresses are not restored into them

CREATE TABLE blog_announcement_attempts (
    id uuid NOT NULL,
    announcement_id uuid NOT NULL,
    integration_id uuid,
    integration_name text NOT NULL,
    state text DEFAULT 'pending'::text NOT NULL,
    tries integer DEFAULT 0 NOT NULL,
    lease_token uuid,
    lease_expires_at timestamp with time zone,
    due_at timestamp with time zone DEFAULT now() NOT NULL,
    settled_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    run integer DEFAULT 1 NOT NULL,
    settled_reason text,
    mention_role boolean DEFAULT false NOT NULL,
    message_id text,
    CONSTRAINT blog_announcement_attempts_lease_check CHECK (((lease_token IS NULL) = (lease_expires_at IS NULL))),
    CONSTRAINT blog_announcement_attempts_name_check CHECK (((char_length(integration_name) >= 1) AND (char_length(integration_name) <= 48))),
    CONSTRAINT blog_announcement_attempts_reason_check CHECK (((settled_reason IS NULL) = (settled_at IS NULL))),
    CONSTRAINT blog_announcement_attempts_run_check CHECK ((run >= 1)),
    CONSTRAINT blog_announcement_attempts_settled_check CHECK (((state = ANY (ARRAY['pending'::text, 'sending'::text])) = (settled_at IS NULL))),
    CONSTRAINT blog_announcement_attempts_settled_reason_check CHECK (((settled_reason IS NULL) OR (settled_reason = ANY (ARRAY['arrived'::text, 'exhausted'::text, 'refused'::text, 'gone'::text, 'removed'::text, 'disabled'::text, 'moved'::text, 'unconfirmed'::text])))),
    CONSTRAINT blog_announcement_attempts_state_check CHECK ((state = ANY (ARRAY['pending'::text, 'sending'::text, 'delivered'::text, 'failed'::text, 'unconfirmed'::text]))),
    CONSTRAINT blog_announcement_attempts_tries_check CHECK ((tries >= 0))
);

CREATE TABLE blog_announcement_tries (
    id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    number integer NOT NULL,
    outcome text NOT NULL,
    status integer,
    detail text DEFAULT ''::text NOT NULL,
    took_ms integer DEFAULT 0 NOT NULL,
    attempted_at timestamp with time zone DEFAULT now() NOT NULL,
    run integer DEFAULT 1 NOT NULL,
    CONSTRAINT blog_announcement_tries_detail_check CHECK ((char_length(detail) <= 200)),
    CONSTRAINT blog_announcement_tries_number_check CHECK ((number >= 1)),
    CONSTRAINT blog_announcement_tries_outcome_check CHECK ((outcome = ANY (ARRAY['delivered'::text, 'refused'::text, 'unreachable'::text, 'unconfirmed'::text]))),
    CONSTRAINT blog_announcement_tries_run_check CHECK ((run >= 1))
);

CREATE TABLE blog_announcements (
    id uuid NOT NULL,
    post_id uuid NOT NULL,
    revision_id uuid NOT NULL,
    type text NOT NULL,
    occurred_at timestamp with time zone DEFAULT now() NOT NULL,
    note text DEFAULT ''::text NOT NULL,
    CONSTRAINT blog_announcements_note_check CHECK ((char_length(note) <= 500)),
    CONSTRAINT blog_announcements_type_check CHECK ((type = ANY (ARRAY['blog.post.published.v1'::text, 'blog.post.updated.v1'::text, 'blog.post.unpublished.v1'::text])))
);

CREATE TABLE blog_integrations (
    id uuid NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    host text NOT NULL,
    address bytea NOT NULL,
    signing_secret bytea,
    state text DEFAULT 'unverified'::text NOT NULL,
    verified_at timestamp with time zone,
    disabled_at timestamp with time zone,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    announcements text[] DEFAULT '{publication.post.published.v1}'::text[] NOT NULL,
    previous_secret bytea,
    previous_secret_until timestamp with time zone,
    signing_secret_set_at timestamp with time zone DEFAULT now() NOT NULL,
    guild_id text,
    channel_id text,
    webhook_name text,
    role_id text,
    role_name text,
    CONSTRAINT blog_integrations_announcements_check CHECK ((announcements <@ ARRAY['blog.post.published.v1'::text, 'blog.post.updated.v1'::text, 'blog.post.unpublished.v1'::text])),
    CONSTRAINT blog_integrations_announces_check CHECK (((type <> 'discord'::text) OR (announcements = ARRAY['blog.post.published.v1'::text]))),
    CONSTRAINT blog_integrations_channel_check CHECK (((type = 'discord'::text) = ((guild_id IS NOT NULL) AND (channel_id IS NOT NULL)))),
    CONSTRAINT blog_integrations_disabled_check CHECK (((state = 'disabled'::text) = (disabled_at IS NOT NULL))),
    CONSTRAINT blog_integrations_host_check CHECK (((char_length(host) >= 1) AND (char_length(host) <= 253))),
    CONSTRAINT blog_integrations_name_check CHECK (((char_length(name) >= 1) AND (char_length(name) <= 48))),
    CONSTRAINT blog_integrations_overlap_check CHECK (((previous_secret IS NULL) = (previous_secret_until IS NULL))),
    CONSTRAINT blog_integrations_role_check CHECK ((((role_id IS NULL) = (role_name IS NULL)) AND ((role_id IS NULL) OR (type = 'discord'::text)))),
    CONSTRAINT blog_integrations_role_name_check CHECK (((role_name IS NULL) OR ((char_length(role_name) >= 1) AND (char_length(role_name) <= 48)))),
    CONSTRAINT blog_integrations_secret_check CHECK (((type = 'webhook'::text) = (signing_secret IS NOT NULL))),
    CONSTRAINT blog_integrations_state_check CHECK ((state = ANY (ARRAY['unverified'::text, 'active'::text, 'disabled'::text]))),
    CONSTRAINT blog_integrations_type_check CHECK ((type = ANY (ARRAY['webhook'::text, 'discord'::text]))),
    CONSTRAINT blog_integrations_verified_check CHECK (((state <> 'active'::text) OR (verified_at IS NOT NULL)))
);

CREATE TABLE post_schedule_integrations (
    schedule_id uuid NOT NULL,
    integration_id uuid NOT NULL,
    mention_role boolean DEFAULT false NOT NULL
);

CREATE TABLE work_announcement_attempts (
    id uuid NOT NULL,
    announcement_id uuid NOT NULL,
    integration_id uuid,
    integration_name text NOT NULL,
    integration_type text NOT NULL,
    state text DEFAULT 'pending'::text NOT NULL,
    settled_reason text,
    message_id text,
    run integer DEFAULT 1 NOT NULL,
    tries integer DEFAULT 0 NOT NULL,
    lease_token uuid,
    lease_expires_at timestamp with time zone,
    due_at timestamp with time zone DEFAULT now() NOT NULL,
    settled_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT work_announcement_attempts_lease_check CHECK (((lease_token IS NULL) = (lease_expires_at IS NULL))),
    CONSTRAINT work_announcement_attempts_name_check CHECK (((char_length(integration_name) >= 1) AND (char_length(integration_name) <= 48))),
    CONSTRAINT work_announcement_attempts_reason_check CHECK (((settled_reason IS NULL) = (settled_at IS NULL))),
    CONSTRAINT work_announcement_attempts_run_check CHECK ((run >= 1)),
    CONSTRAINT work_announcement_attempts_settled_check CHECK (((state = ANY (ARRAY['pending'::text, 'sending'::text])) = (settled_at IS NULL))),
    CONSTRAINT work_announcement_attempts_settled_reason_check CHECK (((settled_reason IS NULL) OR (settled_reason = ANY (ARRAY['arrived'::text, 'exhausted'::text, 'refused'::text, 'gone'::text, 'removed'::text, 'disabled'::text, 'moved'::text, 'unconfirmed'::text, 'taken_down'::text, 'withdrawn'::text, 'unlisted'::text, 'deleted'::text])))),
    CONSTRAINT work_announcement_attempts_state_check CHECK ((state = ANY (ARRAY['pending'::text, 'sending'::text, 'delivered'::text, 'failed'::text, 'unconfirmed'::text]))),
    CONSTRAINT work_announcement_attempts_tries_check CHECK ((tries >= 0)),
    CONSTRAINT work_announcement_attempts_type_check CHECK ((integration_type = ANY (ARRAY['webhook'::text, 'discord'::text])))
);

CREATE TABLE work_announcement_tries (
    id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    run integer DEFAULT 1 NOT NULL,
    number integer NOT NULL,
    outcome text NOT NULL,
    status integer,
    detail text DEFAULT ''::text NOT NULL,
    took_ms integer DEFAULT 0 NOT NULL,
    attempted_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT work_announcement_tries_detail_check CHECK ((char_length(detail) <= 200)),
    CONSTRAINT work_announcement_tries_number_check CHECK ((number >= 1)),
    CONSTRAINT work_announcement_tries_outcome_check CHECK ((outcome = ANY (ARRAY['delivered'::text, 'refused'::text, 'unreachable'::text, 'unconfirmed'::text]))),
    CONSTRAINT work_announcement_tries_run_check CHECK ((run >= 1))
);

CREATE TABLE work_announcements (
    id uuid NOT NULL,
    work_id uuid NOT NULL,
    version_id uuid NOT NULL,
    type text NOT NULL,
    occurred_at timestamp with time zone DEFAULT now() NOT NULL,
    unlisted_consent boolean DEFAULT false NOT NULL,
    payload jsonb NOT NULL,
    CONSTRAINT work_announcements_type_check CHECK ((type = 'work.update.published.v1'::text))
);

CREATE TABLE work_integration_defaults (
    work_id uuid NOT NULL,
    integration_id uuid NOT NULL
);

CREATE TABLE work_integrations (
    id uuid NOT NULL,
    owner_id uuid NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    host text NOT NULL,
    address bytea NOT NULL,
    signing_secret bytea,
    signing_secret_set_at timestamp with time zone,
    previous_secret bytea,
    previous_secret_until timestamp with time zone,
    guild_id text,
    channel_id text,
    state text DEFAULT 'unverified'::text NOT NULL,
    verified_at timestamp with time zone,
    disabled_at timestamp with time zone,
    version bigint DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT work_integrations_check CHECK (((type = 'webhook'::text) = (signing_secret IS NOT NULL))),
    CONSTRAINT work_integrations_check1 CHECK (((type = 'webhook'::text) = (signing_secret_set_at IS NOT NULL))),
    CONSTRAINT work_integrations_check2 CHECK (((type = 'discord'::text) = ((guild_id IS NOT NULL) AND (channel_id IS NOT NULL)))),
    CONSTRAINT work_integrations_check3 CHECK (((previous_secret IS NULL) = (previous_secret_until IS NULL))),
    CONSTRAINT work_integrations_check4 CHECK (((state <> 'active'::text) OR (verified_at IS NOT NULL))),
    CONSTRAINT work_integrations_check5 CHECK (((state = 'disabled'::text) = (disabled_at IS NOT NULL))),
    CONSTRAINT work_integrations_name_check CHECK (((char_length(name) >= 1) AND (char_length(name) <= 48))),
    CONSTRAINT work_integrations_state_check CHECK ((state = ANY (ARRAY['unverified'::text, 'active'::text, 'disabled'::text]))),
    CONSTRAINT work_integrations_type_check CHECK ((type = ANY (ARRAY['webhook'::text, 'discord'::text])))
);

ALTER TABLE ONLY blog_announcement_attempts
    ADD CONSTRAINT blog_announcement_attempts_pkey PRIMARY KEY (id);

ALTER TABLE ONLY blog_announcement_tries
    ADD CONSTRAINT blog_announcement_tries_attempt_id_number_key UNIQUE (attempt_id, number);

ALTER TABLE ONLY blog_announcement_tries
    ADD CONSTRAINT blog_announcement_tries_pkey PRIMARY KEY (id);

ALTER TABLE ONLY blog_announcements
    ADD CONSTRAINT blog_announcements_pkey PRIMARY KEY (id);

ALTER TABLE ONLY blog_integrations
    ADD CONSTRAINT blog_integrations_pkey PRIMARY KEY (id);

ALTER TABLE ONLY post_schedule_integrations
    ADD CONSTRAINT post_schedule_integrations_pkey PRIMARY KEY (schedule_id, integration_id);

ALTER TABLE ONLY work_announcement_attempts
    ADD CONSTRAINT work_announcement_attempts_pkey PRIMARY KEY (id);

ALTER TABLE ONLY work_announcement_tries
    ADD CONSTRAINT work_announcement_tries_attempt_id_number_key UNIQUE (attempt_id, number);

ALTER TABLE ONLY work_announcement_tries
    ADD CONSTRAINT work_announcement_tries_pkey PRIMARY KEY (id);

ALTER TABLE ONLY work_announcements
    ADD CONSTRAINT work_announcements_pkey PRIMARY KEY (id);

ALTER TABLE ONLY work_announcements
    ADD CONSTRAINT work_announcements_version_key UNIQUE (version_id);

ALTER TABLE ONLY work_integration_defaults
    ADD CONSTRAINT work_integration_defaults_pkey PRIMARY KEY (work_id, integration_id);

ALTER TABLE ONLY work_integrations
    ADD CONSTRAINT work_integrations_pkey PRIMARY KEY (id);

CREATE INDEX blog_announcement_attempts_announcement_idx ON blog_announcement_attempts USING btree (announcement_id);

CREATE UNIQUE INDEX blog_announcement_attempts_choice_idx ON blog_announcement_attempts USING btree (announcement_id, integration_id) WHERE (integration_id IS NOT NULL);

CREATE INDEX blog_announcement_attempts_due_idx ON blog_announcement_attempts USING btree (due_at) WHERE (state = ANY (ARRAY['pending'::text, 'sending'::text]));

CREATE INDEX blog_announcement_attempts_settled_idx ON blog_announcement_attempts USING btree (settled_at DESC) WHERE (settled_at IS NOT NULL);

CREATE INDEX blog_announcement_tries_run_idx ON blog_announcement_tries USING btree (attempt_id, run);

CREATE INDEX blog_announcements_post_idx ON blog_announcements USING btree (post_id, occurred_at DESC);

CREATE INDEX blog_integrations_order_idx ON blog_integrations USING btree (name, created_at);

CREATE INDEX work_announcement_attempts_announcement_idx ON work_announcement_attempts USING btree (announcement_id);

CREATE UNIQUE INDEX work_announcement_attempts_choice_idx ON work_announcement_attempts USING btree (announcement_id, integration_id) WHERE (integration_id IS NOT NULL);

CREATE INDEX work_announcement_attempts_due_idx ON work_announcement_attempts USING btree (due_at) WHERE (state = ANY (ARRAY['pending'::text, 'sending'::text]));

CREATE INDEX work_announcement_attempts_integration_idx ON work_announcement_attempts USING btree (integration_id) WHERE (integration_id IS NOT NULL);

CREATE INDEX work_announcement_tries_run_idx ON work_announcement_tries USING btree (attempt_id, run);

CREATE INDEX work_announcements_work_idx ON work_announcements USING btree (work_id, occurred_at DESC);

CREATE INDEX work_integrations_owner_idx ON work_integrations USING btree (owner_id, name, id);

ALTER TABLE ONLY blog_announcement_attempts
    ADD CONSTRAINT blog_announcement_attempts_announcement_id_fkey FOREIGN KEY (announcement_id) REFERENCES blog_announcements(id) ON DELETE CASCADE;

ALTER TABLE ONLY blog_announcement_attempts
    ADD CONSTRAINT blog_announcement_attempts_integration_id_fkey FOREIGN KEY (integration_id) REFERENCES blog_integrations(id) ON DELETE SET NULL;

ALTER TABLE ONLY blog_announcement_tries
    ADD CONSTRAINT blog_announcement_tries_attempt_id_fkey FOREIGN KEY (attempt_id) REFERENCES blog_announcement_attempts(id) ON DELETE CASCADE;

ALTER TABLE ONLY blog_announcements
    ADD CONSTRAINT blog_announcements_post_id_fkey FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE;

ALTER TABLE ONLY blog_announcements
    ADD CONSTRAINT blog_announcements_revision_id_fkey FOREIGN KEY (revision_id) REFERENCES post_revisions(id);

ALTER TABLE ONLY blog_integrations
    ADD CONSTRAINT blog_integrations_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE ONLY post_schedule_integrations
    ADD CONSTRAINT post_schedule_integrations_integration_id_fkey FOREIGN KEY (integration_id) REFERENCES blog_integrations(id) ON DELETE CASCADE;

ALTER TABLE ONLY post_schedule_integrations
    ADD CONSTRAINT post_schedule_integrations_schedule_id_fkey FOREIGN KEY (schedule_id) REFERENCES post_schedules(id) ON DELETE CASCADE;

ALTER TABLE ONLY work_announcement_attempts
    ADD CONSTRAINT work_announcement_attempts_announcement_id_fkey FOREIGN KEY (announcement_id) REFERENCES work_announcements(id) ON DELETE CASCADE;

ALTER TABLE ONLY work_announcement_attempts
    ADD CONSTRAINT work_announcement_attempts_integration_id_fkey FOREIGN KEY (integration_id) REFERENCES work_integrations(id) ON DELETE SET NULL;

ALTER TABLE ONLY work_announcement_tries
    ADD CONSTRAINT work_announcement_tries_attempt_id_fkey FOREIGN KEY (attempt_id) REFERENCES work_announcement_attempts(id) ON DELETE CASCADE;

ALTER TABLE ONLY work_announcements
    ADD CONSTRAINT work_announcements_version_id_fkey FOREIGN KEY (version_id) REFERENCES work_versions(id) ON DELETE CASCADE;

ALTER TABLE ONLY work_announcements
    ADD CONSTRAINT work_announcements_work_id_fkey FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE;

ALTER TABLE ONLY work_integration_defaults
    ADD CONSTRAINT work_integration_defaults_integration_id_fkey FOREIGN KEY (integration_id) REFERENCES work_integrations(id) ON DELETE CASCADE;

ALTER TABLE ONLY work_integration_defaults
    ADD CONSTRAINT work_integration_defaults_work_id_fkey FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE;

ALTER TABLE ONLY work_integrations
    ADD CONSTRAINT work_integrations_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE;


alter table post_schedules drop column post_to_discord;
drop table discord_posts;
drop table discord_webhooks;
