-- +goose Up
drop trigger record_publish on works;
drop trigger record_uploaded_publish on works;
create trigger record_publish after insert on work_versions
    for each row when (not new.initial_recorded)
    execute function record_event('publish', 'work_id');

-- +goose Down
drop trigger record_publish on work_versions;
create trigger record_publish after update of lifecycle on works
    for each row when (old.lifecycle = 'draft' and new.lifecycle = 'published')
    execute function record_event('publish', 'id');
create trigger record_uploaded_publish after insert on works
    for each row when (new.lifecycle = 'published')
    execute function record_event('publish', 'id');
