-- +goose Up
-- +goose StatementBegin
ALTER TABLE shld_projects
ADD COLUMN 2fa_enabled BOOL NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE shld_projects SET 2fa_enabled = false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE shld_projects DROP COLUMN IF EXISTS 2fa_enabled;
-- +goose StatementEnd
