-- +goose Up
ALTER TABLE shld_projects ADD COLUMN disabled_at TIMESTAMPTZ DEFAULT NULL;

-- +goose Down
ALTER TABLE shld_projects DROP COLUMN IF EXISTS disabled_at;
