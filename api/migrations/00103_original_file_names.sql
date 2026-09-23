-- +goose Up
alter table work_original_files add column filename text;

update work_original_files original
   set filename = operation.filename
  from upload_operations operation
 where original.blob_id = operation.blob_id
   and original.work_id = coalesce(operation.work_id, operation.target_work_id);

-- +goose Down
alter table work_original_files drop column filename;
