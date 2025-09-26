-- +goose Up

ALTER TABLE shld_projects
ADD COLUMN enable_2fa BOOL NOT NULL DEFAULT false;
UPDATE shld_projects SET enable_2fa = false;
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down

    ALTER TABLE shld_projects DROP COLUMN IF EXISTS enable_2fa;
-- +goose StatementBegin
-- +goose StatementEnd
