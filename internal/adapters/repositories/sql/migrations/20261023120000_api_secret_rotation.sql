-- +goose Up
ALTER TABLE shld_projects ADD COLUMN pending_api_secret VARCHAR(255) DEFAULT NULL;
ALTER TABLE shld_projects ADD COLUMN pending_api_secret_expires_at TIMESTAMP DEFAULT NULL;
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
ALTER TABLE shld_projects DROP COLUMN IF EXISTS pending_api_secret_expires_at;
ALTER TABLE shld_projects DROP COLUMN IF EXISTS pending_api_secret;
-- +goose StatementBegin
-- +goose StatementEnd
