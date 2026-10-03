-- +goose Up

-- Whether the reader sees the site's artwork; it follows them across browsers.
alter table users add column artwork boolean not null default true;

-- +goose Down
alter table users drop column artwork;
