-- +goose Up
-- +goose StatementBegin
ALTER TABLE shld_projects
ADD COLUMN enable_2fa BOOL NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE shld_projects SET enable_2fa = false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE shld_projects DROP COLUMN IF EXISTS enable_2fa;
-- +goose StatementEnd
