-- +goose Up
alter table ingest_operations drop constraint ingest_operations_result_check;
alter table ingest_operations drop constraint ingest_operations_status_check;
alter table ingest_operations
    add constraint ingest_operations_status_check
        check (status in ('pending', 'processing', 'preview', 'cancelled', 'failed', 'success')),
    add constraint ingest_operations_result_check
        check (
            (status = 'success' and asset_id is not null and failure_reason is null)
            or (status = 'failed' and asset_id is null and failure_reason is not null)
            or (status = 'preview' and asset_id is null and failure_reason is null and replacement_preview is not null)
            or (status = 'cancelled' and asset_id is null and failure_reason is null)
            or (status in ('pending', 'processing') and asset_id is null and failure_reason is null)
        );

-- +goose Down
alter table ingest_operations drop constraint ingest_operations_result_check;
alter table ingest_operations drop constraint ingest_operations_status_check;
alter table ingest_operations
    add constraint ingest_operations_status_check
        check (status in ('pending', 'processing', 'preview', 'failed', 'success')),
    add constraint ingest_operations_result_check
        check (
            (status = 'success' and asset_id is not null and failure_reason is null)
            or (status = 'failed' and asset_id is null and failure_reason is not null)
            or (status = 'preview' and asset_id is null and failure_reason is null and replacement_preview is not null)
            or (status in ('pending', 'processing') and asset_id is null and failure_reason is null)
        );
