-- +goose Up
create trigger record_uploaded_publish after insert on works
    for each row when (new.lifecycle = 'published')
    execute function record_event('publish', 'id');

-- +goose Down
drop trigger record_uploaded_publish on works;
