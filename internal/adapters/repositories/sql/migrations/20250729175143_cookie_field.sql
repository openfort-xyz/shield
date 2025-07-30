-- +goose Up
ALTER TABLE shld_custom_providers
ADD COLUMN cookie_field_name VARCHAR(64) DEFAULT NULL;


-- +goose Down
ALTER TABLE shld_custom_providers
DROP COLUMN IF EXISTS cookie_field_name;