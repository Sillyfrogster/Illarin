-- +goose Up
alter table linked_instances
    add column library_application_version text,
    add constraint linked_instances_library_application_version_check
        check (library_application_version is null
            or (library_application_version = btrim(library_application_version)
            and char_length(library_application_version) between 1 and 64));

-- +goose Down
alter table linked_instances
    drop constraint linked_instances_library_application_version_check,
    drop column library_application_version;
