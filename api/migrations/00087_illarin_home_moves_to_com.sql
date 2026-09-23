-- +goose Up
update publication_apps
set home_url = 'https://illarin.com'
where id = '9d3f1c00-0000-4000-8000-000000000001'
  and home_url = 'https://illarin.xyz';

-- +goose Down
update publication_apps
set home_url = 'https://illarin.xyz'
where id = '9d3f1c00-0000-4000-8000-000000000001'
  and home_url = 'https://illarin.com';
