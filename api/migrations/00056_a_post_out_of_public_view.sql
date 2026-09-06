-- +goose Up
alter table posts drop constraint posts_status_check;
alter table posts add constraint posts_status_check
    check (status in ('draft', 'published', 'withdrawn'));

alter table posts drop constraint posts_published_check;
alter table posts add constraint posts_published_check
    check ((published_at is not null and slug is not null)
           = (status in ('published', 'withdrawn')));

alter table publication_events drop constraint publication_events_type_check;
alter table publication_events add constraint publication_events_type_check
    check (type in ('publication.post.published.v1', 'publication.post.updated.v1',
                    'publication.post.withdrawn.v1'));

alter table publication_audits drop constraint publication_audits_state_check;
alter table publication_audits add constraint publication_audits_state_check
    check (before_state is null or before_state in ('draft', 'published', 'withdrawn'));
alter table publication_audits drop constraint publication_audits_next_state_check;
alter table publication_audits add constraint publication_audits_next_state_check
    check (after_state is null or after_state in ('draft', 'published', 'withdrawn'));

create table post_withdrawals (
    id           uuid primary key,
    post_id      uuid not null references posts (id) on delete cascade,
    revision_id  uuid not null references post_revisions (id),
    reason       text not null,
    explanation  text not null default '',
    withdrawn_by uuid references users (id) on delete set null,
    withdrawn_at timestamptz not null default now(),
    constraint post_withdrawals_reason_check check (char_length(reason) between 1 and 500),
    constraint post_withdrawals_explanation_check check (char_length(explanation) <= 500)
);

create index post_withdrawals_post_idx on post_withdrawals (post_id, withdrawn_at desc);

-- +goose Down
drop table post_withdrawals;
alter table publication_audits drop constraint publication_audits_next_state_check;
alter table publication_audits add constraint publication_audits_next_state_check
    check (after_state is null or after_state in ('draft', 'published'));
alter table publication_audits drop constraint publication_audits_state_check;
alter table publication_audits add constraint publication_audits_state_check
    check (before_state is null or before_state in ('draft', 'published'));
update posts set status = 'published' where status = 'withdrawn';
delete from publication_events where type = 'publication.post.withdrawn.v1';
alter table publication_events drop constraint publication_events_type_check;
alter table publication_events add constraint publication_events_type_check
    check (type in ('publication.post.published.v1', 'publication.post.updated.v1'));
alter table posts drop constraint posts_published_check;
alter table posts add constraint posts_published_check
    check ((status = 'published') = (published_at is not null and slug is not null));
alter table posts drop constraint posts_status_check;
alter table posts add constraint posts_status_check check (status in ('draft', 'published'));
