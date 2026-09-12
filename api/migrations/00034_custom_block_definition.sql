-- +goose Up
update asset_blocks set definition = 'custom_block' where definition = 'custom_section';

-- +goose Down
update asset_blocks set definition = 'custom_section' where definition = 'custom_block';
